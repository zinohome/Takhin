// Copyright 2025 Takhin Data, Inc.

package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/takhin-data/takhin/pkg/config"
)

// LoadTLSConfig loads and configures TLS from config
func LoadTLSConfig(cfg *config.TLSConfig) (*tls.Config, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		PreferServerCipherSuites: cfg.PreferServerCipher,
		MinVersion:               parseTLSVersion(cfg.MinVersion),
	}

	// Configure cipher suites if specified
	if len(cfg.CipherSuites) > 0 {
		suites, err := parseCipherSuites(cfg.CipherSuites)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cipher suites: %w", err)
		}
		tlsConfig.CipherSuites = suites
	}

	// Configure client authentication
	switch cfg.ClientAuth {
	case "none":
		tlsConfig.ClientAuth = tls.NoClientCert
	case "request":
		tlsConfig.ClientAuth = tls.RequestClientCert
	case "require":
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	default:
		return nil, fmt.Errorf("invalid client auth mode: %s", cfg.ClientAuth)
	}

	// Load CA certificate for client verification
	if cfg.CAFile != "" {
		caCert, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.ClientCAs = caCertPool
		if cfg.VerifyClientCert {
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}

	return tlsConfig, nil
}

// parseTLSVersion converts string to TLS version constant
func parseTLSVersion(version string) uint16 {
	switch version {
	case "TLS1.0":
		return tls.VersionTLS10
	case "TLS1.1":
		return tls.VersionTLS11
	case "TLS1.2":
		return tls.VersionTLS12
	case "TLS1.3":
		return tls.VersionTLS13
	default:
		return tls.VersionTLS12 // Default to TLS 1.2
	}
}

// parseCipherSuites converts cipher suite names to constants
func parseCipherSuites(suites []string) ([]uint16, error) {
	cipherMap := map[string]uint16{
		"TLS_RSA_WITH_AES_128_CBC_SHA":            tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"TLS_RSA_WITH_AES_256_CBC_SHA":            tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_AES_128_GCM_SHA256":                  tls.TLS_AES_128_GCM_SHA256,
		"TLS_AES_256_GCM_SHA384":                  tls.TLS_AES_256_GCM_SHA384,
		"TLS_CHACHA20_POLY1305_SHA256":            tls.TLS_CHACHA20_POLY1305_SHA256,
	}

	result := make([]uint16, 0, len(suites))
	for _, name := range suites {
		if cipher, ok := cipherMap[name]; ok {
			result = append(result, cipher)
		} else {
			return nil, fmt.Errorf("unknown cipher suite: %s", name)
		}
	}

	return result, nil
}

// VerifyCertificate verifies a certificate against a CA
func VerifyCertificate(cert *x509.Certificate, caFile string) error {
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to parse CA certificate")
	}

	opts := x509.VerifyOptions{
		Roots: caCertPool,
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("certificate verification failed: %w", err)
	}

	return nil
}
