package payroll

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/diegofxm/payroll/contracts"
	"github.com/diegofxm/payroll/database/seed"
	"github.com/diegofxm/payroll/employees"
	"github.com/diegofxm/payroll/payslips"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed database/migrations/*.sql
var migrationsFS embed.FS

// Core agrupa todos los servicios del módulo de nómina.
// Se construye una sola vez al arrancar y se pasa a las capas HTTP que lo necesiten.
type Core struct {
	Employees *employees.Service
	Contracts *contracts.Service
	Payslips  *payslips.Service
	pool      *pgxpool.Pool
}

// New conecta cada servicio con su repositorio PostgreSQL.
func New(pool *pgxpool.Pool) *Core {
	employeesSvc := employees.NewService(employees.NewPostgresRepository(pool))
	contractsSvc := contracts.NewService(contracts.NewPostgresRepository(pool))
	payslipsSvc := payslips.NewService(payslips.NewPostgresRepository(pool), contractsSvc)

	return &Core{
		Employees: employeesSvc,
		Contracts: contractsSvc,
		Payslips:  payslipsSvc,
		pool:      pool,
	}
}

// Migrate ejecuta las migraciones SQL embebidas.
// Usa payroll_schema_migrations para no interferir con otros módulos en la misma BD.
func Migrate(databaseURL string) error {
	sep := "?"
	if strings.Contains(databaseURL, "?") {
		sep = "&"
	}
	url := databaseURL + sep + "x-migrations-table=payroll_schema_migrations"

	src, err := iofs.New(migrationsFS, "database/migrations")
	if err != nil {
		return fmt.Errorf("payroll migrate: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, url)
	if err != nil {
		return fmt.Errorf("payroll migrate: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("payroll migrate: %w", err)
	}
	return nil
}

// Seed carga los catálogos de ley: SMMLV histórico y tasas ARL por clase de riesgo.
// Es idempotente y se puede llamar en cada arranque.
func (c *Core) Seed(ctx context.Context) error {
	if err := seed.SMMLV(ctx, c.pool); err != nil {
		return err
	}
	return seed.ARLRates(ctx, c.pool)
}
