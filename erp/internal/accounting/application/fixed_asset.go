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
	Code                string
	Name                string
	Description         string
	AssetAccount        string
	DepreciationAccount string
	AccumulatedAccount  string
	// GainAccount/LossAccount son opcionales al crear -- solo hacen falta si el activo se
	// termina dando de baja con utilidad/pérdida (ver DisposeFixedAssetUseCase). No se exigen
	// acá porque muchos activos nunca se venden, y pedirlas siempre sería fricción sin uso real.
	GainAccount          string
	LossAccount          string
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
	if req.GainAccount != "" {
		if _, err := uc.accounts.GetPostable(ctx, req.GainAccount); err != nil {
			return nil, fmt.Errorf("cuenta de utilidad en venta de activos %q: %w", req.GainAccount, err)
		}
	}
	if req.LossAccount != "" {
		if _, err := uc.accounts.GetPostable(ctx, req.LossAccount); err != nil {
			return nil, fmt.Errorf("cuenta de pérdida en venta de activos %q: %w", req.LossAccount, err)
		}
	}
	return uc.assets.Create(ctx, domain.FixedAsset{
		CompanyID: companyID, Code: req.Code, Name: req.Name, Description: req.Description,
		AssetAccount: req.AssetAccount, DepreciationAccount: req.DepreciationAccount, AccumulatedAccount: req.AccumulatedAccount,
		GainAccount: req.GainAccount, LossAccount: req.LossAccount,
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

// DisposeFixedAssetUseCase da de baja o vende un activo fijo: retira el costo y la depreciación
// acumulada del mayor, y contabiliza la diferencia entre lo recibido (0 si es una baja sin venta)
// y el valor en libros como utilidad o pérdida -- GainAccount/LossAccount ya existían en el
// dominio y se persistían, pero ningún caso de uso los usaba.
type DisposeFixedAssetUseCase struct {
	assets   domain.FixedAssetRepository
	accounts domain.AccountRepository
	periods  domain.PeriodRepository
	journals domain.JournalRepository
}

func NewDisposeFixedAssetUseCase(
	assets domain.FixedAssetRepository,
	accounts domain.AccountRepository,
	periods domain.PeriodRepository,
	journals domain.JournalRepository,
) *DisposeFixedAssetUseCase {
	return &DisposeFixedAssetUseCase{assets: assets, accounts: accounts, periods: periods, journals: journals}
}

type DisposeFixedAssetRequest struct {
	DisposalDate time.Time
	// ProceedsCents es lo recibido por el activo -- 0 si es una baja sin venta (activo dañado,
	// obsoleto, etc.), mayor a 0 si se vendió.
	ProceedsCents int64
	// ProceedsAccountCode es la cuenta PUC que recibe el dinero (caja/banco/cuenta por cobrar) --
	// requerida solo cuando ProceedsCents > 0.
	ProceedsAccountCode string
	Description         string
}

func (uc *DisposeFixedAssetUseCase) Execute(ctx context.Context, companyID uuid.UUID, assetID uuid.UUID, req DisposeFixedAssetRequest) (*domain.FixedAsset, error) {
	asset, err := uc.assets.GetByID(ctx, companyID, assetID)
	if err != nil {
		return nil, err
	}
	if asset.Status == domain.AssetDisposed {
		return nil, domain.ErrAssetAlreadyDisposed
	}
	if req.ProceedsCents > 0 && req.ProceedsAccountCode == "" {
		return nil, fmt.Errorf("proceeds_account_code es requerido cuando hay valor de venta")
	}

	period, err := getOrCreatePeriod(ctx, uc.periods, companyID, req.DisposalDate)
	if err != nil {
		return nil, err
	}
	if period.Status == domain.PeriodClosed {
		return nil, domain.ErrPeriodClosed
	}

	accumulated, err := uc.assets.GetAccumulatedDepreciation(ctx, asset.ID)
	if err != nil {
		return nil, err
	}
	netBookValue := asset.AcquisitionCost - accumulated
	gain := req.ProceedsCents - netBookValue
	loss := int64(0)
	if gain < 0 {
		loss = -gain
		gain = 0
	}

	acctAsset, err := uc.accounts.GetPostable(ctx, asset.AssetAccount)
	if err != nil {
		return nil, fmt.Errorf("cuenta del activo %q: %w", asset.AssetAccount, err)
	}

	desc := req.Description
	if desc == "" {
		desc = "Baja de activo " + asset.Name
		if req.ProceedsCents > 0 {
			desc = "Venta de activo " + asset.Name
		}
	}

	var lines []*domain.JournalLine
	if accumulated > 0 {
		acctAccumulated, err := uc.accounts.GetPostable(ctx, asset.AccumulatedAccount)
		if err != nil {
			return nil, fmt.Errorf("cuenta de depreciación acumulada %q: %w", asset.AccumulatedAccount, err)
		}
		lines = append(lines, &domain.JournalLine{AccountID: acctAccumulated.ID, AccountCode: acctAccumulated.Code, Debit: accumulated, Description: desc})
	}
	if req.ProceedsCents > 0 {
		acctProceeds, err := uc.accounts.GetPostable(ctx, req.ProceedsAccountCode)
		if err != nil {
			return nil, fmt.Errorf("cuenta de cobro %q: %w", req.ProceedsAccountCode, err)
		}
		lines = append(lines, &domain.JournalLine{AccountID: acctProceeds.ID, AccountCode: acctProceeds.Code, Debit: req.ProceedsCents, Description: desc})
	}
	if loss > 0 {
		if asset.LossAccount == "" {
			return nil, fmt.Errorf("este activo no tiene configurada una cuenta de pérdida en venta de activos, y esta baja genera una pérdida de %d centavos -- edítalo en la base de datos o vuelve a crearlo con loss_account", loss)
		}
		acctLoss, err := uc.accounts.GetPostable(ctx, asset.LossAccount)
		if err != nil {
			return nil, fmt.Errorf("cuenta de pérdida en venta de activos %q: %w", asset.LossAccount, err)
		}
		lines = append(lines, &domain.JournalLine{AccountID: acctLoss.ID, AccountCode: acctLoss.Code, Debit: loss, Description: desc})
	}
	lines = append(lines, &domain.JournalLine{AccountID: acctAsset.ID, AccountCode: acctAsset.Code, Credit: asset.AcquisitionCost, Description: desc})
	if gain > 0 {
		if asset.GainAccount == "" {
			return nil, fmt.Errorf("este activo no tiene configurada una cuenta de utilidad en venta de activos, y esta baja genera una utilidad de %d centavos -- edítalo en la base de datos o vuelve a crearlo con gain_account", gain)
		}
		acctGain, err := uc.accounts.GetPostable(ctx, asset.GainAccount)
		if err != nil {
			return nil, fmt.Errorf("cuenta de utilidad en venta de activos %q: %w", asset.GainAccount, err)
		}
		lines = append(lines, &domain.JournalLine{AccountID: acctGain.ID, AccountCode: acctGain.Code, Credit: gain, Description: desc})
	}

	sourceType := "BAJA_ACTIVO"
	if req.ProceedsCents > 0 {
		sourceType = "VENTA_ACTIVO"
	}
	if _, err := uc.journals.Create(ctx, domain.JournalEntry{
		CompanyID: companyID, PeriodID: period.ID, Date: req.DisposalDate,
		Description: desc, Status: domain.StatusPosted, Source: "fixed_assets", EntryType: domain.EntryAutomatic,
		SourceDocumentType: sourceType, Book: domain.BookBoth, Lines: lines,
	}); err != nil {
		return nil, err
	}

	if err := uc.assets.UpdateStatus(ctx, companyID, asset.ID, domain.AssetDisposed); err != nil {
		return nil, err
	}
	asset.Status = domain.AssetDisposed
	return asset, nil
}
