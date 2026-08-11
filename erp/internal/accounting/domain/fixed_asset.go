package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type AssetStatus string

const (
	AssetActive           AssetStatus = "ACTIVE"
	AssetFullyDepreciated AssetStatus = "FULLY_DEPRECIATED"
	AssetDisposed         AssetStatus = "DISPOSED"
)

// FixedAsset es un activo fijo depreciable. Las cuentas PUC (asset/depreciation/accumulated)
// se capturan por activo, no por categoría fija, porque distintas categorías de activo usan
// subcuentas distintas del mismo grupo (152405 Maquinaria vs 152410 Equipo de oficina, etc).
// Los montos van en centavos (int64), igual que journal_lines.
type FixedAsset struct {
	ID                  uuid.UUID
	CompanyID           uuid.UUID
	Code                string
	Name                string
	Description         string
	AssetAccount        string // cuenta PUC del activo (ej. 152405)
	DepreciationAccount string // cuenta PUC del gasto (ej. 516010)
	AccumulatedAccount  string // cuenta PUC de depreciación acumulada (ej. 159210)
	GainAccount         string
	LossAccount         string
	AcquisitionDate     time.Time
	AcquisitionCost     int64 // centavos
	SalvageValue        int64 // centavos, valor residual
	UsefulLifeMonths    int
	DepreciationMethod  string // "STRAIGHT_LINE" — único método soportado por ahora
	Status              AssetStatus
	ThirdPartyNIT       string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// MonthlyDepreciation calcula la cuota mensual por línea recta, en centavos.
func (a *FixedAsset) MonthlyDepreciation() int64 {
	if a.UsefulLifeMonths <= 0 {
		return 0
	}
	return (a.AcquisitionCost - a.SalvageValue) / int64(a.UsefulLifeMonths)
}

type DepreciationRunStatus string

const (
	RunCompleted DepreciationRunStatus = "COMPLETED"
	RunReversed  DepreciationRunStatus = "REVERSED"
)

// DepreciationRun es una corrida de depreciación mensual — genera un único asiento con una
// línea de gasto/acumulada por cada activo depreciado ese mes.
type DepreciationRun struct {
	ID        uuid.UUID
	CompanyID uuid.UUID
	PeriodID  uuid.UUID
	RunDate   time.Time
	Status    DepreciationRunStatus
	JournalID *uuid.UUID
	CreatedAt time.Time
}

// DepreciationEntry es la cuota depreciada de un activo específico dentro de una corrida.
type DepreciationEntry struct {
	ID        uuid.UUID
	RunID     uuid.UUID
	AssetID   uuid.UUID
	Amount    int64 // centavos
	CreatedAt time.Time
}

var (
	ErrFixedAssetNotFound   = errors.New("activo fijo no encontrado")
	ErrDepreciationExists   = errors.New("ya se corrió la depreciación de este período")
	ErrNoDepreciableAssets  = errors.New("no hay activos con saldo por depreciar en este período")
	ErrAssetAlreadyDisposed = errors.New("el activo ya fue dado de baja")
)

type FixedAssetRepository interface {
	Create(ctx context.Context, a FixedAsset) (*FixedAsset, error)
	List(ctx context.Context, companyID uuid.UUID) ([]FixedAsset, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*FixedAsset, error)
	// GetAccumulatedDepreciation suma las cuotas ya corridas para el activo — fuente de verdad
	// en vez de un contador en la fila del activo (mismo criterio que los saldos del mayor).
	GetAccumulatedDepreciation(ctx context.Context, assetID uuid.UUID) (int64, error)
	UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status AssetStatus) error
}

type DepreciationRepository interface {
	CreateRun(ctx context.Context, run DepreciationRun, entries []DepreciationEntry) (*DepreciationRun, error)
	ListRuns(ctx context.Context, companyID uuid.UUID) ([]DepreciationRun, error)
	HasRunForPeriod(ctx context.Context, companyID, periodID uuid.UUID) (bool, error)
}
