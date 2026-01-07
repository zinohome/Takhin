// Copyright 2025 Takhin Data, Inc.

package profiler

import (
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/takhin-data/takhin/pkg/config"
	"github.com/takhin-data/takhin/pkg/logger"
)

type Server struct {
	config *config.Config
	logger *logger.Logger
	server *http.Server
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		config: cfg,
		logger: logger.Default().WithComponent("profiler-server"),
	}
}

func (s *Server) Start() error {
	if !s.config.Profiler.Enabled {
		s.logger.Info("profiler server disabled")
		return nil
	}

	addr := fmt.Sprintf("%s:%d", s.config.Profiler.Host, s.config.Profiler.Port)

	mux := http.NewServeMux()
	
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))

	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.logger.Info("starting profiler server",
		"address", addr,
		"endpoints", []string{
			"/debug/pprof/",
			"/debug/pprof/heap",
			"/debug/pprof/goroutine",
			"/debug/pprof/profile",
			"/debug/pprof/trace",
		},
	)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("profiler server error", "error", err)
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	if s.server != nil {
		s.logger.Info("stopping profiler server")
		return s.server.Close()
	}
	return nil
}
