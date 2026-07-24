package accounting

import (
	"context"
	"embed"
	"fmt"

	"github.com/diegofxm/accounting/accounts"
	"github.com/diegofxm/accounting/database/seed"
	"github.com/diegofxm/accounting/journals"
	"github.com/diegofxm/accounting/periods"
	"github.com/diegofxm/accounting/reports"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed database/migrations/*.sql
var migrationsFS embed.FS

// Core agrupa todos los servicios del motor contable. Se crea una sola vez al arrancar
// la aplicación y se pasa a las capas que necesiten registrar o consultar asientos.
type Core struct {
	Accounts *accounts.Service
	Journals *journals.Service
	Periods  *periods.Service
	Reports  *reports.Service
	pool     *pgxpool.Pool
}

// New construye el Core conectando cada servicio con su repositorio PostgreSQL.
func New(pool *pgxpool.Pool) *Core {
	accountsSvc := accounts.NewService(accounts.NewPostgresRepository(pool))
	periodsSvc := periods.NewService(periods.NewPostgresRepository(pool))
	journalRepo := journals.NewPostgresRepository(pool)

	return &Core{
		Accounts: accountsSvc,
		Periods:  periodsSvc,
		Journals: journals.NewService(journalRepo, accountsSvc, periodsSvc),
		Reports:  reports.NewService(pool),
		pool:     pool,
	}
}

// Migrate ejecuta las migraciones SQL embebidas del módulo contable.
// Llamar una vez al arrancar, antes de New.
func Migrate(databaseURL string) error {
	src, err := iofs.New(migrationsFS, "database/migrations")
	if err != nil {
		return fmt.Errorf("accounting migrate: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, databaseURL)
	if err != nil {
		return fmt.Errorf("accounting migrate: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("accounting migrate: %w", err)
	}
	return nil
}

// Seed carga el PUC completo desde el CSV embebido. Es idempotente: se puede llamar
// en cada arranque sin riesgo de duplicar datos.
func (c *Core) Seed(ctx context.Context) error {
	return seed.Accounts(ctx, c.pool)
}
