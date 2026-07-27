package assets

import (
	"time"

	"github.com/google/uuid"
)

// DepreciationMethod indica el método de cálculo de depreciación.
type DepreciationMethod string

const (
	MethodStraightLine DepreciationMethod = "STRAIGHT_LINE"
)

// AssetStatus indica el estado del activo dentro del ciclo de vida.
type AssetStatus string

const (
	StatusActive           AssetStatus = "ACTIVE"
	StatusDisposed         AssetStatus = "DISPOSED"
	StatusFullyDepreciated AssetStatus = "FULLY_DEPRECIATED"
)

// FixedAsset representa un bien de propiedad, planta y equipo (PPE).
// Los campos *Account apuntan a los códigos del PUC que se usarán en los asientos
// de depreciación mensual y baja del activo.
type FixedAsset struct {
	ID                  uuid.UUID
	CompanyID           uuid.UUID
	Code                string
	Name                string
	Description         string
	AssetAccount        string // código PUC del activo     (ej. 152001)
	DepreciationAccount string // gasto depreciación         (ej. 516010)
	AccumulatedAccount  string // depreciación acumulada     (ej. 159220)
	GainAccount         string // ganancia en baja           (ej. 424505)
	LossAccount         string // pérdida en baja            (ej. 529005)
	AcquisitionDate     time.Time
	AcquisitionCost     int64 // centavos
	SalvageValue        int64 // centavos — valor residual al final de la vida útil
	UsefulLifeMonths    int
	DepreciationMethod  DepreciationMethod
	Status              AssetStatus
	ThirdPartyNIT       string    // NIT del proveedor del activo
	SourceDocumentID    uuid.UUID // UUID del DS/FE de compra que originó el activo; uuid.Nil si ninguno
	SourceDocumentType  string    // discriminador: "FE", "DS", etc.
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// DepreciationRun es el registro de una corrida de depreciación mensual.
// JournalID apunta al asiento contable generado para esa corrida.
type DepreciationRun struct {
	ID        uuid.UUID
	CompanyID uuid.UUID
	PeriodID  uuid.UUID
	RunDate   time.Time
	Status    string
	JournalID uuid.UUID // uuid.Nil si aún no se ha generado el asiento
	CreatedAt time.Time
}

// DepreciationEntry es el detalle por activo dentro de una corrida.
type DepreciationEntry struct {
	ID        uuid.UUID
	RunID     uuid.UUID
	AssetID   uuid.UUID
	Amount    int64 // centavos depreciadados en esta entrada
	CreatedAt time.Time
}

// ScheduleRow es una fila del plan de depreciación proyectado.
type ScheduleRow struct {
	MonthNumber int
	Date        time.Time
	Amount      int64 // depreciación del mes en centavos
	Accumulated int64 // total acumulado hasta este mes
	BookValue   int64 // valor en libros = costo - acumulado
}

// DisposeRequest contiene los parámetros para dar de baja un activo.
type DisposeRequest struct {
	CompanyID       uuid.UUID
	AssetID         uuid.UUID
	Date            time.Time
	Proceeds        int64  // 0 si es baja sin venta
	ProceedsAccount string // cuenta de caja/banco; requerido si Proceeds > 0
	VoucherType     string // opcional — tipo de comprobante del asiento de baja
}
