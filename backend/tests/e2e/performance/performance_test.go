// Copyright 2025 Takhin Data, Inc.

// +build e2e

package performance

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/takhin-data/takhin/tests/e2e/testutil"
)

func TestProduceThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "throughput-test"
	err := srv.CreateTopic(topicName, 4)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Measure produce throughput
	numMessages := 10000
	messageSize := 1024 // 1KB
	value := make([]byte, messageSize)

	startTime := time.Now()
	successCount := 0

	for i := 0; i < numMessages; i++ {
		partition := int32(i % 4)
		err := client.Produce(topicName, partition, []byte(fmt.Sprintf("key%d", i)), value)
		if err == nil {
			successCount++
		}
	}

	duration := time.Since(startTime)
	throughputMBps := float64(successCount*messageSize) / duration.Seconds() / (1024 * 1024)
	throughputMsgSec := float64(successCount) / duration.Seconds()

	t.Logf("Produce Throughput: %.2f MB/s, %.2f msg/s", throughputMBps, throughputMsgSec)
	t.Logf("Duration: %v, Messages: %d/%d", duration, successCount, numMessages)

	assert.Greater(t, throughputMsgSec, 100.0, "Should achieve at least 100 msg/s")
}

func TestConsumeThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "consume-throughput-test"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Produce test data
	numMessages := 5000
	messageSize := 1024
	value := make([]byte, messageSize)

	for i := 0; i < numMessages; i++ {
		_ = client.Produce(topicName, 0, []byte(fmt.Sprintf("key%d", i)), value)
	}

	time.Sleep(500 * time.Millisecond)

	// Measure consume throughput
	startTime := time.Now()
	totalRecords := 0
	offset := int64(0)

	for {
		records, err := client.Fetch(topicName, 0, offset, 1024*1024)
		require.NoError(t, err)

		if len(records) == 0 {
			break
		}

		totalRecords += len(records)
		offset += int64(len(records))

		if totalRecords >= numMessages/2 {
			break // Read at least half
		}
	}

	duration := time.Since(startTime)
	throughputMBps := float64(totalRecords*messageSize) / duration.Seconds() / (1024 * 1024)
	throughputMsgSec := float64(totalRecords) / duration.Seconds()

	t.Logf("Consume Throughput: %.2f MB/s, %.2f msg/s", throughputMBps, throughputMsgSec)
	t.Logf("Duration: %v, Records: %d", duration, totalRecords)

	assert.Greater(t, throughputMsgSec, 100.0, "Should achieve at least 100 msg/s")
}

func TestConcurrentProducers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "concurrent-producers-test"
	numPartitions := 8
	err := srv.CreateTopic(topicName, numPartitions)
	require.NoError(t, err)

	numProducers := 10
	messagesPerProducer := 500

	var wg sync.WaitGroup
	var totalSuccess int64

	startTime := time.Now()

	for p := 0; p < numProducers; p++ {
		wg.Add(1)
		go func(producerID int) {
			defer wg.Done()

			client, err := testutil.NewKafkaClient(srv.Address())
			if err != nil {
				t.Logf("Producer %d failed to connect: %v", producerID, err)
				return
			}
			defer client.Close()

			for i := 0; i < messagesPerProducer; i++ {
				partition := int32(i % numPartitions)
				key := fmt.Sprintf("p%d-k%d", producerID, i)
				value := fmt.Sprintf("p%d-v%d", producerID, i)

				err := client.Produce(topicName, partition, []byte(key), []byte(value))
				if err == nil {
					atomic.AddInt64(&totalSuccess, 1)
				}
			}
		}(p)
	}

	wg.Wait()
	duration := time.Since(startTime)

	totalMessages := int64(numProducers * messagesPerProducer)
	throughput := float64(totalSuccess) / duration.Seconds()

	t.Logf("Concurrent Producers: %d producers, %d messages each", numProducers, messagesPerProducer)
	t.Logf("Success: %d/%d (%.1f%%)", totalSuccess, totalMessages, float64(totalSuccess)/float64(totalMessages)*100)
	t.Logf("Throughput: %.2f msg/s, Duration: %v", throughput, duration)

	assert.Greater(t, float64(totalSuccess)/float64(totalMessages), 0.8, "At least 80% should succeed")
}

