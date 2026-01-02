// Copyright 2025 Takhin Data, Inc.

package handler

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ProduceWaiter manages produce requests waiting for ISR acknowledgment
type ProduceWaiter struct {
	mu       sync.RWMutex
	waiters  map[string]map[int32]map[int64]*produceWaitEntry // topic -> partition -> offset -> waiter
	notifyCh chan notifyEvent
}

// produceWaitEntry represents a produce request waiting for acknowledgment
type produceWaitEntry struct {
	offset     int64
	acks       int16
	doneCh     chan error
	cancelFunc context.CancelFunc
}

// notifyEvent represents a notification that HWM has advanced
type notifyEvent struct {
	topic     string
	partition int32
	hwm       int64
}

// NewProduceWaiter creates a new produce waiter
func NewProduceWaiter() *ProduceWaiter {
	pw := &ProduceWaiter{
		waiters:  make(map[string]map[int32]map[int64]*produceWaitEntry),
		notifyCh: make(chan notifyEvent, 1000),
	}
	go pw.notifyLoop()
	return pw
}

// WaitForAck waits for ISR acknowledgment of a produce request
// Returns error if timeout or acknowledgment fails
func (pw *ProduceWaiter) WaitForAck(ctx context.Context, topic string, partition int32, offset int64, acks int16, timeout time.Duration) error {
	// acks=0 or acks=1: no waiting needed
	if acks >= 0 {
		return nil
	}

	// Create waiter entry
	doneCh := make(chan error, 1)
	timeoutCtx, cancelFunc := context.WithTimeout(ctx, timeout)

	entry := &produceWaitEntry{
		offset:     offset,
		acks:       acks,
		doneCh:     doneCh,
		cancelFunc: cancelFunc,
	}

	// Register waiter
	pw.mu.Lock()
	if _, exists := pw.waiters[topic]; !exists {
		pw.waiters[topic] = make(map[int32]map[int64]*produceWaitEntry)
	}
	if _, exists := pw.waiters[topic][partition]; !exists {
		pw.waiters[topic][partition] = make(map[int64]*produceWaitEntry)
	}
	pw.waiters[topic][partition][offset] = entry
	pw.mu.Unlock()

	// Wait for acknowledgment or timeout
	select {
	case err := <-doneCh:
		cancelFunc()
		return err
	case <-timeoutCtx.Done():
		// Timeout: remove waiter
		pw.removeWaiter(topic, partition, offset)
		return fmt.Errorf("produce request timeout after %v", timeout)
	}
}

// NotifyHWMAdvanced notifies waiters that HWM has advanced
func (pw *ProduceWaiter) NotifyHWMAdvanced(topic string, partition int32, hwm int64) {
	select {
	case pw.notifyCh <- notifyEvent{topic: topic, partition: partition, hwm: hwm}:
	default:
		// Channel full, skip notification (will be handled on next update)
	}
}

// notifyLoop processes HWM advancement notifications
func (pw *ProduceWaiter) notifyLoop() {
	for event := range pw.notifyCh {
		pw.processHWMAdvance(event.topic, event.partition, event.hwm)
	}
}

// processHWMAdvance checks all waiters and completes those that are satisfied
func (pw *ProduceWaiter) processHWMAdvance(topic string, partition int32, hwm int64) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	partWaiters, exists := pw.waiters[topic][partition]
	if !exists {
		return
	}

	// Complete all waiters with offset <= HWM
	for offset, entry := range partWaiters {
		if offset <= hwm {
			select {
			case entry.doneCh <- nil:
				entry.cancelFunc()
				delete(partWaiters, offset)
			default:
				// Channel already closed or consumed
			}
		}
	}

	// Cleanup empty maps
	if len(partWaiters) == 0 {
		delete(pw.waiters[topic], partition)
	}
	if len(pw.waiters[topic]) == 0 {
		delete(pw.waiters, topic)
	}
}

// removeWaiter removes a waiter (called on timeout)
func (pw *ProduceWaiter) removeWaiter(topic string, partition int32, offset int64) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if partWaiters, exists := pw.waiters[topic][partition]; exists {
		if entry, exists := partWaiters[offset]; exists {
			entry.cancelFunc()
			delete(partWaiters, offset)
		}

		if len(partWaiters) == 0 {
			delete(pw.waiters[topic], partition)
		}
	}
	if len(pw.waiters[topic]) == 0 {
		delete(pw.waiters, topic)
	}
}

// GetWaitingCount returns the number of waiting produce requests
func (pw *ProduceWaiter) GetWaitingCount() int {
	pw.mu.RLock()
	defer pw.mu.RUnlock()

	count := 0
	for _, partitions := range pw.waiters {
		for _, offsets := range partitions {
			count += len(offsets)
		}
	}
	return count
}

// Close stops the waiter and cancels all pending requests
func (pw *ProduceWaiter) Close() {
	close(pw.notifyCh)

	pw.mu.Lock()
	defer pw.mu.Unlock()

	// Cancel all pending waiters
	for _, partitions := range pw.waiters {
		for _, offsets := range partitions {
			for _, entry := range offsets {
				entry.cancelFunc()
				select {
				case entry.doneCh <- fmt.Errorf("produce waiter closed"):
				default:
				}
			}
		}
	}

	pw.waiters = make(map[string]map[int32]map[int64]*produceWaitEntry)
}
