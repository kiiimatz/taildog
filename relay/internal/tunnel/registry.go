package tunnel

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Tunnel describes a single forwarding rule on the relay.
type Tunnel struct {
	ID         string    `json:"id"`
	ClientID   string    `json:"clientID"`
	Protocol   string    `json:"protocol"`
	LocalPort  int       `json:"localPort"`
	RemotePort int       `json:"remotePort"`
	CreatedAt  time.Time `json:"createdAt"`
	Active     bool      `json:"active"`
}

// Client represents a connected daemon.
type Client struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	IP          string    `json:"ip"`
	ConnectedAt time.Time `json:"connectedAt"`
	Online      bool      `json:"online"`
	Tunnels     []*Tunnel `json:"tunnels"`
}

// Registry is the in-memory store of clients and their tunnels.
type Registry struct {
	mu      sync.RWMutex
	pool    *Pool
	clients map[string]*Client
	tunnels map[string]*Tunnel
}

// NewRegistry creates a Registry backed by the given port Pool.
func NewRegistry(pool *Pool) *Registry {
	return &Registry{
		pool:    pool,
		clients: make(map[string]*Client),
		tunnels: make(map[string]*Tunnel),
	}
}

// AddClient registers a new connected client, or returns the existing one if
// it reconnects with the same ID.
func (r *Registry) AddClient(id, name, ip string) *Client {
	r.mu.Lock()
	defer r.mu.Unlock()

	if c, ok := r.clients[id]; ok {
		c.Online = true
		c.IP = ip
		return c
	}
	c := &Client{
		ID:          id,
		Name:        name,
		IP:          ip,
		ConnectedAt: time.Now(),
		Online:      true,
	}
	r.clients[id] = c
	return c
}

// MarkOffline marks a client as offline without touching its tunnels or
// releasing ports.  Proxy listeners remain active so that when the daemon
// reconnects, in-flight and new connections are served immediately.
func (r *Registry) MarkOffline(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.clients[id]; ok {
		c.Online = false
	}
}

// RemoveClient marks a client offline and releases all its tunnels.
// Returns the tunnels that were removed so callers can stop proxy listeners.
// Use MarkOffline instead when you want tunnels to persist across reconnects.
func (r *Registry) RemoveClient(id string) []*Tunnel {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.clients[id]
	if !ok {
		return nil
	}
	c.Online = false
	removed := make([]*Tunnel, 0, len(c.Tunnels))
	for _, t := range c.Tunnels {
		t.Active = false
		r.pool.Release(t.RemotePort)
		delete(r.tunnels, t.ID)
		removed = append(removed, t)
	}
	c.Tunnels = nil
	return removed
}

// GetClient returns the client with the given ID.
func (r *Registry) GetClient(id string) (*Client, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.clients[id]
	return c, ok
}

// ListClients returns a snapshot of all known clients.
func (r *Registry) ListClients() []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Client, 0, len(r.clients))
	for _, c := range r.clients {
		out = append(out, c)
	}
	return out
}

// AddTunnel allocates a remote port and registers the tunnel.
// assignedRemotePort may differ from remotePort if there was a conflict.
func (r *Registry) AddTunnel(clientID, proto string, localPort, remotePort int) (*Tunnel, int, error) {
	assigned, err := r.pool.Allocate(remotePort)
	if err != nil {
		return nil, 0, fmt.Errorf("registry: allocate port: %w", err)
	}

	t := &Tunnel{
		ID:         uuid.New().String(),
		ClientID:   clientID,
		Protocol:   proto,
		LocalPort:  localPort,
		RemotePort: assigned,
		CreatedAt:  time.Now(),
		Active:     true,
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.tunnels[t.ID] = t
	if c, ok := r.clients[clientID]; ok {
		c.Tunnels = append(c.Tunnels, t)
	}
	return t, assigned, nil
}

// RemoveTunnel deactivates and removes a tunnel, releasing its port.
func (r *Registry) RemoveTunnel(tunnelID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.tunnels[tunnelID]
	if !ok {
		return fmt.Errorf("registry: tunnel %s not found", tunnelID)
	}

	t.Active = false
	r.pool.Release(t.RemotePort)

	// Remove from the owning client's slice.
	if c, ok := r.clients[t.ClientID]; ok {
		filtered := c.Tunnels[:0]
		for _, ct := range c.Tunnels {
			if ct.ID != tunnelID {
				filtered = append(filtered, ct)
			}
		}
		c.Tunnels = filtered
	}

	delete(r.tunnels, tunnelID)
	return nil
}

// GetTunnel returns the tunnel with the given ID.
func (r *Registry) GetTunnel(id string) (*Tunnel, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tunnels[id]
	return t, ok
}

// ListTunnels returns a snapshot of all registered tunnels.
func (r *Registry) ListTunnels() []*Tunnel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Tunnel, 0, len(r.tunnels))
	for _, t := range r.tunnels {
		out = append(out, t)
	}
	return out
}
