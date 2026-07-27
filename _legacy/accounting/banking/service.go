package banking

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service encapsula la lógica de conciliación bancaria.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateBankAccount registra una nueva cuenta bancaria para la empresa.
// accountID es el UUID de la cuenta contable del PUC asociada (ej. 1110 Bancos).
func (s *Service) CreateBankAccount(ctx context.Context, companyID uuid.UUID, name, bankName, accountNo string, accountID uuid.UUID) (*BankAccount, error) {
	return s.repo.CreateBankAccount(ctx, BankAccount{
		CompanyID: companyID,
		Name:      name,
		BankName:  bankName,
		AccountNo: accountNo,
		AccountID: accountID,
	})
}

// GetBankAccount devuelve una cuenta bancaria por UUID.
func (s *Service) GetBankAccount(ctx context.Context, id uuid.UUID) (*BankAccount, error) {
	return s.repo.GetBankAccount(ctx, id)
}

// ListBankAccounts devuelve todas las cuentas bancarias de la empresa.
func (s *Service) ListBankAccounts(ctx context.Context, companyID uuid.UUID) ([]*BankAccount, error) {
	return s.repo.ListBankAccounts(ctx, companyID)
}

// ImportStatementLines importa líneas del extracto bancario en lote.
// Es idempotente siempre que el llamador no duplique las líneas — no hay
// deduplicación automática porque dos débitos del mismo valor y fecha
// pueden ser movimientos distintos.
func (s *Service) ImportStatementLines(ctx context.Context, bankAccountID uuid.UUID, imports []ImportLine) ([]*StatementLine, error) {
	if len(imports) == 0 {
		return nil, nil
	}
	lines := make([]StatementLine, len(imports))
	for i, imp := range imports {
		lines[i] = StatementLine{
			BankAccountID: bankAccountID,
			Date:          imp.Date,
			Description:   imp.Description,
			Debit:         imp.Debit,
			Credit:        imp.Credit,
			Reference:     imp.Reference,
		}
	}
	return s.repo.CreateStatementLines(ctx, lines)
}

// Reconcile marca una línea del extracto como conciliada contra una línea de asiento.
func (s *Service) Reconcile(ctx context.Context, statementLineID, journalLineID uuid.UUID) error {
	return s.repo.Reconcile(ctx, statementLineID, journalLineID)
}

// Unreconcile desmarca una línea del extracto previamente conciliada.
func (s *Service) Unreconcile(ctx context.Context, statementLineID uuid.UUID) error {
	return s.repo.Unreconcile(ctx, statementLineID)
}

// GetReport genera el informe de conciliación bancaria para un período.
// El informe sigue la estructura estándar colombiana:
//
//	Saldo según extracto
//	(−) Partidas en extracto no en libros
//	(+) Depósitos en tránsito
//	= Saldo ajustado ≈ Saldo en libros
func (s *Service) GetReport(ctx context.Context, bankAccountID uuid.UUID, from, to time.Time) (*ReconciliationReport, error) {
	ba, err := s.repo.GetBankAccount(ctx, bankAccountID)
	if err != nil {
		return nil, fmt.Errorf("reporte conciliación: obtener cuenta: %w", err)
	}

	stmtBalance, err := s.repo.StatementBalance(ctx, bankAccountID, to)
	if err != nil {
		return nil, fmt.Errorf("reporte conciliación: saldo extracto: %w", err)
	}

	bookBalance, err := s.repo.BookBalance(ctx, bankAccountID, to)
	if err != nil {
		return nil, fmt.Errorf("reporte conciliación: saldo libros: %w", err)
	}

	unreconciledStmt, err := s.repo.ListUnreconciledStatement(ctx, bankAccountID, from, to)
	if err != nil {
		return nil, fmt.Errorf("reporte conciliación: partidas extracto no conciliadas: %w", err)
	}

	unreconciledBook, err := s.repo.ListUnreconciledBook(ctx, bankAccountID, from, to)
	if err != nil {
		return nil, fmt.Errorf("reporte conciliación: partidas libros no conciliadas: %w", err)
	}

	return &ReconciliationReport{
		BankAccount:           ba,
		PeriodFrom:            from,
		PeriodTo:              to,
		StatementBalance:      stmtBalance,
		UnreconciledStatement: unreconciledStmt,
		BookBalance:           bookBalance,
		UnreconciledBook:      unreconciledBook,
		Difference:            stmtBalance - bookBalance,
	}, nil
}
