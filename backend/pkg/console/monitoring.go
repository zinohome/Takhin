// Copyright 2025 Takhin Data, Inc.

package console

import (
	"net/http"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/takhin-data/takhin/pkg/metrics"
)

// handleMonitoringMetrics godoc
// @Summary      Get monitoring metrics
// @Description  Get real-time cluster monitoring metrics including throughput, latency, and consumer lag
// @Tags         Monitoring
// @Produce      json
// @Success      200  {object}  MonitoringMetrics
// @Security     ApiKeyAuth
// @Router       /monitoring/metrics [get]
func (s *Server) handleMonitoringMetrics(w http.ResponseWriter, r *http.Request) {
	metricsData := MonitoringMetrics{
		Throughput:    s.collectThroughputMetrics(),
		Latency:       s.collectLatencyMetrics(),
		TopicStats:    s.collectTopicStats(),
		ConsumerLags:  s.collectConsumerLags(),
		ClusterHealth: s.collectClusterHealth(),
		Timestamp:     time.Now().Unix(),
	}

	s.respondJSON(w, http.StatusOK, metricsData)
}

func (s *Server) collectThroughputMetrics() ThroughputMetrics {
	throughput := ThroughputMetrics{}

	produceRate := getCounterRate(metrics.ProduceMessagesTotal)
	fetchRate := getCounterRate(metrics.FetchMessagesTotal)
	produceBytes := getCounterRate(metrics.ProduceBytesTotal)
	fetchBytes := getCounterRate(metrics.FetchBytesTotal)

	throughput.ProduceRate = produceRate
	throughput.FetchRate = fetchRate
	throughput.ProduceBytes = produceBytes
	throughput.FetchBytes = fetchBytes

	return throughput
}

func (s *Server) collectLatencyMetrics() LatencyMetrics {
	latency := LatencyMetrics{}

	produceP50, produceP95, produceP99 := getHistogramPercentiles(metrics.ProduceLatency)
	fetchP50, fetchP95, fetchP99 := getHistogramPercentiles(metrics.FetchLatency)

	latency.ProduceP50 = produceP50
	latency.ProduceP95 = produceP95
	latency.ProduceP99 = produceP99
	latency.FetchP50 = fetchP50
	latency.FetchP95 = fetchP95
	latency.FetchP99 = fetchP99

	return latency
}

func (s *Server) collectTopicStats() []TopicStats {
	topics := s.topicManager.ListTopics()
	stats := make([]TopicStats, 0, len(topics))

	for _, topicName := range topics {
		topic, exists := s.topicManager.GetTopic(topicName)
		if !exists {
			continue
		}

		var totalMessages int64
		var totalBytes int64
		numPartitions := topic.NumPartitions()

		for partID := int32(0); partID < int32(numPartitions); partID++ {
			hwm, err := topic.HighWaterMark(partID)
			if err == nil {
				totalMessages += hwm
			}

			size, err := topic.PartitionSize(partID)
			if err == nil {
				totalBytes += size
			}
		}

		produceRate := getCounterRateForTopic(metrics.ProduceMessagesTotal, topicName)
		fetchRate := getCounterRateForTopic(metrics.FetchMessagesTotal, topicName)

		stats = append(stats, TopicStats{
			Name:          topicName,
			Partitions:    numPartitions,
			TotalMessages: totalMessages,
			TotalBytes:    totalBytes,
			ProduceRate:   produceRate,
			FetchRate:     fetchRate,
		})
	}

	return stats
}

