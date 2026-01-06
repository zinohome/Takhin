// Copyright 2025 Takhin Data, Inc.

package debug

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/logger"
)

func TestNewBundle(t *testing.T) {
	cfg := &config.Config{}
	log := logger.Default()

	bundle := NewBundle(cfg, log)
	assert.NotNil(t, bundle)
	assert.Equal(t, cfg, bundle.cfg)
	assert.Equal(t, log, bundle.logger)
}

func TestDefaultBundleOptions(t *testing.T) {
	opts := DefaultBundleOptions()
	assert.True(t, opts.IncludeLogs)
	assert.True(t, opts.IncludeConfig)
	assert.True(t, opts.IncludeMetrics)
	assert.True(t, opts.IncludeSystem)
	assert.Equal(t, int64(100), opts.LogsMaxSizeMB)
	assert.Equal(t, 24*time.Hour, opts.LogsSince)
}

func TestBundleGenerate(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tmpDir,
		},
		Logging: config.LoggingConfig{
			Level: "info",
		},
	}
	log := logger.Default()

	bundle := NewBundle(cfg, log)
	bundle.outputDir = tmpDir

	opts := &BundleOptions{
		IncludeLogs:    true,
		IncludeConfig:  true,
		IncludeMetrics: true,
		IncludeSystem:  true,
		OutputPath:     filepath.Join(tmpDir, "test-bundle.tar.gz"),
	}

	path, err := bundle.Generate(context.Background(), opts)
	require.NoError(t, err)
	assert.Equal(t, opts.OutputPath, path)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestCollectSystemInfo(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{}
	log := logger.Default()

	bundle := NewBundle(cfg, log)

	err := bundle.CollectSystemInfo(context.Background(), tmpDir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(tmpDir, "system-info.json"))
	require.NoError(t, err)

	var info SystemInfo
	err = json.Unmarshal(data, &info)
	require.NoError(t, err)

	assert.NotEmpty(t, info.OS)
	assert.NotEmpty(t, info.Architecture)
	assert.NotEmpty(t, info.GoVersion)
	assert.Greater(t, info.NumCPU, 0)
	assert.Greater(t, info.NumGoroutines, 0)
}

func TestCollectConfig(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 9092,
		},
		Sasl: config.SaslConfig{
			Enabled:    true,
			PlainUsers: "/etc/takhin/users",
		},
	}
	log := logger.Default()

	bundle := NewBundle(cfg, log)

	err := bundle.collectConfig(context.Background(), tmpDir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(tmpDir, "config.json"))
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Check that we have some config data
	assert.NotEmpty(t, result)
}

func TestCollectLogs(t *testing.T) {
	tmpDir := t.TempDir()
	logsDir := filepath.Join(tmpDir, "logs")
	err := os.MkdirAll(logsDir, 0755)
	require.NoError(t, err)

	logFile := filepath.Join(logsDir, "test.log")
	err = os.WriteFile(logFile, []byte("test log content"), 0644)
	require.NoError(t, err)

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tmpDir,
		},
	}
	log := logger.Default()

	bundle := NewBundle(cfg, log)
	bundleDir := filepath.Join(tmpDir, "bundle")
	err = os.MkdirAll(bundleDir, 0755)
	require.NoError(t, err)

	// Change working directory to tmpDir
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	opts := &BundleOptions{
		LogsMaxSizeMB: 100,
		LogsSince:     24 * time.Hour,
	}

	err = bundle.collectLogs(context.Background(), bundleDir, opts)
	require.NoError(t, err)
}

func TestCollectStorageInfo(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	err := os.MkdirAll(dataDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(dataDir, "test.dat")
	err = os.WriteFile(testFile, []byte("test data"), 0644)
	require.NoError(t, err)

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: dataDir,
		},
	}
	log := logger.Default()

	bundle := NewBundle(cfg, log)
	bundleDir := filepath.Join(tmpDir, "bundle")
	err = os.MkdirAll(bundleDir, 0755)
	require.NoError(t, err)

	opts := &BundleOptions{
		StorageMaxSizeMB: 50,
	}

	err = bundle.collectStorageInfo(context.Background(), bundleDir, opts)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(bundleDir, "storage", "storage-info.json"))
	require.NoError(t, err)

	var info map[string]interface{}
	err = json.Unmarshal(data, &info)
	require.NoError(t, err)

	assert.Equal(t, dataDir, info["data_dir"])
	assert.Greater(t, info["total_size_bytes"].(float64), float64(0))
}

func TestCreateTarball(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	err := os.MkdirAll(sourceDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(sourceDir, "test.txt")
	testContent := "test content"
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	cfg := &config.Config{}
	log := logger.Default()
	bundle := NewBundle(cfg, log)

	outputPath := filepath.Join(tmpDir, "test.tar.gz")
	err = bundle.createTarball(sourceDir, outputPath)
	require.NoError(t, err)

	file, err := os.Open(outputPath)
	require.NoError(t, err)
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	require.NoError(t, err)
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	foundFile := false
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		if header.Name == "test.txt" {
			foundFile = true
			data, err := io.ReadAll(tr)
			require.NoError(t, err)
			assert.Equal(t, testContent, string(data))
		}
	}

	assert.True(t, foundFile, "test.txt not found in tarball")
}

func TestSanitizeConfig(t *testing.T) {
	cfg := &config.Config{
		Sasl: config.SaslConfig{
			Enabled:    true,
			PlainUsers: "/etc/takhin/users",
		},
	}
	log := logger.Default()
	bundle := NewBundle(cfg, log)

	sanitized := bundle.sanitizeConfig(cfg)
	assert.NotNil(t, sanitized)
}

func TestSanitizeValue(t *testing.T) {
	cfg := &config.Config{}
	log := logger.Default()
	bundle := NewBundle(cfg, log)

	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{
			name:     "password should be redacted",
			key:      "TAKHIN_PASSWORD",
			value:    "secret123",
			expected: "***REDACTED***",
		},
		{
			name:     "secret should be redacted",
			key:      "TAKHIN_SECRET_KEY",
			value:    "mysecret",
			expected: "***REDACTED***",
		},
		{
			name:     "normal value should not be redacted",
			key:      "TAKHIN_SERVER_PORT",
			value:    "9092",
			expected: "9092",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bundle.sanitizeValue(tt.key, tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetRelevantEnvVars(t *testing.T) {
	os.Setenv("TAKHIN_TEST_VAR", "test-value")
	os.Setenv("TAKHIN_SECRET", "secret-value")
	os.Setenv("OTHER_VAR", "other-value")
	defer func() {
		os.Unsetenv("TAKHIN_TEST_VAR")
		os.Unsetenv("TAKHIN_SECRET")
		os.Unsetenv("OTHER_VAR")
	}()

	cfg := &config.Config{}
	log := logger.Default()
	bundle := NewBundle(cfg, log)

	envVars := bundle.getRelevantEnvVars()

	assert.Contains(t, envVars, "TAKHIN_TEST_VAR")
	assert.Equal(t, "test-value", envVars["TAKHIN_TEST_VAR"])

	assert.Contains(t, envVars, "TAKHIN_SECRET")
	assert.Equal(t, "***REDACTED***", envVars["TAKHIN_SECRET"])

	assert.NotContains(t, envVars, "OTHER_VAR")
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "source.txt")
	dstPath := filepath.Join(tmpDir, "dest.txt")

	content := "test file content"
	err := os.WriteFile(srcPath, []byte(content), 0644)
	require.NoError(t, err)

	err = copyFile(srcPath, dstPath)
	require.NoError(t, err)

	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}
