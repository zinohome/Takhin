// Copyright 2025 Takhin Data, Inc.

package throttle

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/takhin-data/takhin/pkg/logger"
	"github.com/takhin-data/takhin/pkg/metrics"
	"golang.org/x/time/rate"
)

// Type represents the throttle type
type Type string

const (
	// TypeProducer represents producer throttling
	TypeProducer Type = "producer"
	// TypeConsumer represents consumer throttling
	TypeConsumer Type = "consumer"
)

// Throttler manages rate limiting for producers and consumers
type Throttler struct {
	producerLimiter *rate.Limiter
	consumerLimiter *rate.Limiter
	
	// Dynamic adjustment state
	producerRate atomic.Int64 // bytes per second
	consumerRate atomic.Int64 // bytes per second
	
	// Statistics
	producerThrottled atomic.Int64
	consumerThrottled atomic.Int64
	producerAllowed   atomic.Int64
	consumerAllowed   atomic.Int64
	
	// Configuration
	config *Config
	logger *logger.Logger
	
	// Dynamic adjustment
	adjustmentEnabled bool
	adjustmentMu      sync.RWMutex
	stopChan          chan struct{}
	wg                sync.WaitGroup
}

// Config holds throttle configuration
type Config struct {
	// Producer throttling
	ProducerBytesPerSecond int64 `koanf:"producer.bytes.per.second"`
	ProducerBurst          int   `koanf:"producer.burst"`
	
	// Consumer throttling
	ConsumerBytesPerSecond int64 `koanf:"consumer.bytes.per.second"`
	ConsumerBurst          int   `koanf:"consumer.burst"`
	
	// Dynamic adjustment
	DynamicEnabled        bool    `koanf:"dynamic.enabled"`
	DynamicCheckInterval  int     `koanf:"dynamic.check.interval.ms"`
	DynamicMinRate        int64   `koanf:"dynamic.min.rate"`
	DynamicMaxRate        int64   `koanf:"dynamic.max.rate"`
	DynamicTargetUtilPct  float64 `koanf:"dynamic.target.util.pct"`
	DynamicAdjustmentStep float64 `koanf:"dynamic.adjustment.step"`
}

// New creates a new Throttler with the given configuration
func New(cfg *Config) *Throttler {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	
	setConfigDefaults(cfg)
	
	t := &Throttler{
		config:            cfg,
		logger:            logger.Default().WithComponent("throttle"),
		adjustmentEnabled: cfg.DynamicEnabled,
		stopChan:          make(chan struct{}),
	}
	
	// Initialize rate limiters
	t.producerLimiter = rate.NewLimiter(rate.Limit(cfg.ProducerBytesPerSecond), cfg.ProducerBurst)
	t.consumerLimiter = rate.NewLimiter(rate.Limit(cfg.ConsumerBytesPerSecond), cfg.ConsumerBurst)
	
	t.producerRate.Store(cfg.ProducerBytesPerSecond)
	t.consumerRate.Store(cfg.ConsumerBytesPerSecond)
	
	// Start dynamic adjustment if enabled
	if cfg.DynamicEnabled {
		t.wg.Add(1)
		go t.dynamicAdjustmentLoop()
	}
	
	t.logger.Info("throttler initialized",
		"producer_rate", cfg.ProducerBytesPerSecond,
		"consumer_rate", cfg.ConsumerBytesPerSecond,
		"dynamic_enabled", cfg.DynamicEnabled,
	)
	
	return t
}

// DefaultConfig returns default throttle configuration
func DefaultConfig() *Config {
	return &Config{
		ProducerBytesPerSecond: 10 * 1024 * 1024, // 10 MB/s
		ProducerBurst:          20 * 1024 * 1024, // 20 MB burst
		ConsumerBytesPerSecond: 10 * 1024 * 1024, // 10 MB/s
		ConsumerBurst:          20 * 1024 * 1024, // 20 MB burst
		DynamicEnabled:         false,
		DynamicCheckInterval:   5000,  // 5 seconds
		DynamicMinRate:         1024 * 1024,      // 1 MB/s
		DynamicMaxRate:         100 * 1024 * 1024, // 100 MB/s
		DynamicTargetUtilPct:   0.80,              // 80% target utilization
		DynamicAdjustmentStep:  0.10,              // 10% adjustment step
	}
}

