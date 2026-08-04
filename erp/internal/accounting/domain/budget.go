package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type BudgetStatus string

const (
	BudgetDraft    BudgetStatus = "DRAFT"
	BudgetApproved BudgetStatus = "APPROVED"
	BudgetClosed   BudgetStatus = "CLOSED"
)

type Budget struct {
	ID        uuid.UUID
	CompanyID uuid.UUID
	Year      int
	Name      string
	Status    BudgetStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BudgetLine es el presupuesto mensual de una cuenta dentro de un presupuesto anual —
// centavos, mismos 12 meses que la tabla (jan..dec).
type BudgetLine struct {
	ID          uuid.UUID
	BudgetID    uuid.UUID
	AccountID   uuid.UUID
	AccountCode string
	AccountName string
	Months      [12]int64
}

func (l BudgetLine) Total() int64 {
	var t int64
	for _, m := range l.Months {
		t += m
	}
	return t
}

// BudgetActualRow compara lo presupuestado contra lo realmente movido en el mayor, mes a mes,
// para una cuenta dentro de un presupuesto.
type BudgetActualRow struct {
	AccountCode    string
	AccountName    string
	BudgetedMonths [12]int64
	ActualMonths   [12]int64
}

var (
	ErrBudgetNotFound = errors.New("presupuesto no encontrado")
)

type BudgetRepository interface {
	Create(ctx context.Context, b Budget) (*Budget, error)
	List(ctx context.Context, companyID uuid.UUID, year int) ([]Budget, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*Budget, error)
	// UpsertLine crea o reemplaza los 12 meses de una cuenta dentro del presupuesto.
	UpsertLine(ctx context.Context, line BudgetLine) (*BudgetLine, error)
	ListLines(ctx context.Context, budgetID uuid.UUID) ([]BudgetLine, error)
	UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status BudgetStatus) error
	// GetActualMonths devuelve el movimiento real (débito-crédito) de la cuenta por cada mes
	// del año dado — base para el comparativo presupuesto vs. real.
	GetActualMonths(ctx context.Context, companyID, accountID uuid.UUID, year int) ([12]int64, error)
}
