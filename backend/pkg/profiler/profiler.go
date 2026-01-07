// Copyright 2025 Takhin Data, Inc.

package profiler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"time"

	"github.com/takhin-data/takhin/pkg/logger"
)

type ProfileType string

const (
	ProfileTypeCPU       ProfileType = "cpu"
	ProfileTypeMemory    ProfileType = "memory"
	ProfileTypeHeap      ProfileType = "heap"
	ProfileTypeAllocs    ProfileType = "allocs"
	ProfileTypeBlock     ProfileType = "block"
	ProfileTypeMutex     ProfileType = "mutex"
	ProfileTypeGoroutine ProfileType = "goroutine"
	ProfileTypeTrace     ProfileType = "trace"
)

type ProfileOptions struct {
	Type        ProfileType
	Duration    time.Duration
	OutputPath  string
	SampleRate  int
	MemProfileRate int
}

type Profiler struct {
	logger *logger.Logger
}

func New() *Profiler {
	return &Profiler{
		logger: logger.Default().WithComponent("profiler"),
	}
}

func (p *Profiler) Profile(ctx context.Context, opts *ProfileOptions) (string, error) {
	if opts.OutputPath == "" {
		timestamp := time.Now().Format("20060102_150405")
		opts.OutputPath = filepath.Join(os.TempDir(), fmt.Sprintf("takhin_%s_%s.prof", opts.Type, timestamp))
	}

	dir := filepath.Dir(opts.OutputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	p.logger.Info("starting profile",
		"type", opts.Type,
		"duration", opts.Duration,
		"output", opts.OutputPath,
	)

	switch opts.Type {
	case ProfileTypeCPU:
		return p.profileCPU(ctx, opts)
	case ProfileTypeMemory, ProfileTypeHeap:
		return p.profileHeap(ctx, opts)
	case ProfileTypeAllocs:
		return p.profileAllocs(ctx, opts)
	case ProfileTypeBlock:
		return p.profileBlock(ctx, opts)
	case ProfileTypeMutex:
		return p.profileMutex(ctx, opts)
	case ProfileTypeGoroutine:
		return p.profileGoroutine(ctx, opts)
	case ProfileTypeTrace:
		return p.profileTrace(ctx, opts)
	default:
		return "", fmt.Errorf("unsupported profile type: %s", opts.Type)
	}
}

func (p *Profiler) profileCPU(ctx context.Context, opts *ProfileOptions) (string, error) {
	f, err := os.Create(opts.OutputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create profile file: %w", err)
	}
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		return "", fmt.Errorf("failed to start CPU profile: %w", err)
	}
	defer pprof.StopCPUProfile()

	select {
	case <-time.After(opts.Duration):
	case <-ctx.Done():
		return "", ctx.Err()
	}

	p.logger.Info("CPU profile completed", "output", opts.OutputPath)
	return opts.OutputPath, nil
}

