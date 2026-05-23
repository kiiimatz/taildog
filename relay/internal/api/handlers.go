package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/kiiimatz/taildog/relay/internal/auth"
	"github.com/kiiimatz/taildog/relay/internal/db"
	"github.com/kiiimatz/taildog/relay/internal/tunnel"
)

const relayVersion = "0.1.0"

// claimsCtxKey is the context key used to store validated JWT claims.
type claimsCtxKey struct{}

// Server holds the shared state needed by all HTTP handlers.
type Server struct {
	DB              *db.DB
	Registry        *tunnel.Registry
	Hub             *Hub
	DashboardOrigin string
	StartTime       time.Time
}

// NewRouter builds and returns the gorilla/mux router wired to s.
func NewRouter(s *Server) http.Handler {
	r := mux.NewRouter()

	// CORS + common middleware applied to every route.
	r.Use(s.corsMiddleware)

	// Public routes.
	r.HandleFunc("/api/auth/login", s.handleLogin).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/api/auth/refresh", s.handleRefresh).Methods(http.MethodPost, http.MethodOptions)

	// Authenticated routes.
	authed := r.PathPrefix("/api").Subrouter()
	authed.Use(s.authMiddleware)

	authed.HandleFunc("/auth/logout", s.handleLogout).Methods(http.MethodPost, http.MethodOptions)
	authed.HandleFunc("/auth/me", s.handleMe).Methods(http.MethodGet, http.MethodOptions)

	authed.HandleFunc("/clients", s.handleListClients).Methods(http.MethodGet, http.MethodOptions)
	authed.HandleFunc("/clients/{id}/tunnels", s.handleClientTunnels).Methods(http.MethodGet, http.MethodOptions)
	authed.HandleFunc("/tunnels", s.handleCreateTunnel).Methods(http.MethodPost, http.MethodOptions)
	authed.HandleFunc("/tunnels/{id}", s.handleDeleteTunnel).Methods(http.MethodDelete, http.MethodOptions)

	authed.HandleFunc("/server/info", s.handleServerInfo).Methods(http.MethodGet, http.MethodOptions)

	// Admin-only routes.
	admin := authed.PathPrefix("/admin").Subrouter()
	admin.Use(s.adminMiddleware)

	admin.HandleFunc("/audit-log", s.handleAuditLog).Methods(http.MethodGet, http.MethodOptions)
	admin.HandleFunc("/change-password", s.handleChangePassword).Methods(http.MethodPost, http.MethodOptions)
	admin.HandleFunc("/revoke-all-sessions", s.handleRevokeAllSessions).Methods(http.MethodPost, http.MethodOptions)

	// WebSocket — auth checked inside via ?token= query param.
	authed.HandleFunc("/events", s.handleEvents).Methods(http.MethodGet)

	return r
}

