// Copyright 2025 Takhin Data, Inc.

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name       string
		configFile string
		wantErr    bool
		validate   func(*testing.T, *Config)
	}{
		{
			name:       "load with defaults",
			configFile: "",
			wantErr:    false,
			validate: func(t *testing.T, cfg *Config) {
				assert.Equal(t, "0.0.0.0", cfg.Server.Host)
				assert.Equal(t, 9092, cfg.Server.Port)
				assert.Equal(t, 1, cfg.Kafka.BrokerID)
				assert.Equal(t, "info", cfg.Logging.Level)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(tt.configFile)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)

			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Server: ServerConfig{
					Port: 9092,
				},
				Kafka: KafkaConfig{
					BrokerID: 1,
				},
				Storage: StorageConfig{
					LogSegmentSize: 1024,
				},
				Logging: LoggingConfig{
					Level: "info",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			cfg: &Config{
				Server: ServerConfig{
					Port: -1,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
