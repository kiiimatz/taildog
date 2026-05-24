package server

import (
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"sync"
)

// sessionStore maps clientID → *clientSession.
type sessionStore struct {
	mu   sync.RWMutex
	data map[string]*clientSession
}

func newSessionStore() *sessionStore {
	return &sessionStore{data: make(map[string]*clientSession)}
}

func (s *sessionStore) set(clientID string, sess *clientSession) {
	s.mu.Lock()
	s.data[clientID] = sess
	s.mu.Unlock()
}

func (s *sessionStore) del(clientID string) {
	s.mu.Lock()
	delete(s.data, clientID)
	s.mu.Unlock()
}

func (s *sessionStore) get(clientID string) (*clientSession, bool) {
	s.mu.RLock()
	sess, ok := s.data[clientID]
	s.mu.RUnlock()
	return sess, ok
}

// sessions is the global per-binary daemon session registry.
var sessions = newSessionStore()

// proxyEntry holds a tunnel's TCP listener and its remote port number.
type proxyEntry struct {
	ln         net.Listener
	remotePort int
}

// proxyStore manages tunnel TCP listeners.
type proxyStore struct {
	mu   sync.Mutex
	data map[string]proxyEntry // tunnelID → entry
}

// proxies is the global tunnel proxy registry.
var proxies = &proxyStore{data: make(map[string]proxyEntry)}

func (p *proxyStore) start(remotePort int, tunnelID, clientID string, localPort int) error {
	addr := fmt.Sprintf(":%d", remotePort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("proxy: listen %s: %w", addr, err)
	}
	p.mu.Lock()
	p.data[tunnelID] = proxyEntry{ln: ln, remotePort: remotePort}
	p.mu.Unlock()

	ufwAllow(remotePort, "tcp")

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleProxyConn(conn, tunnelID, clientID, localPort)
		}
	}()
	log.Printf("proxy: listening on :%d for tunnel %s (local:%d)", remotePort, tunnelID, localPort)
	return nil
}

func (p *proxyStore) stop(tunnelID string) {
	p.mu.Lock()
	entry, ok := p.data[tunnelID]
	delete(p.data, tunnelID)
	p.mu.Unlock()
	if ok {
		entry.ln.Close()
		ufwDelete(entry.remotePort, "tcp")
		log.Printf("proxy: stopped listener for tunnel %s", tunnelID)
	}
}

// stopAll closes every active proxy listener and removes their UFW rules.
// Called during server shutdown.
func (p *proxyStore) stopAll() {
	p.mu.Lock()
	entries := make(map[string]proxyEntry, len(p.data))
	for k, v := range p.data {
		entries[k] = v
	}
	p.data = make(map[string]proxyEntry)
	p.mu.Unlock()
	for tunnelID, entry := range entries {
		entry.ln.Close()
		ufwDelete(entry.remotePort, "tcp")
		log.Printf("proxy: stopped listener for tunnel %s (shutdown)", tunnelID)
	}
}

func handleProxyConn(extConn net.Conn, tunnelID, clientID string, localPort int) {
	defer extConn.Close()
	log.Printf("proxy: new conn from %s → tunnel %s clientID %s local:%d", extConn.RemoteAddr(), tunnelID, clientID, localPort)

	sess, ok := sessions.get(clientID)
	if !ok {
		log.Printf("proxy: no session for client %s — daemon not connected?", clientID)
		return
	}
	log.Printf("proxy: session found for client %s, sending open frame", clientID)

	connID, err := sess.openConn(tunnelID, extConn, localPort)
	if err != nil {
		log.Printf("proxy: openConn failed: %v", err)
		return
	}
	log.Printf("proxy: open frame sent, connID=%s", connID)
	defer sess.closeConn(connID, true)

	// Read from external connection, send data frames to daemon.
	buf := make([]byte, 32*1024)
	for {
		n, err := extConn.Read(buf)
		if n > 0 {
			payload := base64.StdEncoding.EncodeToString(buf[:n])
			if sendErr := sess.send(map[string]string{
				"type": "data", "connID": connID, "payload": payload,
			}); sendErr != nil {
				break
			}
		}
		if err != nil {
			break
		}
	}
}
