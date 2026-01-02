// Copyright 2025 Takhin Data, Inc.

package grpcapi

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/takhin-data/takhin/pkg/coordinator"
	"github.com/takhin-data/takhin/pkg/logger"
	"github.com/takhin-data/takhin/pkg/storage/topic"
)

// GRPCServer manages the gRPC server lifecycle
type GRPCServer struct {
	server       *grpc.Server
	listener     net.Listener
	apiServer    *Server
	logger       *logger.Logger
	healthServer *health.Server
}

// NewGRPCServer creates a new gRPC server
func NewGRPCServer(addr string, topicManager *topic.Manager, coord *coordinator.Coordinator, version string) (*GRPCServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	// Configure gRPC server options
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(10 * 1024 * 1024), // 10MB
		grpc.MaxSendMsgSize(10 * 1024 * 1024), // 10MB
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Minute,
			MaxConnectionAge:      30 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Minute,
			Time:                  5 * time.Minute,
			Timeout:               1 * time.Minute,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             1 * time.Minute,
			PermitWithoutStream: true,
		}),
	}

	grpcServer := grpc.NewServer(opts...)

	// Create API server
	apiServer := NewServer(topicManager, coord, version)

	// Register Takhin service (would be generated from proto)
	// RegisterTakhinServiceServer(grpcServer, apiServer)

	// Register health check service
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("takhin.v1.TakhinService", healthpb.HealthCheckResponse_SERVING)

	// Register reflection service for debugging
	reflection.Register(grpcServer)

	return &GRPCServer{
		server:       grpcServer,
		listener:     listener,
		apiServer:    apiServer,
		logger:       logger.Default().WithComponent("grpc-server"),
		healthServer: healthServer,
	}, nil
}

// Start starts the gRPC server
func (s *GRPCServer) Start() error {
	s.logger.Info("Starting gRPC server", "addr", s.listener.Addr().String())

	if err := s.server.Serve(s.listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Stop gracefully stops the gRPC server
func (s *GRPCServer) Stop() {
	s.logger.Info("Stopping gRPC server")
	
	// Mark as not serving
	s.healthServer.SetServingStatus("takhin.v1.TakhinService", healthpb.HealthCheckResponse_NOT_SERVING)
	
	// Graceful stop with timeout
	stopped := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		s.logger.Info("gRPC server stopped gracefully")
	case <-time.After(30 * time.Second):
		s.logger.Warn("Graceful stop timeout, forcing stop")
		s.server.Stop()
	}
}

// Addr returns the server's listening address
func (s *GRPCServer) Addr() net.Addr {
	return s.listener.Addr()
}

// HealthCheck performs a health check
func (s *GRPCServer) HealthCheck(ctx context.Context) error {
	// Basic health check - can be extended
	return nil
}
