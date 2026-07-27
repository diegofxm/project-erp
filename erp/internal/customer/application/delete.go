package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/customer/domain"
)

type DeleteUseCase struct {
	repo domain.Repository
}

func NewDeleteUseCase(repo domain.Repository) *DeleteUseCase {
	return &DeleteUseCase{repo: repo}
}

func (uc *DeleteUseCase) Execute(ctx context.Context, companyID, id uuid.UUID) error {
	return uc.repo.Delete(ctx, companyID, id)
}
