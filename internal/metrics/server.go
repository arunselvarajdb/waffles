package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/waffles/mcp-gateway/pkg/logger"
)

// Server is a standalone HTTP server for exposing Prometheus metrics
type Server struct {
	registry   *Registry
	httpServer *http.Server
	logger     logger.Logger
	port       int
}

// NewServer creates a new metrics server
func NewServer(registry *Registry, port int, log logger.Logger) *Server {
	return &Server{
		registry: registry,
		port:     port,
		logger:   log.With().Str("component", "metrics-server").Logger(),
	}
}

// Start starts the metrics HTTP server on the configured port
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.HandlerFor(
		s.registry.GetRegistry(),
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	))

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info().
		Int("port", s.port).
		Msg("Starting metrics server")

	// Start server in a goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error().
				Err(err).
				Msg("Metrics server error")
		}
	}()

	s.logger.Info().
		Int("port", s.port).
		Msg("Metrics server started successfully")

	return nil
}

// Shutdown gracefully shuts down the metrics server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	s.logger.Info().Msg("Shutting down metrics server")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		s.logger.Error().
			Err(err).
			Msg("Error shutting down metrics server")
		return err
	}

	s.logger.Info().Msg("Metrics server shut down successfully")
	return nil
}
