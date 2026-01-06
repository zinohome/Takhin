// Copyright 2025 Takhin Data, Inc.

package config

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config represents the application configuration
type Config struct {
	Server      ServerConfig      `koanf:"server"`
	Kafka       KafkaConfig       `koanf:"kafka"`
	Storage     StorageConfig     `koanf:"storage"`
	Replication ReplicationConfig `koanf:"replication"`
	Raft        RaftConfig        `koanf:"raft"`
	Logging     LoggingConfig     `koanf:"logging"`
	Metrics     MetricsConfig     `koanf:"metrics"`
	ACL         ACLConfig         `koanf:"acl"`
	Throttle    ThrottleConfig    `koanf:"throttle"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host string    `koanf:"host"`
	Port int       `koanf:"port"`
	TLS  TLSConfig `koanf:"tls"`
}

// TLSConfig holds TLS/SSL configuration
type TLSConfig struct {
	Enabled            bool     `koanf:"enabled"`
	CertFile           string   `koanf:"cert.file"`
	KeyFile            string   `koanf:"key.file"`
	CAFile             string   `koanf:"ca.file"`
	ClientAuth         string   `koanf:"client.auth"`        // none, request, require
	VerifyClientCert   bool     `koanf:"verify.client.cert"` // For mTLS
	MinVersion         string   `koanf:"min.version"`        // TLS1.2, TLS1.3
	CipherSuites       []string `koanf:"cipher.suites"`
	PreferServerCipher bool     `koanf:"prefer.server.cipher"`
}

// KafkaConfig holds Kafka protocol configuration
type KafkaConfig struct {
	BrokerID          int         `koanf:"broker.id"`
	Listeners         []string    `koanf:"listeners"`
	AdvertisedHost    string      `koanf:"advertised.host"`
	AdvertisedPort    int         `koanf:"advertised.port"`
	MaxMessageBytes   int         `koanf:"max.message.bytes"`
	MaxConnections    int         `koanf:"max.connections"`
	RequestTimeout    int         `koanf:"request.timeout.ms"`
	ConnectionTimeout int         `koanf:"connection.timeout.ms"`
	ClusterBrokers    []int       `koanf:"cluster.brokers"` // List of all broker IDs in cluster
	Batch             BatchConfig `koanf:"batch"`           // Batch processing configuration
}

// BatchConfig holds batch processing configuration
type BatchConfig struct {
	MaxSize         int    `koanf:"max.size"`          // Max records per batch (0=unlimited)
	MaxBytes        int    `koanf:"max.bytes"`         // Max bytes per batch
	LingerMs        int    `koanf:"linger.ms"`         // Time to wait for batching
	AdaptiveEnabled bool   `koanf:"adaptive.enabled"`  // Enable adaptive batch sizing
	AdaptiveMinSize int    `koanf:"adaptive.min.size"` // Min batch size for adaptive mode
	AdaptiveMaxSize int    `koanf:"adaptive.max.size"` // Max batch size for adaptive mode
	CompressionType string `koanf:"compression.type"`  // Compression: none, gzip, snappy, lz4, zstd
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	DataDir            string           `koanf:"data.dir"`
	LogSegmentSize     int64            `koanf:"log.segment.size"`
	LogRetentionHours  int              `koanf:"log.retention.hours"`
	LogRetentionBytes  int64            `koanf:"log.retention.bytes"`
	LogCleanupInterval int              `koanf:"log.cleanup.interval.ms"`
	LogFlushInterval   int              `koanf:"log.flush.interval.ms"`
	LogFlushMessages   int              `koanf:"log.flush.messages"`
	CleanerEnabled     bool             `koanf:"cleaner.enabled"`
	CompactionInterval int              `koanf:"compaction.interval.ms"`
	MinCleanableRatio  float64          `koanf:"compaction.min.cleanable.ratio"`
	Encryption         EncryptionConfig `koanf:"encryption"`
}

// EncryptionConfig holds encryption at rest configuration
type EncryptionConfig struct {
	Enabled   bool   `koanf:"enabled"`
	Algorithm string `koanf:"algorithm"` // none, aes-128-gcm, aes-256-gcm, chacha20-poly1305
	KeyDir    string `koanf:"key.dir"`
}

// ReplicationConfig holds replication configuration
type ReplicationConfig struct {
	DefaultReplicationFactor int16 `koanf:"default.replication.factor"`
	ReplicaLagTimeMaxMs      int64 `koanf:"replica.lag.time.max.ms"`
	ReplicaFetchWaitMaxMs    int   `koanf:"replica.fetch.wait.max.ms"`
	ReplicaFetchMaxBytes     int   `koanf:"replica.fetch.max.bytes"`
}

// RaftConfig holds Raft consensus configuration
type RaftConfig struct {
	HeartbeatTimeoutMs   int  `koanf:"heartbeat.timeout.ms"`
	ElectionTimeoutMs    int  `koanf:"election.timeout.ms"`
	LeaderLeaseTimeoutMs int  `koanf:"leader.lease.timeout.ms"`
	CommitTimeoutMs      int  `koanf:"commit.timeout.ms"`
	SnapshotIntervalMs   int  `koanf:"snapshot.interval.ms"`
	SnapshotThreshold    int  `koanf:"snapshot.threshold"`
	PreVoteEnabled       bool `koanf:"prevote.enabled"`
	MaxAppendEntries     int  `koanf:"max.append.entries"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled bool   `koanf:"enabled"`
	Host    string `koanf:"host"`
	Port    int    `koanf:"port"`
	Path    string `koanf:"path"`
}

