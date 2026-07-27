// Package accounting es el corazón del ERP: todo movimiento de dinero genera un asiento.
// Motor central: PostDocument(event) recibe eventos de otros módulos y crea asientos.
// Ver _legacy/accounting/ para la implementación de referencia (plan de cuentas colombiano).
package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// JournalEntry es un asiento contable: un conjunto de líneas (débitos y créditos)
// que siempre deben sumar cero (principio de partida doble).
type JournalEntry struct {
	ID          uuid.UUID
	CompanyID   uuid.UUID
	PeriodID    uuid.UUID
	Reference   string    // "FE-SETP-001", "NOM-2026-01", etc.
	Description string
	PostedAt    time.Time
	Lines       []JournalLine
}

// JournalLine es una línea del asiento (un movimiento en una cuenta contable).
type JournalLine struct {
	ID          uuid.UUID
	EntryID     uuid.UUID
	AccountCode string // código del plan de cuentas
	AccountName string
	DebitCents  int64
	CreditCents int64
	Description string
}

type Repository interface {
	SaveEntry(ctx context.Context, e JournalEntry) (*JournalEntry, error)
	GetEntry(ctx context.Context, id uuid.UUID) (*JournalEntry, error)
	ListByPeriod(ctx context.Context, companyID, periodID uuid.UUID) ([]*JournalEntry, error)
}

// PostingEngine recibe eventos de dominio y genera los asientos contables correspondientes.
// Cada método sabe la mecánica contable de su evento (qué cuentas debitar/acreditar y por qué).
type PostingEngine interface {
	PostInvoice(ctx context.Context, invoiceID, companyID uuid.UUID) error
	PostPayroll(ctx context.Context, payrollID, companyID uuid.UUID) error
	PostPayment(ctx context.Context, paymentID, companyID uuid.UUID) error
	PostPurchase(ctx context.Context, purchaseID, companyID uuid.UUID) error
}
