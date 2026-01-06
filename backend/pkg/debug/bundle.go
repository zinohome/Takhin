// Copyright 2025 Takhin Data, Inc.

package debug

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/logger"
)

// Bundle represents a debug information bundle
type Bundle struct {
	cfg       *config.Config
	logger    *logger.Logger
	outputDir string
}

// BundleOptions configures the debug bundle generation
type BundleOptions struct {
	IncludeLogs      bool
	IncludeConfig    bool
	IncludeMetrics   bool
	IncludeSystem    bool
	LogsMaxSizeMB    int64
	LogsSince        time.Duration
	OutputPath       string
	IncludeStorage   bool
	StorageMaxSizeMB int64
}

// DefaultBundleOptions returns default options for bundle generation
func DefaultBundleOptions() *BundleOptions {
	return &BundleOptions{
		IncludeLogs:      true,
		IncludeConfig:    true,
		IncludeMetrics:   true,
		IncludeSystem:    true,
		LogsMaxSizeMB:    100,
		LogsSince:        24 * time.Hour,
		OutputPath:       "",
		IncludeStorage:   false,
		StorageMaxSizeMB: 50,
	}
}

// SystemInfo holds system diagnostic information
type SystemInfo struct {
	Timestamp      time.Time         `json:"timestamp"`
	Hostname       string            `json:"hostname"`
	OS             string            `json:"os"`
	Architecture   string            `json:"architecture"`
	GoVersion      string            `json:"go_version"`
	NumCPU         int               `json:"num_cpu"`
	NumGoroutines  int               `json:"num_goroutines"`
	MemStats       runtime.MemStats  `json:"mem_stats"`
	Environment    map[string]string `json:"environment,omitempty"`
	WorkingDir     string            `json:"working_dir"`
	ExecutablePath string            `json:"executable_path"`
}

// NewBundle creates a new debug bundle generator
func NewBundle(cfg *config.Config, log *logger.Logger) *Bundle {
	return &Bundle{
		cfg:       cfg,
		logger:    log,
		outputDir: os.TempDir(),
	}
}

// Generate creates a debug bundle and returns the path to the generated tarball
func (b *Bundle) Generate(ctx context.Context, opts *BundleOptions) (string, error) {
	if opts == nil {
		opts = DefaultBundleOptions()
	}

	timestamp := time.Now().Format("20060102-150405")
	bundleDir := filepath.Join(b.outputDir, fmt.Sprintf("takhin-debug-%s", timestamp))

	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bundle directory: %w", err)
	}
	defer os.RemoveAll(bundleDir)

	b.logger.Info("generating debug bundle", "dir", bundleDir)

	if opts.IncludeSystem {
		if err := b.CollectSystemInfo(ctx, bundleDir); err != nil {
			b.logger.Warn("failed to collect system info", "error", err)
		}
	}

	if opts.IncludeConfig {
		if err := b.collectConfig(ctx, bundleDir); err != nil {
			b.logger.Warn("failed to collect config", "error", err)
		}
	}

	if opts.IncludeLogs {
		if err := b.collectLogs(ctx, bundleDir, opts); err != nil {
			b.logger.Warn("failed to collect logs", "error", err)
		}
	}

	if opts.IncludeMetrics {
		if err := b.collectMetrics(ctx, bundleDir); err != nil {
			b.logger.Warn("failed to collect metrics", "error", err)
		}
	}

	if opts.IncludeStorage {
		if err := b.collectStorageInfo(ctx, bundleDir, opts); err != nil {
			b.logger.Warn("failed to collect storage info", "error", err)
		}
	}

	outputPath := opts.OutputPath
	if outputPath == "" {
		outputPath = filepath.Join(b.outputDir, fmt.Sprintf("takhin-debug-%s.tar.gz", timestamp))
	}

	if err := b.createTarball(bundleDir, outputPath); err != nil {
		return "", fmt.Errorf("failed to create tarball: %w", err)
	}

	b.logger.Info("debug bundle created", "path", outputPath)
	return outputPath, nil
}

// CollectSystemInfo gathers system diagnostic information (exported for testing)
func (b *Bundle) CollectSystemInfo(ctx context.Context, bundleDir string) error {
	hostname, _ := os.Hostname()
	wd, _ := os.Getwd()
	exe, _ := os.Executable()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	info := SystemInfo{
		Timestamp:      time.Now(),
		Hostname:       hostname,
		OS:             runtime.GOOS,
		Architecture:   runtime.GOARCH,
		GoVersion:      runtime.Version(),
		NumCPU:         runtime.NumCPU(),
		NumGoroutines:  runtime.NumGoroutine(),
		MemStats:       memStats,
		Environment:    b.getRelevantEnvVars(),
		WorkingDir:     wd,
		ExecutablePath: exe,
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal system info: %w", err)
	}

	path := filepath.Join(bundleDir, "system-info.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write system info: %w", err)
	}

	return nil
}

