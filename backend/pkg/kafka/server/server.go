// Copyright 2025 Takhin Data, Inc.

package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"

	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/kafka/handler"
	"github.com/takhin-data/takhin/pkg/logger"
	"github.com/takhin-data/takhin/pkg/storage/topic"
	tlsutil "github.com/takhin-data/takhin/pkg/tls"
)

// Server represents a Kafka protocol server
type Server struct {
	config   *config.Config
	handler  *handler.Handler
	logger   logger.Logger
	listener net.Listener
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// New creates a new Kafka server
func New(cfg *config.Config, topicMgr *topic.Manager) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		config:  cfg,
		handler: handler.New(cfg, topicMgr),
		logger:  *logger.Default().WithComponent("kafka-server"),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts the Kafka server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Kafka.AdvertisedHost, s.config.Kafka.AdvertisedPort)
	
	var listener net.Listener
	var err error

	// Check if TLS is enabled
	if s.config.Server.TLS.Enabled {
		tlsConfig, err := tlsutil.LoadTLSConfig(&s.config.Server.TLS)
		if err != nil {
			return fmt.Errorf("failed to load TLS config: %w", err)
		}

		listener, err = tls.Listen("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to listen on %s with TLS: %w", addr, err)
		}
		s.logger.Info("kafka server started with TLS", "address", addr, "client_auth", s.config.Server.TLS.ClientAuth)
	} else {
		listener, err = net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", addr, err)
		}
		s.logger.Info("kafka server started", "address", addr)
	}

	s.listener = listener

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.acceptLoop()
	}()

	return nil
}

// acceptLoop accepts incoming connections
func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				s.logger.Error("failed to accept connection", "error", err)
				continue
			}
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConnection(conn)
		}()
	}
}

// handleConnection handles a single connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	s.logger.Info("new connection", "remote", conn.RemoteAddr())

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// Read message size (4 bytes)
		sizeBuf := make([]byte, 4)
		if _, err := conn.Read(sizeBuf); err != nil {
			if err.Error() != "EOF" {
				s.logger.Error("failed to read message size", "error", err)
			}
			return
		}

		size := int32(sizeBuf[0])<<24 | int32(sizeBuf[1])<<16 | int32(sizeBuf[2])<<8 | int32(sizeBuf[3])

		// Read message body
		msgBuf := make([]byte, size)
		if _, err := conn.Read(msgBuf); err != nil {
			s.logger.Error("failed to read message", "error", err)
			return
		}

		// Check if this is a Fetch request (API Key 1) for zero-copy optimization
		// We need at least request header to determine the API key
		if len(msgBuf) >= 8 {
			apiKey := int16(msgBuf[0])<<8 | int16(msgBuf[1])
			
			// Fetch API key is 1
			if apiKey == 1 {
				// Try zero-copy path for Fetch requests
				err := s.handler.HandleFetchZeroCopy(msgBuf, conn)
				if err != nil {
					s.logger.Error("failed to handle fetch with zero-copy", "error", err)
					return
				}
				continue
			}
		}

		// Handle request normally for non-Fetch requests
		resp, err := s.handler.HandleRequest(msgBuf)
		if err != nil {
			s.logger.Error("failed to handle request", "error", err)
			return
		}

		// Write response size
		respSize := len(resp)
		respSizeBuf := []byte{
			byte(respSize >> 24),
			byte(respSize >> 16),
			byte(respSize >> 8),
			byte(respSize),
		}
		if _, err := conn.Write(respSizeBuf); err != nil {
			s.logger.Error("failed to write response size", "error", err)
			return
		}

		// Write response
		if _, err := conn.Write(resp); err != nil {
			s.logger.Error("failed to write response", "error", err)
			return
		}
	}
}

// Stop stops the Kafka server
func (s *Server) Stop() {
	s.cancel()
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
	s.logger.Info("kafka server stopped")
}
