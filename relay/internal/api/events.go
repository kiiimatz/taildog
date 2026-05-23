// Package api implements the REST API and WebSocket event hub.
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Origin checking is handled by the CORS middleware.
		return true
	},
}

// Event is the envelope sent over the WebSocket channel.
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Hub manages active WebSocket connections and broadcasts events to them.
type Hub struct {
	mu      sync.RWMutex
	clients map[*wsClient]struct{}
}

type wsClient struct {
	conn   *websocket.Conn
	send   chan []byte
	closed chan struct{}
}

// NewHub creates an empty Hub.
func NewHub() *Hub {
	return &Hub{clients: make(map[*wsClient]struct{})}
}

// Broadcast serialises the event and sends it to every connected WS client.
func (h *Hub) Broadcast(eventType string, data interface{}) {
	payload, err := json.Marshal(Event{Type: eventType, Data: data})
	if err != nil {
		log.Printf("hub: marshal event %s: %v", eventType, err)
		return
	}

	h.mu.RLock()
	targets := make([]*wsClient, 0, len(h.clients))
	for c := range h.clients {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		select {
		case c.send <- payload:
		case <-c.closed:
			// Client already gone; remove it.
			h.remove(c)
		default:
			// Slow consumer — drop and disconnect.
			h.remove(c)
			c.conn.Close()
		}
	}
}

func (h *Hub) add(c *wsClient) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) remove(c *wsClient) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

// ServeWS upgrades an HTTP connection to WebSocket and registers the client
// with the hub. The caller must have already validated the auth token.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws: upgrade: %v", err)
		return
	}

	c := &wsClient{
		conn:   conn,
		send:   make(chan []byte, 64),
		closed: make(chan struct{}),
	}
	h.add(c)

	// Writer goroutine.
	go func() {
		defer func() {
			close(c.closed)
			h.remove(c)
			conn.Close()
		}()
		for {
			select {
			case msg, ok := <-c.send:
				if !ok {
					conn.WriteMessage(websocket.CloseMessage, nil) //nolint:errcheck
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			}
		}
	}()

	// Reader goroutine — we only need it to detect disconnects.
	go func() {
		defer func() {
			h.remove(c)
			conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}
