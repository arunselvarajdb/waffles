package database

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/waffles/mcp-gateway/pkg/logger"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrateUp runs all pending migrations
func MigrateUp(databaseURL string, log logger.Logger) error {
	m, err := newMigrate(databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()

	log.Info().Msg("Running database migrations...")

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info().Msg("No new migrations to apply")
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	log.Info().
		Uint("version", version).
		Bool("dirty", dirty).
		Msg("Migrations applied successfully")

	return nil
}

// MigrateDown rolls back the last migration
func MigrateDown(databaseURL string, log logger.Logger) error {
	m, err := newMigrate(databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()

	log.Warn().Msg("Rolling back last migration...")

	if err := m.Steps(-1); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info().Msg("No migrations to roll back")
			return nil
		}
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	log.Info().
		Uint("version", version).
		Bool("dirty", dirty).
		Msg("Migration rolled back successfully")

	return nil
}

// MigrateTo migrates to a specific version
func MigrateTo(databaseURL string, version uint, log logger.Logger) error {
	m, err := newMigrate(databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()

	log.Info().Uint("target_version", version).Msg("Migrating to specific version...")

	if err := m.Migrate(version); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info().Msg("Already at target version")
			return nil
		}
		return fmt.Errorf("failed to migrate to version %d: %w", version, err)
	}

	log.Info().Uint("version", version).Msg("Migrated to version successfully")
	return nil
}

// MigrateStatus returns the current migration status
func MigrateStatus(databaseURL string) (version uint, dirty bool, err error) {
	m, err := newMigrate(databaseURL)
	if err != nil {
		return 0, false, err
	}
	defer m.Close()

	version, dirty, err = m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}

	return version, dirty, err
}

// newMigrate creates a new migrate instance
func newMigrate(databaseURL string) (*migrate.Migrate, error) {
	// Create iofs driver from embedded filesystem
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return m, nil
}
