package assets

import (
	"context"
	"fmt"
	"time"

	"github.com/diegofxm/accounting/journals"
	"github.com/diegofxm/accounting/periods"
	"github.com/google/uuid"
)

// Service expone las operaciones de activos fijos y depreciación automática.
type Service struct {
	repo        Repository
	journalsSvc *journals.Service
	periodsSvc  *periods.Service
}

func NewService(repo Repository, journalsSvc *journals.Service, periodsSvc *periods.Service) *Service {
	return &Service{repo: repo, journalsSvc: journalsSvc, periodsSvc: periodsSvc}
}

// Create registra un nuevo activo fijo con validaciones básicas.
func (s *Service) Create(ctx context.Context, asset FixedAsset) (*FixedAsset, error) {
	if asset.AcquisitionCost <= 0 {
		return nil, ErrInvalidAcquisitionCost
	}
	if asset.SalvageValue >= asset.AcquisitionCost {
		return nil, ErrInvalidSalvageValue
	}
	if asset.AssetAccount == "" || asset.DepreciationAccount == "" || asset.AccumulatedAccount == "" {
		return nil, ErrMissingAccounts
	}
	if asset.DepreciationMethod == "" {
		asset.DepreciationMethod = MethodStraightLine
	}
	if asset.Status == "" {
		asset.Status = StatusActive
	}
	if asset.GainAccount == "" {
		asset.GainAccount = "424505"
	}
	if asset.LossAccount == "" {
		asset.LossAccount = "529005"
	}
	return s.repo.Create(ctx, asset)
}

// GetByID devuelve un activo por su UUID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*FixedAsset, error) {
	return s.repo.GetByID(ctx, id)
}

// List devuelve todos los activos de una empresa. activeOnly filtra solo los ACTIVE.
func (s *Service) List(ctx context.Context, companyID uuid.UUID, activeOnly bool) ([]*FixedAsset, error) {
	return s.repo.ListByCompany(ctx, companyID, activeOnly)
}

// GetSchedule proyecta el plan de depreciación completo de un activo (línea recta).
// Devuelve una fila por cada mes de vida útil con el monto, acumulado y valor en libros.
func (s *Service) GetSchedule(ctx context.Context, assetID uuid.UUID) ([]*ScheduleRow, error) {
	asset, err := s.repo.GetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}

	depreciable := asset.AcquisitionCost - asset.SalvageValue
	monthly := depreciable / int64(asset.UsefulLifeMonths)
	// El último mes absorbe el centavo residual de la división entera.
	lastMonthExtra := depreciable - monthly*int64(asset.UsefulLifeMonths)

	rows := make([]*ScheduleRow, asset.UsefulLifeMonths)
	var accumulated int64
	for i := 0; i < asset.UsefulLifeMonths; i++ {
		amount := monthly
		if i == asset.UsefulLifeMonths-1 {
			amount += lastMonthExtra
		}
		accumulated += amount
		date := asset.AcquisitionDate.AddDate(0, i+1, 0)
		rows[i] = &ScheduleRow{
			MonthNumber: i + 1,
			Date:        date,
			Amount:      amount,
			Accumulated: accumulated,
			BookValue:   asset.AcquisitionCost - accumulated,
		}
	}
	return rows, nil
}