func TestConcurrentConsumers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "concurrent-consumers-test"
	numPartitions := 4
	err := srv.CreateTopic(topicName, numPartitions)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)

	// Produce test data
	for i := 0; i < 1000; i++ {
		partition := int32(i % numPartitions)
		_ = client.Produce(topicName, partition, []byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("value%d", i)))
	}
	client.Close()

	time.Sleep(300 * time.Millisecond)

	// Concurrent consumers
	numConsumers := 4
	var wg sync.WaitGroup
	var totalRead int64

	startTime := time.Now()

	for c := 0; c < numConsumers; c++ {
		wg.Add(1)
		go func(consumerID int) {
			defer wg.Done()

			client, err := testutil.NewKafkaClient(srv.Address())
			if err != nil {
				return
			}
			defer client.Close()

			// Each consumer reads from one partition
			partition := int32(consumerID % numPartitions)
			records, err := client.Fetch(topicName, partition, 0, 1024*1024)
			if err == nil {
				atomic.AddInt64(&totalRead, int64(len(records)))
			}
		}(c)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("Concurrent Consumers: %d consumers, read %d records total", numConsumers, totalRead)
	t.Logf("Duration: %v, Throughput: %.2f msg/s", duration, float64(totalRead)/duration.Seconds())

	assert.Greater(t, totalRead, int64(500), "Should read at least half of produced messages")
}

func TestLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "latency-test"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Measure end-to-end latency
	numSamples := 100
	latencies := make([]time.Duration, 0, numSamples)

	for i := 0; i < numSamples; i++ {
		start := time.Now()

		// Produce
		key := fmt.Sprintf("latency-key-%d", i)
		value := fmt.Sprintf("latency-value-%d", i)
		err := client.Produce(topicName, 0, []byte(key), []byte(value))
		require.NoError(t, err)

		// Immediate fetch
		_, err = client.Fetch(topicName, 0, int64(i), 1024)
		require.NoError(t, err)

		latency := time.Since(start)
		latencies = append(latencies, latency)

		time.Sleep(10 * time.Millisecond) // Small delay between samples
	}

	// Calculate statistics
	var total time.Duration
	var max time.Duration
	min := latencies[0]

	for _, lat := range latencies {
		total += lat
		if lat > max {
			max = lat
		}
		if lat < min {
			min = lat
		}
	}

	avg := total / time.Duration(len(latencies))

	t.Logf("Latency Statistics (produce + fetch):")
	t.Logf("  Average: %v", avg)
	t.Logf("  Min: %v", min)
	t.Logf("  Max: %v", max)

	assert.Less(t, avg, 100*time.Millisecond, "Average latency should be under 100ms")
}

func TestBackpressure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "backpressure-test"
	err := srv.CreateTopic(topicName, 1)
	require.NoError(t, err)

	client, err := testutil.NewKafkaClient(srv.Address())
	require.NoError(t, err)
	defer client.Close()

	// Produce rapidly without consuming
	numMessages := 5000
	largeValue := make([]byte, 10*1024) // 10KB messages

	startTime := time.Now()
	successCount := 0

	for i := 0; i < numMessages; i++ {
		err := client.Produce(topicName, 0, []byte(fmt.Sprintf("key%d", i)), largeValue)
		if err == nil {
			successCount++
		} else if i > numMessages/2 {
			// Expect some failures due to backpressure
			t.Logf("Backpressure detected at message %d", i)
			break
		}
	}

	duration := time.Since(startTime)

	t.Logf("Backpressure Test: %d/%d messages succeeded", successCount, numMessages)
	t.Logf("Duration: %v, Rate: %.2f msg/s", duration, float64(successCount)/duration.Seconds())

	// System should handle at least some messages before backpressure
	assert.Greater(t, successCount, numMessages/10, "Should handle at least 10% before backpressure")
}

func TestLongRunningProducerConsumer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	srv := testutil.NewTestServer(t)
	defer srv.Close()

	topicName := "long-running-test"
	err := srv.CreateTopic(topicName, 2)
	require.NoError(t, err)

	var wg sync.WaitGroup
	stopChan := make(chan struct{})
	var produceCount, consumeCount int64

	// Producer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		client, _ := testutil.NewKafkaClient(srv.Address())
		defer client.Close()

		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				err := client.Produce(topicName, 0, []byte("key"), []byte("value"))
				if err == nil {
					atomic.AddInt64(&produceCount, 1)
				}
			}
		}
	}()

	// Consumer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		client, _ := testutil.NewKafkaClient(srv.Address())
		defer client.Close()

		offset := int64(0)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				records, err := client.Fetch(topicName, 0, offset, 1024*1024)
				if err == nil && len(records) > 0 {
					atomic.AddInt64(&consumeCount, int64(len(records)))
					offset += int64(len(records))
				}
			}
		}
	}()

	// Run for 10 seconds
	time.Sleep(10 * time.Second)
	close(stopChan)
	wg.Wait()

	t.Logf("Long Running Test (10s):")
	t.Logf("  Produced: %d messages", produceCount)
	t.Logf("  Consumed: %d messages", consumeCount)
	t.Logf("  Lag: %d messages", produceCount-consumeCount)

	assert.Greater(t, produceCount, int64(100), "Should produce significant messages")
	assert.Greater(t, float64(consumeCount)/float64(produceCount), 0.5, "Should consume at least 50% of produced")
}
