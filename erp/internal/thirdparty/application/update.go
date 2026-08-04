package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/thirdparty/domain"
)

type UpdateUseCase struct {
	repo domain.Repository
}

func NewUpdateUseCase(repo domain.Repository) *UpdateUseCase {
	return &UpdateUseCase{repo: repo}
}

func (uc *UpdateUseCase) Execute(ctx context.Context, companyID, id uuid.UUID, role domain.Role, req SaveRequest) (*domain.Party, error) {
	existing, err := uc.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, notFoundForRole(err, role)
	}
	applyShared(existing, req, resolveCheckDigit(req))
	applyRole(existing, role, req)
	return uc.repo.Update(ctx, *existing)
}