// RunDepreciation genera el asiento mensual de depreciación para todos los activos
// activos de la empresa. Operación idempotente por periodo: falla si el periodo
// ya tiene una corrida completada.
func (s *Service) RunDepreciation(ctx context.Context, companyID uuid.UUID, date time.Time) (*DepreciationRun, error) {
	// Resolver o crear el periodo contable para la fecha.
	period, err := s.periodsSvc.GetOrCreate(ctx, companyID, date)
	if err != nil {
		return nil, fmt.Errorf("run depreciation: obtener periodo: %w", err)
	}

	// Verificar que no existe ya una corrida para este periodo.
	existing, err := s.repo.GetRunForPeriod(ctx, companyID, period.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrAlreadyDepreciated
	}

	// Obtener activos activos.
	activeAssets, err := s.repo.ListByCompany(ctx, companyID, true)
	if err != nil {
		return nil, fmt.Errorf("run depreciation: listar activos: %w", err)
	}

	type assetDepreciation struct {
		asset  *FixedAsset
		amount int64
	}
	var toDepreciate []assetDepreciation

	for _, a := range activeAssets {
		accumulated, err := s.repo.GetAccumulatedDepreciation(ctx, a.ID)
		if err != nil {
			return nil, fmt.Errorf("run depreciation: acumulado activo %s: %w", a.Code, err)
		}

		depreciable := a.AcquisitionCost - a.SalvageValue
		remaining := depreciable - accumulated

		if remaining <= 0 {
			// Ya está totalmente depreciado — actualizar estado.
			_ = s.repo.UpdateStatus(ctx, a.ID, StatusFullyDepreciated)
			continue
		}

		monthly := depreciable / int64(a.UsefulLifeMonths)
		if monthly <= 0 {
			monthly = 1
		}
		amount := monthly
		if amount > remaining {
			amount = remaining
		}
		toDepreciate = append(toDepreciate, assetDepreciation{asset: a, amount: amount})
	}

	if len(toDepreciate) == 0 {
		return nil, ErrNoAssetsToDepreciate
	}

	// Construir las líneas del asiento de depreciación.
	lines := make([]journals.LineRequest, 0, len(toDepreciate)*2)
	entries := make([]DepreciationEntry, 0, len(toDepreciate))

	for _, d := range toDepreciate {
		desc := fmt.Sprintf("Depreciación %s — %s", d.asset.Code, date.Format("2006-01"))
		lines = append(lines,
			journals.LineRequest{
				AccountCode: d.asset.DepreciationAccount,
				Debit:       d.amount,
				Description: desc,
			},
			journals.LineRequest{
				AccountCode: d.asset.AccumulatedAccount,
				Credit:      d.amount,
				Description: desc,
			},
		)
		entries = append(entries, DepreciationEntry{
			AssetID: d.asset.ID,
			Amount:  d.amount,
		})
	}

	entry, err := s.journalsSvc.Post(ctx, journals.PostRequest{
		CompanyID:   companyID,
		Date:        date,
		Description: fmt.Sprintf("Depreciación mensual %s", date.Format("2006-01")),
		Source:      "depreciation",
		EntryType:   journals.EntryAutomatic,
		VoucherType: "DA",
		Lines:       lines,
	})
	if err != nil {
		return nil, fmt.Errorf("run depreciation: registrar asiento: %w", err)
	}

	run := DepreciationRun{
		CompanyID: companyID,
		PeriodID:  period.ID,
		RunDate:   date,
		Status:    "COMPLETED",
		JournalID: entry.ID,
	}

	savedRun, err := s.repo.CreateRun(ctx, run, entries)
	if err != nil {
		return nil, fmt.Errorf("run depreciation: guardar corrida: %w", err)
	}

	// Marcar como totalmente depreciados los activos que cerraron su vida útil.
	for _, d := range toDepreciate {
		accumulated, _ := s.repo.GetAccumulatedDepreciation(ctx, d.asset.ID)
		depreciable := d.asset.AcquisitionCost - d.asset.SalvageValue
		if accumulated >= depreciable {
			_ = s.repo.UpdateStatus(ctx, d.asset.ID, StatusFullyDepreciated)
		}
	}

	return savedRun, nil
}

// Dispose da de baja un activo y genera el asiento contable de retiro.
// Si Proceeds > 0 se registra la venta con ganancia o pérdida.
// Si Proceeds = 0 se registra la pérdida total del valor en libros.
func (s *Service) Dispose(ctx context.Context, req DisposeRequest) (*journals.JournalEntry, error) {
	asset, err := s.repo.GetByID(ctx, req.AssetID)
	if err != nil {
		return nil, err
	}
	if asset.Status == StatusDisposed {
		return nil, ErrAssetAlreadyDisposed
	}
	if req.Proceeds > 0 && req.ProceedsAccount == "" {
		return nil, ErrProceedsAccountRequired
	}

	accumulated, err := s.repo.GetAccumulatedDepreciation(ctx, req.AssetID)
	if err != nil {
		return nil, fmt.Errorf("dispose: obtener acumulado: %w", err)
	}

	bookValue := asset.AcquisitionCost - accumulated
	gainLoss := req.Proceeds - bookValue // positivo = ganancia, negativo = pérdida

	var lines []journals.LineRequest

	// Reversa de la depreciación acumulada.
	if accumulated > 0 {
		lines = append(lines, journals.LineRequest{
			AccountCode: asset.AccumulatedAccount,
			Debit:       accumulated,
			Description: "Baja de depreciación acumulada — " + asset.Name,
		})
	}

	// Ingresos por venta (si hay).
	if req.Proceeds > 0 {
		lines = append(lines, journals.LineRequest{
			AccountCode:   req.ProceedsAccount,
			Debit:         req.Proceeds,
			ThirdPartyNIT: asset.ThirdPartyNIT,
			Description:   "Ingresos por venta de " + asset.Name,
		})
	}

	// Ganancia o pérdida.
	switch {
	case gainLoss > 0:
		lines = append(lines, journals.LineRequest{
			AccountCode: asset.GainAccount,
			Credit:      gainLoss,
			Description: fmt.Sprintf("Utilidad en venta de %s", asset.Name),
		})
	case gainLoss < 0:
		lines = append(lines, journals.LineRequest{
			AccountCode: asset.LossAccount,
			Debit:       -gainLoss,
			Description: fmt.Sprintf("Pérdida en baja de %s", asset.Name),
		})
	}

	// Retiro del activo al costo.
	lines = append(lines, journals.LineRequest{
		AccountCode: asset.AssetAccount,
		Credit:      asset.AcquisitionCost,
		Description: "Baja del activo — " + asset.Name,
	})

	entry, err := s.journalsSvc.Post(ctx, journals.PostRequest{
		CompanyID:   req.CompanyID,
		Date:        req.Date,
		Description: fmt.Sprintf("Baja de activo fijo: %s — %s", asset.Code, asset.Name),
		Source:      "asset_disposal",
		EntryType:   journals.EntryAutomatic,
		VoucherType: req.VoucherType,
		Lines:       lines,
	})
	if err != nil {
		return nil, fmt.Errorf("dispose: registrar asiento: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, req.AssetID, StatusDisposed); err != nil {
		return nil, fmt.Errorf("dispose: actualizar estado: %w", err)
	}

	return entry, nil
}