func (p *Profiler) profileHeap(ctx context.Context, opts *ProfileOptions) (string, error) {
	if opts.MemProfileRate > 0 {
		oldRate := runtime.MemProfileRate
		runtime.MemProfileRate = opts.MemProfileRate
		defer func() { runtime.MemProfileRate = oldRate }()
	}

	if opts.Duration > 0 {
		select {
		case <-time.After(opts.Duration):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	runtime.GC()

	f, err := os.Create(opts.OutputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create profile file: %w", err)
	}
	defer f.Close()

	if err := pprof.WriteHeapProfile(f); err != nil {
		return "", fmt.Errorf("failed to write heap profile: %w", err)
	}

	p.logger.Info("heap profile completed", "output", opts.OutputPath)
	return opts.OutputPath, nil
}

func (p *Profiler) profileAllocs(ctx context.Context, opts *ProfileOptions) (string, error) {
	if opts.MemProfileRate > 0 {
		oldRate := runtime.MemProfileRate
		runtime.MemProfileRate = opts.MemProfileRate
		defer func() { runtime.MemProfileRate = oldRate }()
	}

	if opts.Duration > 0 {
		select {
		case <-time.After(opts.Duration):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	f, err := os.Create(opts.OutputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create profile file: %w", err)
	}
	defer f.Close()

	prof := pprof.Lookup("allocs")
	if prof == nil {
		return "", fmt.Errorf("allocs profile not found")
	}

	if err := prof.WriteTo(f, 0); err != nil {
		return "", fmt.Errorf("failed to write allocs profile: %w", err)
	}

	p.logger.Info("allocs profile completed", "output", opts.OutputPath)
	return opts.OutputPath, nil
}

func (p *Profiler) profileBlock(ctx context.Context, opts *ProfileOptions) (string, error) {
	if opts.SampleRate > 0 {
		runtime.SetBlockProfileRate(opts.SampleRate)
		defer runtime.SetBlockProfileRate(0)
	}

	if opts.Duration > 0 {
		select {
		case <-time.After(opts.Duration):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	f, err := os.Create(opts.OutputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create profile file: %w", err)
	}
	defer f.Close()

	prof := pprof.Lookup("block")
	if prof == nil {
		return "", fmt.Errorf("block profile not found")
	}

	if err := prof.WriteTo(f, 0); err != nil {
		return "", fmt.Errorf("failed to write block profile: %w", err)
	}

	p.logger.Info("block profile completed", "output", opts.OutputPath)
	return opts.OutputPath, nil
}

func (p *Profiler) profileMutex(ctx context.Context, opts *ProfileOptions) (string, error) {
	if opts.SampleRate > 0 {
		runtime.SetMutexProfileFraction(opts.SampleRate)
		defer runtime.SetMutexProfileFraction(0)
	}

	if opts.Duration > 0 {
		select {
		case <-time.After(opts.Duration):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	f, err := os.Create(opts.OutputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create profile file: %w", err)
	}
	defer f.Close()

	prof := pprof.Lookup("mutex")
	if prof == nil {
		return "", fmt.Errorf("mutex profile not found")
	}

	if err := prof.WriteTo(f, 0); err != nil {
		return "", fmt.Errorf("failed to write mutex profile: %w", err)
	}

	p.logger.Info("mutex profile completed", "output", opts.OutputPath)
	return opts.OutputPath, nil
}

func (p *Profiler) profileGoroutine(ctx context.Context, opts *ProfileOptions) (string, error) {
	f, err := os.Create(opts.OutputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create profile file: %w", err)
	}
	defer f.Close()

	prof := pprof.Lookup("goroutine")
	if prof == nil {
		return "", fmt.Errorf("goroutine profile not found")
	}

	if err := prof.WriteTo(f, 0); err != nil {
		return "", fmt.Errorf("failed to write goroutine profile: %w", err)
	}

	p.logger.Info("goroutine profile completed", "output", opts.OutputPath)
	return opts.OutputPath, nil
}

func (p *Profiler) profileTrace(ctx context.Context, opts *ProfileOptions) (string, error) {
	f, err := os.Create(opts.OutputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create trace file: %w", err)
	}
	defer f.Close()

	if err := trace.Start(f); err != nil {
		return "", fmt.Errorf("failed to start trace: %w", err)
	}
	defer trace.Stop()

	select {
	case <-time.After(opts.Duration):
	case <-ctx.Done():
		return "", ctx.Err()
	}

	p.logger.Info("trace completed", "output", opts.OutputPath)
	return opts.OutputPath, nil
}

func (p *Profiler) ProfileAll(ctx context.Context, basePath string, duration time.Duration) (map[ProfileType]string, error) {
	results := make(map[ProfileType]string)
	timestamp := time.Now().Format("20060102_150405")

	types := []ProfileType{
		ProfileTypeCPU,
		ProfileTypeHeap,
		ProfileTypeAllocs,
		ProfileTypeGoroutine,
	}

	for _, ptype := range types {
		outputPath := filepath.Join(basePath, fmt.Sprintf("takhin_%s_%s.prof", ptype, timestamp))
		opts := &ProfileOptions{
			Type:       ptype,
			Duration:   duration,
			OutputPath: outputPath,
		}

		path, err := p.Profile(ctx, opts)
		if err != nil {
			p.logger.Error("failed to create profile", "type", ptype, "error", err)
			continue
		}
		results[ptype] = path
	}

	return results, nil
}
