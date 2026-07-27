package catalogs

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/diegofxm/catalogs/database/seed"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed database/migrations/*.sql
var migrationsFS embed.FS

// NewPostgresRepository es el punto de entrada que usan los módulos consumidores
// (apidian, edocuments) para obtener un Repository sobre PostgreSQL.
// El alias permite usar catalogs.NewPostgresRepository sin subcarpetas.
// La implementación real está en postgres.go de este mismo paquete.

// Migrate ejecuta las migraciones SQL del schema catalogs.*.
// Usa catalogs_schema_migrations para no interferir con otros módulos.
// La migración usa IF NOT EXISTS — es idempotente si el schema ya existe.
func Migrate(databaseURL string) error {
	sep := "?"
	if strings.Contains(databaseURL, "?") {
		sep = "&"
	}
	url := databaseURL + sep + "x-migrations-table=catalogs_schema_migrations"

	src, err := iofs.New(migrationsFS, "database/migrations")
	if err != nil {
		return fmt.Errorf("catalogs migrate: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, url)
	if err != nil {
		return fmt.Errorf("catalogs migrate: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("catalogs migrate: %w", err)
	}
	return nil
}

// Seed carga los 13 catálogos DIAN/DANE desde los CSVs embebidos.
// Es idempotente y se puede llamar en cada arranque.
func Seed(ctx context.Context, pool *pgxpool.Pool) error {
	return seed.All(ctx, pool)
}
