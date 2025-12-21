package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/waffles/mcp-gateway/internal/config"
	"github.com/waffles/mcp-gateway/internal/database"
	"github.com/waffles/mcp-gateway/internal/metrics"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

// Server represents the HTTP server
type Server struct {
	config        *config.Config
	router        *gin.Engine
	httpServer    *http.Server
	db            *database.DB
	logger        logger.Logger
	metrics       *metrics.Registry
	metricsServer *metrics.Server
}

// New creates a new HTTP server instance
func New(cfg *config.Config, db *database.DB, log logger.Logger, metricsReg *metrics.Registry, metricsSrv *metrics.Server) *Server {
	// Set Gin mode based on environment
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg.Server.Environment == "development" {
		gin.SetMode(gin.DebugMode)
	}

	// Create Gin router
	router := gin.New()

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return &Server{
		config:        cfg,
		router:        router,
		httpServer:    httpServer,
		db:            db,
		logger:        log,
		metrics:       metricsReg,
		metricsServer: metricsSrv,
	}
}

// Router returns the Gin router for route registration
func (s *Server) Router() *gin.Engine {
	return s.router
}

// Start starts the HTTP server and metrics server
func (s *Server) Start(ctx context.Context) error {
	// Start metrics server if enabled
	if s.metricsServer != nil {
		if err := s.metricsServer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start metrics server: %w", err)
		}
	}

	s.logger.Info().
		Str("host", s.config.Server.Host).
		Int("port", s.config.Server.Port).
		Str("environment", s.config.Server.Environment).
		Msg("Starting HTTP server")

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		s.logger.Info().Msg("Shutdown signal received")
		return s.Shutdown()
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	s.logger.Info().Msg("Shutting down HTTP server gracefully...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Server.ShutdownTimeout)
	defer cancel()

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error().Err(err).Msg("HTTP server shutdown error")
		return err
	}

	// Shutdown metrics server
	if s.metricsServer != nil {
		if err := s.metricsServer.Shutdown(ctx); err != nil {
			s.logger.Error().Err(err).Msg("Metrics server shutdown error")
		}
	}

	// Close database connection
	if s.db != nil {
		s.db.Close()
	}

	s.logger.Info().Msg("Server shutdown complete")
	return nil
}

// Addr returns the server address
func (s *Server) Addr() string {
	return s.httpServer.Addr
}
