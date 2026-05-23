// Package db provides SQLite-backed persistence for taildog relay.
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// schema contains all DDL statements executed on startup.
const schema = `
CREATE TABLE IF NOT EXISTS users (
  id           TEXT PRIMARY KEY,
  username     TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  role         TEXT NOT NULL DEFAULT 'admin',
  created_at   DATETIME NOT NULL,
  last_login   DATETIME
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
  id          TEXT PRIMARY KEY,
  user_id     TEXT NOT NULL,
  token_hash  TEXT NOT NULL,
  expires_at  DATETIME NOT NULL,
  created_at  DATETIME NOT NULL,
  FOREIGN KEY(user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS rate_limit (
  ip           TEXT NOT NULL UNIQUE,
  attempts     INTEGER NOT NULL DEFAULT 0,
  locked_until DATETIME,
  last_attempt DATETIME
);

CREATE TABLE IF NOT EXISTS audit_log (
  id         TEXT PRIMARY KEY,
  timestamp  DATETIME NOT NULL,
  event_type TEXT NOT NULL,
  actor      TEXT,
  detail     TEXT,
  ip_address TEXT
);
`

// DB wraps an *sql.DB with domain-specific operations.
type DB struct {
	conn *sql.DB
}

// User represents a row from the users table.
type User struct {
	ID           string
	Username     string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
	LastLogin    *time.Time
}

// RefreshToken represents a row from the refresh_tokens table.
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// AuditEntry represents a row from the audit_log table.
type AuditEntry struct {
	ID        string
	Timestamp time.Time
	EventType string
	Actor     string
	Detail    string
	IPAddress string
}

// New opens (or creates) the SQLite database at path and runs migrations.
func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}
	conn.SetMaxOpenConns(1) // sqlite3 is not safe for concurrent writes
	if _, err := conn.Exec(schema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("db migrate: %w", err)
	}
	return &DB{conn: conn}, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.conn.Close()
}

// IsFirstRun returns true when the users table contains no rows.
func (d *DB) IsFirstRun() (bool, error) {
	var count int
	err := d.conn.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("IsFirstRun: %w", err)
	}
	return count == 0, nil
}

// CreateUser inserts a new user record.
func (d *DB) CreateUser(username, passwordHash, role string) error {
	id := uuid.New().String()
	_, err := d.conn.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, username, passwordHash, role, time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("CreateUser: %w", err)
	}
	return nil
}

// GetUserByUsername fetches a user by username. Returns nil, nil when not found.
func (d *DB) GetUserByUsername(username string) (*User, error) {
	row := d.conn.QueryRow(
		`SELECT id, username, password_hash, role, created_at, last_login FROM users WHERE username = ?`,
		username,
	)
	return scanUser(row)
}

// GetUserByID fetches a user by primary key. Returns nil, nil when not found.
func (d *DB) GetUserByID(id string) (*User, error) {
	row := d.conn.QueryRow(
		`SELECT id, username, password_hash, role, created_at, last_login FROM users WHERE id = ?`,
		id,
	)
	return scanUser(row)
}

func scanUser(row *sql.Row) (*User, error) {
	u := &User{}
	var lastLogin sql.NullTime
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &lastLogin)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanUser: %w", err)
	}
	if lastLogin.Valid {
		u.LastLogin = &lastLogin.Time
	}
	return u, nil
}

// UpdateLastLogin sets the last_login timestamp for a user.
func (d *DB) UpdateLastLogin(userID string) error {
	_, err := d.conn.Exec(
		`UPDATE users SET last_login = ? WHERE id = ?`,
		time.Now().UTC(), userID,
	)
	return err
}

// UpdateUserPassword updates the bcrypt hash for a user.
func (d *DB) UpdateUserPassword(userID, newHash string) error {
	_, err := d.conn.Exec(
		`UPDATE users SET password_hash = ? WHERE id = ?`,
		newHash, userID,
	)
	return err
}

