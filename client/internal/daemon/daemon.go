// Package daemon manages the taildog background process and relay connection.
package daemon

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kiiimatz/taildog/client/internal/config"
)

const (
	clientVersion = "0.1.0"
	pingPeriod    = 30 * time.Second
)

// pidPath returns the path to the PID file.
func pidPath() string {
	return filepath.Join(config.ConfigDir(), "taildog.pid")
}

// LogPath returns the path to the daemon log file.
func LogPath() string {
	return filepath.Join(config.ConfigDir(), "taildog.log")
}

// writePID writes the current process PID to the PID file.
func writePID() error {
	if err := os.MkdirAll(config.ConfigDir(), 0o700); err != nil {
		return err
	}
	return os.WriteFile(pidPath(), []byte(strconv.Itoa(os.Getpid())), 0o600)
}

// removePID deletes the PID file.
func removePID() {
	_ = os.Remove(pidPath())
}

// readPIDFile parses the PID file and returns the pid it contains.
// Returns (0, nil) if the file does not exist.
func readPIDFile() (int, error) {
	data, err := os.ReadFile(pidPath())
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("reading pid file: %w", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parsing pid file: %w", err)
	}
	return pid, nil
}

// Start launches the daemon in the background and writes the PID file.
// It re-executes the current binary with --foreground so the parent returns
// immediately.
func Start() error {
	alive, pid, err := IsRunning()
	if err != nil {
		return err
	}
	if alive {
		return fmt.Errorf("daemon is already running (PID: %d)", pid)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	// Open (or create) the log file for the child's stdout/stderr.
	logPath := LogPath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		return err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}

	proc, err := os.StartProcess(exe, []string{exe, "up", "--foreground"}, &os.ProcAttr{
		Files: []*os.File{os.Stdin, logFile, logFile},
		Sys:   procAttrs(),
	})
	if err != nil {
		logFile.Close()
		return fmt.Errorf("starting daemon process: %w", err)
	}
	logFile.Close()

	// Give the child a moment to write its PID file.
	time.Sleep(200 * time.Millisecond)

	// Save PID before Release() zeroes it out.
	startedPID := proc.Pid

	// Detach – we don't wait for the child.
	_ = proc.Release()

	fmt.Printf("taildog daemon started (PID: %d)\n", startedPID)
	return nil
}

// RunForeground is the entry-point used when --foreground is passed.
// It writes the PID file, configures logging, and enters the connect loop.
func RunForeground() error {
	logPath := LogPath()
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		return err
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer logFile.Close()

	log.SetOutput(io.MultiWriter(os.Stderr, logFile))
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[taildog] ")

	if err := writePID(); err != nil {
		return fmt.Errorf("writing PID file: %w", err)
	}
	defer removePID()

	log.Printf("daemon started (PID: %d)", os.Getpid())

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	return connectWithBackoff(cfg)
}

// ---------------------------------------------------------------------------
// Relay connection
// ---------------------------------------------------------------------------

type helloMsg struct {
	Type     string `json:"type"`
	ClientID string `json:"clientID"`
	Name     string `json:"name"`
	Version  string `json:"version"`
}

type registerMsg struct {
	Type       string `json:"type"`
	TunnelID   string `json:"tunnelID"`
	Protocol   string `json:"protocol"`
	LocalPort  int    `json:"localPort"`
	RemotePort int    `json:"remotePort"`
}

type pingMsg struct {
	Type string `json:"type"`
}

