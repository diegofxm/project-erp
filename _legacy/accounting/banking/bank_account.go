package banking

import (
	"time"

	"github.com/google/uuid"
)

// BankAccount es una cuenta bancaria de la empresa registrada en el sistema.
type BankAccount struct {
	ID          uuid.UUID
	CompanyID   uuid.UUID
	Name        string
	BankName    string
	AccountNo   string
	AccountID   uuid.UUID // cuenta contable PUC asociada (ej. 1110 Bancos)
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// StatementLine es una línea del extracto bancario.
type StatementLine struct {
	ID            uuid.UUID
	BankAccountID uuid.UUID
	Date          time.Time
	Description   string
	Debit         int64
	Credit        int64
	Reference     string
	IsReconciled  bool
	JournalLineID *uuid.UUID // nil si aún no está conciliada
	CreatedAt     time.Time
}

// ReconciliationReport es el resultado de la conciliación bancaria para un período.
type ReconciliationReport struct {
	BankAccount       *BankAccount
	PeriodFrom        time.Time
	PeriodTo          time.Time

	// Saldo según extracto bancario al final del período.
	StatementBalance int64
	// Partidas en el extracto que aún no están en los libros.
	UnreconciledStatement []*StatementLine
	// Saldo en libros (libro mayor de la cuenta bancaria).
	BookBalance int64
	// Partidas en los libros que aún no figuran en el extracto.
	UnreconciledBook []*UnreconciledBookItem
	// Diferencia; debe ser cero si la conciliación cuadra.
	Difference int64
}

// UnreconciledBookItem representa un movimiento en el libro mayor que aún
// no aparece en el extracto bancario.
type UnreconciledBookItem struct {
	JournalLineID uuid.UUID
	JournalID     uuid.UUID
	Date          time.Time
	Description   string
	Debit         int64
	Credit        int64
}

// ImportLine contiene los datos de una línea del extracto para importar.
type ImportLine struct {
	Date        time.Time
	Description string
	Debit       int64
	Credit      int64
	Reference   string
}
