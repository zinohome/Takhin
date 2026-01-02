// Copyright 2025 Takhin Data, Inc.

package metrics

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/logger"
)

var (
	// Connection metrics
	ConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "takhin_connections_active",
			Help: "Number of active connections",
		},
	)

	ConnectionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "takhin_connections_total",
			Help: "Total number of connections",
		},
	)

	BytesSent = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "takhin_bytes_sent_total",
			Help: "Total bytes sent",
		},
	)

	BytesReceived = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "takhin_bytes_received_total",
			Help: "Total bytes received",
		},
	)

	// Kafka API metrics
	KafkaRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_kafka_requests_total",
			Help: "Total number of Kafka API requests by API key and version",
		},
		[]string{"api_key", "version"},
	)

	KafkaRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "takhin_kafka_request_duration_seconds",
			Help:    "Kafka request processing duration in seconds by API key",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"api_key"},
	)

	KafkaRequestErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_kafka_request_errors_total",
			Help: "Total number of Kafka API errors by API key and error code",
		},
		[]string{"api_key", "error_code"},
	)

	// Producer metrics
	ProduceRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_produce_requests_total",
			Help: "Total number of produce requests by topic",
		},
		[]string{"topic"},
	)

	ProduceMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_produce_messages_total",
			Help: "Total number of messages produced by topic and partition",
		},
		[]string{"topic", "partition"},
	)

	ProduceBytesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_produce_bytes_total",
			Help: "Total bytes produced by topic",
		},
		[]string{"topic"},
	)

	ProduceLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "takhin_produce_latency_seconds",
			Help:    "Produce request latency in seconds by topic",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
		[]string{"topic"},
	)

	// Consumer metrics
	FetchRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_fetch_requests_total",
			Help: "Total number of fetch requests by topic",
		},
		[]string{"topic"},
	)

	FetchMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_fetch_messages_total",
			Help: "Total number of messages fetched by topic and partition",
		},
		[]string{"topic", "partition"},
	)

	FetchBytesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_fetch_bytes_total",
			Help: "Total bytes fetched by topic",
		},
		[]string{"topic"},
	)

	FetchLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "takhin_fetch_latency_seconds",
			Help:    "Fetch request latency in seconds by topic",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
		[]string{"topic"},
	)

	// Storage metrics
	StorageDiskUsageBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "takhin_storage_disk_usage_bytes",
			Help: "Disk usage in bytes by topic and partition",
		},
		[]string{"topic", "partition"},
	)

	StorageLogSegments = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "takhin_storage_log_segments",
			Help: "Number of log segments by topic and partition",
		},
		[]string{"topic", "partition"},
	)

	StorageLogEndOffset = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "takhin_storage_log_end_offset",
			Help: "Log end offset (high water mark) by topic and partition",
		},
		[]string{"topic", "partition"},
	)

	StorageActiveSegmentBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "takhin_storage_active_segment_bytes",
			Help: "Active segment size in bytes by topic and partition",
		},
		[]string{"topic", "partition"},
	)

	StorageIOReads = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_storage_io_reads_total",
			Help: "Total number of storage read operations by topic",
		},
		[]string{"topic"},
	)

	StorageIOWrites = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_storage_io_writes_total",
			Help: "Total number of storage write operations by topic",
		},
		[]string{"topic"},
	)

	StorageIOErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_storage_io_errors_total",
			Help: "Total number of storage I/O errors by topic and operation",
		},
		[]string{"topic", "operation"},
	)

	// Replication metrics
	ReplicationLag = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "takhin_replication_lag_offsets",
			Help: "Replication lag in offsets by topic, partition and follower",
		},
		[]string{"topic", "partition", "follower_id"},
	)

	ReplicationISRSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "takhin_replication_isr_size",
			Help: "Number of in-sync replicas by topic and partition",
		},
		[]string{"topic", "partition"},
	)

	ReplicationReplicasTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "takhin_replication_replicas_total",
			Help: "Total number of replicas by topic and partition",
		},
		[]string{"topic", "partition"},
	)

	ReplicationFetchRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_replication_fetch_requests_total",
			Help: "Total number of replication fetch requests by follower",
		},
		[]string{"follower_id"},
	)

	ReplicationFetchLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "takhin_replication_fetch_latency_seconds",
			Help:    "Replication fetch latency in seconds by follower",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		},
		[]string{"follower_id"},
	)

	// Consumer Group metrics
	ConsumerGroupMembers = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "takhin_consumer_group_members",
			Help: "Number of members in consumer group",
		},
		[]string{"group_id"},
	)

	ConsumerGroupState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "takhin_consumer_group_state",
			Help: "Consumer group state (0=Dead, 1=Empty, 2=PreparingRebalance, 3=CompletingRebalance, 4=Stable)",
		},
		[]string{"group_id", "state"},
	)

	ConsumerGroupRebalances = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_consumer_group_rebalances_total",
			Help: "Total number of consumer group rebalances",
		},
		[]string{"group_id"},
	)

	ConsumerGroupLag = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "takhin_consumer_group_lag_offsets",
			Help: "Consumer group lag in offsets by group, topic and partition",
		},
		[]string{"group_id", "topic", "partition"},
	)

	ConsumerGroupCommitRate = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_consumer_group_commits_total",
			Help: "Total number of offset commits by group and topic",
		},
		[]string{"group_id", "topic"},
	)

	// Go Runtime metrics
	GoRoutines = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "takhin_go_goroutines",
			Help: "Number of goroutines",
		},
	)

	GoThreads = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "takhin_go_threads",
			Help: "Number of OS threads",
		},
	)

	GoMemAllocBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "takhin_go_mem_alloc_bytes",
			Help: "Bytes of allocated heap objects",
		},
	)

	GoMemTotalAllocBytes = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "takhin_go_mem_total_alloc_bytes",
			Help: "Cumulative bytes allocated for heap objects",
		},
	)

	GoMemSysBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "takhin_go_mem_sys_bytes",
			Help: "Total bytes of memory obtained from the OS",
		},
	)

	GoMemHeapAllocBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "takhin_go_mem_heap_alloc_bytes",
			Help: "Bytes of allocated heap objects",
		},
	)

	GoMemHeapIdleBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "takhin_go_mem_heap_idle_bytes",
			Help: "Bytes in idle heap spans",
		},
	)

	GoMemHeapInuseBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "takhin_go_mem_heap_inuse_bytes",
			Help: "Bytes in in-use heap spans",
		},
	)

	GoGCPauseSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "takhin_go_gc_pause_seconds",
			Help:    "GC pause duration in seconds",
			Buckets: []float64{.00001, .00005, .0001, .0005, .001, .005, .01, .05, .1},
		},
	)

	GoGCTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "takhin_go_gc_total",
			Help: "Total number of GC runs",
		},
	)

	// Legacy metrics (kept for backward compatibility)
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_requests_total",
			Help: "Total number of requests by API key (deprecated, use takhin_kafka_requests_total)",
		},
		[]string{"api_key"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "takhin_request_duration_seconds",
			Help:    "Request duration in seconds by API key (deprecated, use takhin_kafka_request_duration_seconds)",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"api_key"},
	)

	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_errors_total",
			Help: "Total number of errors by type (deprecated, use takhin_kafka_request_errors_total)",
		},
		[]string{"type"},
	)

	// Raft election metrics
	RaftElectionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "takhin_raft_elections_total",
			Help: "Total number of leader elections initiated",
		},
	)

	RaftElectionDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "takhin_raft_election_duration_seconds",
			Help:    "Duration of leader elections in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 3.0, 5.0, 10.0},
		},
	)

	RaftLeaderChanges = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "takhin_raft_leader_changes_total",
			Help: "Total number of leader changes",
		},
	)

	RaftState = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "takhin_raft_state",
			Help: "Current Raft state (0=follower, 1=candidate, 2=leader)",
		},
	)

	RaftPreVoteRequestsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "takhin_raft_prevote_requests_total",
			Help: "Total number of PreVote requests sent",
		},
	)

	RaftPreVoteGrantedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "takhin_raft_prevote_granted_total",
			Help: "Total number of PreVote requests granted",
		},
	)
)

