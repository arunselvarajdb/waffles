package database

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/waffles/waffles/internal/config"
	"github.com/waffles/waffles/pkg/logger"
)

// DB wraps the pgxpool.Pool
type DB struct {
	Pool   *pgxpool.Pool
	logger logger.Logger
}

// NewPostgresDB creates a new PostgreSQL connection pool
func NewPostgresDB(cfg config.DatabaseConfig, log logger.Logger) (*DB, error) {
	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Database,
	)

	// Parse config
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Set connection pool settings with bounds checking to avoid integer overflow
	maxConns := cfg.MaxOpenConns
	if maxConns < 0 {
		maxConns = 0
	} else if maxConns > math.MaxInt32 {
		maxConns = math.MaxInt32
	}
	poolConfig.MaxConns = int32(maxConns) // #nosec G115 -- bounds checked above

	minConns := cfg.MaxIdleConns
	if minConns < 0 {
		minConns = 0
	} else if minConns > math.MaxInt32 {
		minConns = math.MaxInt32
	}
	poolConfig.MinConns = int32(minConns) // #nosec G115 -- bounds checked above

	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime
	poolConfig.MaxConnIdleTime = cfg.ConnMaxIdleTime

	// Create connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Ping to verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Database).
		Int("max_conns", cfg.MaxOpenConns).
		Msg("Database connection pool established")

	return &DB{
		Pool:   pool,
		logger: log,
	}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	if db.Pool != nil {
		db.logger.Info().Msg("Closing database connection pool")
		db.Pool.Close()
	}
}

// Ping checks if the database is accessible
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Stats returns connection pool statistics
func (db *DB) Stats() *pgxpool.Stat {
	return db.Pool.Stat()
}
