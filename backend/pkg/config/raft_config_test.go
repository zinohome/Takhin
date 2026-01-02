// Copyright 2025 Takhin Data, Inc.

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRaftConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid raft config",
			cfg: &Config{
				Server: ServerConfig{Port: 9092},
				Kafka:  KafkaConfig{BrokerID: 1},
				Storage: StorageConfig{
					LogSegmentSize: 1024,
				},
				Raft: RaftConfig{
					HeartbeatTimeoutMs:   1000,
					ElectionTimeoutMs:    3000,
					LeaderLeaseTimeoutMs: 500,
					CommitTimeoutMs:      50,
					PreVoteEnabled:       true,
				},
				Logging: LoggingConfig{Level: "info"},
			},
			wantErr: false,
		},
		{
			name: "heartbeat timeout too low",
			cfg: &Config{
				Server: ServerConfig{Port: 9092},
				Kafka:  KafkaConfig{BrokerID: 1},
				Storage: StorageConfig{
					LogSegmentSize: 1024,
				},
				Raft: RaftConfig{
					HeartbeatTimeoutMs:   50, // Too low
					ElectionTimeoutMs:    3000,
					LeaderLeaseTimeoutMs: 500,
				},
				Logging: LoggingConfig{Level: "info"},
			},
			wantErr: true,
			errMsg:  "invalid heartbeat timeout: 50ms (minimum 100ms)",
		},
		{
			name: "election timeout less than heartbeat",
			cfg: &Config{
				Server: ServerConfig{Port: 9092},
				Kafka:  KafkaConfig{BrokerID: 1},
				Storage: StorageConfig{
					LogSegmentSize: 1024,
				},
				Raft: RaftConfig{
					HeartbeatTimeoutMs:   2000,
					ElectionTimeoutMs:    1000, // Less than heartbeat
					LeaderLeaseTimeoutMs: 500,
				},
				Logging: LoggingConfig{Level: "info"},
			},
			wantErr: true,
			errMsg:  "election timeout (1000ms) must be >= heartbeat timeout (2000ms)",
		},
		{
			name: "leader lease timeout too low",
			cfg: &Config{
				Server: ServerConfig{Port: 9092},
				Kafka:  KafkaConfig{BrokerID: 1},
				Storage: StorageConfig{
					LogSegmentSize: 1024,
				},
				Raft: RaftConfig{
					HeartbeatTimeoutMs:   1000,
					ElectionTimeoutMs:    3000,
					LeaderLeaseTimeoutMs: 50, // Too low
				},
				Logging: LoggingConfig{Level: "info"},
			},
			wantErr: true,
			errMsg:  "invalid leader lease timeout: 50ms (minimum 100ms)",
		},
		{
			name: "no raft config (should pass - uses defaults)",
			cfg: &Config{
				Server: ServerConfig{Port: 9092},
				Kafka:  KafkaConfig{BrokerID: 1},
				Storage: StorageConfig{
					LogSegmentSize: 1024,
				},
				Logging: LoggingConfig{Level: "info"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRaftConfigDefaults(t *testing.T) {
	cfg := &Config{}
	setDefaults(cfg)

	// Verify Raft defaults are set correctly
	assert.Equal(t, 1000, cfg.Raft.HeartbeatTimeoutMs, "heartbeat timeout default")
	assert.Equal(t, 3000, cfg.Raft.ElectionTimeoutMs, "election timeout default")
	assert.Equal(t, 500, cfg.Raft.LeaderLeaseTimeoutMs, "leader lease timeout default")
	assert.Equal(t, 50, cfg.Raft.CommitTimeoutMs, "commit timeout default")
	assert.Equal(t, 120000, cfg.Raft.SnapshotIntervalMs, "snapshot interval default")
	assert.Equal(t, 8192, cfg.Raft.SnapshotThreshold, "snapshot threshold default")
	assert.Equal(t, 64, cfg.Raft.MaxAppendEntries, "max append entries default")
	// PreVoteEnabled defaults to false (zero value)
	assert.False(t, cfg.Raft.PreVoteEnabled, "prevote enabled default")
}

func TestRaftConfigValidationInLoad(t *testing.T) {
	// Test that Load() calls setDefaults before validate
	cfg, err := Load("")
	require.NoError(t, err)

	// Config should have valid Raft settings after Load
	assert.GreaterOrEqual(t, cfg.Raft.HeartbeatTimeoutMs, 100)
	assert.GreaterOrEqual(t, cfg.Raft.ElectionTimeoutMs, cfg.Raft.HeartbeatTimeoutMs)
}
