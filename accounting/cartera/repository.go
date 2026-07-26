package cartera

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository abstrae el acceso a datos del módulo de cartera.
type Repository interface {
	// GetMovements devuelve todos los movimientos en las cuentas indicadas para
	// una empresa hasta la fecha asOf, ordenados por NIT + fecha. Solo incluye
	// líneas con third_party_nit registrado.
	GetMovements(ctx context.Context, companyID uuid.UUID, asOf time.Time, accountPrefixes []string) ([]*Movement, error)

	// GetNITMovements devuelve los movimientos de un NIT específico en un rango de fechas.
	GetNITMovements(ctx context.Context, companyID uuid.UUID, nit string, accountPrefixes []string, from, to time.Time) ([]*Movement, error)

	// MarkReconciled registra la conciliación de una línea contra otra.
	MarkReconciled(ctx context.Context, mark ReconciliationMark) (*ReconciliationMark, error)

	// UnmarkReconciled elimina la marca de conciliación de una línea.
	UnmarkReconciled(ctx context.Context, journalLineID uuid.UUID) error
}
