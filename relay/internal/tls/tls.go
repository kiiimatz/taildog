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
// <dataDir>/tls/ if they do not already exist (or if they were generated with
// the old CA-cert format that causes TLS handshake failures on some clients).
// It returns the paths to the certificate and key files.
func EnsureCerts(dataDir string) (certFile, keyFile string, err error) {
	tlsDir := filepath.Join(dataDir, "tls")
	if err := os.MkdirAll(tlsDir, 0o700); err != nil {
		return "", "", fmt.Errorf("tls: mkdir %s: %w", tlsDir, err)
	}

	certFile = filepath.Join(tlsDir, "server.crt")
	keyFile = filepath.Join(tlsDir, "server.key")

	// Return early only if both files exist AND the cert is the new format.
	// Old certs had IsCA:true which causes TLS handshake errors.
	if fileExists(certFile) && fileExists(keyFile) && !isLegacyCert(certFile) {
		return certFile, keyFile, nil
	}

	if err := generate(certFile, keyFile); err != nil {
		return "", "", err
	}
	return certFile, keyFile, nil
}

// isLegacyCert returns true if the certificate at path was generated with the
// old IsCA:true template (which causes "tls: bad certificate" on clients).
func isLegacyCert(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return true
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return true
	}
	return cert.IsCA
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// localIPs collects all unicast IP addresses on the machine so they can be
// embedded in the certificate's Subject Alternative Names.
func localIPs() []net.IP {
	ips := []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}
	seen := map[string]bool{"127.0.0.1": true, "::1": true}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			s := ip.String()
			if !seen[s] {
				seen[s] = true
				ips = append(ips, ip)
			}
		}
	}
	return ips
}

func generate(certFile, keyFile string) error {
	// Generate ECDSA P-256 key pair.
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("tls: generate key: %w", err)
	}

	serialMax := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialMax)
	if err != nil {
		return fmt.Errorf("tls: serial: %w", err)
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"renode relay"},
			CommonName:   "renode-relay",
		},
		NotBefore: now.Add(-time.Minute), // slight backdating for clock skew
		NotAfter:  now.Add(10 * 365 * 24 * time.Hour),

		// Server leaf certificate — NOT a CA.
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,

		// Include all local IPs so the cert is valid regardless of which
		// interface the client reaches the relay through.
		IPAddresses: localIPs(),
		DNSNames:    []string{"localhost"},
	}

	// Self-sign.
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("tls: create cert: %w", err)
	}

	if err := writePEM(certFile, "CERTIFICATE", certDER, 0o644); err != nil {
		return err
	}

	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("tls: marshal key: %w", err)
	}

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
