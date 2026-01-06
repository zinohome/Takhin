// Copyright 2025 Takhin Data, Inc.

package throttle

import (
	"context"
	"testing"
)

func BenchmarkThrottleProducer(b *testing.B) {
	cfg := &Config{
		ProducerBytesPerSecond: 100 * 1024 * 1024, // 100 MB/s
		ProducerBurst:          200 * 1024 * 1024,
		ConsumerBytesPerSecond: 100 * 1024 * 1024,
		ConsumerBurst:          200 * 1024 * 1024,
		DynamicEnabled:         false,
	}
	
	throttler := New(cfg)
	defer throttler.Close()
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		throttler.AllowProducer(ctx, 1024)
	}
}

func BenchmarkThrottleConsumer(b *testing.B) {
	cfg := &Config{
		ProducerBytesPerSecond: 100 * 1024 * 1024,
		ProducerBurst:          200 * 1024 * 1024,
		ConsumerBytesPerSecond: 100 * 1024 * 1024, // 100 MB/s
		ConsumerBurst:          200 * 1024 * 1024,
		DynamicEnabled:         false,
	}
	
	throttler := New(cfg)
	defer throttler.Close()
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		throttler.AllowConsumer(ctx, 1024)
	}
}

func BenchmarkThrottleConcurrent(b *testing.B) {
	cfg := &Config{
		ProducerBytesPerSecond: 100 * 1024 * 1024,
		ProducerBurst:          200 * 1024 * 1024,
		ConsumerBytesPerSecond: 100 * 1024 * 1024,
		ConsumerBurst:          200 * 1024 * 1024,
		DynamicEnabled:         false,
	}
	
	throttler := New(cfg)
	defer throttler.Close()
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			throttler.AllowProducer(ctx, 1024)
		}
	})
}

func BenchmarkThrottleDisabled(b *testing.B) {
	cfg := &Config{
		ProducerBytesPerSecond: 0, // Disabled
		ConsumerBytesPerSecond: 0, // Disabled
	}
	
	throttler := New(cfg)
	defer throttler.Close()
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		throttler.AllowProducer(ctx, 1024)
	}
}

func BenchmarkUpdateRate(b *testing.B) {
	cfg := &Config{
		ProducerBytesPerSecond: 10 * 1024 * 1024,
		ProducerBurst:          20 * 1024 * 1024,
		ConsumerBytesPerSecond: 10 * 1024 * 1024,
		ConsumerBurst:          20 * 1024 * 1024,
		DynamicMinRate:         1024 * 1024,
		DynamicMaxRate:         100 * 1024 * 1024,
	}
	
	throttler := New(cfg)
	defer throttler.Close()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rate := int64((i%10 + 1) * 1024 * 1024)
		throttler.UpdateProducerRate(rate, 0)
	}
}
