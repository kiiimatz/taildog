package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.RelayPort != 7700 {
		t.Fatalf("expected relay_port 7700, got %d", cfg.RelayPort)
	}
	if cfg.Tunnels == nil {
		t.Fatal("expected non-nil tunnels slice")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	// Override config path for test
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	defer func() {
		if origHome != "" {
			os.Setenv("HOME", origHome)
		}
	}()

	// Create config dir
	cfgDir := filepath.Join(dir, ".renode")
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	cfg.RelayHost = "relay.example.com"
	cfg.RelayPort = 7700
	cfg.Tunnels = []Tunnel{
		{ID: "abc", Protocol: "tcp", LocalPort: 5432, RemotePort: 15432},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.RelayHost != "relay.example.com" {
		t.Fatalf("relay_host mismatch: %s", loaded.RelayHost)
	}
	if len(loaded.Tunnels) != 1 {
		t.Fatalf("expected 1 tunnel, got %d", len(loaded.Tunnels))
	}
	if loaded.Tunnels[0].Protocol != "tcp" {
		t.Fatalf("protocol mismatch: %s", loaded.Tunnels[0].Protocol)
	}
	if loaded.Tunnels[0].RemotePort != 15432 {
		t.Fatalf("remote_port mismatch: %d", loaded.Tunnels[0].RemotePort)
	}
}

func TestLoadMissing(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load on missing file should return default, got: %v", err)
	}
	if cfg.RelayPort != 7700 {
		t.Fatalf("expected default relay_port 7700")
	}
}
