package server

import (
	"encoding/json"
	"net"
	"sync"

	"github.com/google/uuid"
)

// extConn abstracts over TCP connections and UDP virtual sessions so that
// control.go can write data back to an external peer without caring about
// the underlying transport.
type extConn interface {
	Write(data []byte) error
	Close() error
}

// tcpExtConn wraps a TCP net.Conn.
type tcpExtConn struct{ conn net.Conn }

func (t *tcpExtConn) Write(data []byte) error { _, err := t.conn.Write(data); return err }
func (t *tcpExtConn) Close() error            { return t.conn.Close() }

// udpExtConn wraps a shared UDP PacketConn and the peer's source address.
// Write sends a datagram back to the peer.
// Close is a no-op: the PacketConn is shared and must be closed by the proxy.
type udpExtConn struct {
	pc   net.PacketConn
	addr net.Addr
}

func (u *udpExtConn) Write(data []byte) error { _, err := u.pc.WriteTo(data, u.addr); return err }
func (u *udpExtConn) Close() error            { return nil }

type clientSession struct {
	clientID string
	encMu    sync.Mutex
	enc      *json.Encoder
	connMu   sync.Mutex
	conns    map[string]extConn // connID → external connection (TCP or UDP)
}

func newClientSession(clientID string, conn net.Conn) *clientSession {
	return &clientSession{
		clientID: clientID,
		enc:      json.NewEncoder(conn),
		conns:    make(map[string]extConn),
	}
}

func (s *clientSession) send(v interface{}) error {
	s.encMu.Lock()
	err := s.enc.Encode(v)
	s.encMu.Unlock()
	return err
}

// openConn sends an "open" frame to the daemon and registers the external conn.
func (s *clientSession) openConn(tunnelID string, extC extConn, localPort int) (string, error) {
	connID := uuid.New().String()
	err := s.send(map[string]interface{}{"type": "open", "tunnelID": tunnelID, "connID": connID, "localPort": localPort})
	if err != nil {
		return "", err
	}
	s.connMu.Lock()
	s.conns[connID] = extC
	s.connMu.Unlock()
	return connID, nil
}

// closeConn removes the external conn and optionally sends a "close" frame.
func (s *clientSession) closeConn(connID string, sendFrame bool) {
	s.connMu.Lock()
	conn, ok := s.conns[connID]
	delete(s.conns, connID)
	s.connMu.Unlock()
	if ok {
		conn.Close()
	}
	if sendFrame {
		s.send(map[string]string{"type": "close", "connID": connID}) //nolint:errcheck
	}
}

// writeToExternal writes data to the external connection for connID.
func (s *clientSession) writeToExternal(connID string, data []byte) {
	s.connMu.Lock()
	conn, ok := s.conns[connID]
	s.connMu.Unlock()
	if ok {
		conn.Write(data) //nolint:errcheck
	}
}

// closeAllExternal closes all external connections.
func (s *clientSession) closeAllExternal() {
	s.connMu.Lock()
	for _, conn := range s.conns {
		conn.Close()
	}
	s.conns = make(map[string]extConn)
	s.connMu.Unlock()
}
