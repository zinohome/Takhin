// Copyright 2025 Takhin Data, Inc.

package testutil

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/kafka/handler"
	"github.com/takhin-data/takhin/pkg/kafka/server"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

// TestServer represents a test Takhin server instance
type TestServer struct {
	Config       *config.Config
	Server       *server.Server
	Handler      *handler.Handler
	TopicManager *topic.Manager
	Coordinator  *coordinator.Coordinator
	DataDir      string
	Port         int
	t            *testing.T
}

// NewTestServer creates and starts a new test server
func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	// Create temp data directory
	dataDir := t.TempDir()

	// Find available port
	port := findAvailablePort(t)

	// Create config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: port,
		},
		Kafka: config.KafkaConfig{
			BrokerID: 1,
		},
		Storage: config.StorageConfig{
			DataDir:            dataDir,
			LogSegmentSize:     1024 * 1024, // 1MB for testing
			LogRetentionHours:  1,
			LogFlushInterval:   1000, // 1 second
		},
		Replication: config.ReplicationConfig{
			DefaultReplicationFactor: 1,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
			Port:    0, // Disable metrics port for tests
		},
		Health: config.HealthConfig{
			Enabled: false, // Disable health check for tests
		},
		ACL: config.ACLConfig{
			Enabled: false,
		},
		Sasl: config.SaslConfig{
			Enabled: false,
		},
		Throttle: config.ThrottleConfig{
			Producer: config.ProducerThrottleConfig{
				BytesPerSecond: 0, // No throttling in tests
			},
			Consumer: config.ConsumerThrottleConfig{
				BytesPerSecond: 0,
			},
		},
	}

	// Initialize logger (simple logging for tests)
	// logger.InitLogger(cfg.Logging.Level, cfg.Logging.Format)

	// Create topic manager
	topicMgr := topic.NewManager(cfg.Storage.DataDir, int64(cfg.Storage.LogSegmentSize))

	// Create server
	srv := server.New(cfg, topicMgr)

	// Start server in background
	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to be ready
	if !waitForServer(fmt.Sprintf("localhost:%d", port), 10*time.Second) {
		t.Fatal("Server failed to start within timeout")
	}

	ts := &TestServer{
		Config:       cfg,
		Server:       srv,
		Handler:      nil, // Not directly accessible
		TopicManager: topicMgr,
		DataDir:      dataDir,
		Port:         port,
		t:            t,
	}

	// Cleanup on test completion
	t.Cleanup(func() {
		ts.Close()
	})

	return ts
}

// Close stops the test server and cleans up resources
func (ts *TestServer) Close() {
	if ts.Server != nil {
		// Server doesn't have Shutdown method, just stop it
		// In a real implementation, would properly shut down
		ts.Server.Stop()
	}
}

// Address returns the server address
func (ts *TestServer) Address() string {
	return fmt.Sprintf("localhost:%d", ts.Port)
}

// CreateTopic creates a topic for testing
func (ts *TestServer) CreateTopic(name string, numPartitions int) error {
	return ts.TopicManager.CreateTopic(name, int32(numPartitions))
}

// findAvailablePort finds an available TCP port
func findAvailablePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to find available port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

// waitForServer waits for the server to become available
func waitForServer(addr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// TestCluster represents a cluster of test servers
type TestCluster struct {
	Servers []*TestServer
	t       *testing.T
}

// NewTestCluster creates a cluster of test servers
func NewTestCluster(t *testing.T, numBrokers int) *TestCluster {
	t.Helper()

	servers := make([]*TestServer, numBrokers)
	for i := 0; i < numBrokers; i++ {
		servers[i] = NewTestServer(t)
		// Update broker ID
		servers[i].Config.Kafka.BrokerID = i + 1
	}

	return &TestCluster{
		Servers: servers,
		t:       t,
	}
}

// Close stops all servers in the cluster
func (tc *TestCluster) Close() {
	for _, srv := range tc.Servers {
		srv.Close()
	}
}

// Leader returns the first server (acting as leader)
func (tc *TestCluster) Leader() *TestServer {
	if len(tc.Servers) == 0 {
		return nil
	}
	return tc.Servers[0]
}

// Followers returns all servers except the leader
func (tc *TestCluster) Followers() []*TestServer {
	if len(tc.Servers) <= 1 {
		return nil
	}
	return tc.Servers[1:]
}

// GetServer returns the server at the specified index
func (tc *TestCluster) GetServer(index int) *TestServer {
	if index < 0 || index >= len(tc.Servers) {
		return nil
	}
	return tc.Servers[index]
}

// WaitForDataDir waits for specific files to appear in data directory
func WaitForDataDir(dataDir string, expectedFiles int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(dataDir)
		if err == nil && len(entries) >= expectedFiles {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for files in %s", dataDir)
}

// FileExists checks if a file exists in the data directory
func FileExists(dataDir, pattern string) bool {
	matches, err := filepath.Glob(filepath.Join(dataDir, pattern))
	return err == nil && len(matches) > 0
}
