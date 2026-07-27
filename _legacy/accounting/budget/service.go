package budget

import (
	"context"
	"fmt"

	"github.com/diegofxm/accounting/accounts"
	"github.com/google/uuid"
)

// Service gestiona presupuestos anuales y el reporte Presupuesto vs. Real (BvR).
type Service struct {
	repo        Repository
	accountsSvc *accounts.Service
}

func NewService(repo Repository, accountsSvc *accounts.Service) *Service {
	return &Service{repo: repo, accountsSvc: accountsSvc}
}

// Create crea el encabezado de un presupuesto en estado DRAFT.
func (s *Service) Create(ctx context.Context, companyID uuid.UUID, year int, name string) (*Budget, error) {
	return s.repo.CreateBudget(ctx, Budget{
		CompanyID: companyID,
		Year:      year,
		Name:      name,
		Status:    StatusDraft,
	})
}

// Get devuelve el encabezado de un presupuesto por ID.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Budget, error) {
	return s.repo.GetBudget(ctx, id)
}

// List devuelve todos los presupuestos de una empresa para un año dado.
func (s *Service) List(ctx context.Context, companyID uuid.UUID, year int) ([]*Budget, error) {
	return s.repo.ListBudgets(ctx, companyID, year)
}

// Approve cambia el estado de DRAFT a APPROVED. Un presupuesto aprobado no puede modificarse.
func (s *Service) Approve(ctx context.Context, id uuid.UUID) error {
	return s.repo.ApproveBudget(ctx, id)
}

// SetLine crea o reemplaza la línea mensual de una cuenta en el presupuesto.
// La cuenta se resuelve por código (debe ser posteable).
// Falla si el presupuesto ya está APPROVED.
func (s *Service) SetLine(ctx context.Context, req SetLineRequest) (*BudgetLine, error) {
	b, err := s.repo.GetBudget(ctx, req.BudgetID)
	if err != nil {
		return nil, err
	}
	if b.Status == StatusApproved {
		return nil, ErrBudgetApproved
	}

	acct, err := s.accountsSvc.GetPostable(ctx, req.AccountCode)
	if err != nil {
		return nil, fmt.Errorf("cuenta %q: %w", req.AccountCode, err)
	}

	return s.repo.SetLine(ctx, acct.ID, req)
}

// Lines devuelve todas las líneas de un presupuesto con código y nombre de cuenta.
func (s *Service) Lines(ctx context.Context, budgetID uuid.UUID) ([]*BudgetLine, error) {
	return s.repo.Lines(ctx, budgetID)
}

// BvR genera el reporte Presupuesto vs. Real para un rango de meses (1–12 inclusive).
// Incluye tanto las cuentas con línea presupuestada como las que solo tienen ejecución real.
func (s *Service) BvR(ctx context.Context, companyID, budgetID uuid.UUID, fromMonth, toMonth int) (*BvRReport, error) {
	if fromMonth < 1 || toMonth > 12 || fromMonth > toMonth {
		return nil, ErrInvalidMonths
	}

	b, err := s.repo.GetBudget(ctx, budgetID)
	if err != nil {
		return nil, err
	}
	if b.CompanyID != companyID {
		return nil, ErrBudgetNotFound
	}

	lines, err := s.repo.Lines(ctx, budgetID)
	if err != nil {
		return nil, err
	}

	actuals, err := s.repo.ActualsByMonth(ctx, companyID, b.Year, fromMonth, toMonth)
	if err != nil {
		return nil, err
	}

	// Índice de actuals por account_id para O(1) lookup
	actualsMap := make(map[uuid.UUID]ActualRow, len(actuals))
	for _, a := range actuals {
		actualsMap[a.AccountID] = a
	}

	var bvrLines []BvRLine

	// Líneas con presupuesto (pueden o no tener ejecución real)
	for _, l := range lines {
		actual := actualsMap[l.AccountID]
		delete(actualsMap, l.AccountID)
		budgetAmt := l.RangeAmount(fromMonth, toMonth)
		bvrLines = append(bvrLines, BvRLine{
			AccountID:   l.AccountID,
			AccountCode: l.AccountCode,
			AccountName: l.AccountName,
			Category:    l.Category,
			Budget:      budgetAmt,
			Actual:      actual.Net,
			Variance:    actual.Net - budgetAmt,
		})
	}

	// Cuentas con ejecución real pero sin línea presupuestada
	for _, a := range actualsMap {
		bvrLines = append(bvrLines, BvRLine{
			AccountID:   a.AccountID,
			AccountCode: a.AccountCode,
			AccountName: a.AccountName,
			Category:    a.Category,
			Budget:      0,
			Actual:      a.Net,
			Variance:    a.Net,
		})
	}

	report := &BvRReport{
		BudgetID:   budgetID,
		BudgetName: b.Name,
		Year:       b.Year,
		FromMonth:  fromMonth,
		ToMonth:    toMonth,
		Lines:      bvrLines,
	}
	for _, l := range bvrLines {
		report.TotalBudget += l.Budget
		report.TotalActual += l.Actual
		report.TotalVariance += l.Variance
	}

	return report, nil
}
