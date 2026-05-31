// Package server exposes the renode relay as a callable library so that
// other binaries (e.g. the combined renode CLI) can embed it.
package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kiiimatz/renode/relay/internal/api"
	"github.com/kiiimatz/renode/relay/internal/auth"
	"github.com/kiiimatz/renode/relay/internal/db"
	internaltls "github.com/kiiimatz/renode/relay/internal/tls"
	"github.com/kiiimatz/renode/relay/internal/tunnel"
)

// Config holds all configuration for the relay server.
type Config struct {
	DataDir         string
	ControlPort     int
	APIPort         int
	APIHost         string
	DashboardOrigin string
	Version         string
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		DataDir:         "./data",
		ControlPort:     7700,
		APIPort:         7701,
		APIHost:         "0.0.0.0",
		DashboardOrigin: "http://localhost:5173",
		Version:         "0.1.0",
	}
}

// Run starts the relay server and blocks until SIGINT/SIGTERM.
func Run(cfg Config) error {
	// ── Database ──────────────────────────────────────────────────────────
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return fmt.Errorf("mkdir data: %w", err)
	}
	database, err := db.New(cfg.DataDir + "/relay.db")
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer database.Close()

	// ── First-run wizard ──────────────────────────────────────────────────
	firstRun, err := database.IsFirstRun()
	if err != nil {
		return fmt.Errorf("first-run check: %w", err)
	}
	if firstRun {
		if err := auth.RunWizard(database); err != nil {
			return fmt.Errorf("wizard: %w", err)
		}
	}

	// ── TLS ───────────────────────────────────────────────────────────────
	certFile, keyFile, err := internaltls.EnsureCerts(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("tls: %w", err)
	}
	tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("load certs: %w", err)
	}
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{tlsCert}, MinVersion: tls.VersionTLS12}

	// ── JWT secret ────────────────────────────────────────────────────────
	if err := auth.LoadSecret(cfg.DataDir); err != nil {
		return fmt.Errorf("jwt secret: %w", err)
	}

	// ── UFW (Linux only) ──────────────────────────────────────────────────
	ufwAllow(cfg.ControlPort, "tcp")
	ufwAllow(cfg.APIPort, "tcp")
	defer func() {
		ufwDelete(cfg.ControlPort, "tcp")
		ufwDelete(cfg.APIPort, "tcp")
		proxies.stopAll() // also removes UFW rules for all tunnel remote ports
	}()

	// ── Tunnel infrastructure ─────────────────────────────────────────────
	pool := tunnel.NewPool(10000, 60000)
	registry := tunnel.NewRegistry(pool)
	hub := api.NewHub()

	// ── REST API + embedded dashboard (plain HTTP) ────────────────────────
	// The control port uses TLS for security; the API/dashboard uses plain
	// HTTP so browsers can connect without certificate warnings.
	srv := &api.Server{
		DB:              database,
		Registry:        registry,
		Hub:             hub,
		DashboardOrigin: cfg.DashboardOrigin,
		StartTime:       time.Now(),
		StartTunnelProxy: func(remotePort int, tunnelID, clientID string, localPort int) error {
			return proxies.start(remotePort, tunnelID, clientID, localPort)
		},
		StopTunnelProxy: func(tunnelID string) {
			proxies.stop(tunnelID)
		},
	}
	apiAddr := fmt.Sprintf("%s:%d", cfg.APIHost, cfg.APIPort)
	httpServer := &http.Server{
		Addr:         apiAddr,
		Handler:      api.NewRouter(srv),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("api: %v", err)
		}
	}()

	// ── Control server ────────────────────────────────────────────────────
	controlAddr := fmt.Sprintf("0.0.0.0:%d", cfg.ControlPort)
	controlListener, err := tls.Listen("tcp", controlAddr, tlsCfg)
	if err != nil {
		return fmt.Errorf("control listen %s: %w", controlAddr, err)
	}
	go AcceptControlConnections(controlListener, registry, hub, database)

	log.Printf("renode relay  control :%d  api+dashboard :%d", cfg.ControlPort, cfg.APIPort)

	// ── Graceful shutdown ─────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down…")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)  //nolint:errcheck
	controlListener.Close()   //nolint:errcheck
	log.Println("bye.")
	return nil
}

// AcceptControlConnections loops accepting client daemon connections.
func AcceptControlConnections(ln net.Listener, registry *tunnel.Registry, hub *api.Hub, database *db.DB) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go HandleControlConn(conn, registry, hub, database)
	}
}
