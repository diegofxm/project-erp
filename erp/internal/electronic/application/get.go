package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/electronic/domain"
)

// GetDocumentUseCase devuelve un documento por ID.
type GetDocumentUseCase struct {
	documents domain.DocumentRepository
}

func NewGetDocumentUseCase(documents domain.DocumentRepository) *GetDocumentUseCase {
	return &GetDocumentUseCase{documents: documents}
}

func (uc *GetDocumentUseCase) Get(ctx context.Context, companyID, id uuid.UUID) (*domain.Document, error) {
	d, err := uc.documents.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d.CompanyID != companyID {
		return nil, domain.ErrDocumentNotFound
	}
	return d, nil
}

// ListDocumentsUseCase lista documentos de una empresa.
type ListDocumentsUseCase struct {
	documents domain.DocumentRepository
}

func NewListDocumentsUseCase(documents domain.DocumentRepository) *ListDocumentsUseCase {
	return &ListDocumentsUseCase{documents: documents}
}

func (uc *ListDocumentsUseCase) List(ctx context.Context, companyID uuid.UUID, filter domain.ListFilter) ([]*domain.Document, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}
	return uc.documents.ListByCompany(ctx, companyID, filter)
}

// ManageNumberingUseCase gestiona rangos de numeración.
type ManageNumberingUseCase struct {
	numbering domain.NumberingRepository
}

func NewManageNumberingUseCase(numbering domain.NumberingRepository) *ManageNumberingUseCase {
	return &ManageNumberingUseCase{numbering: numbering}
}

func (uc *ManageNumberingUseCase) Create(ctx context.Context, nr domain.NumberingRange) (*domain.NumberingRange, error) {
	return uc.numbering.Create(ctx, nr)
}

func (uc *ManageNumberingUseCase) List(ctx context.Context, companyID uuid.UUID, dianDocType string) ([]*domain.NumberingRange, error) {
	return uc.numbering.ListByCompany(ctx, companyID, dianDocType)
}

func (uc *ManageNumberingUseCase) Deactivate(ctx context.Context, companyID, id uuid.UUID) error {
	nr, err := uc.numbering.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if nr.CompanyID != companyID {
		return domain.ErrRangeNotFound
	}
	return uc.numbering.Deactivate(ctx, id)
}

func (uc *ManageNumberingUseCase) Activate(ctx context.Context, companyID, id uuid.UUID) (*domain.NumberingRange, error) {
	nr, err := uc.numbering.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if nr.CompanyID != companyID {
		return nil, domain.ErrRangeNotFound
	}
	if err := uc.numbering.Activate(ctx, id); err != nil {
		return nil, err
	}
	return uc.numbering.GetByID(ctx, id)
}
