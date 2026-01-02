// Copyright 2025 Takhin Data, Inc.

package tls

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/config"
)

func TestLoadTLSConfig(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *config.TLSConfig
		setupFunc func(t *testing.T, dir string) *config.TLSConfig
		wantErr   bool
		validate  func(t *testing.T, tlsConfig *tls.Config)
	}{
		{
			name: "disabled TLS",
			cfg: &config.TLSConfig{
				Enabled: false,
			},
			wantErr: false,
			validate: func(t *testing.T, tlsConfig *tls.Config) {
				assert.Nil(t, tlsConfig)
			},
		},
		{
			name: "basic TLS with valid certificates",
			setupFunc: func(t *testing.T, dir string) *config.TLSConfig {
				certFile, keyFile, _, err := GenerateTestCertificates(dir)
				require.NoError(t, err)
				return &config.TLSConfig{
					Enabled:    true,
					CertFile:   certFile,
					KeyFile:    keyFile,
					ClientAuth: "none",
					MinVersion: "TLS1.2",
				}
			},
			wantErr: false,
			validate: func(t *testing.T, tlsConfig *tls.Config) {
				assert.NotNil(t, tlsConfig)
				assert.Equal(t, tls.NoClientCert, tlsConfig.ClientAuth)
				assert.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MinVersion)
				assert.Len(t, tlsConfig.Certificates, 1)
			},
		},
		{
			name: "TLS with client authentication",
			setupFunc: func(t *testing.T, dir string) *config.TLSConfig {
				certFile, keyFile, caFile, err := GenerateTestCertificates(dir)
				require.NoError(t, err)
				return &config.TLSConfig{
					Enabled:    true,
					CertFile:   certFile,
					KeyFile:    keyFile,
					CAFile:     caFile,
					ClientAuth: "require",
					MinVersion: "TLS1.3",
				}
			},
			wantErr: false,
			validate: func(t *testing.T, tlsConfig *tls.Config) {
				assert.NotNil(t, tlsConfig)
				assert.Equal(t, tls.RequireAndVerifyClientCert, tlsConfig.ClientAuth)
				assert.Equal(t, uint16(tls.VersionTLS13), tlsConfig.MinVersion)
				assert.NotNil(t, tlsConfig.ClientCAs)
			},
		},
		{
			name: "TLS with custom cipher suites",
			setupFunc: func(t *testing.T, dir string) *config.TLSConfig {
				certFile, keyFile, _, err := GenerateTestCertificates(dir)
				require.NoError(t, err)
				return &config.TLSConfig{
					Enabled:    true,
					CertFile:   certFile,
					KeyFile:    keyFile,
					ClientAuth: "none",
					MinVersion: "TLS1.2",
					CipherSuites: []string{
						"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
						"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
					},
				}
			},
			wantErr: false,
			validate: func(t *testing.T, tlsConfig *tls.Config) {
				assert.NotNil(t, tlsConfig)
				assert.Len(t, tlsConfig.CipherSuites, 2)
			},
		},
		{
			name: "TLS with invalid cipher suite",
			setupFunc: func(t *testing.T, dir string) *config.TLSConfig {
				certFile, keyFile, _, err := GenerateTestCertificates(dir)
				require.NoError(t, err)
				return &config.TLSConfig{
					Enabled:      true,
					CertFile:     certFile,
					KeyFile:      keyFile,
					ClientAuth:   "none",
					CipherSuites: []string{"INVALID_CIPHER"},
				}
			},
			wantErr: true,
		},
		{
			name: "missing certificate file",
			cfg: &config.TLSConfig{
				Enabled:    true,
				CertFile:   "/nonexistent/cert.pem",
				KeyFile:    "/nonexistent/key.pem",
				ClientAuth: "none",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			cfg := tt.cfg
			if tt.setupFunc != nil {
				cfg = tt.setupFunc(t, dir)
			}

			tlsConfig, err := LoadTLSConfig(cfg)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, tlsConfig)
			}
		})
	}
}

func TestParseTLSVersion(t *testing.T) {
	tests := []struct {
		version string
		want    uint16
	}{
		{"TLS1.0", tls.VersionTLS10},
		{"TLS1.1", tls.VersionTLS11},
		{"TLS1.2", tls.VersionTLS12},
		{"TLS1.3", tls.VersionTLS13},
		{"invalid", tls.VersionTLS12}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := parseTLSVersion(tt.version)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseCipherSuites(t *testing.T) {
	tests := []struct {
		name    string
		suites  []string
		wantErr bool
		wantLen int
	}{
		{
			name: "valid cipher suites",
			suites: []string{
				"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
				"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			},
			wantErr: false,
			wantLen: 2,
		},
		{
			name:    "TLS 1.3 cipher suites",
			suites:  []string{"TLS_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384"},
			wantErr: false,
			wantLen: 2,
		},
		{
			name:    "invalid cipher suite",
			suites:  []string{"INVALID_CIPHER"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCipherSuites(tt.suites)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result, tt.wantLen)
		})
	}
}

func TestGenerateTestCertificates(t *testing.T) {
	dir := t.TempDir()

	certFile, keyFile, caFile, err := GenerateTestCertificates(dir)
	require.NoError(t, err)

	// Verify files exist
	assert.FileExists(t, certFile)
	assert.FileExists(t, keyFile)
	assert.FileExists(t, caFile)

	// Verify we can load the certificate
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	require.NoError(t, err)
	assert.NotNil(t, cert)

	// Verify CA certificate
	caData, err := os.ReadFile(caFile)
	require.NoError(t, err)
	assert.NotEmpty(t, caData)
}

func TestGenerateClientCertificate(t *testing.T) {
	dir := t.TempDir()

	// First generate server certificates
	_, _, caFile, err := GenerateTestCertificates(dir)
	require.NoError(t, err)

	// Generate client certificate
	clientCertFile, clientKeyFile, err := GenerateClientCertificate(dir, caFile)
	require.NoError(t, err)

	// Verify files exist
	assert.FileExists(t, clientCertFile)
	assert.FileExists(t, clientKeyFile)

	// Verify we can load the certificate
	cert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
	require.NoError(t, err)
	assert.NotNil(t, cert)
}

func TestVerifyCertificate(t *testing.T) {
	dir := t.TempDir()

	certFile, _, caFile, err := GenerateTestCertificates(dir)
	require.NoError(t, err)

	// Load the certificate
	certData, err := os.ReadFile(certFile)
	require.NoError(t, err)

	// Parse certificate
	block, _ := pem.Decode(certData)
	require.NotNil(t, block)

	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	// Verify certificate
	err = VerifyCertificate(cert, caFile)
	assert.NoError(t, err)
}

func TestTLSConfigWithMTLS(t *testing.T) {
	dir := t.TempDir()

	certFile, keyFile, caFile, err := GenerateTestCertificates(dir)
	require.NoError(t, err)

	cfg := &config.TLSConfig{
		Enabled:          true,
		CertFile:         certFile,
		KeyFile:          keyFile,
		CAFile:           caFile,
		ClientAuth:       "require",
		VerifyClientCert: true,
		MinVersion:       "TLS1.2",
	}

	tlsConfig, err := LoadTLSConfig(cfg)
	require.NoError(t, err)
	assert.NotNil(t, tlsConfig)
	assert.Equal(t, tls.RequireAndVerifyClientCert, tlsConfig.ClientAuth)
	assert.NotNil(t, tlsConfig.ClientCAs)
}
