package server

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net"
	"time"

	"github.com/kiiimatz/renode/relay/internal/api"
	"github.com/kiiimatz/renode/relay/internal/db"
	"github.com/kiiimatz/renode/relay/internal/tunnel"
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

	// Single decoder for the entire session lifetime.
	dec := json.NewDecoder(bufio.NewReader(conn))

	var hello helloMsg
	if err := dec.Decode(&hello); err != nil {
		log.Printf("control: handshake from %s: %v", remoteIP, err)
		return
	}
	if hello.Type != "hello" || hello.ClientID == "" {
		log.Printf("control: unexpected hello type %q from %s", hello.Type, remoteIP)
		return
	}

	sess := newClientSession(hello.ClientID, conn)
	if err := sess.send(welcomeMsg{Type: "welcome", RelayVersion: "0.1.0"}); err != nil {
		log.Printf("control: send welcome to %s: %v", remoteIP, err)
		return
	}
	conn.SetDeadline(time.Time{}) //nolint:errcheck

	client := registry.AddClient(hello.ClientID, hello.Name, remoteIP)
	hub.Broadcast("CLIENT_CONNECTED", client)
	database.AddAuditLog("CLIENT_CONNECTED", hello.ClientID, hello.Name, remoteIP) //nolint:errcheck
	log.Printf("control: client %s (%s) connected from %s", hello.Name, hello.ClientID, remoteIP)

	sessions.set(hello.ClientID, sess)
	defer func() {
		// Remove the session first so new proxy connections don't reach a
		// half-closed pipe while we're tearing down in-flight connections.
		sessions.del(hello.ClientID)
		// Close any external TCP connections that are currently bridged through
		// this daemon session (they will be re-opened on the next request once
		// the daemon reconnects).
		sess.closeAllExternal()
		// Mark the client offline but keep all tunnel records and proxy
		// listeners alive.  When the daemon reconnects, AddClient() flips
		// Online back to true and the proxy listeners resume serving traffic.
		registry.MarkOffline(hello.ClientID)
		hub.Broadcast("CLIENT_DISCONNECTED", map[string]string{"id": hello.ClientID})
		database.AddAuditLog("CLIENT_DISCONNECTED", hello.ClientID, "", remoteIP) //nolint:errcheck
		log.Printf("control: client %s (%s) disconnected", hello.Name, hello.ClientID)
	}()

	// Main read loop — decode JSON frames from daemon.
	for {
		conn.SetReadDeadline(time.Now().Add(2 * time.Minute)) //nolint:errcheck
		var raw map[string]json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			if err != io.EOF {
				log.Printf("control: client %s read: %v", hello.ClientID, err)
			}
			break
		}

		var frameType string
		if err := json.Unmarshal(raw["type"], &frameType); err != nil {
			continue
		}

		switch frameType {
		case "ping":
			sess.send(map[string]string{"type": "pong"}) //nolint:errcheck

		case "data":
			var connID, payload string
			json.Unmarshal(raw["connID"], &connID)   //nolint:errcheck
			json.Unmarshal(raw["payload"], &payload) //nolint:errcheck
			data, err := base64.StdEncoding.DecodeString(payload)
			if err == nil {
				sess.writeToExternal(connID, data)
			}

		case "close":
			var connID string
			json.Unmarshal(raw["connID"], &connID) //nolint:errcheck
			sess.closeConn(connID, false)

		// "register" frames from daemon are informational only;
		// tunnels are managed via the REST API.
		}
	}
}
