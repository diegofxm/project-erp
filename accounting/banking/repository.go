package banking

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia del módulo de conciliación.
type Repository interface {
	// BankAccounts
	CreateBankAccount(ctx context.Context, ba BankAccount) (*BankAccount, error)
	GetBankAccount(ctx context.Context, id uuid.UUID) (*BankAccount, error)
	ListBankAccounts(ctx context.Context, companyID uuid.UUID) ([]*BankAccount, error)

	// StatementLines
	CreateStatementLines(ctx context.Context, lines []StatementLine) ([]*StatementLine, error)
	GetStatementLine(ctx context.Context, id uuid.UUID) (*StatementLine, error)
	ListStatementLines(ctx context.Context, bankAccountID uuid.UUID, from, to time.Time) ([]*StatementLine, error)
	ListUnreconciledStatement(ctx context.Context, bankAccountID uuid.UUID, from, to time.Time) ([]*StatementLine, error)
	Reconcile(ctx context.Context, statementLineID, journalLineID uuid.UUID) error
	Unreconcile(ctx context.Context, statementLineID uuid.UUID) error

	// Book movements for the bank account that are NOT reconciled.
	ListUnreconciledBook(ctx context.Context, bankAccountID uuid.UUID, from, to time.Time) ([]*UnreconciledBookItem, error)

	// Balance queries
	StatementBalance(ctx context.Context, bankAccountID uuid.UUID, asOf time.Time) (float64, error)
	BookBalance(ctx context.Context, bankAccountID uuid.UUID, asOf time.Time) (float64, error)
}
