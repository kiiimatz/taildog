// Package daemon manages the renode background process and relay connection.
package daemon

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
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
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kiiimatz/renode/client/internal/config"
)

const (
	clientVersion = "0.1.0"
	pingPeriod    = 30 * time.Second
)

// pidPath returns the path to the PID file.
func pidPath() string {
	return filepath.Join(config.ConfigDir(), "renode.pid")
}

// LogPath returns the path to the daemon log file.
func LogPath() string {
	return filepath.Join(config.ConfigDir(), "renode.log")
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

	// Use /dev/null (NUL on Windows) for stdin so that the detached daemon
	// does not inherit the parent's console input handle.
	nullFile, err := os.Open(os.DevNull)
	if err != nil {
		nullFile = os.Stdin // fallback
	}

	proc, err := os.StartProcess(exe, []string{exe, "up", "--foreground"}, &os.ProcAttr{
		Files: []*os.File{nullFile, logFile, logFile},
		Sys:   procAttrs(),
	})
	nullFile.Close()
	logFile.Close()
	if err != nil {
		return fmt.Errorf("starting daemon process: %w", err)
	}

	// On Linux, prime the sudo password cache so the detached daemon can
	// run ufw commands without prompting (sudo caches credentials for ~15 min).
	ufwPrimeSudo()

	// Give the child a moment to write its PID file.
	time.Sleep(300 * time.Millisecond)

	// Save PID before Release() zeroes it out.
	startedPID := proc.Pid

	// Detach – we don't wait for the child.
	_ = proc.Release()

	// Verify the daemon actually started and wrote its PID file.
	running, _, _ := IsRunning()
	if !running {
		return fmt.Errorf("daemon process (PID: %d) exited immediately — check logs: %s", startedPID, logPath)
	}

	fmt.Printf("renode daemon started (PID: %d)\n", startedPID)
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

	// Write logs to the file only.  The parent already set fd-1/fd-2 of this
	// child process to logFile, so os.Stderr IS logFile; using MultiWriter
	// would write every line twice to the same physical file.
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[renode] ")

	if err := writePID(); err != nil {
		return fmt.Errorf("writing PID file: %w", err)
	}
	defer removePID()

	log.Printf("daemon started (PID: %d)", os.Getpid())

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// UFW: open local service ports for any pre-configured tunnels (Linux only).
	for _, t := range cfg.Tunnels {
		ufwAllow(t.LocalPort, t.Protocol)
	}
	defer func() {
		for _, t := range cfg.Tunnels {
			ufwDelete(t.LocalPort, t.Protocol)
		}
	}()

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
	// Only set ServerName (SNI) when actually verifying the certificate.
	// With InsecureSkipVerify (no ca_cert_path), setting ServerName to an IP
	// address can trigger unexpected TLS handshake behavior.
	if cfg.CACertPath != "" {
		tlsCfg.ServerName = cfg.RelayHost
	}

	log.Printf("connecting to relay %s", addr)
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("dialing relay: %w", err)
	}
	defer conn.Close()
	log.Printf("connected to relay %s", addr)

	// Mutex-protected encoder — shared by ping goroutine and tunnel conn readers.
	var encMu sync.Mutex
	enc := json.NewEncoder(conn)
	safeEncode := func(v interface{}) error {
		encMu.Lock()
		err := enc.Encode(v)
		encMu.Unlock()
		return err
	}

	dec := json.NewDecoder(conn)

	// Hello
	hostname, _ := os.Hostname()
	if err := safeEncode(helloMsg{
		Type:     "hello",
		ClientID: clientIDFromConfig(),
		Name:     hostname,
		Version:  clientVersion,
	}); err != nil {
		return fmt.Errorf("sending hello: %w", err)
	}

	// Welcome
	var welcome map[string]interface{}
	if err := dec.Decode(&welcome); err != nil {
		return fmt.Errorf("reading welcome: %w", err)
	}
	log.Printf("welcome received: %v", welcome)

	// Register tunnels
	for _, t := range cfg.Tunnels {
		if err := safeEncode(registerMsg{
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

	// Active local connections: connID → net.Conn
	var localMu sync.Mutex
	localConns := make(map[string]net.Conn)

	closeLocal := func(connID string, sendFrame bool) {
		localMu.Lock()
		c, ok := localConns[connID]
		delete(localConns, connID)
		localMu.Unlock()
		if ok {
			c.Close()
		}
		if sendFrame {
			safeEncode(map[string]string{"type": "close", "connID": connID}) //nolint:errcheck
		}
	}

	// Ping loop
	pingErrCh := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for range ticker.C {
			if err := safeEncode(pingMsg{Type: "ping"}); err != nil {
				pingErrCh <- err
				return
			}
		}
	}()

	// Frame reader
	frameCh := make(chan error, 1)
	go func() {
		for {
			var raw map[string]json.RawMessage
			if err := dec.Decode(&raw); err != nil {
				frameCh <- err
				return
			}
			var frameType string
			if err := json.Unmarshal(raw["type"], &frameType); err != nil {
				continue
			}

			switch frameType {
			case "pong":
				// expected keep-alive reply

			case "open":
				var tunnelID, connID string
				var localPort int
				json.Unmarshal(raw["tunnelID"], &tunnelID)   //nolint:errcheck
				json.Unmarshal(raw["connID"], &connID)       //nolint:errcheck
				json.Unmarshal(raw["localPort"], &localPort) //nolint:errcheck
				log.Printf("open frame: tunnelID=%s connID=%s localPort=%d", tunnelID, connID, localPort)

				// Fall back to config lookup if relay didn't send localPort.
				var localProto string
				if localPort == 0 {
					for _, t := range cfg.Tunnels {
						if t.ID == tunnelID {
							localPort = t.LocalPort
							localProto = t.Protocol
							break
						}
					}
				} else {
					for _, t := range cfg.Tunnels {
						if t.ID == tunnelID {
							localProto = t.Protocol
							break
						}
					}
				}
				if localPort == 0 {
					log.Printf("open: localPort unknown for tunnel %s — sending close", tunnelID)
					safeEncode(map[string]string{"type": "close", "connID": connID}) //nolint:errcheck
					continue
				}

				dialProto := "tcp"
				if localProto == "udp" {
					dialProto = "udp"
				}
				localConn, err := net.Dial(dialProto, fmt.Sprintf("localhost:%d", localPort))
				if err != nil {
					log.Printf("open: dial localhost:%d failed: %v", localPort, err)
					safeEncode(map[string]string{"type": "close", "connID": connID}) //nolint:errcheck
					continue
				}

				localMu.Lock()
				localConns[connID] = localConn
				localMu.Unlock()
				log.Printf("open: connected localhost:%d connID=%s", localPort, connID)

				// Read from local connection, send data frames to relay.
				go func(connID string, localConn net.Conn) {
					defer closeLocal(connID, true)
					buf := make([]byte, 32*1024)
					for {
						n, err := localConn.Read(buf)
						if n > 0 {
							payload := base64.StdEncoding.EncodeToString(buf[:n])
							if sendErr := safeEncode(map[string]string{
								"type": "data", "connID": connID, "payload": payload,
							}); sendErr != nil {
								return
							}
						}
						if err != nil {
							return
						}
					}
				}(connID, localConn)

			case "data":
				var connID, payload string
				json.Unmarshal(raw["connID"], &connID)   //nolint:errcheck
				json.Unmarshal(raw["payload"], &payload) //nolint:errcheck
				data, err := base64.StdEncoding.DecodeString(payload)
				if err != nil {
					continue
				}
				localMu.Lock()
				c, ok := localConns[connID]
				localMu.Unlock()
				if ok {
					c.Write(data) //nolint:errcheck
				}

			case "close":
				var connID string
				json.Unmarshal(raw["connID"], &connID) //nolint:errcheck
				closeLocal(connID, false)
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
// Otherwise, all certificate verification is bypassed to accept the relay's
// self-signed cert.  The channel is still TLS-encrypted.
func buildTLSConfig(caCertPath string) (*tls.Config, error) {
	if caCertPath == "" {
		return &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true, //nolint:gosec
			// VerifyPeerCertificate provides a belt-and-suspenders bypass on
			// platforms (e.g. Windows) where InsecureSkipVerify alone may not
			// suppress all certificate-level TLS alerts.
			VerifyPeerCertificate: func([][]byte, [][]*x509.Certificate) error {
				return nil
			},
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
