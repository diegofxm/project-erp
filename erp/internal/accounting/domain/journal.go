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
