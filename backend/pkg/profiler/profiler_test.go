// Copyright 2025 Takhin Data, Inc.

package profiler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfiler_CPU(t *testing.T) {
	prof := New()
	tmpDir := t.TempDir()

	opts := &ProfileOptions{
		Type:       ProfileTypeCPU,
		Duration:   100 * time.Millisecond,
		OutputPath: filepath.Join(tmpDir, "cpu.prof"),
	}

	ctx := context.Background()
	path, err := prof.Profile(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, opts.OutputPath, path)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestProfiler_Heap(t *testing.T) {
	prof := New()
	tmpDir := t.TempDir()

	opts := &ProfileOptions{
		Type:       ProfileTypeHeap,
		Duration:   50 * time.Millisecond,
		OutputPath: filepath.Join(tmpDir, "heap.prof"),
	}

	ctx := context.Background()
	path, err := prof.Profile(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, opts.OutputPath, path)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestProfiler_Goroutine(t *testing.T) {
	prof := New()
	tmpDir := t.TempDir()

	opts := &ProfileOptions{
		Type:       ProfileTypeGoroutine,
		OutputPath: filepath.Join(tmpDir, "goroutine.prof"),
	}

	ctx := context.Background()
	path, err := prof.Profile(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, opts.OutputPath, path)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestProfiler_Allocs(t *testing.T) {
	prof := New()
	tmpDir := t.TempDir()

	opts := &ProfileOptions{
		Type:       ProfileTypeAllocs,
		Duration:   50 * time.Millisecond,
		OutputPath: filepath.Join(tmpDir, "allocs.prof"),
	}

	ctx := context.Background()
	path, err := prof.Profile(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, opts.OutputPath, path)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestProfiler_Trace(t *testing.T) {
	prof := New()
	tmpDir := t.TempDir()

	opts := &ProfileOptions{
		Type:       ProfileTypeTrace,
		Duration:   100 * time.Millisecond,
		OutputPath: filepath.Join(tmpDir, "trace.out"),
	}

	ctx := context.Background()
	path, err := prof.Profile(ctx, opts)
	require.NoError(t, err)
	assert.Equal(t, opts.OutputPath, path)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestProfiler_ProfileAll(t *testing.T) {
	prof := New()
	tmpDir := t.TempDir()

	ctx := context.Background()
	results, err := prof.ProfileAll(ctx, tmpDir, 100*time.Millisecond)
	require.NoError(t, err)

	expectedTypes := []ProfileType{
		ProfileTypeCPU,
		ProfileTypeHeap,
		ProfileTypeAllocs,
		ProfileTypeGoroutine,
	}

	for _, ptype := range expectedTypes {
		path, ok := results[ptype]
		assert.True(t, ok, "missing profile type: %s", ptype)
		
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0))
	}
}

func TestProfiler_InvalidType(t *testing.T) {
	prof := New()
	tmpDir := t.TempDir()

	opts := &ProfileOptions{
		Type:       ProfileType("invalid"),
		OutputPath: filepath.Join(tmpDir, "invalid.prof"),
	}

	ctx := context.Background()
	_, err := prof.Profile(ctx, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported profile type")
}

func TestProfiler_ContextCancellation(t *testing.T) {
	prof := New()
	tmpDir := t.TempDir()

	opts := &ProfileOptions{
		Type:       ProfileTypeCPU,
		Duration:   5 * time.Second,
		OutputPath: filepath.Join(tmpDir, "cpu_cancelled.prof"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := prof.Profile(ctx, opts)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}
