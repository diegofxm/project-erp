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

// Migrate aplica las migraciones del módulo catalog.
// Usa su propia tabla de versiones para no interferir con otros módulos.
func Migrate(databaseURL string) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return err
	}

	// golang-migrate pgx/v5 usa el scheme "pgx5://"
	dbURL := databaseURL
	dbURL = strings.TrimPrefix(dbURL, "postgresql://")
	dbURL = strings.TrimPrefix(dbURL, "postgres://")
	dbURL = "pgx5://" + dbURL

	sep := "?"
	if strings.Contains(dbURL, "?") {
		sep = "&"
	}
	dbURL += sep + "x-migrations-table=catalog_schema_migrations"

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
