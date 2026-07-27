package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/company/domain"
)

type UpdateCredentialsUseCase struct {
	repo domain.Repository
}

func NewUpdateCredentialsUseCase(repo domain.Repository) *UpdateCredentialsUseCase {
	return &UpdateCredentialsUseCase{repo: repo}
}

type UpdateCredentialsRequest struct {
	SoftwareID          string
	SoftwarePIN         string
	Certificate         []byte
	CertificatePassword string
	NeSoftwareID        string
	NeSoftwarePIN       string
}

func (uc *UpdateCredentialsUseCase) Execute(ctx context.Context, id uuid.UUID, req UpdateCredentialsRequest) error {
	return uc.repo.UpdateCredentials(ctx, id,
		req.SoftwareID, req.SoftwarePIN,
		req.Certificate, req.CertificatePassword,
		req.NeSoftwareID, req.NeSoftwarePIN,
	)
}
