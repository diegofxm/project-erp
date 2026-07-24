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
type JournalLine struct {
	ID          uuid.UUID
	JournalID   uuid.UUID
	AccountID   uuid.UUID
	AccountCode string
	Debit       float64
	Credit      float64
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
type LineRequest struct {
	AccountCode string
	Debit       float64
	Credit      float64
	CostCenter  string
	Description string
}
