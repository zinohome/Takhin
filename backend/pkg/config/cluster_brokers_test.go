// Copyright 2025 Takhin Data, Inc.

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterBrokersValidation(t *testing.T) {
	tests := []struct {
		name           string
		brokerID       int
		clusterBrokers []int
		shouldFail     bool
		errorContains  string
	}{
		{
			name:           "current broker in cluster list",
			brokerID:       2,
			clusterBrokers: []int{1, 2, 3},
			shouldFail:     false,
		},
		{
			name:           "current broker not in cluster list",
			brokerID:       4,
			clusterBrokers: []int{1, 2, 3},
			shouldFail:     true,
			errorContains:  "not found in cluster.brokers list",
		},
		{
			name:           "empty cluster list (single broker mode)",
			brokerID:       1,
			clusterBrokers: []int{},
			shouldFail:     false,
		},
		{
			name:           "single broker in cluster list",
			brokerID:       1,
			clusterBrokers: []int{1},
			shouldFail:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 9092,
				},
				Kafka: KafkaConfig{
					BrokerID:       tt.brokerID,
					ClusterBrokers: tt.clusterBrokers,
				},
				Storage: StorageConfig{
					LogSegmentSize: 1024 * 1024,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			}

			err := validate(cfg)
			if tt.shouldFail {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClusterBrokersDefault(t *testing.T) {
	cfg := &Config{
		Kafka: KafkaConfig{
			BrokerID: 1,
			// ClusterBrokers not set
		},
	}

	// setDefaults should not modify ClusterBrokers
	setDefaults(cfg)
	assert.Nil(t, cfg.Kafka.ClusterBrokers)
}