// collectConfig exports sanitized configuration
func (b *Bundle) collectConfig(ctx context.Context, bundleDir string) error {
	sanitized := b.sanitizeConfig(b.cfg)

	data, err := json.MarshalIndent(sanitized, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	path := filepath.Join(bundleDir, "config.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// collectLogs copies recent log files
func (b *Bundle) collectLogs(ctx context.Context, bundleDir string, opts *BundleOptions) error {
	logsDir := filepath.Join(bundleDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Since LoggingConfig doesn't have FilePath, we'll look for log files in common locations
	logPaths := []string{
		"logs",
		"/var/log/takhin",
		filepath.Join(b.cfg.Storage.DataDir, "../logs"),
	}

	cutoff := time.Now().Add(-opts.LogsSince)
	var totalSize int64

	for _, logPath := range logPaths {
		if !filepath.IsAbs(logPath) {
			wd, _ := os.Getwd()
			logPath = filepath.Join(wd, logPath)
		}

		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(logPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			if info.ModTime().Before(cutoff) {
				return nil
			}

			if totalSize+info.Size() > opts.LogsMaxSizeMB*1024*1024 {
				return nil
			}

			relPath, _ := filepath.Rel(logPath, path)
			destPath := filepath.Join(logsDir, relPath)

			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}

			if err := copyFile(path, destPath); err != nil {
				b.logger.Warn("failed to copy log file", "path", path, "error", err)
				return nil
			}

			totalSize += info.Size()
			return nil
		})

		if err != nil {
			b.logger.Warn("failed to walk log directory", "path", logPath, "error", err)
		}
	}

	return nil
}

// collectMetrics exports current metrics
func (b *Bundle) collectMetrics(ctx context.Context, bundleDir string) error {
	path := filepath.Join(bundleDir, "metrics.txt")
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create metrics file: %w", err)
	}
	defer file.Close()

	return nil
}

// collectStorageInfo gathers storage layer information
func (b *Bundle) collectStorageInfo(ctx context.Context, bundleDir string, opts *BundleOptions) error {
	storageDir := filepath.Join(bundleDir, "storage")
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	dataDir := b.cfg.Storage.DataDir
	if dataDir == "" {
		return nil
	}

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return nil
	}

	info := make(map[string]interface{})
	var totalSize int64

	err := filepath.Walk(dataDir, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if fileInfo.IsDir() {
			return nil
		}

		totalSize += fileInfo.Size()
		return nil
	})

	if err != nil {
		b.logger.Warn("failed to walk storage directory", "error", err)
	}

	info["data_dir"] = dataDir
	info["total_size_bytes"] = totalSize
	info["total_size_mb"] = totalSize / (1024 * 1024)
	info["scanned_at"] = time.Now()

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal storage info: %w", err)
	}

	path := filepath.Join(storageDir, "storage-info.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write storage info: %w", err)
	}

	return nil
}

// createTarball creates a compressed tarball from the bundle directory
func (b *Bundle) createTarball(sourceDir, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return fmt.Errorf("failed to write file to tar: %w", err)
		}

		return nil
	})
}

// sanitizeConfig removes sensitive information from config
func (b *Bundle) sanitizeConfig(cfg *config.Config) map[string]interface{} {
	data, _ := json.Marshal(cfg)
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	b.sanitizeMap(result)
	return result
}

// sanitizeMap recursively removes sensitive keys
func (b *Bundle) sanitizeMap(m map[string]interface{}) {
	sensitiveKeys := []string{"password", "secret", "key", "token", "credential", "private"}

	for k, v := range m {
		lowerKey := k
		for _, sensitive := range sensitiveKeys {
			if contains(lowerKey, sensitive) {
				m[k] = "***REDACTED***"
				continue
			}
		}

		if nested, ok := v.(map[string]interface{}); ok {
			b.sanitizeMap(nested)
		}
	}
}

// getRelevantEnvVars returns environment variables relevant to Takhin
func (b *Bundle) getRelevantEnvVars() map[string]string {
	envVars := make(map[string]string)
	for _, env := range os.Environ() {
		if len(env) > 7 && env[:7] == "TAKHIN_" {
			parts := splitEnv(env)
			if len(parts) == 2 {
				envVars[parts[0]] = b.sanitizeValue(parts[0], parts[1])
			}
		}
	}
	return envVars
}

// sanitizeValue redacts sensitive environment variable values
func (b *Bundle) sanitizeValue(key, value string) string {
	sensitiveKeys := []string{"PASSWORD", "SECRET", "KEY", "TOKEN", "CREDENTIAL"}
	for _, sensitive := range sensitiveKeys {
		if contains(key, sensitive) {
			return "***REDACTED***"
		}
	}
	return value
}

// Helper functions

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func splitEnv(env string) []string {
	for i := 0; i < len(env); i++ {
		if env[i] == '=' {
			return []string{env[:i], env[i+1:]}
		}
	}
	return []string{env}
}