// Connect performs the full relay handshake and then blocks, maintaining the
// connection until it drops.
func Connect(cfg *config.Config) error {
	if cfg.RelayHost == "" {
		return fmt.Errorf("relay_host is not configured")
	}

	addr := net.JoinHostPort(cfg.RelayHost, strconv.Itoa(cfg.RelayPort))
	tlsCfg, err := buildTLSConfig(cfg.CACertPath)
	if err != nil {
		return fmt.Errorf("building TLS config: %w", err)
	}
	tlsCfg.ServerName = cfg.RelayHost

	log.Printf("connecting to relay %s", addr)
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("dialing relay: %w", err)
	}
	defer conn.Close()
	log.Printf("connected to relay %s", addr)

	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)

	// Step 3: hello
	hostname, _ := os.Hostname()
	if err := enc.Encode(helloMsg{
		Type:     "hello",
		ClientID: clientIDFromConfig(),
		Name:     hostname,
		Version:  clientVersion,
	}); err != nil {
		return fmt.Errorf("sending hello: %w", err)
	}

	// Step 4: welcome
	var welcome map[string]interface{}
	if err := dec.Decode(&welcome); err != nil {
		return fmt.Errorf("reading welcome: %w", err)
	}
	log.Printf("welcome received: %v", welcome)

	// Step 5: register tunnels
	for _, t := range cfg.Tunnels {
		if err := enc.Encode(registerMsg{
			Type:       "register",
			TunnelID:   t.ID,
			Protocol:   t.Protocol,
			LocalPort:  t.LocalPort,
			RemotePort: t.RemotePort,
		}); err != nil {
			return fmt.Errorf("registering tunnel %s: %w", t.ID, err)
		}
		log.Printf("registered tunnel %s (%s) local:%d remote:%d",
			t.ID, t.Protocol, t.LocalPort, t.RemotePort)
	}

	// Step 6: ping loop
	pingErrCh := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for range ticker.C {
			if err := enc.Encode(pingMsg{Type: "ping"}); err != nil {
				pingErrCh <- err
				return
			}
		}
	}()

	// Step 7 (stub): inbound frame reader
	frameCh := make(chan error, 1)
	go func() {
		for {
			var frame map[string]interface{}
			if err := dec.Decode(&frame); err != nil {
				frameCh <- err
				return
			}
			frameType, _ := frame["type"].(string)
			switch frameType {
			case "pong":
				// expected keep-alive reply, ignore
			case "data":
				// TODO: proxy frame payload to the local port.
				log.Printf("data frame for tunnel %v (TCP proxy not yet implemented)",
					frame["tunnelID"])
			default:
				log.Printf("unhandled frame type: %q", frameType)
			}
		}
	}()

	select {
	case err := <-pingErrCh:
		return fmt.Errorf("ping failed: %w", err)
	case err := <-frameCh:
		if err == io.EOF {
			return fmt.Errorf("relay closed connection")
		}
		return fmt.Errorf("reading frame: %w", err)
	}
}

// connectWithBackoff retries Connect with exponential backoff, capped at 60s.
func connectWithBackoff(cfg *config.Config) error {
	const maxBackoff = 60 * time.Second
	attempt := 0
	for {
		if err := Connect(cfg); err != nil {
			log.Printf("connection lost: %v", err)
		}
		attempt++
		backoff := time.Duration(math.Min(
			float64(maxBackoff),
			float64(time.Second)*math.Pow(2, float64(attempt)),
		))
		log.Printf("reconnecting in %s (attempt %d)", backoff, attempt)
		time.Sleep(backoff)

		// Hot-reload config before each reconnect.
		if newCfg, err := config.Load(); err != nil {
			log.Printf("reloading config: %v", err)
		} else {
			cfg = newCfg
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildTLSConfig returns a TLS config.
// If caCertPath is set, the given CA bundle is trusted.
// Otherwise, InsecureSkipVerify is used to accept the relay's self-signed cert.
func buildTLSConfig(caCertPath string) (*tls.Config, error) {
	if caCertPath == "" {
		// Self-hosted relays use self-signed certificates; skip verification.
		// The control channel is still encrypted — only cert authenticity is
		// not checked. Use --ca-cert for pinned verification in production.
		return &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true, //nolint:gosec
		}, nil
	}
	caPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, fmt.Errorf("reading CA cert %s: %w", caCertPath, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("no valid certificates found in %s", caCertPath)
	}
	return &tls.Config{
		RootCAs:    pool,
		MinVersion: tls.VersionTLS12,
	}, nil
}

// clientIDFromConfig returns a stable UUID for this client, generating and
// persisting one on first run.
func clientIDFromConfig() string {
	idPath := filepath.Join(config.ConfigDir(), "client.id")
	if data, err := os.ReadFile(idPath); err == nil {
		if id := strings.TrimSpace(string(data)); id != "" {
			return id
		}
	}
	id := uuid.NewString()
	_ = os.WriteFile(idPath, []byte(id), 0o600)
	return id
}

// TailLog writes the last n lines of the daemon log to w, then follows new
// writes indefinitely (like tail -f).
func TailLog(w io.Writer, n int) error {
	logPath := LogPath()
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(w, "(log file does not exist yet — daemon has not run)")
			return nil
		}
		return err
	}
	defer f.Close()

	// Collect all lines then print the last n.
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	start := 0
	if len(lines) > n {
		start = len(lines) - n
	}
	for _, l := range lines[start:] {
		fmt.Fprintln(w, l)
	}

	// Follow new content.
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			fmt.Fprint(w, line)
		}
		if err == io.EOF {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if err != nil {
			return err
		}
	}
}
