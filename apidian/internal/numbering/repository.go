package numbering

import (
	"context"

	"github.com/google/uuid"
)

// Repository define las operaciones de persistencia de rangos de numeración.
type Repository interface {
	Create(ctx context.Context, nr NumberingRange) (*NumberingRange, error)
	GetByID(ctx context.Context, id uuid.UUID) (*NumberingRange, error)

	// ClaimNext reclama el siguiente número del rango de forma atómica: bajo concurrencia,
	// dos llamadas simultáneas nunca deben recibir el mismo número ni dejar un hueco.
	// Devuelve ErrRangeExhausted si RangeTo ya fue alcanzado.
	ClaimNext(ctx context.Context, id uuid.UUID) (int64, error)

	// ReleaseIfCurrent revierte el ÚLTIMO ClaimNext de este rango — pero SOLO si current_number
	// sigue siendo exactamente number, es decir, nadie reclamó otro número desde entonces. Si
	// no se cumple esa condición, no hace nada (alguien más ya avanzó; retroceder ahí sería
	// arriesgar que dos documentos terminen con el mismo número). Pensado para un documento
	// rechazado por la DIAN o que nunca se logró transmitir (send_error) — ese número nunca
	// quedó realmente registrado, así que el siguiente intento lo puede reclamar de nuevo en
	// vez de avanzar y dejar un hueco (ver docs/apidian-architecture.md sección 9.33).
	ReleaseIfCurrent(ctx context.Context, id uuid.UUID, number int64) error

	// ListByIssuer devuelve los rangos de un emisor, opcionalmente filtrados por tipo de
	// documento DIAN ("" = todos). Sin paginación a propósito: el volumen esperado por emisor
	// es bajo (resoluciones de numeración, no documentos emitidos).
	ListByIssuer(ctx context.Context, issuerID uuid.UUID, dianDocumentTypeCode string) ([]*NumberingRange, error)
}
