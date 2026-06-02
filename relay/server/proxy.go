package server

import (
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
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

// proxyEntry holds a tunnel's listener and its metadata.
// For TCP tunnels ln is set; for UDP tunnels pc is set.
type proxyEntry struct {
	protocol   string
	remotePort int
	ln         net.Listener   // TCP
	pc         net.PacketConn // UDP
}

// proxyStore manages tunnel proxy listeners (TCP and UDP).
type proxyStore struct {
	mu   sync.Mutex
	data map[string]proxyEntry // tunnelID → entry
}

// proxies is the global tunnel proxy registry.
var proxies = &proxyStore{data: make(map[string]proxyEntry)}

// start opens a proxy listener for the given protocol.
func (p *proxyStore) start(proto string, remotePort int, tunnelID, clientID string, localPort int) error {
	if proto == "udp" {
		return p.startUDP(remotePort, tunnelID, clientID, localPort)
	}
	return p.startTCP(remotePort, tunnelID, clientID, localPort)
}

func (p *proxyStore) startTCP(remotePort int, tunnelID, clientID string, localPort int) error {
	addr := fmt.Sprintf(":%d", remotePort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("proxy: listen tcp %s: %w", addr, err)
	}
	p.mu.Lock()
	p.data[tunnelID] = proxyEntry{protocol: "tcp", ln: ln, remotePort: remotePort}
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
	log.Printf("proxy: listening on tcp :%d for tunnel %s (local:%d)", remotePort, tunnelID, localPort)
	return nil
}

func (p *proxyStore) startUDP(remotePort int, tunnelID, clientID string, localPort int) error {
	addr := fmt.Sprintf(":%d", remotePort)
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return fmt.Errorf("proxy: listen udp %s: %w", addr, err)
	}
	p.mu.Lock()
	p.data[tunnelID] = proxyEntry{protocol: "udp", pc: pc, remotePort: remotePort}
	p.mu.Unlock()

	ufwAllow(remotePort, "udp")

	go handleUDPProxy(pc, tunnelID, clientID, localPort)
	log.Printf("proxy: listening on udp :%d for tunnel %s (local:%d)", remotePort, tunnelID, localPort)
	return nil
}

func (p *proxyStore) stop(tunnelID string) {
	p.mu.Lock()
	entry, ok := p.data[tunnelID]
	delete(p.data, tunnelID)
	p.mu.Unlock()
	if !ok {
		return
	}
	if entry.ln != nil {
		entry.ln.Close()
	}
	if entry.pc != nil {
		entry.pc.Close()
	}
	ufwDelete(entry.remotePort, entry.protocol)
	log.Printf("proxy: stopped listener for tunnel %s", tunnelID)
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
		if entry.ln != nil {
			entry.ln.Close()
		}
		if entry.pc != nil {
			entry.pc.Close()
		}
		ufwDelete(entry.remotePort, entry.protocol)
		log.Printf("proxy: stopped listener for tunnel %s (shutdown)", tunnelID)
	}
}

// ── TCP proxy ────────────────────────────────────────────────────────────────

func handleProxyConn(extConn net.Conn, tunnelID, clientID string, localPort int) {
	defer extConn.Close()
	log.Printf("proxy: new tcp conn from %s → tunnel %s clientID %s local:%d", extConn.RemoteAddr(), tunnelID, clientID, localPort)

	sess, ok := sessions.get(clientID)
	if !ok {
		log.Printf("proxy: no session for client %s — daemon not connected?", clientID)
		return
	}

	connID, err := sess.openConn(tunnelID, &tcpExtConn{conn: extConn}, localPort)
	if err != nil {
		log.Printf("proxy: openConn failed: %v", err)
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

// ── UDP proxy ────────────────────────────────────────────────────────────────

const udpIdleTimeout = 5 * time.Minute

type udpVirtualSession struct {
	connID   string
	lastSeen time.Time
}

// handleUDPProxy reads datagrams from pc and multiplexes them over the daemon
// control channel as virtual connections keyed by source address.
// Each unique (srcAddr) is a separate virtual session; the daemon side dials
// a connected UDP socket to localPort for each one.
func handleUDPProxy(pc net.PacketConn, tunnelID, clientID string, localPort int) {
	addrToSess := make(map[string]*udpVirtualSession) // srcAddr.String() → virtual session
	lastCleanup := time.Now()
	buf := make([]byte, 65536)

	for {
		n, srcAddr, err := pc.ReadFrom(buf)
		if err != nil {
			return // pc was closed (tunnel deleted or server shutdown)
		}

		sess, ok := sessions.get(clientID)
		if !ok {
			// Daemon not connected; drop the datagram.
			continue
		}

		addrKey := srcAddr.String()
		vs, exists := addrToSess[addrKey]
		if !exists {
			connID, err := sess.openConn(tunnelID, &udpExtConn{pc: pc, addr: srcAddr}, localPort)
			if err != nil {
				log.Printf("proxy udp: openConn for %s failed: %v", addrKey, err)
				continue
			}
			vs = &udpVirtualSession{connID: connID, lastSeen: time.Now()}
			addrToSess[addrKey] = vs
			log.Printf("proxy udp: new session %s → connID %s local:%d", addrKey, connID, localPort)
		}
		vs.lastSeen = time.Now()

		payload := base64.StdEncoding.EncodeToString(buf[:n])
		if err := sess.send(map[string]string{
			"type": "data", "connID": vs.connID, "payload": payload,
		}); err != nil {
			log.Printf("proxy udp: send data frame for %s: %v", addrKey, err)
		}

		// Periodic idle-session cleanup (runs in the read goroutine to avoid locks).
		if time.Since(lastCleanup) > time.Minute {
			lastCleanup = time.Now()
			for key, s := range addrToSess {
				if time.Since(s.lastSeen) > udpIdleTimeout {
					if sess2, ok2 := sessions.get(clientID); ok2 {
						sess2.closeConn(s.connID, true)
					}
					delete(addrToSess, key)
					log.Printf("proxy udp: expired idle session %s", key)
				}
			}
		}
	}
}
