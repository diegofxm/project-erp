package database

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps a pgxpool with migration support.
type DB struct {
	Pool *pgxpool.Pool
	url  string
}

// Config holds connection pool parameters.
type Config struct {
	URL     string
	MaxConn int32
	MinConn int32
}

// New opens and validates a connection pool to PostgreSQL.
func New(ctx context.Context, cfg Config) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConn
	poolCfg.MinConns = cfg.MinConn
	poolCfg.MaxConnLifetime = 30 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute
	poolCfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{Pool: pool, url: cfg.URL}, nil
}

// Migrate runs all pending up-migrations using embedded SQL files.
func (d *DB) Migrate() error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("load migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, d.url)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

// Ping checks database connectivity.
func (d *DB) Ping(ctx context.Context) error {
	return d.Pool.Ping(ctx)
}

// Close releases all connections in the pool.
func (d *DB) Close() {
	d.Pool.Close()
}