// ── Middleware ──────────────────────────────────────────────────────────────

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == s.DashboardOrigin || s.DashboardOrigin == "*" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// WebSocket events endpoint uses a query-param token; handled inside the handler.
		if r.URL.Path == "/api/events" {
			next.ServeHTTP(w, r)
			return
		}

		tokenStr := ""
		if h := r.Header.Get("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
			tokenStr = h[7:]
		}
		if tokenStr == "" {
			jsonError(w, http.StatusUnauthorized, "missing token")
			return
		}
		claims, err := auth.ValidateToken(tokenStr)
		if err != nil {
			jsonError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), claimsCtxKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFromCtx(r.Context())
		if claims == nil || claims.Role != "admin" {
			jsonError(w, http.StatusForbidden, "admin required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── Auth handlers ───────────────────────────────────────────────────────────

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid body")
		return
	}

	ip := clientIP(r)

	// Rate-limit check.
	locked, err := s.DB.CheckRateLimit(ip)
	if err != nil {
		log.Printf("login: rate-limit check: %v", err)
	}
	if locked {
		jsonError(w, http.StatusTooManyRequests, "too many failed attempts — try again later")
		return
	}

	fail := func(msg string) {
		// Fixed 400 ms delay on every failure to blunt timing attacks.
		time.Sleep(400 * time.Millisecond)
		s.DB.RecordFailedAttempt(ip)                             //nolint:errcheck
		s.DB.AddAuditLog("LOGIN_FAILED", body.Username, msg, ip) //nolint:errcheck
		jsonError(w, http.StatusUnauthorized, "invalid credentials")
	}

	user, err := s.DB.GetUserByUsername(body.Username)
	if err != nil || user == nil {
		fail("user not found")
		return
	}
	if !auth.CheckPassword(user.PasswordHash, body.Password) {
		fail("wrong password")
		return
	}

	// Success.
	s.DB.ResetRateLimit(ip)       //nolint:errcheck
	s.DB.UpdateLastLogin(user.ID) //nolint:errcheck

	accessToken, err := auth.IssueAccessToken(user.ID, user.Role)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	refreshToken, err := auth.IssueRefreshToken(user.ID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "could not issue refresh token")
		return
	}

	// Store the SHA-256 hash of the refresh token.
	h := sha256Token(refreshToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if _, err := s.DB.StoreRefreshToken(user.ID, h, expiresAt); err != nil {
		log.Printf("login: store refresh token: %v", err)
	}

	s.DB.AddAuditLog("LOGIN_SUCCESS", user.Username, "", ip) //nolint:errcheck

	jsonOK(w, map[string]interface{}{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
		"expiresIn":    900,
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
		jsonError(w, http.StatusBadRequest, "missing refreshToken")
		return
	}

	claims, err := auth.ValidateToken(body.RefreshToken)
	if err != nil {
		jsonError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	h := sha256Token(body.RefreshToken)
	rt, err := s.DB.GetRefreshToken(h)
	if err != nil || rt == nil {
		jsonError(w, http.StatusUnauthorized, "refresh token not found or revoked")
		return
	}
	if time.Now().After(rt.ExpiresAt) {
		s.DB.DeleteRefreshToken(h) //nolint:errcheck
		jsonError(w, http.StatusUnauthorized, "refresh token expired")
		return
	}

	user, err := s.DB.GetUserByID(claims.UserID)
	if err != nil || user == nil {
		jsonError(w, http.StatusUnauthorized, "user not found")
		return
	}

	accessToken, err := auth.IssueAccessToken(user.ID, user.Role)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "could not issue token")
		return
	}

	jsonOK(w, map[string]interface{}{
		"accessToken": accessToken,
		"expiresIn":   900,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck

	if body.RefreshToken != "" {
		h := sha256Token(body.RefreshToken)
		s.DB.DeleteRefreshToken(h) //nolint:errcheck
	}

	claims := claimsFromCtx(r.Context())
	if claims != nil {
		s.DB.AddAuditLog("LOGOUT", claims.UserID, "", clientIP(r)) //nolint:errcheck
	}
	jsonOK(w, map[string]string{"status": "ok"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	user, err := s.DB.GetUserByID(claims.UserID)
	if err != nil || user == nil {
		jsonError(w, http.StatusNotFound, "user not found")
		return
	}
	jsonOK(w, map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

// ── Client / tunnel handlers ────────────────────────────────────────────────

func (s *Server) handleListClients(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, s.Registry.ListClients())
}

func (s *Server) handleClientTunnels(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	client, ok := s.Registry.GetClient(id)
	if !ok {
		jsonError(w, http.StatusNotFound, "client not found")
		return
	}
	if client.Tunnels == nil {
		jsonOK(w, []*tunnel.Tunnel{})
		return
	}
	jsonOK(w, client.Tunnels)
}

func (s *Server) handleCreateTunnel(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClientID   string `json:"clientID"`
		Protocol   string `json:"protocol"`
		LocalPort  int    `json:"localPort"`
		RemotePort int    `json:"remotePort"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.ClientID == "" || body.Protocol == "" || body.LocalPort == 0 {
		jsonError(w, http.StatusBadRequest, "clientID, protocol and localPort are required")
		return
	}

	t, assigned, err := s.Registry.AddTunnel(body.ClientID, body.Protocol, body.LocalPort, body.RemotePort)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.Hub.Broadcast("TUNNEL_CREATED", t)
	claims := claimsFromCtx(r.Context())
	if claims != nil {
		s.DB.AddAuditLog("TUNNEL_CREATED", claims.UserID, //nolint:errcheck
			"port="+strconv.Itoa(assigned), clientIP(r))
	}

	jsonCreated(w, t)
}

func (s *Server) handleDeleteTunnel(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if _, ok := s.Registry.GetTunnel(id); !ok {
		jsonError(w, http.StatusNotFound, "tunnel not found")
		return
	}
	if err := s.Registry.RemoveTunnel(id); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.Hub.Broadcast("TUNNEL_DELETED", map[string]string{"id": id})
	claims := claimsFromCtx(r.Context())
	if claims != nil {
		s.DB.AddAuditLog("TUNNEL_DELETED", claims.UserID, "tunnelID="+id, clientIP(r)) //nolint:errcheck
	}
	jsonOK(w, map[string]string{"status": "deleted"})
}

// ── Server info ─────────────────────────────────────────────────────────────

func (s *Server) handleServerInfo(w http.ResponseWriter, r *http.Request) {
	clients := s.Registry.ListClients()
	online := 0
	for _, c := range clients {
		if c.Online {
			online++
		}
	}
	// Parse host + IP from request
	serverName := r.Host
	serverIP := clientIP(r)
	if serverName == "" {
		serverName = "taildog-relay"
	}
	jsonOK(w, map[string]interface{}{
		"version":          relayVersion,
		"uptime":           int(time.Since(s.StartTime).Seconds()),
		"connectedClients": online,
		"serverName":       serverName,
		"serverIP":         serverIP,
	})
}

// ── Admin handlers ───────────────────────────────────────────────────────────

func (s *Server) handleAuditLog(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}
	entries, err := s.DB.GetAuditLog(limit)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "db error")
		return
	}
	jsonOK(w, entries)
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid body")
		return
	}

	claims := claimsFromCtx(r.Context())
	user, err := s.DB.GetUserByID(claims.UserID)
	if err != nil || user == nil {
		jsonError(w, http.StatusNotFound, "user not found")
		return
	}
	if !auth.CheckPassword(user.PasswordHash, body.CurrentPassword) {
		jsonError(w, http.StatusUnauthorized, "current password incorrect")
		return
	}
	if len(body.NewPassword) < 12 {
		jsonError(w, http.StatusBadRequest, "new password too short (min 12 chars)")
		return
	}

	hash, err := auth.HashPassword(body.NewPassword)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "hash error")
		return
	}
	if err := s.DB.UpdateUserPassword(user.ID, hash); err != nil {
		jsonError(w, http.StatusInternalServerError, "db error")
		return
	}
	s.DB.AddAuditLog("PASSWORD_CHANGED", user.Username, "", clientIP(r)) //nolint:errcheck
	jsonOK(w, map[string]string{"status": "ok"})
}

func (s *Server) handleRevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	if err := s.DB.DeleteAllRefreshTokens(claims.UserID); err != nil {
		jsonError(w, http.StatusInternalServerError, "db error")
		return
	}
	s.DB.AddAuditLog("REVOKE_ALL_SESSIONS", claims.UserID, "", clientIP(r)) //nolint:errcheck
	jsonOK(w, map[string]string{"status": "ok"})
}

// ── WebSocket ─────────────────────────────────────────────────────────────

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	// Validate ?token= query parameter since the JS WebSocket API cannot set headers.
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		jsonError(w, http.StatusUnauthorized, "missing token")
		return
	}
	if _, err := auth.ValidateToken(tokenStr); err != nil {
		jsonError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	s.Hub.ServeWS(w, r)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func jsonCreated(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func jsonError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	return r.RemoteAddr
}

func sha256Token(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func claimsFromCtx(ctx context.Context) *auth.Claims {
	c, _ := ctx.Value(claimsCtxKey{}).(*auth.Claims)
	return c
}
