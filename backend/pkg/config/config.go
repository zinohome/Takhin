// Copyright 2025 Takhin Data, Inc.

package config

import (
	"fmt"
	"log/slog"
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
	Logging     LoggingConfig     `koanf:"logging"`
	Metrics     MetricsConfig     `koanf:"metrics"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host string `koanf:"host"`
	Port int    `koanf:"port"`
}

// KafkaConfig holds Kafka protocol configuration
type KafkaConfig struct {
	BrokerID          int      `koanf:"broker.id"`
	Listeners         []string `koanf:"listeners"`
	AdvertisedHost    string   `koanf:"advertised.host"`
	AdvertisedPort    int      `koanf:"advertised.port"`
	MaxMessageBytes   int      `koanf:"max.message.bytes"`
	MaxConnections    int      `koanf:"max.connections"`
	RequestTimeout    int      `koanf:"request.timeout.ms"`
	ConnectionTimeout int      `koanf:"connection.timeout.ms"`
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	DataDir            string  `koanf:"data.dir"`
	LogSegmentSize     int64   `koanf:"log.segment.size"`
	LogRetentionHours  int     `koanf:"log.retention.hours"`
	LogRetentionBytes  int64   `koanf:"log.retention.bytes"`
	LogCleanupInterval int     `koanf:"log.cleanup.interval.ms"`
	LogFlushInterval   int     `koanf:"log.flush.interval.ms"`
	LogFlushMessages   int     `koanf:"log.flush.messages"`
	CleanerEnabled     bool    `koanf:"cleaner.enabled"`
	CompactionInterval int     `koanf:"compaction.interval.ms"`
	MinCleanableRatio  float64 `koanf:"compaction.min.cleanable.ratio"`
}

// ReplicationConfig holds replication configuration
type ReplicationConfig struct {
	DefaultReplicationFactor int16 `koanf:"default.replication.factor"`
	ReplicaLagTimeMaxMs      int64 `koanf:"replica.lag.time.max.ms"`
	ReplicaFetchWaitMaxMs    int   `koanf:"replica.fetch.wait.max.ms"`
	ReplicaFetchMaxBytes     int   `koanf:"replica.fetch.max.bytes"`
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
}

func validate(cfg *Config) error {
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	if cfg.Kafka.BrokerID < 0 {
		return fmt.Errorf("invalid broker ID: %d", cfg.Kafka.BrokerID)
	}

	if cfg.Storage.LogSegmentSize <= 0 {
		return fmt.Errorf("invalid log segment size: %d", cfg.Storage.LogSegmentSize)
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

	return nil
}
