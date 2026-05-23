// Package auth handles JWT issuance/validation, password hashing and the JWT secret.
package auth

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost      = 14
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	secretBytes     = 64
)

// Claims is the JWT payload used for both access and refresh tokens.
type Claims struct {
	UserID string `json:"uid"`
	Role   string `json:"role,omitempty"`
	jwt.RegisteredClaims
}

var jwtSecret []byte

// LoadSecret reads (or generates) the JWT signing secret at <dataDir>/.secret.
func LoadSecret(dataDir string) error {
	path := filepath.Join(dataDir, ".secret")

	data, err := os.ReadFile(path)
	if err == nil && len(data) == secretBytes {
		jwtSecret = data
		return nil
	}

	// Generate a new secret.
	secret := make([]byte, secretBytes)
	if _, err := rand.Read(secret); err != nil {
		return fmt.Errorf("auth: generate secret: %w", err)
	}

	perm := os.FileMode(0o600)
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return fmt.Errorf("auth: mkdir %s: %w", dataDir, err)
	}
	if err := os.WriteFile(path, secret, perm); err != nil {
		return fmt.Errorf("auth: write secret: %w", err)
	}

	// On Unix, ensure the file permissions are strictly 0600.
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0o600); err != nil {
			return fmt.Errorf("auth: chmod secret: %w", err)
		}
	}

	jwtSecret = secret
	return nil
}

// IssueAccessToken mints a signed HS256 JWT valid for 15 minutes.
func IssueAccessToken(userID, role string) (string, error) {
	if jwtSecret == nil {
		return "", fmt.Errorf("auth: JWT secret not loaded")
	}
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// IssueRefreshToken mints a signed HS256 JWT valid for 7 days.
func IssueRefreshToken(userID string) (string, error) {
	if jwtSecret == nil {
		return "", fmt.Errorf("auth: JWT secret not loaded")
	}
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(refreshTokenTTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken parses and validates a JWT string, returning its claims.
func ValidateToken(tokenStr string) (*Claims, error) {
	if jwtSecret == nil {
		return nil, fmt.Errorf("auth: JWT secret not loaded")
	}
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method %v", t.Header["alg"])
		}
		return jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth: validate token: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("auth: invalid token claims")
	}
	return claims, nil
}

// HashPassword returns a bcrypt hash of password. The plaintext slice is
// zeroed before the function returns.
func HashPassword(password string) (string, error) {
	plain := []byte(password)
	defer zeroBytes(plain)

	hash, err := bcrypt.GenerateFromPassword(plain, bcryptCost)
	if err != nil {
		return "", fmt.Errorf("auth: hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword reports whether password matches the stored bcrypt hash.
func CheckPassword(hash, password string) bool {
	plain := []byte(password)
	defer zeroBytes(plain)
	return bcrypt.CompareHashAndPassword([]byte(hash), plain) == nil
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
