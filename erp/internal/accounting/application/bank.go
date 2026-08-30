package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

// ManageBankUseCase agrupa cuentas bancarias, extractos y conciliación manual — un solo caso
// de uso porque las tres operaciones comparten los mismos repos y son pequeñas individualmente
// (mismo criterio que ManagePeriodUseCase).
type ManageBankUseCase struct {
	accounts    domain.BankAccountRepository
	statements  domain.BankStatementRepository
	pucAccounts domain.AccountRepository
}

func NewManageBankUseCase(
	accounts domain.BankAccountRepository,
	statements domain.BankStatementRepository,
	pucAccounts domain.AccountRepository,
) *ManageBankUseCase {
	return &ManageBankUseCase{accounts: accounts, statements: statements, pucAccounts: pucAccounts}
}

type CreateBankAccountRequest struct {
	Name        string
	BankName    string
	AccountNo   string
	AccountCode string // código PUC, ej. "111005"
}

func (uc *ManageBankUseCase) CreateAccount(ctx context.Context, companyID uuid.UUID, req CreateBankAccountRequest) (*domain.BankAccount, error) {
	if req.Name == "" || req.AccountNo == "" {
		return nil, fmt.Errorf("nombre y número de cuenta son obligatorios")
	}
	acct, err := uc.pucAccounts.GetPostable(ctx, req.AccountCode)
	if err != nil {
		return nil, err
	}
	return uc.accounts.Create(ctx, domain.BankAccount{
		CompanyID: companyID, Name: req.Name, BankName: req.BankName, AccountNo: req.AccountNo, AccountID: acct.ID,
	})
}

func (uc *ManageBankUseCase) ListAccounts(ctx context.Context, companyID uuid.UUID) ([]domain.BankAccount, error) {
	return uc.accounts.List(ctx, companyID)
}

type UpdateBankAccountRequest struct {
	Name      string
	BankName  string
	AccountNo string
}

func (uc *ManageBankUseCase) UpdateAccount(ctx context.Context, companyID, id uuid.UUID, req UpdateBankAccountRequest) (*domain.BankAccount, error) {
	if req.Name == "" || req.AccountNo == "" {
		return nil, fmt.Errorf("nombre y número de cuenta son obligatorios")
	}
	return uc.accounts.Update(ctx, companyID, id, req.Name, req.BankName, req.AccountNo)
}

func (uc *ManageBankUseCase) DeactivateAccount(ctx context.Context, companyID, id uuid.UUID) error {
	return uc.accounts.Deactivate(ctx, companyID, id)
}

type AddStatementLineRequest struct {
	Date        time.Time
	Description string
	DebitCents  int64
	CreditCents int64
	Reference   string
}

func (uc *ManageBankUseCase) AddStatementLine(ctx context.Context, companyID, bankAccountID uuid.UUID, req AddStatementLineRequest) (*domain.BankStatementLine, error) {
	if _, err := uc.accounts.GetByID(ctx, companyID, bankAccountID); err != nil {
		return nil, err
	}
	return uc.statements.AddLine(ctx, domain.BankStatementLine{
		BankAccountID: bankAccountID, Date: req.Date, Description: req.Description,
		Debit: req.DebitCents, Credit: req.CreditCents, Reference: req.Reference,
	})
}

func (uc *ManageBankUseCase) ListStatement(ctx context.Context, companyID, bankAccountID uuid.UUID) ([]domain.BankStatementLine, error) {
	if _, err := uc.accounts.GetByID(ctx, companyID, bankAccountID); err != nil {
		return nil, err
	}
	return uc.statements.ListByAccount(ctx, bankAccountID)
}

// Candidates devuelve las líneas de asiento candidatas a conciliar contra el movimiento dado.
func (uc *ManageBankUseCase) Candidates(ctx context.Context, companyID, statementLineID uuid.UUID) ([]domain.ReconciliationCandidate, error) {
	line, err := uc.statements.GetLine(ctx, statementLineID)
	if err != nil {
		return nil, err
	}
	bankAcct, err := uc.accounts.GetByID(ctx, companyID, line.BankAccountID)
	if err != nil {
		return nil, err
	}
	amount := line.Debit
	if line.Credit > 0 {
		amount = line.Credit
	}
	return uc.statements.FindCandidates(ctx, companyID, bankAcct.AccountID, amount, line.Date)
}

func (uc *ManageBankUseCase) Reconcile(ctx context.Context, companyID, statementLineID, journalLineID uuid.UUID) error {
	// Verificar que el movimiento pertenece a una cuenta bancaria de esta empresa (tenant check).
	line, err := uc.statements.GetLine(ctx, statementLineID)
	if err != nil {
		return err
	}
	if _, err := uc.accounts.GetByID(ctx, companyID, line.BankAccountID); err != nil {
		return err
	}
	return uc.statements.Reconcile(ctx, statementLineID, journalLineID)
}
