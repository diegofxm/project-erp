package budget

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrBudgetNotFound = errors.New("budget: presupuesto no encontrado")
	ErrBudgetApproved = errors.New("budget: el presupuesto ya está aprobado y no se puede modificar")
	ErrInvalidMonths  = errors.New("budget: fromMonth debe ser <= toMonth y ambos entre 1 y 12")
)

type BudgetStatus string

const (
	StatusDraft    BudgetStatus = "DRAFT"
	StatusApproved BudgetStatus = "APPROVED"
	StatusClosed   BudgetStatus = "CLOSED"
)

// Budget es el encabezado del presupuesto anual de una empresa.
type Budget struct {
	ID        uuid.UUID
	CompanyID uuid.UUID
	Year      int
	Name      string
	Status    BudgetStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BudgetLine es la asignación mensual por cuenta dentro de un presupuesto.
// Todos los montos en centavos (int64).
type BudgetLine struct {
	ID          uuid.UUID
	BudgetID    uuid.UUID
	AccountID   uuid.UUID
	AccountCode string
	AccountName string
	Category    string
	Jan, Feb, Mar, Apr, May, Jun int64
	Jul, Aug, Sep, Oct, Nov, Dec int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// MonthAmount devuelve el presupuesto para el mes m (1–12). Retorna 0 si m está fuera de rango.
func (l *BudgetLine) MonthAmount(m int) int64 {
	months := [12]int64{l.Jan, l.Feb, l.Mar, l.Apr, l.May, l.Jun,
		l.Jul, l.Aug, l.Sep, l.Oct, l.Nov, l.Dec}
	if m < 1 || m > 12 {
		return 0
	}
	return months[m-1]
}

// RangeAmount devuelve el total presupuestado para los meses from–to inclusive (1–12).
func (l *BudgetLine) RangeAmount(from, to int) int64 {
	var total int64
	for m := from; m <= to; m++ {
		total += l.MonthAmount(m)
	}
	return total
}

// ActualRow es el saldo neto de una cuenta en el período de actuals del BvR.
type ActualRow struct {
	AccountID   uuid.UUID
	AccountCode string
	AccountName string
	Category    string
	Net         int64 // SUM(debit) - SUM(credit) de los asientos POSTED
}

// BvRLine es una fila del reporte Presupuesto vs. Real.
type BvRLine struct {
	AccountID   uuid.UUID
	AccountCode string
	AccountName string
	Category    string
	Budget      int64 // presupuestado en el rango de meses (centavos)
	Actual      int64 // ejecutado real: SUM(debit)-SUM(credit) (centavos)
	Variance    int64 // Actual - Budget (positivo = ejecución sobre presupuesto)
}

// BvRReport es el reporte completo Presupuesto vs. Real.
type BvRReport struct {
	BudgetID      uuid.UUID
	BudgetName    string
	Year          int
	FromMonth     int
	ToMonth       int
	Lines         []BvRLine
	TotalBudget   int64
	TotalActual   int64
	TotalVariance int64
}

// SetLineRequest se usa para crear o reemplazar una línea mensual en el presupuesto.
type SetLineRequest struct {
	BudgetID    uuid.UUID
	AccountCode string
	Jan, Feb, Mar, Apr, May, Jun int64
	Jul, Aug, Sep, Oct, Nov, Dec int64
}
