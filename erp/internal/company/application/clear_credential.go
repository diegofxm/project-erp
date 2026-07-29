package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/company/domain"
)

type CredentialKind string

const (
	CredentialSoftware    CredentialKind = "software"
	CredentialNeSoftware  CredentialKind = "ne_software"
	CredentialCertificate CredentialKind = "certificate"
)

type ClearCredentialUseCase struct {
	repo domain.Repository
}

func NewClearCredentialUseCase(repo domain.Repository) *ClearCredentialUseCase {
	return &ClearCredentialUseCase{repo: repo}
}

func (uc *ClearCredentialUseCase) Execute(ctx context.Context, id uuid.UUID, kind CredentialKind) error {
	switch kind {
	case CredentialSoftware:
		return uc.repo.ClearSoftware(ctx, id)
	case CredentialNeSoftware:
		return uc.repo.ClearNeSoftware(ctx, id)
	case CredentialCertificate:
		return uc.repo.ClearCertificate(ctx, id)
	default:
		return fmt.Errorf("tipo de credencial desconocido: %s", kind)
	}
}
