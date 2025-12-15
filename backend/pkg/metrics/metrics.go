// Copyright 2025 Takhin Data, Inc.

package metrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/logger"
)

var (
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_requests_total",
			Help: "Total number of requests by API key",
		},
		[]string{"api_key"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "takhin_request_duration_seconds",
			Help:    "Request duration in seconds by API key",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"api_key"},
	)

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

	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "takhin_errors_total",
			Help: "Total number of errors by type",
		},
		[]string{"type"},
	)
)

type Server struct {
	config *config.Config
	logger *logger.Logger
	server *http.Server
}

func New(cfg *config.Config) *Server {
	return &Server{
		config: cfg,
		logger: logger.Default().WithComponent("metrics"),
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

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("metrics server error", "error", err)
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	if s.server != nil {
		s.logger.Info("stopping metrics server")
		return s.server.Close()
	}
	return nil
}
