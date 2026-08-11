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
	// ErrBudgetNotDraft: renombrar/borrar un presupuesto o quitarle una línea solo tiene sentido
	// mientras sigue en DRAFT -- uno ya APPROVED/CLOSED pudo haberse usado en comparativos ya
	// revisados por alguien más.
	ErrBudgetNotDraft = errors.New("el presupuesto debe estar en borrador para esta operación")
)

type BudgetRepository interface {
	Create(ctx context.Context, b Budget) (*Budget, error)
	List(ctx context.Context, companyID uuid.UUID, year int) ([]Budget, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*Budget, error)
	// Rename corrige el nombre de un presupuesto EN BORRADOR -- ErrBudgetNotDraft si no.
	Rename(ctx context.Context, companyID, id uuid.UUID, name string) (*Budget, error)
	// Delete elimina un presupuesto EN BORRADOR (con sus líneas) -- ErrBudgetNotDraft si no.
	Delete(ctx context.Context, companyID, id uuid.UUID) error
	// UpsertLine crea o reemplaza los 12 meses de una cuenta dentro del presupuesto.
	UpsertLine(ctx context.Context, line BudgetLine) (*BudgetLine, error)
	// DeleteLine quita una cuenta del presupuesto (en vez del workaround de poner los 12 meses
	// en cero) -- valida DRAFT del lado de la aplicación (BudgetUseCase.DeleteLine), no acá.
	DeleteLine(ctx context.Context, budgetID, accountID uuid.UUID) error
	ListLines(ctx context.Context, budgetID uuid.UUID) ([]BudgetLine, error)
	UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status BudgetStatus) error
	// GetActualMonths devuelve el movimiento real (débito-crédito) de la cuenta por cada mes
	// del año dado — base para el comparativo presupuesto vs. real.
	GetActualMonths(ctx context.Context, companyID, accountID uuid.UUID, year int) ([12]int64, error)
}
