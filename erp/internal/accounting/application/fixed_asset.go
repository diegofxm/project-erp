package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

type FixedAssetUseCase struct {
	assets   domain.FixedAssetRepository
	accounts domain.AccountRepository
}

func NewFixedAssetUseCase(assets domain.FixedAssetRepository, accounts domain.AccountRepository) *FixedAssetUseCase {
	return &FixedAssetUseCase{assets: assets, accounts: accounts}
}

type CreateFixedAssetRequest struct {
	Code                 string
	Name                 string
	Description          string
	AssetAccount         string
	DepreciationAccount  string
	AccumulatedAccount   string
	AcquisitionDate      time.Time
	AcquisitionCostCents int64
	SalvageValueCents    int64
	UsefulLifeMonths     int
	ThirdPartyNIT        string
}

func (uc *FixedAssetUseCase) Create(ctx context.Context, companyID uuid.UUID, req CreateFixedAssetRequest) (*domain.FixedAsset, error) {
	if req.Code == "" || req.Name == "" {
		return nil, fmt.Errorf("código y nombre son obligatorios")
	}
	if req.AcquisitionCostCents <= 0 {
		return nil, fmt.Errorf("el costo de adquisición debe ser mayor a cero")
	}
	if req.UsefulLifeMonths <= 0 {
		return nil, fmt.Errorf("la vida útil debe ser mayor a cero meses")
	}
	// Validar las tres cuentas ahora — si se dejan sin validar hasta la corrida de
	// depreciación, un código mal escrito rompe silenciosamente meses después de creado el activo.
	if _, err := uc.accounts.GetPostable(ctx, req.AssetAccount); err != nil {
		return nil, fmt.Errorf("cuenta del activo %q: %w", req.AssetAccount, err)
	}
	if _, err := uc.accounts.GetPostable(ctx, req.DepreciationAccount); err != nil {
		return nil, fmt.Errorf("cuenta de gasto %q: %w", req.DepreciationAccount, err)
	}
	if _, err := uc.accounts.GetPostable(ctx, req.AccumulatedAccount); err != nil {
		return nil, fmt.Errorf("cuenta de depreciación acumulada %q: %w", req.AccumulatedAccount, err)
	}
	return uc.assets.Create(ctx, domain.FixedAsset{
		CompanyID: companyID, Code: req.Code, Name: req.Name, Description: req.Description,
		AssetAccount: req.AssetAccount, DepreciationAccount: req.DepreciationAccount, AccumulatedAccount: req.AccumulatedAccount,
		AcquisitionDate: req.AcquisitionDate, AcquisitionCost: req.AcquisitionCostCents, SalvageValue: req.SalvageValueCents,
		UsefulLifeMonths: req.UsefulLifeMonths, ThirdPartyNIT: req.ThirdPartyNIT,
	})
}

func (uc *FixedAssetUseCase) List(ctx context.Context, companyID uuid.UUID) ([]domain.FixedAsset, error) {
	return uc.assets.List(ctx, companyID)
}

// assetWithAccumulated es un FixedAsset con su depreciación acumulada resuelta — evita otra
// consulta desde el handler para calcular saldo pendiente en la UI.
type AssetWithAccumulated struct {
	domain.FixedAsset
	Accumulated int64
}

func (uc *FixedAssetUseCase) ListWithAccumulated(ctx context.Context, companyID uuid.UUID) ([]AssetWithAccumulated, error) {
	assets, err := uc.assets.List(ctx, companyID)
	if err != nil {
		return nil, err
	}
	out := make([]AssetWithAccumulated, len(assets))
	for i, a := range assets {
		acc, err := uc.assets.GetAccumulatedDepreciation(ctx, a.ID)
		if err != nil {
			return nil, err
		}
		out[i] = AssetWithAccumulated{FixedAsset: a, Accumulated: acc}
	}
	return out, nil
}

