// Copyright 2025 Takhin Data, Inc.

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
	"os"
	"path/filepath"
	"time"
)

// GenerateTestCertificates generates test certificates for TLS testing
func GenerateTestCertificates(dir string) (certFile, keyFile, caFile string, err error) {
	// Generate CA
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate CA key: %w", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Takhin Test CA"},
			CommonName:   "Takhin Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create CA certificate: %w", err)
	}

	// Write CA certificate
	caFile = filepath.Join(dir, "ca.pem")
	caOut, err := os.Create(caFile)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create CA file: %w", err)
	}
	defer caOut.Close()

	if err := pem.Encode(caOut, &pem.Block{Type: "CERTIFICATE", Bytes: caCertBytes}); err != nil {
		return "", "", "", fmt.Errorf("failed to write CA certificate: %w", err)
	}

	// Write CA key (for client certificate generation)
	caKeyFile := filepath.Join(dir, "ca-key.pem")
	caKeyOut, err := os.Create(caKeyFile)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create CA key file: %w", err)
	}
	defer caKeyOut.Close()

	caKeyBytes, err := x509.MarshalECPrivateKey(caKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to marshal CA key: %w", err)
	}

	if err := pem.Encode(caKeyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: caKeyBytes}); err != nil {
		return "", "", "", fmt.Errorf("failed to write CA key: %w", err)
	}

	// Generate server key
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate server key: %w", err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Takhin Test Server"},
			CommonName:   "localhost",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"localhost"},
	}

	serverCertBytes, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create server certificate: %w", err)
	}

	// Write server certificate
	certFile = filepath.Join(dir, "server.pem")
	certOut, err := os.Create(certFile)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create cert file: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: serverCertBytes}); err != nil {
		return "", "", "", fmt.Errorf("failed to write server certificate: %w", err)
	}

	// Write server key
	keyFile = filepath.Join(dir, "server-key.pem")
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyOut.Close()

	keyBytes, err := x509.MarshalECPrivateKey(serverKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to marshal server key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return "", "", "", fmt.Errorf("failed to write server key: %w", err)
	}

	return certFile, keyFile, caFile, nil
}

// GenerateClientCertificate generates a client certificate for mTLS testing
func GenerateClientCertificate(dir, caFile string) (certFile, keyFile string, err error) {
	// Read CA certificate
	caData, err := os.ReadFile(caFile)
	if err != nil {
		return "", "", fmt.Errorf("failed to read CA file: %w", err)
	}

	block, _ := pem.Decode(caData)
	if block == nil {
		return "", "", fmt.Errorf("failed to decode CA certificate")
	}

	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Read CA key
	caKeyFile := filepath.Join(dir, "ca-key.pem")
	caKeyData, err := os.ReadFile(caKeyFile)
	if err != nil {
		return "", "", fmt.Errorf("failed to read CA key file: %w", err)
	}

	caKeyBlock, _ := pem.Decode(caKeyData)
	if caKeyBlock == nil {
		return "", "", fmt.Errorf("failed to decode CA key")
	}

	caKey, err := x509.ParseECPrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse CA key: %w", err)
	}

	// Generate client key
	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate client key: %w", err)
	}

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			Organization: []string{"Takhin Test Client"},
			CommonName:   "test-client",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// Sign with CA key
	clientCertBytes, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create client certificate: %w", err)
	}

	// Write client certificate
	certFile = filepath.Join(dir, "client.pem")
	certOut, err := os.Create(certFile)
	if err != nil {
		return "", "", fmt.Errorf("failed to create client cert file: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: clientCertBytes}); err != nil {
		return "", "", fmt.Errorf("failed to write client certificate: %w", err)
	}

	// Write client key
	keyFile = filepath.Join(dir, "client-key.pem")
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return "", "", fmt.Errorf("failed to create client key file: %w", err)
	}
	defer keyOut.Close()

	keyBytes, err := x509.MarshalECPrivateKey(clientKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal client key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return "", "", fmt.Errorf("failed to write client key: %w", err)
	}

	return certFile, keyFile, nil
}
