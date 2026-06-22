package documents

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ListFilter acota una consulta de listado de documentos — todos los campos son opcionales,
// el valor cero significa "sin ese filtro". Limit/Offset siempre llegan ya normalizados
// (nunca cero/negativos, nunca por encima del máximo permitido) porque Service.ListDocuments
// los normaliza antes de llamar al repositorio — ningún Repository debe confiar en que el
// llamador ya los validó.
type ListFilter struct {
	DianDocumentTypeCode string
	Status               Status
	From, To             time.Time
	Limit, Offset        int
}

// Repository define las operaciones de persistencia de documentos.
type Repository interface {
	Create(ctx context.Context, d Document) (*Document, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Document, error)
	UpdateDianStatus(ctx context.Context, id uuid.UUID, status Status, trackID, statusCode, statusDescription, statusMessage string) error

	// ListByIssuer devuelve los documentos de un emisor que cumplan filter, más recientes
	// primero (por fecha de emisión, luego por creación).
	ListByIssuer(ctx context.Context, issuerID uuid.UUID, filter ListFilter) ([]*Document, error)
}