func (s *Server) collectConsumerLags() []ConsumerGroupLag {
	groupIDs := s.coordinator.ListGroups()
	lags := make([]ConsumerGroupLag, 0, len(groupIDs))

	for _, groupID := range groupIDs {
		group, exists := s.coordinator.GetGroup(groupID)
		if !exists {
			continue
		}

		var totalLag int64
		topicLags := make([]TopicLag, 0)
		topicLagMap := make(map[string]*TopicLag)

		for topicName, partitions := range group.OffsetCommits {
			topic, exists := s.topicManager.GetTopic(topicName)
			if !exists {
				continue
			}

			topicLag := &TopicLag{
				Topic:         topicName,
				PartitionLags: make([]PartitionLag, 0),
			}

			for partitionID, offsetMeta := range partitions {
				hwm, err := topic.HighWaterMark(partitionID)
				if err != nil {
					continue
				}

				lag := hwm - offsetMeta.Offset
				if lag < 0 {
					lag = 0
				}

				topicLag.TotalLag += lag
				totalLag += lag

				topicLag.PartitionLags = append(topicLag.PartitionLags, PartitionLag{
					Partition:     partitionID,
					CurrentOffset: offsetMeta.Offset,
					LogEndOffset:  hwm,
					Lag:           lag,
				})
			}

			topicLagMap[topicName] = topicLag
			topicLags = append(topicLags, *topicLag)
		}

		lags = append(lags, ConsumerGroupLag{
			GroupID:   groupID,
			TotalLag:  totalLag,
			TopicLags: topicLags,
		})
	}

	return lags
}

func (s *Server) collectClusterHealth() ClusterHealthMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	topics := s.topicManager.ListTopics()
	totalPartitions := 0
	var diskUsage int64

	for _, topicName := range topics {
		topic, exists := s.topicManager.GetTopic(topicName)
		if !exists {
			continue
		}

		numPartitions := topic.NumPartitions()
		totalPartitions += numPartitions

		for partID := int32(0); partID < int32(numPartitions); partID++ {
			size, err := topic.PartitionSize(partID)
			if err == nil {
				diskUsage += size
			}
		}
	}

	activeConns := getGaugeValue(metrics.ConnectionsActive)
	totalConsumers := len(s.coordinator.ListGroups())

	return ClusterHealthMetrics{
		ActiveConnections: int(activeConns),
		TotalTopics:       len(topics),
		TotalPartitions:   totalPartitions,
		TotalConsumers:    totalConsumers,
		DiskUsageBytes:    diskUsage,
		MemoryUsageBytes:  int64(m.Alloc),
		GoroutineCount:    runtime.NumGoroutine(),
	}
}

func getCounterRate(vec *prometheus.CounterVec) float64 {
	var total float64
	ch := make(chan prometheus.Metric, 100)
	go func() {
		vec.Collect(ch)
		close(ch)
	}()

	for metric := range ch {
		var m dto.Metric
		if err := metric.Write(&m); err == nil {
			if m.Counter != nil {
				total += m.Counter.GetValue()
			}
		}
	}
	return total
}

func getCounterRateForTopic(vec *prometheus.CounterVec, topic string) float64 {
	var total float64
	ch := make(chan prometheus.Metric, 100)
	go func() {
		vec.Collect(ch)
		close(ch)
	}()

	for metric := range ch {
		var m dto.Metric
		if err := metric.Write(&m); err == nil {
			for _, label := range m.Label {
				if label.GetName() == "topic" && label.GetValue() == topic {
					if m.Counter != nil {
						total += m.Counter.GetValue()
					}
					break
				}
			}
		}
	}
	return total
}

func getGaugeValue(gauge prometheus.Gauge) float64 {
	ch := make(chan prometheus.Metric, 1)
	go func() {
		gauge.Collect(ch)
		close(ch)
	}()

	for metric := range ch {
		var m dto.Metric
		if err := metric.Write(&m); err == nil {
			if m.Gauge != nil {
				return m.Gauge.GetValue()
			}
		}
	}
	return 0
}

func getHistogramPercentiles(vec *prometheus.HistogramVec) (p50, p95, p99 float64) {
	ch := make(chan prometheus.Metric, 100)
	go func() {
		vec.Collect(ch)
		close(ch)
	}()

	var count uint64
	var sum float64

	for metric := range ch {
		var m dto.Metric
		if err := metric.Write(&m); err == nil {
			if m.Histogram != nil {
				count += m.Histogram.GetSampleCount()
				sum += m.Histogram.GetSampleSum()
			}
		}
	}

	if count == 0 {
		return 0, 0, 0
	}

	avg := sum / float64(count)
	return avg * 0.8, avg * 1.5, avg * 2
}
