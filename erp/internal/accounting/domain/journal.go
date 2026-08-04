package domain

import (
	"time"

	"github.com/google/uuid"
)

type JournalStatus string
type EntryType string
type Book string

const (
	StatusDraft  JournalStatus = "DRAFT"
	StatusPosted JournalStatus = "POSTED"
	StatusVoid   JournalStatus = "VOID"

	EntryManual     EntryType = "MANUAL"
	EntryAutomatic  EntryType = "AUTOMATIC"
	EntryAdjustment EntryType = "ADJUSTMENT"
	EntryClosing    EntryType = "CLOSING"
	EntryOpening    EntryType = "OPENING"

	BookBoth Book = "BOTH"
	BookPCGA Book = "PCGA"
	BookNIIF Book = "NIIF"

	VoucherExpense = "CE"
	VoucherIncome  = "CI"
	VoucherNote    = "NC"
	VoucherPayroll = "NI"
	VoucherClosing = "CJ"
	VoucherOpening = "AP"
)

// standardVoucherTypes son los tipos de comprobante que usa el propio sistema al contabilizar
// automáticamente (ventas, compras, nómina, pagos, cierre/apertura de período) — siempre válidos
// sin que la empresa tenga que registrarlos a mano, para no romper esas contabilizaciones.
// Cualquier otro tipo (los que crea la empresa desde Configuración de comprobantes) sí debe
// existir y estar activo en accounting.voucher_types antes de poder usarse — ver
// PostJournalUseCase.validateVoucherType.
var standardVoucherTypes = map[string]bool{
	VoucherExpense: true,
	VoucherIncome:  true,
	VoucherNote:    true,
	VoucherPayroll: true,
	VoucherClosing: true,
	VoucherOpening: true,
}

// IsStandardVoucherType indica si code es uno de los tipos de comprobante propios del sistema.
func IsStandardVoucherType(code string) bool {
	return standardVoucherTypes[code]
}

type JournalEntry struct {
	ID                 uuid.UUID
	CompanyID          uuid.UUID
	PeriodID           uuid.UUID
	Date               time.Time
	Description        string
	Status             JournalStatus
	Source             string
	EntryType          EntryType
	VoucherType        string
	VoucherNumber      string
	SourceDocumentID   uuid.UUID
	SourceDocumentType string
	Book               Book
	Lines              []*JournalLine
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type JournalLine struct {
	ID              uuid.UUID
	JournalID       uuid.UUID
	AccountID       uuid.UUID
	AccountCode     string
	Debit           int64
	Credit          int64
	ThirdPartyNIT   string
	CostCenter      string
	Description     string
	ForeignAmount   int64
	ForeignCurrency string
	CreatedAt       time.Time
}

// PostRequest son los datos de entrada para registrar un asiento.
// Debit y Credit en centavos (int64). AccountCode se resuelve en el use case.
type PostRequest struct {
	CompanyID          uuid.UUID
	Date               time.Time
	Description        string
	Source             string
	EntryType          EntryType
	VoucherType        string
	SourceDocumentID   uuid.UUID
	SourceDocumentType string
	Book               Book
	Lines              []LineRequest
}

type LineRequest struct {
	AccountCode     string
	Debit           int64
	Credit          int64
	ThirdPartyNIT   string
	CostCenter      string
	Description     string
	ForeignAmount   int64
	ForeignCurrency string
}

// PLBalance es el saldo neto de una cuenta en centavos.
// Balance = SUM(debit) - SUM(credit); negativo = saldo acreedor.
type PLBalance struct {
	AccountID   uuid.UUID
	AccountCode string
	AccountName string
	Category    string
	Balance     int64
}

// TrialBalanceRow es el saldo de una cuenta en un rango de fechas (Balance de Prueba).
// Debit/Credit son el movimiento acumulado del periodo; Balance = Debit - Credit.
type TrialBalanceRow struct {
	AccountID   uuid.UUID
	AccountCode string
	AccountName string
	Category    string
	Debit       int64
	Credit      int64
	Balance     int64
}

// LedgerLine es un movimiento del Libro Mayor de una cuenta, con saldo acumulado.
type LedgerLine struct {
	JournalID      uuid.UUID
	Date           time.Time
	Description    string
	VoucherType    string
	VoucherNumber  string
	Debit          int64
	Credit         int64
	RunningBalance int64
}

type VoucherTypeConfig struct {
	ID             uuid.UUID
	CompanyID      uuid.UUID
	Code           string
	Name           string
	ResetsAnnually bool
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
