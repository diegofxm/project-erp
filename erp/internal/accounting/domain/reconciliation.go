package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ReconciliationMark cruza dos líneas de asiento de la misma cuenta que se cancelan entre sí
// (ej. el débito de una factura de cartera contra el crédito de su pago, cuando no quedaron en
// el mismo asiento) — distinto de la conciliación bancaria (bank_statement_lines), que cruza
// contra un extracto externo. ReconciledWith puede ser nil: marcar una línea sola como "ya
// revisada" sin cruzarla contra otra todavía.
type ReconciliationMark struct {
	ID             uuid.UUID
	CompanyID      uuid.UUID
	JournalLineID  uuid.UUID
	ReconciledWith *uuid.UUID
	Note           string
	ReconciledAt   time.Time
}

// OpenLine es una línea de asiento sin conciliar de una cuenta — candidata a cruzarse contra
// otra (ej. todas las líneas de la cuenta 1305 Clientes sin marcar todavía).
type OpenLine struct {
	LineID        uuid.UUID
	JournalID     uuid.UUID
	Date          time.Time
	Description   string
	VoucherNumber string
	ThirdPartyNIT string
	Debit         int64
	Credit        int64
}

var ErrReconciliationLineNotFound = errors.New("línea de asiento no encontrada en esa cuenta")

type ReconciliationRepository interface {
	// Mark cruza journalLineID contra reconciledWith (puede ser nil) — upsert por journal_line_id
	// (UNIQUE), así que volver a marcar la misma línea reemplaza la marca anterior.
	Mark(ctx context.Context, companyID, journalLineID uuid.UUID, reconciledWith *uuid.UUID, note string) (*ReconciliationMark, error)
	Unmark(ctx context.Context, companyID, journalLineID uuid.UUID) error
	// ListOpenLines devuelve las líneas de accountCode sin marca de conciliación, más antiguas
	// primero — lo que el usuario ve para elegir qué cruzar contra qué.
	ListOpenLines(ctx context.Context, companyID uuid.UUID, accountCode string) ([]OpenLine, error)
}
