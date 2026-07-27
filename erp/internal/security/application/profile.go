package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/security/domain"
)

type UpdateProfileUseCase struct {
	repo domain.Repository
}

func NewUpdateProfileUseCase(repo domain.Repository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{repo: repo}
}

func (uc *UpdateProfileUseCase) Execute(ctx context.Context, userID uuid.UUID, name, email string) (*domain.User, error) {
	return uc.repo.UpdateProfile(ctx, userID, name, email)
}

type GetProfileUseCase struct {
	repo domain.Repository
}

func NewGetProfileUseCase(repo domain.Repository) *GetProfileUseCase {
	return &GetProfileUseCase{repo: repo}
}

func (uc *GetProfileUseCase) Execute(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return uc.repo.GetByID(ctx, userID)
}
