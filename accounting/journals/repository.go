package journals

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PLBalance es el saldo neto de una cuenta para un período dado (en centavos).
// Balance = SUM(debit) - SUM(credit); negativo indica saldo acreedor (típico en ingresos).
type PLBalance struct {
	AccountID   uuid.UUID
	AccountCode string
	AccountName string
	Category    string
	Balance     int64
}

// Repository define las operaciones de persistencia de asientos.
type Repository interface {
	Create(ctx context.Context, entry JournalEntry) (*JournalEntry, error)
	GetByID(ctx context.Context, id uuid.UUID) (*JournalEntry, error)
	Void(ctx context.Context, id uuid.UUID) error
	ListByCompany(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]*JournalEntry, error)

	// GetYearPLBalances devuelve los saldos de las cuentas de Ingresos, Gastos y Costos
	// para un año completo (1 ene – 31 dic). Solo incluye cuentas con movimiento.
	GetYearPLBalances(ctx context.Context, companyID uuid.UUID, year int) ([]PLBalance, error)

	// GetBSBalances devuelve los saldos acumulados de Activo, Pasivo y Patrimonio
	// hasta la fecha indicada. Solo incluye cuentas con saldo distinto de cero.
	GetBSBalances(ctx context.Context, companyID uuid.UUID, asOf time.Time) ([]PLBalance, error)
}