// StoreRefreshToken inserts a hashed refresh token and returns the new row ID.
func (d *DB) StoreRefreshToken(userID, tokenHash string, expiresAt time.Time) (string, error) {
	id := uuid.New().String()
	_, err := d.conn.Exec(
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, userID, tokenHash, expiresAt.UTC(), time.Now().UTC(),
	)
	if err != nil {
		return "", fmt.Errorf("StoreRefreshToken: %w", err)
	}
	return id, nil
}

// GetRefreshToken looks up a refresh token by its hash.
func (d *DB) GetRefreshToken(tokenHash string) (*RefreshToken, error) {
	row := d.conn.QueryRow(
		`SELECT id, user_id, token_hash, expires_at, created_at FROM refresh_tokens WHERE token_hash = ?`,
		tokenHash,
	)
	rt := &RefreshToken{}
	err := row.Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetRefreshToken: %w", err)
	}
	return rt, nil
}

// DeleteRefreshToken removes a single refresh token by hash.
func (d *DB) DeleteRefreshToken(tokenHash string) error {
	_, err := d.conn.Exec(`DELETE FROM refresh_tokens WHERE token_hash = ?`, tokenHash)
	return err
}

// DeleteAllRefreshTokens removes all refresh tokens for a user (logout-all).
func (d *DB) DeleteAllRefreshTokens(userID string) error {
	_, err := d.conn.Exec(`DELETE FROM refresh_tokens WHERE user_id = ?`, userID)
	return err
}

// CheckRateLimit returns true if the IP is currently locked out.
func (d *DB) CheckRateLimit(ip string) (bool, error) {
	row := d.conn.QueryRow(
		`SELECT locked_until FROM rate_limit WHERE ip = ?`, ip,
	)
	var lockedUntil sql.NullTime
	err := row.Scan(&lockedUntil)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("CheckRateLimit: %w", err)
	}
	if lockedUntil.Valid && time.Now().UTC().Before(lockedUntil.Time) {
		return true, nil
	}
	return false, nil
}

// RecordFailedAttempt increments the failure counter and locks the IP after 5 attempts.
func (d *DB) RecordFailedAttempt(ip string) error {
	now := time.Now().UTC()

	// Upsert the row (SQLite ≥ 3.24).
	_, err := d.conn.Exec(`
		INSERT INTO rate_limit (ip, attempts, last_attempt)
		VALUES (?, 1, ?)
		ON CONFLICT(ip) DO UPDATE SET
		  attempts     = attempts + 1,
		  last_attempt = excluded.last_attempt,
		  locked_until = CASE WHEN attempts + 1 >= 5
		                      THEN datetime('now', '+10 minutes')
		                      ELSE locked_until
		                 END
	`, ip, now)
	return err
}

// ResetRateLimit clears failure state for an IP after a successful login.
func (d *DB) ResetRateLimit(ip string) error {
	_, err := d.conn.Exec(
		`UPDATE rate_limit SET attempts = 0, locked_until = NULL WHERE ip = ?`, ip,
	)
	return err
}

// AddAuditLog appends an entry to the audit log.
func (d *DB) AddAuditLog(eventType, actor, detail, ip string) error {
	id := uuid.New().String()
	_, err := d.conn.Exec(
		`INSERT INTO audit_log (id, timestamp, event_type, actor, detail, ip_address)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, time.Now().UTC(), eventType, actor, detail, ip,
	)
	return err
}

// GetAuditLog returns the most recent limit entries, newest first.
func (d *DB) GetAuditLog(limit int) ([]*AuditEntry, error) {
	rows, err := d.conn.Query(
		`SELECT id, timestamp, event_type, actor, detail, ip_address
		   FROM audit_log ORDER BY timestamp DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("GetAuditLog: %w", err)
	}
	defer rows.Close()

	var entries []*AuditEntry
	for rows.Next() {
		e := &AuditEntry{}
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.EventType, &e.Actor, &e.Detail, &e.IPAddress); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
