// Copyright 2025 Takhin Data, Inc.

package audit

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Rotator handles log file rotation
type Rotator struct {
	mu     sync.Mutex
	file   *os.File
	config RotatorConfig
	size   int64
}

// RotatorConfig holds configuration for log rotation
type RotatorConfig struct {
	Filename   string // Full path to log file
	MaxSize    int64  // Max size in bytes before rotation
	MaxBackups int    // Max number of backup files to keep
	MaxAge     int    // Max age in days to keep backups
	Compress   bool   // Compress rotated files
}

// NewRotator creates a new log rotator
func NewRotator(config RotatorConfig) (*Rotator, error) {
	r := &Rotator{
		config: config,
	}

	if err := r.openFile(); err != nil {
		return nil, err
	}

	return r, nil
}

// Write implements io.Writer
func (r *Rotator) Write(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if rotation is needed
	if r.size+int64(len(p)) > r.config.MaxSize {
		if err := r.rotate(); err != nil {
			return 0, fmt.Errorf("rotate log file: %w", err)
		}
	}

	n, err = r.file.Write(p)
	r.size += int64(n)

	return n, err
}

// Close closes the rotator
func (r *Rotator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file != nil {
		if err := r.file.Close(); err != nil {
			return err
		}
		r.file = nil
	}

	return nil
}

// openFile opens the log file for writing
func (r *Rotator) openFile() error {
	info, err := os.Stat(r.config.Filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("stat log file: %w", err)
		}
	}

	file, err := os.OpenFile(r.config.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	r.file = file
	if info != nil {
		r.size = info.Size()
	}

	return nil
}

// rotate rotates the log file
func (r *Rotator) rotate() error {
	// Close current file
	if r.file != nil {
		if err := r.file.Close(); err != nil {
			return fmt.Errorf("close log file: %w", err)
		}
	}

	// Generate backup filename
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	backupName := fmt.Sprintf("%s.%s", r.config.Filename, timestamp)

	// Rename current file to backup
	if err := os.Rename(r.config.Filename, backupName); err != nil {
		return fmt.Errorf("rename log file: %w", err)
	}

	// Compress if configured
	if r.config.Compress {
		go r.compressFile(backupName)
	}

	// Open new file
	if err := r.openFile(); err != nil {
		return err
	}

	// Clean up old backups
	go r.cleanupOldBackups()

	return nil
}

// compressFile compresses a backup file
func (r *Rotator) compressFile(filename string) {
	compressed := filename + ".gz"

	in, err := os.Open(filename)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(compressed)
	if err != nil {
		return
	}
	defer out.Close()

	gzWriter := gzip.NewWriter(out)
	defer gzWriter.Close()

	if _, err := io.Copy(gzWriter, in); err != nil {
		os.Remove(compressed)
		return
	}

	// Remove original file after successful compression
	os.Remove(filename)
}

// cleanupOldBackups removes old backup files
func (r *Rotator) cleanupOldBackups() {
	dir := filepath.Dir(r.config.Filename)
	base := filepath.Base(r.config.Filename)

	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// Find all backup files
	backups := make([]os.DirEntry, 0)
	for _, file := range files {
		if strings.HasPrefix(file.Name(), base+".") && file.Name() != base {
			backups = append(backups, file)
		}
	}

	// Sort by modification time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		iInfo, _ := backups[i].Info()
		jInfo, _ := backups[j].Info()
		return iInfo.ModTime().After(jInfo.ModTime())
	})

	// Remove old backups based on count
	if r.config.MaxBackups > 0 && len(backups) > r.config.MaxBackups {
		for _, file := range backups[r.config.MaxBackups:] {
			os.Remove(filepath.Join(dir, file.Name()))
		}
		backups = backups[:r.config.MaxBackups]
	}

	// Remove old backups based on age
	if r.config.MaxAge > 0 {
		cutoff := time.Now().AddDate(0, 0, -r.config.MaxAge)
		for _, file := range backups {
			info, err := file.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				os.Remove(filepath.Join(dir, file.Name()))
			}
		}
	}
}