type Server struct {
	config      *config.Config
	logger      *logger.Logger
	server      *http.Server
	stopChan    chan struct{}
	lastGCPause uint64
	lastNumGC   uint32
}

func New(cfg *config.Config) *Server {
	return &Server{
		config:   cfg,
		logger:   logger.Default().WithComponent("metrics"),
		stopChan: make(chan struct{}),
	}
}

func (s *Server) Start() error {
	if !s.config.Metrics.Enabled {
		s.logger.Info("metrics server disabled")
		return nil
	}

	addr := fmt.Sprintf("%s:%d", s.config.Metrics.Host, s.config.Metrics.Port)

	mux := http.NewServeMux()
	mux.Handle(s.config.Metrics.Path, promhttp.Handler())

	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.logger.Info("starting metrics server",
		"address", addr,
		"path", s.config.Metrics.Path,
	)

	// Start runtime metrics collector
	go s.collectRuntimeMetrics()

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("metrics server error", "error", err)
		}
	}()

	return nil
}

func (s *Server) collectRuntimeMetrics() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			// Update goroutine and thread counts
			GoRoutines.Set(float64(runtime.NumGoroutine()))
			GoThreads.Set(float64(runtime.GOMAXPROCS(0)))

			// Update memory stats
			GoMemAllocBytes.Set(float64(m.Alloc))
			GoMemTotalAllocBytes.Add(float64(m.TotalAlloc))
			GoMemSysBytes.Set(float64(m.Sys))
			GoMemHeapAllocBytes.Set(float64(m.HeapAlloc))
			GoMemHeapIdleBytes.Set(float64(m.HeapIdle))
			GoMemHeapInuseBytes.Set(float64(m.HeapInuse))

			// Update GC stats
			if m.NumGC > s.lastNumGC {
				// Record new GC pauses
				for i := s.lastNumGC; i < m.NumGC; i++ {
					pause := m.PauseNs[i%256]
					GoGCPauseSeconds.Observe(float64(pause) / 1e9)
					GoGCTotal.Inc()
				}
				s.lastNumGC = m.NumGC
			}

		case <-s.stopChan:
			return
		}
	}
}

func (s *Server) Stop() error {
	close(s.stopChan)
	if s.server != nil {
		s.logger.Info("stopping metrics server")
		return s.server.Close()
	}
	return nil
}