// ACLConfig holds ACL configuration
type ACLConfig struct {
	Enabled bool `koanf:"enabled"`
}

// ThrottleConfig holds throttle configuration
type ThrottleConfig struct {
	Producer ProducerThrottleConfig `koanf:"producer"`
	Consumer ConsumerThrottleConfig `koanf:"consumer"`
	Dynamic  DynamicThrottleConfig  `koanf:"dynamic"`
}

// ProducerThrottleConfig holds producer throttle configuration
type ProducerThrottleConfig struct {
	BytesPerSecond int64 `koanf:"bytes.per.second"`
	Burst          int   `koanf:"burst"`
}

// ConsumerThrottleConfig holds consumer throttle configuration
type ConsumerThrottleConfig struct {
	BytesPerSecond int64 `koanf:"bytes.per.second"`
	Burst          int   `koanf:"burst"`
}

// DynamicThrottleConfig holds dynamic throttle adjustment configuration
type DynamicThrottleConfig struct {
	Enabled         bool    `koanf:"enabled"`
	CheckIntervalMs int     `koanf:"check.interval.ms"`
	MinRate         int64   `koanf:"min.rate"`
	MaxRate         int64   `koanf:"max.rate"`
	TargetUtilPct   float64 `koanf:"target.util.pct"`
	AdjustmentStep  float64 `koanf:"adjustment.step"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	k := koanf.New(".")

	if configPath != "" {
		if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("load config file: %w", err)
		}
		slog.Info("loaded config from file", "path", configPath)
	}

	if err := k.Load(env.Provider("TAKHIN_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "TAKHIN_")), "_", ".", -1)
	}), nil); err != nil {
		return nil, fmt.Errorf("load environment variables: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	setDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(cfg *Config) {
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 9092
	}

	if cfg.Kafka.BrokerID == 0 {
		cfg.Kafka.BrokerID = 1
	}
	if cfg.Kafka.AdvertisedHost == "" {
		cfg.Kafka.AdvertisedHost = "localhost"
	}
	if cfg.Kafka.AdvertisedPort == 0 {
		cfg.Kafka.AdvertisedPort = cfg.Server.Port
	}
	if cfg.Kafka.MaxMessageBytes == 0 {
		cfg.Kafka.MaxMessageBytes = 1024 * 1024
	}
	if cfg.Kafka.MaxConnections == 0 {
		cfg.Kafka.MaxConnections = 1000
	}
	if cfg.Kafka.RequestTimeout == 0 {
		cfg.Kafka.RequestTimeout = 30000
	}
	if cfg.Kafka.ConnectionTimeout == 0 {
		cfg.Kafka.ConnectionTimeout = 60000
	}

	// Batch processing defaults
	if cfg.Kafka.Batch.MaxBytes == 0 {
		cfg.Kafka.Batch.MaxBytes = 1048576 // 1MB
	}
	if cfg.Kafka.Batch.LingerMs == 0 {
		cfg.Kafka.Batch.LingerMs = 10 // 10ms
	}
	if cfg.Kafka.Batch.AdaptiveMinSize == 0 {
		cfg.Kafka.Batch.AdaptiveMinSize = 16
	}
	if cfg.Kafka.Batch.AdaptiveMaxSize == 0 {
		cfg.Kafka.Batch.AdaptiveMaxSize = 10000
	}
	if cfg.Kafka.Batch.CompressionType == "" {
		cfg.Kafka.Batch.CompressionType = "none"
	}

	if cfg.Storage.DataDir == "" {
		cfg.Storage.DataDir = "/tmp/takhin-data"
	}
	if cfg.Storage.LogSegmentSize == 0 {
		cfg.Storage.LogSegmentSize = 1024 * 1024 * 1024
	}
	if cfg.Storage.LogRetentionHours == 0 {
		cfg.Storage.LogRetentionHours = 168
	}
	if cfg.Storage.LogCleanupInterval == 0 {
		cfg.Storage.LogCleanupInterval = 300000
	}
	if cfg.Storage.LogFlushInterval == 0 {
		cfg.Storage.LogFlushInterval = 1000
	}
	if cfg.Storage.LogFlushMessages == 0 {
		cfg.Storage.LogFlushMessages = 10000
	}
	// Cleaner defaults
	// CleanerEnabled defaults to false if not set (explicit opt-in)
	if cfg.Storage.CompactionInterval == 0 {
		cfg.Storage.CompactionInterval = 600000 // 10 minutes
	}
	if cfg.Storage.MinCleanableRatio == 0 {
		cfg.Storage.MinCleanableRatio = 0.5 // 50%
	}
	
	// Encryption defaults
	if cfg.Storage.Encryption.Algorithm == "" {
		cfg.Storage.Encryption.Algorithm = "none"
	}
	if cfg.Storage.Encryption.KeyDir == "" {
		cfg.Storage.Encryption.KeyDir = filepath.Join(cfg.Storage.DataDir, "keys")
	}
	
	// Replication defaults
	if cfg.Replication.DefaultReplicationFactor == 0 {
		cfg.Replication.DefaultReplicationFactor = 1 // Single replica by default
	}
	if cfg.Replication.ReplicaLagTimeMaxMs == 0 {
		cfg.Replication.ReplicaLagTimeMaxMs = 10000 // 10 seconds
	}
	if cfg.Replication.ReplicaFetchWaitMaxMs == 0 {
		cfg.Replication.ReplicaFetchWaitMaxMs = 500 // 500ms
	}
	if cfg.Replication.ReplicaFetchMaxBytes == 0 {
		cfg.Replication.ReplicaFetchMaxBytes = 1048576 // 1MB
	}

	// Raft defaults - optimized for fast leader election
	if cfg.Raft.HeartbeatTimeoutMs == 0 {
		cfg.Raft.HeartbeatTimeoutMs = 1000 // 1 second
	}
	if cfg.Raft.ElectionTimeoutMs == 0 {
		cfg.Raft.ElectionTimeoutMs = 3000 // 3 seconds
	}
	if cfg.Raft.LeaderLeaseTimeoutMs == 0 {
		cfg.Raft.LeaderLeaseTimeoutMs = 500 // 500ms
	}
	if cfg.Raft.CommitTimeoutMs == 0 {
		cfg.Raft.CommitTimeoutMs = 50 // 50ms
	}
	if cfg.Raft.SnapshotIntervalMs == 0 {
		cfg.Raft.SnapshotIntervalMs = 120000 // 2 minutes
	}
	if cfg.Raft.SnapshotThreshold == 0 {
		cfg.Raft.SnapshotThreshold = 8192 // 8192 log entries
	}
	if cfg.Raft.MaxAppendEntries == 0 {
		cfg.Raft.MaxAppendEntries = 64
	}
	// PreVote is enabled by default (explicit opt-out)
	// No need to set default as false is the zero value

	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}

	if cfg.Metrics.Path == "" {
		cfg.Metrics.Path = "/metrics"
	}
	if cfg.Metrics.Port == 0 {
		cfg.Metrics.Port = 9090
	}

	// TLS defaults
	if cfg.Server.TLS.ClientAuth == "" {
		cfg.Server.TLS.ClientAuth = "none"
	}
	if cfg.Server.TLS.MinVersion == "" {
		cfg.Server.TLS.MinVersion = "TLS1.2"
	}

	// Throttle defaults
	if cfg.Throttle.Producer.BytesPerSecond == 0 {
		cfg.Throttle.Producer.BytesPerSecond = 10 * 1024 * 1024 // 10 MB/s
	}
	if cfg.Throttle.Producer.Burst == 0 {
		cfg.Throttle.Producer.Burst = int(cfg.Throttle.Producer.BytesPerSecond * 2)
	}
	if cfg.Throttle.Consumer.BytesPerSecond == 0 {
		cfg.Throttle.Consumer.BytesPerSecond = 10 * 1024 * 1024 // 10 MB/s
	}
	if cfg.Throttle.Consumer.Burst == 0 {
		cfg.Throttle.Consumer.Burst = int(cfg.Throttle.Consumer.BytesPerSecond * 2)
	}
	if cfg.Throttle.Dynamic.CheckIntervalMs == 0 {
		cfg.Throttle.Dynamic.CheckIntervalMs = 5000 // 5 seconds
	}
	if cfg.Throttle.Dynamic.MinRate == 0 {
		cfg.Throttle.Dynamic.MinRate = 1024 * 1024 // 1 MB/s
	}
	if cfg.Throttle.Dynamic.MaxRate == 0 {
		cfg.Throttle.Dynamic.MaxRate = 100 * 1024 * 1024 // 100 MB/s
	}
	if cfg.Throttle.Dynamic.TargetUtilPct == 0 {
		cfg.Throttle.Dynamic.TargetUtilPct = 0.80 // 80%
	}
	if cfg.Throttle.Dynamic.AdjustmentStep == 0 {
		cfg.Throttle.Dynamic.AdjustmentStep = 0.10 // 10%
	}
}

func validate(cfg *Config) error {
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	if cfg.Kafka.BrokerID < 0 {
		return fmt.Errorf("invalid broker ID: %d", cfg.Kafka.BrokerID)
	}

	// Validate cluster brokers
	if len(cfg.Kafka.ClusterBrokers) > 0 {
		// Check that current broker is in the list
		found := false
		for _, brokerID := range cfg.Kafka.ClusterBrokers {
			if brokerID == cfg.Kafka.BrokerID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("current broker ID %d not found in cluster.brokers list", cfg.Kafka.BrokerID)
		}
	}

	if cfg.Storage.LogSegmentSize <= 0 {
		return fmt.Errorf("invalid log segment size: %d", cfg.Storage.LogSegmentSize)
	}

	// Validate Raft configuration (only if any Raft values are set)
	if cfg.Raft.HeartbeatTimeoutMs > 0 {
		if cfg.Raft.HeartbeatTimeoutMs < 100 {
			return fmt.Errorf("invalid heartbeat timeout: %dms (minimum 100ms)", cfg.Raft.HeartbeatTimeoutMs)
		}
		if cfg.Raft.ElectionTimeoutMs < cfg.Raft.HeartbeatTimeoutMs {
			return fmt.Errorf("election timeout (%dms) must be >= heartbeat timeout (%dms)",
				cfg.Raft.ElectionTimeoutMs, cfg.Raft.HeartbeatTimeoutMs)
		}
		if cfg.Raft.LeaderLeaseTimeoutMs > 0 && cfg.Raft.LeaderLeaseTimeoutMs < 100 {
			return fmt.Errorf("invalid leader lease timeout: %dms (minimum 100ms)", cfg.Raft.LeaderLeaseTimeoutMs)
		}
	}

	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[cfg.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", cfg.Logging.Level)
	}

	// Validate TLS configuration
	if cfg.Server.TLS.Enabled {
		if cfg.Server.TLS.CertFile == "" {
			return fmt.Errorf("TLS cert file is required when TLS is enabled")
		}
		if cfg.Server.TLS.KeyFile == "" {
			return fmt.Errorf("TLS key file is required when TLS is enabled")
		}

		validClientAuth := map[string]bool{
			"none":    true,
			"request": true,
			"require": true,
		}
		if !validClientAuth[cfg.Server.TLS.ClientAuth] {
			return fmt.Errorf("invalid client auth mode: %s (must be none, request, or require)", cfg.Server.TLS.ClientAuth)
		}

		validMinVersion := map[string]bool{
			"TLS1.0": true,
			"TLS1.1": true,
			"TLS1.2": true,
			"TLS1.3": true,
		}
		if !validMinVersion[cfg.Server.TLS.MinVersion] {
			return fmt.Errorf("invalid TLS min version: %s", cfg.Server.TLS.MinVersion)
		}

		// If client auth is required or client cert verification is enabled, CA file is required
		if (cfg.Server.TLS.ClientAuth == "require" || cfg.Server.TLS.VerifyClientCert) && cfg.Server.TLS.CAFile == "" {
			return fmt.Errorf("CA file is required when client authentication is required or client cert verification is enabled")
		}
	}

	// Validate encryption configuration
	if cfg.Storage.Encryption.Enabled {
		validAlgorithms := map[string]bool{
			"aes-128-gcm":        true,
			"aes-256-gcm":        true,
			"chacha20-poly1305":  true,
		}
		if !validAlgorithms[cfg.Storage.Encryption.Algorithm] {
			return fmt.Errorf("invalid encryption algorithm: %s (must be aes-128-gcm, aes-256-gcm, or chacha20-poly1305)", cfg.Storage.Encryption.Algorithm)
		}
		if cfg.Storage.Encryption.KeyDir == "" {
			return fmt.Errorf("encryption key directory is required when encryption is enabled")
		}
	}

	return nil
}
