package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/waffles/waffles/internal/config"
	"github.com/waffles/waffles/internal/database"
	"github.com/waffles/waffles/pkg/logger"
)

var (
	configPath = flag.String("config", "", "path to config file")
	direction  = flag.String("direction", "up", "migration direction (up or down)")
	version    = flag.Uint("version", 0, "migrate to specific version (0 = latest)")
	status     = flag.Bool("status", false, "show migration status")
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

	// Build database URL
	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)

	// Show status and exit
	if *status {
		ver, dirty, err := database.MigrateStatus(dbURL)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get migration status")
			os.Exit(1)
		}

		if ver == 0 {
			fmt.Println("No migrations applied yet")
		} else {
			fmt.Printf("Current version: %d\n", ver)
			fmt.Printf("Dirty: %v\n", dirty)
		}
		return
	}

	// Run migrations
	log.Info().
		Str("host", cfg.Database.Host).
		Int("port", cfg.Database.Port).
		Str("database", cfg.Database.Database).
		Str("direction", *direction).
		Msg("Starting database migration")

	var migrationErr error

	switch *direction {
	case "up":
		if *version > 0 {
			migrationErr = database.MigrateTo(dbURL, *version, log)
		} else {
			migrationErr = database.MigrateUp(dbURL, log)
		}
	case "down":
		migrationErr = database.MigrateDown(dbURL, log)
	default:
		log.Error().Str("direction", *direction).Msg("Invalid migration direction")
		os.Exit(1)
	}

	if migrationErr != nil {
		log.Error().Err(migrationErr).Msg("Migration failed")
		os.Exit(1)
	}

	log.Info().Msg("Migration completed successfully")
}