func setConfigDefaults(cfg *Config) {
	if cfg.ProducerBytesPerSecond == 0 {
		cfg.ProducerBytesPerSecond = 10 * 1024 * 1024
	}
	if cfg.ProducerBurst == 0 {
		cfg.ProducerBurst = int(cfg.ProducerBytesPerSecond * 2)
	}
	if cfg.ConsumerBytesPerSecond == 0 {
		cfg.ConsumerBytesPerSecond = 10 * 1024 * 1024
	}
	if cfg.ConsumerBurst == 0 {
		cfg.ConsumerBurst = int(cfg.ConsumerBytesPerSecond * 2)
	}
	if cfg.DynamicCheckInterval == 0 {
		cfg.DynamicCheckInterval = 5000
	}
	if cfg.DynamicMinRate == 0 {
		cfg.DynamicMinRate = 1024 * 1024
	}
	if cfg.DynamicMaxRate == 0 {
		cfg.DynamicMaxRate = 100 * 1024 * 1024
	}
	if cfg.DynamicTargetUtilPct == 0 {
		cfg.DynamicTargetUtilPct = 0.80
	}
	if cfg.DynamicAdjustmentStep == 0 {
		cfg.DynamicAdjustmentStep = 0.10
	}
}

// AllowProducer checks if producer request can proceed
func (t *Throttler) AllowProducer(ctx context.Context, bytes int) error {
	if t.config.ProducerBytesPerSecond <= 0 {
		// Throttling disabled
		t.producerAllowed.Add(int64(bytes))
		return nil
	}
	
	err := t.producerLimiter.WaitN(ctx, bytes)
	if err != nil {
		t.producerThrottled.Add(int64(bytes))
		metrics.ThrottleRequests.WithLabelValues("producer", "throttled").Inc()
		metrics.ThrottleBytes.WithLabelValues("producer", "throttled").Add(float64(bytes))
		return err
	}
	
	t.producerAllowed.Add(int64(bytes))
	metrics.ThrottleRequests.WithLabelValues("producer", "allowed").Inc()
	metrics.ThrottleBytes.WithLabelValues("producer", "allowed").Add(float64(bytes))
	return nil
}

// AllowConsumer checks if consumer request can proceed
func (t *Throttler) AllowConsumer(ctx context.Context, bytes int) error {
	if t.config.ConsumerBytesPerSecond <= 0 {
		// Throttling disabled
		t.consumerAllowed.Add(int64(bytes))
		return nil
	}
	
	err := t.consumerLimiter.WaitN(ctx, bytes)
	if err != nil {
		t.consumerThrottled.Add(int64(bytes))
		metrics.ThrottleRequests.WithLabelValues("consumer", "throttled").Inc()
		metrics.ThrottleBytes.WithLabelValues("consumer", "throttled").Add(float64(bytes))
		return err
	}
	
	t.consumerAllowed.Add(int64(bytes))
	metrics.ThrottleRequests.WithLabelValues("consumer", "allowed").Inc()
	metrics.ThrottleBytes.WithLabelValues("consumer", "allowed").Add(float64(bytes))
	return nil
}

// UpdateProducerRate dynamically updates the producer rate limit
func (t *Throttler) UpdateProducerRate(bytesPerSecond int64, burst int) {
	if bytesPerSecond < t.config.DynamicMinRate {
		bytesPerSecond = t.config.DynamicMinRate
	}
	if bytesPerSecond > t.config.DynamicMaxRate {
		bytesPerSecond = t.config.DynamicMaxRate
	}
	
	t.adjustmentMu.Lock()
	defer t.adjustmentMu.Unlock()
	
	t.producerLimiter.SetLimit(rate.Limit(bytesPerSecond))
	if burst > 0 {
		t.producerLimiter.SetBurst(burst)
	}
	t.producerRate.Store(bytesPerSecond)
	
	metrics.ThrottleRate.WithLabelValues("producer").Set(float64(bytesPerSecond))
	
	t.logger.Info("updated producer rate",
		"bytes_per_second", bytesPerSecond,
		"burst", burst,
	)
}

// UpdateConsumerRate dynamically updates the consumer rate limit
func (t *Throttler) UpdateConsumerRate(bytesPerSecond int64, burst int) {
	if bytesPerSecond < t.config.DynamicMinRate {
		bytesPerSecond = t.config.DynamicMinRate
	}
	if bytesPerSecond > t.config.DynamicMaxRate {
		bytesPerSecond = t.config.DynamicMaxRate
	}
	
	t.adjustmentMu.Lock()
	defer t.adjustmentMu.Unlock()
	
	t.consumerLimiter.SetLimit(rate.Limit(bytesPerSecond))
	if burst > 0 {
		t.consumerLimiter.SetBurst(burst)
	}
	t.consumerRate.Store(bytesPerSecond)
	
	metrics.ThrottleRate.WithLabelValues("consumer").Set(float64(bytesPerSecond))
	
	t.logger.Info("updated consumer rate",
		"bytes_per_second", bytesPerSecond,
		"burst", burst,
	)
}

