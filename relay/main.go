// taildog relay daemon — entry point.
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kiiimatz/taildog/relay/internal/api"
	"github.com/kiiimatz/taildog/relay/internal/auth"
	"github.com/kiiimatz/taildog/relay/internal/db"
	internaltls "github.com/kiiimatz/taildog/relay/internal/tls"
	"github.com/kiiimatz/taildog/relay/internal/tunnel"
)

const relayVersion = "0.1.0"

func main() {
	// ── Flags ──────────────────────────────────────────────────────────────
	dataDir := flag.String("data-dir", "./data", "directory for DB, certs and secrets")
	controlPort := flag.Int("control-port", 7700, "TLS port for client daemon connections")
	apiPort := flag.Int("api-port", 7701, "TLS port for the REST API")
	apiHost := flag.String("api-host", "0.0.0.0", "listen address for the REST API")
	dashboardOrigin := flag.String("dashboard-origin", "http://localhost:5173", "allowed CORS origin for the dashboard")
	flag.Parse()

	// ── Database ────────────────────────────────────────────────────────────
	dbPath := *dataDir + "/relay.db"
	if err := os.MkdirAll(*dataDir, 0o700); err != nil {
		log.Fatalf("mkdir data: %v", err)
	}
	database, err := db.New(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	// ── First-run wizard ────────────────────────────────────────────────────
	firstRun, err := database.IsFirstRun()
	if err != nil {
		log.Fatalf("first-run check: %v", err)
	}
	if firstRun {
		if err := auth.RunWizard(database); err != nil {
			log.Fatalf("wizard: %v", err)
		}
	}

	// ── TLS certificates ────────────────────────────────────────────────────
	certFile, keyFile, err := internaltls.EnsureCerts(*dataDir)
	if err != nil {
		log.Fatalf("tls: %v", err)
	}

	tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("load certs: %v", err)
	}
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	}

	// ── JWT secret ──────────────────────────────────────────────────────────
	if err := auth.LoadSecret(*dataDir); err != nil {
		log.Fatalf("jwt secret: %v", err)
	}

	// ── Tunnel infrastructure ───────────────────────────────────────────────
	pool := tunnel.NewPool(10000, 60000)
	registry := tunnel.NewRegistry(pool)
	hub := api.NewHub()

	// ── REST API server ─────────────────────────────────────────────────────
	srv := &api.Server{
		DB:              database,
		Registry:        registry,
		Hub:             hub,
		DashboardOrigin: *dashboardOrigin,
		StartTime:       time.Now(),
	}

	apiAddr := fmt.Sprintf("%s:%d", *apiHost, *apiPort)
	apiListener, err := tls.Listen("tcp", apiAddr, tlsCfg)
	if err != nil {
		log.Fatalf("api listen %s: %v", apiAddr, err)
	}

	httpServer := &http.Server{
		Handler:      api.NewRouter(srv),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := httpServer.Serve(apiListener); err != nil && err != http.ErrServerClosed {
			log.Printf("api server: %v", err)
		}
	}()

	// ── Control server ──────────────────────────────────────────────────────
	controlAddr := fmt.Sprintf("0.0.0.0:%d", *controlPort)
	controlListener, err := tls.Listen("tcp", controlAddr, tlsCfg)
	if err != nil {
		log.Fatalf("control listen %s: %v", controlAddr, err)
	}

	go acceptControlConnections(controlListener, registry, hub, database)

	log.Printf("taildog relay listening on :%d, API on :%d", *controlPort, *apiPort)

	// ── Graceful shutdown ───────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down…")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx) //nolint:errcheck
	controlListener.Close()  //nolint:errcheck
	log.Println("bye.")
}

// ── Control plane ────────────────────────────────────────────────────────────

type helloMsg struct {
	Type     string `json:"type"`
	ClientID string `json:"clientID"`
	Name     string `json:"name"`
	Version  string `json:"version"`
}

type welcomeMsg struct {
	Type         string `json:"type"`
	RelayVersion string `json:"relayVersion"`
}

func acceptControlConnections(
	ln net.Listener,
	registry *tunnel.Registry,
	hub *api.Hub,
	database *db.DB,
) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			// Listener closed — normal during shutdown.
			return
		}
		go handleControlConn(conn, registry, hub, database)
	}
}

func handleControlConn(
	conn net.Conn,
	registry *tunnel.Registry,
	hub *api.Hub,
	database *db.DB,
) {
	defer conn.Close()

	remoteIP := conn.RemoteAddr().String()
	conn.SetDeadline(time.Now().Add(30 * time.Second)) //nolint:errcheck

	// ── Handshake ──────────────────────────────────────────────────────────
	var hello helloMsg
	dec := json.NewDecoder(bufio.NewReader(conn))
	if err := dec.Decode(&hello); err != nil {
		log.Printf("control: handshake from %s: %v", remoteIP, err)
		return
	}
	if hello.Type != "hello" || hello.ClientID == "" {
		log.Printf("control: unexpected hello type %q from %s", hello.Type, remoteIP)
		return
	}

	welcome := welcomeMsg{Type: "welcome", RelayVersion: relayVersion}
	if err := json.NewEncoder(conn).Encode(welcome); err != nil {
		log.Printf("control: send welcome to %s: %v", remoteIP, err)
		return
	}

	// Clear the handshake deadline; the connection is now long-lived.
	conn.SetDeadline(time.Time{}) //nolint:errcheck

	// Register the client.
	client := registry.AddClient(hello.ClientID, hello.Name, remoteIP)
	hub.Broadcast("CLIENT_CONNECTED", client)
	database.AddAuditLog("CLIENT_CONNECTED", hello.ClientID, hello.Name, remoteIP) //nolint:errcheck

	log.Printf("control: client %s (%s) connected from %s", hello.Name, hello.ClientID, remoteIP)

	// ── Keep-alive loop ────────────────────────────────────────────────────
	// TODO: implement full tunnel-data forwarding protocol.
	// For now we drain any incoming messages and keep the connection open.
	// When the client disconnects we clean up below.
	buf := make([]byte, 4096)
	for {
		conn.SetReadDeadline(time.Now().Add(2 * time.Minute)) //nolint:errcheck
		_, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("control: client %s read: %v", hello.ClientID, err)
			}
			break
		}
	}

	// ── Cleanup on disconnect ──────────────────────────────────────────────
	registry.RemoveClient(hello.ClientID)
	hub.Broadcast("CLIENT_DISCONNECTED", map[string]string{"id": hello.ClientID})
	database.AddAuditLog("CLIENT_DISCONNECTED", hello.ClientID, "", remoteIP) //nolint:errcheck
	log.Printf("control: client %s (%s) disconnected", hello.Name, hello.ClientID)
}

// ProxyTCP forwards data between two net.Conn connections.
// This is a simple TCP proxy stub — future versions will handle
// UDP/QUIC forwarding and per-tunnel multiplexing.
//
// TODO: replace with proper framed protocol for multi-tunnel multiplexing.
func ProxyTCP(dst, src net.Conn) {
	defer dst.Close()
	defer src.Close()
	io.Copy(dst, src) //nolint:errcheck
}
