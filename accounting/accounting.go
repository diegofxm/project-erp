package accounting

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/diegofxm/accounting/accounts"
	"github.com/diegofxm/accounting/assets"
	"github.com/diegofxm/accounting/banking"
	"github.com/diegofxm/accounting/budget"
	"github.com/diegofxm/accounting/cartera"
	"github.com/diegofxm/accounting/database/seed"
	"github.com/diegofxm/accounting/forex"
	"github.com/diegofxm/accounting/iva"
	"github.com/diegofxm/accounting/journals"
	"github.com/diegofxm/accounting/periods"
	"github.com/diegofxm/accounting/reports"
	"github.com/diegofxm/accounting/tax"
	"github.com/diegofxm/accounting/withholdings"
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
	Accounts     *accounts.Service
	Journals     *journals.Service
	Periods      *periods.Service
	Reports      *reports.Service
	Banking      *banking.Service
	Withholdings *withholdings.Service
	Assets       *assets.Service
	IVA          *iva.Service
	Cartera      *cartera.Service
	Budget       *budget.Service
	Forex        *forex.Service
	Tax          *tax.Service
	pool         *pgxpool.Pool
}

// New construye el Core conectando cada servicio con su repositorio PostgreSQL.
func New(pool *pgxpool.Pool) *Core {
	accountsSvc := accounts.NewService(accounts.NewPostgresRepository(pool))
	periodsSvc := periods.NewService(periods.NewPostgresRepository(pool))
	journalsSvc := journals.NewService(journals.NewPostgresRepository(pool), accountsSvc, periodsSvc)

	return &Core{
		Accounts:     accountsSvc,
		Periods:      periodsSvc,
		Journals:     journalsSvc,
		Reports:      reports.NewService(pool),
		Banking:      banking.NewService(banking.NewPostgresRepository(pool)),
		Withholdings: withholdings.NewService(withholdings.NewPostgresRepository(pool)),
		Assets:       assets.NewService(assets.NewPostgresRepository(pool), journalsSvc, periodsSvc),
		IVA:          iva.NewService(iva.NewPostgresRepository(pool), journalsSvc),
		Cartera:      cartera.NewService(cartera.NewPostgresRepository(pool)),
		Budget:       budget.NewService(budget.NewPostgresRepository(pool), accountsSvc),
		Forex:        forex.NewService(forex.NewPostgresRepository(pool), journalsSvc),
		Tax:          tax.NewService(tax.NewPostgresRepository(pool), journalsSvc),
		pool:         pool,
	}
}

// Migrate ejecuta las migraciones SQL embebidas del módulo contable.
// Usa su propia tabla de seguimiento (accounting_schema_migrations) para no
// interferir con las migraciones de otros módulos que compartan la misma BD.
func Migrate(databaseURL string) error {
	// Agregar x-migrations-table sin romper query params existentes (ej. sslmode=require)
	sep := "?"
	if strings.Contains(databaseURL, "?") {
		sep = "&"
	}
	url := databaseURL + sep + "x-migrations-table=accounting_schema_migrations"

	src, err := iofs.New(migrationsFS, "database/migrations")
	if err != nil {
		return fmt.Errorf("accounting migrate: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, url)
	if err != nil {
		return fmt.Errorf("accounting migrate: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("accounting migrate: %w", err)
	}
	return nil
}

// Seed carga el PUC completo y los catálogos de retenciones desde los CSVs embebidos.
// Es idempotente: se puede llamar en cada arranque sin riesgo de duplicar datos.
func (c *Core) Seed(ctx context.Context) error {
	if err := seed.Accounts(ctx, c.pool); err != nil {
		return err
	}
	if err := seed.WithholdingConcepts(ctx, c.pool); err != nil {
		return err
	}
	if err := seed.UVT(ctx, c.pool); err != nil {
		return err
	}
	return seed.IncomeTaxRates(ctx, c.pool)
}