// GetStats returns current throttle statistics
func (t *Throttler) GetStats() Stats {
	return Stats{
		ProducerRate:      t.producerRate.Load(),
		ProducerThrottled: t.producerThrottled.Load(),
		ProducerAllowed:   t.producerAllowed.Load(),
		ConsumerRate:      t.consumerRate.Load(),
		ConsumerThrottled: t.consumerThrottled.Load(),
		ConsumerAllowed:   t.consumerAllowed.Load(),
	}
}

// Stats holds throttle statistics
type Stats struct {
	ProducerRate      int64
	ProducerThrottled int64
	ProducerAllowed   int64
	ConsumerRate      int64
	ConsumerThrottled int64
	ConsumerAllowed   int64
}

// dynamicAdjustmentLoop periodically adjusts rate limits based on utilization
func (t *Throttler) dynamicAdjustmentLoop() {
	defer t.wg.Done()
	
	ticker := time.NewTicker(time.Duration(t.config.DynamicCheckInterval) * time.Millisecond)
	defer ticker.Stop()
	
	var lastProducerAllowed, lastConsumerAllowed int64
	
	for {
		select {
		case <-ticker.C:
			t.adjustRates(&lastProducerAllowed, &lastConsumerAllowed)
		case <-t.stopChan:
			return
		}
	}
}

// adjustRates adjusts rate limits based on current utilization
func (t *Throttler) adjustRates(lastProducerAllowed, lastConsumerAllowed *int64) {
	currentProducerAllowed := t.producerAllowed.Load()
	currentConsumerAllowed := t.consumerAllowed.Load()
	
	// Calculate bytes transferred in interval
	producerBytes := currentProducerAllowed - *lastProducerAllowed
	consumerBytes := currentConsumerAllowed - *lastConsumerAllowed
	
	*lastProducerAllowed = currentProducerAllowed
	*lastConsumerAllowed = currentConsumerAllowed
	
	// Calculate rates (bytes per second)
	intervalSec := float64(t.config.DynamicCheckInterval) / 1000.0
	producerActualRate := float64(producerBytes) / intervalSec
	consumerActualRate := float64(consumerBytes) / intervalSec
	
	// Adjust producer rate
	currentProducerRate := t.producerRate.Load()
	if currentProducerRate > 0 {
		utilization := producerActualRate / float64(currentProducerRate)
		
		if utilization > t.config.DynamicTargetUtilPct {
			// Increase rate
			newRate := int64(float64(currentProducerRate) * (1.0 + t.config.DynamicAdjustmentStep))
			t.UpdateProducerRate(newRate, 0)
			t.logger.Debug("increased producer rate",
				"utilization", utilization,
				"old_rate", currentProducerRate,
				"new_rate", newRate,
			)
		} else if utilization < t.config.DynamicTargetUtilPct*0.5 {
			// Decrease rate
			newRate := int64(float64(currentProducerRate) * (1.0 - t.config.DynamicAdjustmentStep))
			t.UpdateProducerRate(newRate, 0)
			t.logger.Debug("decreased producer rate",
				"utilization", utilization,
				"old_rate", currentProducerRate,
				"new_rate", newRate,
			)
		}
	}
	
	// Adjust consumer rate
	currentConsumerRate := t.consumerRate.Load()
	if currentConsumerRate > 0 {
		utilization := consumerActualRate / float64(currentConsumerRate)
		
		if utilization > t.config.DynamicTargetUtilPct {
			// Increase rate
			newRate := int64(float64(currentConsumerRate) * (1.0 + t.config.DynamicAdjustmentStep))
			t.UpdateConsumerRate(newRate, 0)
			t.logger.Debug("increased consumer rate",
				"utilization", utilization,
				"old_rate", currentConsumerRate,
				"new_rate", newRate,
			)
		} else if utilization < t.config.DynamicTargetUtilPct*0.5 {
			// Decrease rate
			newRate := int64(float64(currentConsumerRate) * (1.0 - t.config.DynamicAdjustmentStep))
			t.UpdateConsumerRate(newRate, 0)
			t.logger.Debug("decreased consumer rate",
				"utilization", utilization,
				"old_rate", currentConsumerRate,
				"new_rate", newRate,
			)
		}
	}
}

// Close stops the throttler and cleans up resources
func (t *Throttler) Close() error {
	close(t.stopChan)
	t.wg.Wait()
	t.logger.Info("throttler closed")
	return nil
}