// RunDepreciationUseCase corre la depreciación mensual: una cuota por activo activo con saldo
// pendiente, consolidadas en un único asiento (una línea de gasto + una de acumulada por activo).
type RunDepreciationUseCase struct {
	assets        domain.FixedAssetRepository
	depreciations domain.DepreciationRepository
	accounts      domain.AccountRepository
	periods       domain.PeriodRepository
	journals      domain.JournalRepository
}

func NewRunDepreciationUseCase(
	assets domain.FixedAssetRepository,
	depreciations domain.DepreciationRepository,
	accounts domain.AccountRepository,
	periods domain.PeriodRepository,
	journals domain.JournalRepository,
) *RunDepreciationUseCase {
	return &RunDepreciationUseCase{assets: assets, depreciations: depreciations, accounts: accounts, periods: periods, journals: journals}
}

func (uc *RunDepreciationUseCase) Execute(ctx context.Context, companyID uuid.UUID, runDate time.Time) (*domain.DepreciationRun, error) {
	period, err := getOrCreatePeriod(ctx, uc.periods, companyID, runDate)
	if err != nil {
		return nil, err
	}
	if period.Status == domain.PeriodClosed {
		return nil, domain.ErrPeriodClosed
	}

	already, err := uc.depreciations.HasRunForPeriod(ctx, companyID, period.ID)
	if err != nil {
		return nil, err
	}
	if already {
		return nil, domain.ErrDepreciationExists
	}

	assets, err := uc.assets.List(ctx, companyID)
	if err != nil {
		return nil, err
	}

	var entries []domain.DepreciationEntry
	var lines []*domain.JournalLine

	for _, a := range assets {
		if a.Status != domain.AssetActive {
			continue
		}
		accumulated, err := uc.assets.GetAccumulatedDepreciation(ctx, a.ID)
		if err != nil {
			return nil, err
		}
		depreciable := a.AcquisitionCost - a.SalvageValue
		remaining := depreciable - accumulated
		if remaining <= 0 {
			continue
		}
		amount := a.MonthlyDepreciation()
		if amount > remaining {
			amount = remaining
		}
		if amount <= 0 {
			continue
		}

		acctExpense, err := uc.accounts.GetPostable(ctx, a.DepreciationAccount)
		if err != nil {
			return nil, fmt.Errorf("activo %s: %w", a.Code, err)
		}
		acctAccumulated, err := uc.accounts.GetPostable(ctx, a.AccumulatedAccount)
		if err != nil {
			return nil, fmt.Errorf("activo %s: %w", a.Code, err)
		}

		lines = append(lines,
			&domain.JournalLine{AccountID: acctExpense.ID, AccountCode: acctExpense.Code, Debit: amount, Description: "Depreciación " + a.Name},
			&domain.JournalLine{AccountID: acctAccumulated.ID, AccountCode: acctAccumulated.Code, Credit: amount, Description: "Depreciación acumulada " + a.Name},
		)
		entries = append(entries, domain.DepreciationEntry{AssetID: a.ID, Amount: amount})

		if amount == remaining {
			_ = uc.assets.UpdateStatus(ctx, companyID, a.ID, domain.AssetFullyDepreciated)
		}
	}

	if len(entries) == 0 {
		return nil, domain.ErrNoDepreciableAssets
	}

	entry, err := uc.journals.Create(ctx, domain.JournalEntry{
		CompanyID: companyID, PeriodID: period.ID, Date: runDate,
		Description: fmt.Sprintf("Depreciación %s", runDate.Format("2006-01")),
		Status:      domain.StatusPosted, Source: "fixed_assets", EntryType: domain.EntryAutomatic,
		SourceDocumentType: "DEPRECIACION", Book: domain.BookBoth, Lines: lines,
	})
	if err != nil {
		return nil, err
	}

	run, err := uc.depreciations.CreateRun(ctx, domain.DepreciationRun{
		CompanyID: companyID, PeriodID: period.ID, RunDate: runDate, Status: domain.RunCompleted, JournalID: &entry.ID,
	}, entries)
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (uc *RunDepreciationUseCase) ListRuns(ctx context.Context, companyID uuid.UUID) ([]domain.DepreciationRun, error) {
	return uc.depreciations.ListRuns(ctx, companyID)
}
