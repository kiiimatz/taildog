package tunnel

import (
	"testing"
)

func TestRegistryAddRemoveClient(t *testing.T) {
	pool := NewPool(10000, 60000)
	reg := NewRegistry(pool)

	c := reg.AddClient("id1", "host1", "1.2.3.4")
	if c == nil {
		t.Fatal("expected client")
	}
	if len(reg.ListClients()) != 1 {
		t.Fatal("expected 1 client")
	}

	// RemoveClient marks offline; client stays in registry for bookkeeping
	reg.RemoveClient("id1")
	clients := reg.ListClients()
	if len(clients) != 1 {
		t.Fatalf("expected client to remain in registry, got %d clients", len(clients))
	}
	if clients[0].Online {
		t.Fatal("expected client to be marked offline")
	}
}

func TestRegistryAddTunnel(t *testing.T) {
	pool := NewPool(10000, 60000)
	reg := NewRegistry(pool)
	reg.AddClient("c1", "host", "1.1.1.1")

	// Remote port must be within pool range
	tun, assignedPort, err := reg.AddTunnel("c1", "tcp", 3000, 10001)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if assignedPort != 10001 {
		t.Fatalf("expected 10001, got %d", assignedPort)
	}
	if tun.Protocol != "tcp" {
		t.Fatalf("expected tcp, got %s", tun.Protocol)
	}
}

func TestRegistryTunnelConflict(t *testing.T) {
	pool := NewPool(10000, 60000)
	reg := NewRegistry(pool)
	reg.AddClient("c1", "h1", "1.1.1.1")
	reg.AddClient("c2", "h2", "2.2.2.2")

	_, _, err := reg.AddTunnel("c1", "tcp", 3000, 10001)
	if err != nil {
		t.Fatalf("first tunnel: %v", err)
	}

	// Same remote port conflicts — auto-assigned
	_, assigned, err := reg.AddTunnel("c2", "tcp", 3000, 10001)
	if err != nil {
		t.Fatalf("second tunnel (conflict): %v", err)
	}
	if assigned == 10001 {
		t.Fatal("expected auto-reassigned port, got 10001")
	}
}

func TestRegistryRemoveTunnel(t *testing.T) {
	pool := NewPool(10000, 60000)
	reg := NewRegistry(pool)
	reg.AddClient("c1", "h1", "1.1.1.1")

	tun, assigned, err := reg.AddTunnel("c1", "http", 8080, 10080)
	if err != nil {
		t.Fatalf("add tunnel: %v", err)
	}
	if assigned != 10080 {
		t.Fatalf("expected 10080, got %d", assigned)
	}

	err = reg.RemoveTunnel(tun.ID)
	if err != nil {
		t.Fatalf("remove tunnel: %v", err)
	}

	// Port 10080 should be released — reallocating it should succeed
	port, err2 := pool.Allocate(10080)
	if err2 != nil || port != 10080 {
		t.Fatalf("port not released: port=%d err=%v", port, err2)
	}
}
