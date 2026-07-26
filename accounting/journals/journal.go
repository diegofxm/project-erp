package journals

import (
	"time"

	"github.com/google/uuid"
)

type Status string
type EntryType string

const (
	StatusDraft  Status = "DRAFT"
	StatusPosted Status = "POSTED"
	StatusVoid   Status = "VOID"

	EntryManual     EntryType = "MANUAL"
	EntryAutomatic  EntryType = "AUTOMATIC"
	EntryAdjustment EntryType = "ADJUSTMENT"
	EntryClosing    EntryType = "CLOSING"
	EntryOpening    EntryType = "OPENING"
)

// JournalEntry es la cabecera de un asiento contable.
type JournalEntry struct {
	ID          uuid.UUID
	CompanyID   uuid.UUID
	PeriodID    uuid.UUID
	Date        time.Time
	Description string
	Status      Status
	Source      string
	EntryType   EntryType
	Lines       []*JournalLine
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// JournalLine es una línea del asiento — exactamente uno de Debit o Credit > 0.
// Debit y Credit están en centavos (int64) para evitar errores de punto flotante.
// TerceroNIT es el NIT del tercero involucrado (proveedor, cliente, empleado, banco).
// Requerido por la DIAN para Medios Magnéticos / Información Exógena. Vacío solo
// en asientos internos (apertura, cierre, ajustes sin contraparte externa).
type JournalLine struct {
	ID          uuid.UUID
	JournalID   uuid.UUID
	AccountID   uuid.UUID
	AccountCode string
	Debit       int64
	Credit      int64
	TerceroNIT  string
	CostCenter  string
	Description string
	CreatedAt   time.Time
}

// PostRequest contiene los datos necesarios para registrar un asiento nuevo.
type PostRequest struct {
	CompanyID   uuid.UUID
	Date        time.Time
	Description string
	Source      string
	EntryType   EntryType
	Lines       []LineRequest
}

// LineRequest referencia la cuenta por código para que el servicio resuelva el UUID.
// Debit y Credit en centavos (int64). TerceroNIT es opcional para asientos internos.
type LineRequest struct {
	AccountCode string
	Debit       int64
	Credit      int64
	TerceroNIT  string
	CostCenter  string
	Description string
}
