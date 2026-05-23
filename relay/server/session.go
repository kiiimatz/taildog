package server

import (
	"encoding/json"
	"net"
	"sync"

	"github.com/google/uuid"
)

type clientSession struct {
	clientID string
	encMu    sync.Mutex
	enc      *json.Encoder
	connMu   sync.Mutex
	conns    map[string]net.Conn // connID → external TCP connection
}

func newClientSession(clientID string, conn net.Conn) *clientSession {
	return &clientSession{
		clientID: clientID,
		enc:      json.NewEncoder(conn),
		conns:    make(map[string]net.Conn),
	}
}

func (s *clientSession) send(v interface{}) error {
	s.encMu.Lock()
	err := s.enc.Encode(v)
	s.encMu.Unlock()
	return err
}

// openConn sends an "open" frame to the daemon and registers the external conn.
func (s *clientSession) openConn(tunnelID string, extConn net.Conn) (string, error) {
	connID := uuid.New().String()
	err := s.send(map[string]string{"type": "open", "tunnelID": tunnelID, "connID": connID})
	if err != nil {
		return "", err
	}
	s.connMu.Lock()
	s.conns[connID] = extConn
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
	s.conns = make(map[string]net.Conn)
	s.connMu.Unlock()
}
