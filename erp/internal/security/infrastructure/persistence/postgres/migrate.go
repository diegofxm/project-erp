package postgres

import (
	"embed"
	"errors"
	"strings"

	migrate "github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Migrate(databaseURL string) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return err
	}

	dbURL := databaseURL
	dbURL = strings.TrimPrefix(dbURL, "postgresql://")
	dbURL = strings.TrimPrefix(dbURL, "postgres://")
	dbURL = "pgx5://" + dbURL

	sep := "?"
	if strings.Contains(dbURL, "?") {
		sep = "&"
	}
	dbURL += sep + "search_path=public&x-migrations-table=security_schema_migrations"

	m, err := migrate.NewWithSourceInstance("iofs", src, dbURL)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
