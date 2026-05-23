// Package tls handles self-signed certificate generation for the relay.
package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// EnsureCerts creates a self-signed TLS certificate and key under
// <dataDir>/tls/ if they do not already exist.
// It returns the paths to the certificate and key files.
func EnsureCerts(dataDir string) (certFile, keyFile string, err error) {
	tlsDir := filepath.Join(dataDir, "tls")
	if err := os.MkdirAll(tlsDir, 0o700); err != nil {
		return "", "", fmt.Errorf("tls: mkdir %s: %w", tlsDir, err)
	}

	certFile = filepath.Join(tlsDir, "server.crt")
	keyFile = filepath.Join(tlsDir, "server.key")

	// Return early if both files already exist.
	if fileExists(certFile) && fileExists(keyFile) {
		return certFile, keyFile, nil
	}

	if err := generate(certFile, keyFile); err != nil {
		return "", "", err
	}
	return certFile, keyFile, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func generate(certFile, keyFile string) error {
	// Generate ECDSA P-256 key pair.
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("tls: generate key: %w", err)
	}

	// Build the certificate template.
	serialMax := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialMax)
	if err != nil {
		return fmt.Errorf("tls: serial: %w", err)
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"taildog relay"},
			CommonName:   "taildog-relay",
		},
		NotBefore:             now.Add(-time.Minute), // slight backdating for clock skew
		NotAfter:              now.Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		DNSNames:              []string{"localhost"},
	}

	// Self-sign.
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("tls: create cert: %w", err)
	}

	// Write certificate PEM.
	if err := writePEM(certFile, "CERTIFICATE", certDER, 0o644); err != nil {
		return err
	}

	// Marshal private key.
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("tls: marshal key: %w", err)
	}

	// Write key PEM (owner-read only).
	if err := writePEM(keyFile, "EC PRIVATE KEY", keyDER, 0o600); err != nil {
		return err
	}

	return nil
}

func writePEM(path, blockType string, der []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("tls: open %s: %w", path, err)
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{Type: blockType, Bytes: der}); err != nil {
		return fmt.Errorf("tls: write %s: %w", path, err)
	}
	return nil
}
