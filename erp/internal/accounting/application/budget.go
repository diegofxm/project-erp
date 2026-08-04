package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

type BudgetUseCase struct {
	budgets  domain.BudgetRepository
	accounts domain.AccountRepository
}

func NewBudgetUseCase(budgets domain.BudgetRepository, accounts domain.AccountRepository) *BudgetUseCase {
	return &BudgetUseCase{budgets: budgets, accounts: accounts}
}

func (uc *BudgetUseCase) Create(ctx context.Context, companyID uuid.UUID, year int, name string) (*domain.Budget, error) {
	if name == "" {
		return nil, fmt.Errorf("el nombre del presupuesto es obligatorio")
	}
	if year < 2000 {
		return nil, fmt.Errorf("año inválido")
	}
	return uc.budgets.Create(ctx, domain.Budget{CompanyID: companyID, Year: year, Name: name})
}

func (uc *BudgetUseCase) List(ctx context.Context, companyID uuid.UUID, year int) ([]domain.Budget, error) {
	return uc.budgets.List(ctx, companyID, year)
}

type SetBudgetLineRequest struct {
	BudgetID    uuid.UUID
	AccountCode string
	Months      [12]int64
}

func (uc *BudgetUseCase) SetLine(ctx context.Context, companyID uuid.UUID, req SetBudgetLineRequest) (*domain.BudgetLine, error) {
	budget, err := uc.budgets.GetByID(ctx, companyID, req.BudgetID)
	if err != nil {
		return nil, err
	}
	if budget.Status == domain.BudgetClosed {
		return nil, fmt.Errorf("el presupuesto está cerrado")
	}
	acct, err := uc.accounts.GetPostable(ctx, req.AccountCode)
	if err != nil {
		return nil, err
	}
	line, err := uc.budgets.UpsertLine(ctx, domain.BudgetLine{BudgetID: req.BudgetID, AccountID: acct.ID, Months: req.Months})
	if err != nil {
		return nil, err
	}
	line.AccountCode = acct.Code
	line.AccountName = acct.Name
	return line, nil
}

func (uc *BudgetUseCase) SetStatus(ctx context.Context, companyID, budgetID uuid.UUID, status domain.BudgetStatus) error {
	if _, err := uc.budgets.GetByID(ctx, companyID, budgetID); err != nil {
		return err
	}
	return uc.budgets.UpdateStatus(ctx, companyID, budgetID, status)
}

// Compare devuelve, para cada línea del presupuesto, lo presupuestado vs. lo realmente
// movido en el mayor mes a mes.
func (uc *BudgetUseCase) Compare(ctx context.Context, companyID, budgetID uuid.UUID) ([]domain.BudgetActualRow, error) {
	budget, err := uc.budgets.GetByID(ctx, companyID, budgetID)
	if err != nil {
		return nil, err
	}
	lines, err := uc.budgets.ListLines(ctx, budgetID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.BudgetActualRow, len(lines))
	for i, l := range lines {
		actual, err := uc.budgets.GetActualMonths(ctx, companyID, l.AccountID, budget.Year)
		if err != nil {
			return nil, err
		}
		out[i] = domain.BudgetActualRow{AccountCode: l.AccountCode, AccountName: l.AccountName, BudgetedMonths: l.Months, ActualMonths: actual}
	}
	return out, nil
}

func (uc *BudgetUseCase) ListLines(ctx context.Context, companyID, budgetID uuid.UUID) ([]domain.BudgetLine, error) {
	if _, err := uc.budgets.GetByID(ctx, companyID, budgetID); err != nil {
		return nil, err
	}
	return uc.budgets.ListLines(ctx, budgetID)
}
