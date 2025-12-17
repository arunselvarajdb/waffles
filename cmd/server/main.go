package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/waffles/mcp-gateway/internal/config"
	"github.com/waffles/mcp-gateway/internal/database"
	"github.com/waffles/mcp-gateway/internal/metrics"
	"github.com/waffles/mcp-gateway/internal/server"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

//go:embed all:dist
var webAppFS embed.FS

var (
	version    = "dev"
	buildTime  = "unknown"
	configPath = flag.String("config", "", "path to config file")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := logger.NewZerolog(logger.Config{
		Level:  logger.Level(cfg.Logging.Level),
		Format: cfg.Logging.Format,
	})

	log.Info().
		Str("version", version).
		Str("build_time", buildTime).
		Str("environment", cfg.Server.Environment).
		Msg("Starting MCP Gateway")

	// Initialize database
	db, err := database.NewPostgresDB(cfg.Database, log)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to database")
		os.Exit(1)
	}

	// Run migrations
	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)

	if err := database.MigrateUp(dbURL, log); err != nil {
		log.Error().Err(err).Msg("Failed to run migrations")
		os.Exit(1)
	}

	// Initialize metrics
	var metricsRegistry *metrics.Registry
	var metricsServer *metrics.Server

	if cfg.Metrics.Enabled {
		log.Info().
			Int("port", cfg.Metrics.PrometheusPort).
			Msg("Initializing Prometheus metrics")

		metricsRegistry = metrics.NewRegistry()
		metricsServer = metrics.NewServer(metricsRegistry, cfg.Metrics.PrometheusPort, log)
	}

	// Create HTTP server
	srv := server.New(cfg, db, log, metricsRegistry, metricsServer, webAppFS)
	srv.SetupRoutes()

	// Create context that listens for shutdown signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start DB stats collector (collects every 10 seconds)
	if metricsRegistry != nil {
		dbStatsCollector := metrics.NewDBStatsCollector(metricsRegistry, db)
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			// Collect immediately on startup
			dbStatsCollector.Collect()

			for {
				select {
				case <-ticker.C:
					dbStatsCollector.Collect()
				case <-ctx.Done():
					return
				}
			}
		}()
		log.Info().Msg("Database stats collector started")
	}

	// Listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info().Msg("Shutdown signal received")
		cancel()
	}()

	log.Info().
		Str("address", srv.Addr()).
		Msg("MCP Gateway is ready to serve requests")

	// Start server (blocks until shutdown)
	if err := srv.Start(ctx); err != nil {
		log.Error().Err(err).Msg("Server error")
		os.Exit(1)
	}

	log.Info().Msg("MCP Gateway stopped")
}
