package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// BankAccount es una cuenta bancaria de la empresa, ligada a su cuenta PUC de bancos
// (normalmente 111005) para poder conciliar sus movimientos contra el libro mayor.
type BankAccount struct {
	ID        uuid.UUID
	CompanyID uuid.UUID
	Name      string
	BankName  string
	AccountNo string
	AccountID uuid.UUID // cuenta PUC (accounting.accounts.id)
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BankStatementLine es un movimiento del extracto bancario — se concilia contra una línea de
// asiento existente (JournalLineID) cuando coincide con un movimiento ya contabilizado.
type BankStatementLine struct {
	ID            uuid.UUID
	BankAccountID uuid.UUID
	Date          time.Time
	Description   string
	Debit         int64 // centavos
	Credit        int64 // centavos
	Reference     string
	IsReconciled  bool
	JournalLineID *uuid.UUID
	CreatedAt     time.Time
}

var (
	ErrBankAccountNotFound = errors.New("cuenta bancaria no encontrada")
	ErrAlreadyReconciled   = errors.New("el movimiento ya fue conciliado")
)

type BankAccountRepository interface {
	Create(ctx context.Context, a BankAccount) (*BankAccount, error)
	List(ctx context.Context, companyID uuid.UUID) ([]BankAccount, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*BankAccount, error)
}

type BankStatementRepository interface {
	AddLine(ctx context.Context, l BankStatementLine) (*BankStatementLine, error)
	ListByAccount(ctx context.Context, bankAccountID uuid.UUID) ([]BankStatementLine, error)
	GetLine(ctx context.Context, id uuid.UUID) (*BankStatementLine, error)
	Reconcile(ctx context.Context, id, journalLineID uuid.UUID) error
	// FindCandidates busca líneas de asiento sin conciliar en la cuenta contable del banco con
	// el mismo monto, dentro de una ventana de +/-15 días de la fecha del movimiento —
	// candidatas para conciliar manualmente contra la línea de extracto dada.
	FindCandidates(ctx context.Context, companyID, accountID uuid.UUID, amountCents int64, around time.Time) ([]ReconciliationCandidate, error)
}

// ReconciliationCandidate es una línea de asiento candidata a conciliarse contra un movimiento
// de extracto — LineID es lo que se manda de vuelta a Reconcile().
type ReconciliationCandidate struct {
	LineID        uuid.UUID
	JournalID     uuid.UUID
	Date          time.Time
	Description   string
	VoucherType   string
	VoucherNumber string
	Debit         int64
	Credit        int64
}
