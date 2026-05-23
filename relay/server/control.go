package server

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net"
	"time"

	"github.com/kiiimatz/taildog/relay/internal/api"
	"github.com/kiiimatz/taildog/relay/internal/db"
	"github.com/kiiimatz/taildog/relay/internal/tunnel"
)

type helloMsg struct {
	Type     string `json:"type"`
	ClientID string `json:"clientID"`
	Name     string `json:"name"`
	Version  string `json:"version"`
}

type welcomeMsg struct {
	Type         string `json:"type"`
	RelayVersion string `json:"relayVersion"`
}

// HandleControlConn handles a single client daemon connection.
func HandleControlConn(conn net.Conn, registry *tunnel.Registry, hub *api.Hub, database *db.DB) {
	defer conn.Close()

	rawAddr := conn.RemoteAddr().String()
	remoteIP, _, _ := net.SplitHostPort(rawAddr)
	if remoteIP == "" {
		remoteIP = rawAddr
	}
	conn.SetDeadline(time.Now().Add(30 * time.Second)) //nolint:errcheck

	var hello helloMsg
	if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&hello); err != nil {
		log.Printf("control: handshake from %s: %v", remoteIP, err)
		return
	}
	if hello.Type != "hello" || hello.ClientID == "" {
		log.Printf("control: unexpected hello type %q from %s", hello.Type, remoteIP)
		return
	}

	welcome := welcomeMsg{Type: "welcome", RelayVersion: "0.1.0"}
	if err := json.NewEncoder(conn).Encode(welcome); err != nil {
		log.Printf("control: send welcome to %s: %v", remoteIP, err)
		return
	}
	conn.SetDeadline(time.Time{}) //nolint:errcheck

	client := registry.AddClient(hello.ClientID, hello.Name, remoteIP)
	hub.Broadcast("CLIENT_CONNECTED", client)
	database.AddAuditLog("CLIENT_CONNECTED", hello.ClientID, hello.Name, remoteIP) //nolint:errcheck
	log.Printf("control: client %s (%s) connected from %s", hello.Name, hello.ClientID, remoteIP)

	buf := make([]byte, 4096)
	for {
		conn.SetReadDeadline(time.Now().Add(2 * time.Minute)) //nolint:errcheck
		if _, err := conn.Read(buf); err != nil {
			if err != io.EOF {
				log.Printf("control: client %s read: %v", hello.ClientID, err)
			}
			break
		}
	}

	registry.RemoveClient(hello.ClientID)
	hub.Broadcast("CLIENT_DISCONNECTED", map[string]string{"id": hello.ClientID})
	database.AddAuditLog("CLIENT_DISCONNECTED", hello.ClientID, "", remoteIP) //nolint:errcheck
	log.Printf("control: client %s (%s) disconnected", hello.Name, hello.ClientID)
}
