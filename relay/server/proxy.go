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

// proxyStore manages tunnel TCP listeners.
type proxyStore struct {
	mu   sync.Mutex
	data map[string]net.Listener // tunnelID → listener
}

// proxies is the global tunnel proxy registry.
var proxies = &proxyStore{data: make(map[string]net.Listener)}

func (p *proxyStore) start(remotePort int, tunnelID, clientID string, localPort int) error {
	addr := fmt.Sprintf(":%d", remotePort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("proxy: listen %s: %w", addr, err)
	}
	p.mu.Lock()
	p.data[tunnelID] = ln
	p.mu.Unlock()

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
	ln, ok := p.data[tunnelID]
	delete(p.data, tunnelID)
	p.mu.Unlock()
	if ok {
		ln.Close()
		log.Printf("proxy: stopped listener for tunnel %s", tunnelID)
	}
}

func handleProxyConn(extConn net.Conn, tunnelID, clientID string, localPort int) {
	defer extConn.Close()

	sess, ok := sessions.get(clientID)
	if !ok {
		log.Printf("proxy: no session for client %s", clientID)
		return
	}

	connID, err := sess.openConn(tunnelID, extConn, localPort)
	if err != nil {
		log.Printf("proxy: open conn: %v", err)
		return
	}
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
