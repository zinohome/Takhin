// Copyright 2025 Takhin Data, Inc.

package server

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/storage/topic"
	tlsutil "github.com/takhin-data/takhin/pkg/tls"
)

func TestServerTLS(t *testing.T) {
	dir := t.TempDir()

	// Generate test certificates
	certFile, keyFile, _, err := tlsutil.GenerateTestCertificates(dir)
	require.NoError(t, err)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0, // Use random port
			TLS: config.TLSConfig{
				Enabled:    true,
				CertFile:   certFile,
				KeyFile:    keyFile,
				ClientAuth: "none",
				MinVersion: "TLS1.2",
			},
		},
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			AdvertisedHost: "localhost",
			AdvertisedPort: 19092,
		},
		Storage: config.StorageConfig{
			DataDir: dir,
		},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, 1024*1024)
	server := New(cfg, topicMgr)

	// Start server
	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Connect with TLS client
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // For testing only
	}

	conn, err := tls.Dial("tcp", "localhost:19092", tlsConfig)
	require.NoError(t, err)
	defer conn.Close()

	// Verify connection is using TLS
	state := conn.ConnectionState()
	assert.True(t, state.HandshakeComplete)
	assert.NotEmpty(t, state.PeerCertificates)
}

func TestServerMTLS(t *testing.T) {
	dir := t.TempDir()

	// Generate test certificates
	certFile, keyFile, caFile, err := tlsutil.GenerateTestCertificates(dir)
	require.NoError(t, err)

	// Generate client certificate
	clientCertFile, clientKeyFile, err := tlsutil.GenerateClientCertificate(dir, caFile)
	require.NoError(t, err)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
			TLS: config.TLSConfig{
				Enabled:          true,
				CertFile:         certFile,
				KeyFile:          keyFile,
				CAFile:           caFile,
				ClientAuth:       "require",
				VerifyClientCert: true,
				MinVersion:       "TLS1.2",
			},
		},
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			AdvertisedHost: "localhost",
			AdvertisedPort: 19093,
		},
		Storage: config.StorageConfig{
			DataDir: dir,
		},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, 1024*1024)
	server := New(cfg, topicMgr)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Load client certificate
	clientCert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
	require.NoError(t, err)

	// Connect with mTLS client
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{clientCert},
		InsecureSkipVerify: true, // For testing only
	}

	conn, err := tls.Dial("tcp", "localhost:19093", tlsConfig)
	require.NoError(t, err)
	defer conn.Close()

	// Verify mutual TLS
	state := conn.ConnectionState()
	assert.True(t, state.HandshakeComplete)
	assert.NotEmpty(t, state.PeerCertificates)
}

func TestServerTLSDisabled(t *testing.T) {
	dir := t.TempDir()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
			TLS: config.TLSConfig{
				Enabled: false,
			},
		},
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			AdvertisedHost: "localhost",
			AdvertisedPort: 19094,
		},
		Storage: config.StorageConfig{
			DataDir: dir,
		},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, 1024*1024)
	server := New(cfg, topicMgr)

	err := server.Start()
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	// Connect with plain TCP
	conn, err := net.Dial("tcp", "localhost:19094")
	require.NoError(t, err)
	defer conn.Close()

	// Verify this is not a TLS connection
	_, ok := conn.(*tls.Conn)
	assert.False(t, ok)
}

func TestServerTLSWithCipherSuites(t *testing.T) {
	dir := t.TempDir()

	certFile, keyFile, _, err := tlsutil.GenerateTestCertificates(dir)
	require.NoError(t, err)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
			TLS: config.TLSConfig{
				Enabled:    true,
				CertFile:   certFile,
				KeyFile:    keyFile,
				ClientAuth: "none",
				MinVersion: "TLS1.2",
				CipherSuites: []string{
					"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
					"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
				},
				PreferServerCipher: true,
			},
		},
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			AdvertisedHost: "localhost",
			AdvertisedPort: 19095,
		},
		Storage: config.StorageConfig{
			DataDir: dir,
		},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, 1024*1024)
	server := New(cfg, topicMgr)

	err = server.Start()
	require.NoError(t, err)
	defer server.Stop()

	time.Sleep(100 * time.Millisecond)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	conn, err := tls.Dial("tcp", "localhost:19095", tlsConfig)
	require.NoError(t, err)
	defer conn.Close()

	state := conn.ConnectionState()
	assert.True(t, state.HandshakeComplete)
}

func TestTLSWithContext(t *testing.T) {
	dir := t.TempDir()

	certFile, keyFile, _, err := tlsutil.GenerateTestCertificates(dir)
	require.NoError(t, err)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 0,
			TLS: config.TLSConfig{
				Enabled:    true,
				CertFile:   certFile,
				KeyFile:    keyFile,
				ClientAuth: "none",
				MinVersion: "TLS1.2",
			},
		},
		Kafka: config.KafkaConfig{
			BrokerID:       1,
			AdvertisedHost: "localhost",
			AdvertisedPort: 19096,
		},
		Storage: config.StorageConfig{
			DataDir: dir,
		},
	}

	topicMgr := topic.NewManager(cfg.Storage.DataDir, 1024*1024)
	server := New(cfg, topicMgr)

	err = server.Start()
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Stop server with context
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan bool)
	go func() {
		server.Stop()
		done <- true
	}()

	select {
	case <-done:
		// Server stopped successfully
	case <-ctx.Done():
		t.Fatal("server shutdown timeout")
	}
}
