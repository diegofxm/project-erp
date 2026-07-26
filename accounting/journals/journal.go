package journals

import (
	"time"

	"github.com/google/uuid"
)

type Status string
type EntryType string

// Book discrimina el libro contable al que pertenece un asiento.
// BookBoth (defecto) aplica a ambos libros y es válido en cualquier estado del plan.
type Book string

const (
	StatusDraft  Status = "DRAFT"
	StatusPosted Status = "POSTED"
	StatusVoid   Status = "VOID"

	EntryManual     EntryType = "MANUAL"
	EntryAutomatic  EntryType = "AUTOMATIC"
	EntryAdjustment EntryType = "ADJUSTMENT"
	EntryClosing    EntryType = "CLOSING"
	EntryOpening    EntryType = "OPENING"

	// BookBoth aplica a ambos libros (PCGA y NIIF). Es el valor por defecto.
	BookBoth Book = "BOTH"
	// BookPCGA es exclusivo del libro local (Decreto 2649 / PCGA colombiano).
	BookPCGA Book = "PCGA"
	// BookNIIF es exclusivo del libro IFRS (ajustes de convergencia, NIIF 16, etc.).
	BookNIIF Book = "NIIF"
)

// JournalEntry es la cabecera de un asiento contable.
type JournalEntry struct {
	ID                 uuid.UUID
	CompanyID          uuid.UUID
	PeriodID           uuid.UUID
	Date               time.Time
	Description        string
	Status             Status
	Source             string
	EntryType          EntryType
	VoucherType        string    // código del tipo de comprobante (CE, CI, NC…)
	VoucherNumber      string    // número de comprobante formateado (CE-2025-00001)
	SourceDocumentID   uuid.UUID // UUID del documento que originó el asiento (uuid.Nil si ninguno)
	SourceDocumentType string    // discriminador: "FE", "DS", "NC", "NOM", etc.
	Book               Book      // libro contable: "BOTH" (defecto), "PCGA", "NIIF"
	Lines              []*JournalLine
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// JournalLine es una línea del asiento — exactamente uno de Debit o Credit > 0.
// Debit y Credit están en centavos (int64) para evitar errores de punto flotante.
// ThirdPartyNIT es el NIT del tercero involucrado (proveedor, cliente, empleado, banco).
// Requerido por la DIAN para Medios Magnéticos / Información Exógena. Vacío solo
// en asientos internos (apertura, cierre, ajustes sin contraparte externa).
// ForeignAmount y ForeignCurrency son opcionales: solo se populan cuando el documento
// original está en moneda extranjera (USD, EUR). Permiten la revaluación al cierre de período.
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
	ForeignAmount   int64  // centavos en moneda extranjera; 0 si el asiento es en COP
	ForeignCurrency string // ISO 4217 ("USD", "EUR"); vacío si el asiento es en COP
	CreatedAt       time.Time
}

// PostRequest contiene los datos necesarios para registrar un asiento nuevo.
type PostRequest struct {
	CompanyID          uuid.UUID
	Date               time.Time
	Description        string
	Source             string
	EntryType          EntryType
	VoucherType        string    // opcional: si se indica, se asigna el siguiente número consecutivo
	SourceDocumentID   uuid.UUID // UUID del documento fuente (FE, DS, NC…); uuid.Nil si no aplica
	SourceDocumentType string    // discriminador del tipo de documento fuente
	Book               Book      // vacío o "BOTH" usa ambos libros (defecto)
	Lines              []LineRequest
}

// LineRequest referencia la cuenta por código para que el servicio resuelva el UUID.
// Debit y Credit en centavos (int64). ThirdPartyNIT es opcional para asientos internos.
// ForeignAmount y ForeignCurrency son opcionales: solo se usan en documentos en moneda
// extranjera para habilitar la revaluación al cierre del período.
type LineRequest struct {
	AccountCode     string
	Debit           int64
	Credit          int64
	ThirdPartyNIT   string
	CostCenter      string
	Description     string
	ForeignAmount   int64  // centavos en moneda extranjera; 0 si el asiento es en COP
	ForeignCurrency string // ISO 4217 ("USD", "EUR"); vacío si el asiento es en COP
}
