package application

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/company/domain"
)

var ErrInvalidLogoType = errors.New("tipo de imagen no permitido (use image/png, image/jpeg o image/webp)")

type UpdateLogoUseCase struct {
	repo domain.Repository
}

func NewUpdateLogoUseCase(repo domain.Repository) *UpdateLogoUseCase {
	return &UpdateLogoUseCase{repo: repo}
}

func (uc *UpdateLogoUseCase) Update(ctx context.Context, id uuid.UUID, data []byte, contentType string) error {
	if !isAllowedImageType(contentType) {
		return ErrInvalidLogoType
	}
	return uc.repo.UpdateLogo(ctx, id, data, contentType)
}

func (uc *UpdateLogoUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteLogo(ctx, id)
}

func isAllowedImageType(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	return ct == "image/png" || ct == "image/jpeg" || ct == "image/webp"
}
