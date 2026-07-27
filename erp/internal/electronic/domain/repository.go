package domain

import (
	"context"

	"github.com/google/uuid"
)

// DocumentRepository define las operaciones de persistencia de documentos.
type DocumentRepository interface {
	Create(ctx context.Context, d Document) (*Document, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Document, error)
	GetByDocumentKey(ctx context.Context, companyID uuid.UUID, key string) (*Document, error)
	UpdateDraft(ctx context.Context, d Document) (*Document, error)
	Confirm(ctx context.Context, d Document) (*Document, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateDianStatus(ctx context.Context, id uuid.UUID, status Status, trackID, statusCode, statusDescription, statusMessage, applicationResponseXML string) error
	ListByCompany(ctx context.Context, companyID uuid.UUID, filter ListFilter) ([]*Document, error)
	GetRelatedNotes(ctx context.Context, companyID uuid.UUID, prefix string, number int64) ([]RelatedNote, error)
}

// NumberingRepository define las operaciones de persistencia de rangos.
type NumberingRepository interface {
	Create(ctx context.Context, nr NumberingRange) (*NumberingRange, error)
	GetByID(ctx context.Context, id uuid.UUID) (*NumberingRange, error)
	ClaimNext(ctx context.Context, id uuid.UUID) (int64, error)
	ReleaseIfCurrent(ctx context.Context, id uuid.UUID, number int64) error
	ClearTestSetID(ctx context.Context, id uuid.UUID) error
	Deactivate(ctx context.Context, id uuid.UUID) error
	ListByCompany(ctx context.Context, companyID uuid.UUID, dianDocumentTypeCode string) ([]*NumberingRange, error)
}
