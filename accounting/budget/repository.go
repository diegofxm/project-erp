package budget

import (
	"context"

	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia del módulo de presupuesto.
type Repository interface {
	// CreateBudget crea el encabezado de un presupuesto en estado DRAFT.
	// Falla si ya existe (company_id, year, name) duplicado.
	CreateBudget(ctx context.Context, b Budget) (*Budget, error)

	// GetBudget devuelve el encabezado por ID.
	GetBudget(ctx context.Context, id uuid.UUID) (*Budget, error)

	// ListBudgets devuelve los presupuestos de una empresa para un año.
	ListBudgets(ctx context.Context, companyID uuid.UUID, year int) ([]*Budget, error)

	// ApproveBudget cambia el estado de DRAFT a APPROVED.
	// Retorna ErrBudgetNotFound si no existe o ya está en otro estado.
	ApproveBudget(ctx context.Context, id uuid.UUID) error

	// SetLine crea o reemplaza la línea mensual de una cuenta en el presupuesto.
	// accountID debe ser el UUID resuelto de la cuenta; usa ON CONFLICT DO UPDATE.
	SetLine(ctx context.Context, accountID uuid.UUID, req SetLineRequest) (*BudgetLine, error)

	// Lines devuelve todas las líneas de un presupuesto con código y nombre de cuenta.
	Lines(ctx context.Context, budgetID uuid.UUID) ([]*BudgetLine, error)

	// ActualsByMonth devuelve el saldo neto por cuenta (SUM debit − SUM credit) de los
	// asientos POSTED de la empresa en el año y rango de meses indicados.
	ActualsByMonth(ctx context.Context, companyID uuid.UUID, year, fromMonth, toMonth int) ([]ActualRow, error)
}
