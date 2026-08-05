package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/saas/domain"
)

type ProspectUseCase struct {
	repo domain.ProspectRepository
}

func NewProspectUseCase(repo domain.ProspectRepository) *ProspectUseCase {
	return &ProspectUseCase{repo: repo}
}

type SubmitProspectRequest struct {
	Name              string
	Email             string
	NIT               string
	CedulaFile        []byte
	CedulaContentType string
	RUTFile           []byte
	RUTContentType    string
}

// Submit registra una solicitud de acceso pública (sin cuenta) — queda "pending" hasta que un
// superadmin la revise.
func (uc *ProspectUseCase) Submit(ctx context.Context, req SubmitProspectRequest) (*domain.Prospect, error) {
	return uc.repo.Create(ctx, domain.Prospect{
		Name: req.Name, Email: req.Email, NIT: req.NIT,
		CedulaFile: req.CedulaFile, CedulaContentType: req.CedulaContentType,
		RUTFile: req.RUTFile, RUTContentType: req.RUTContentType,
		Status: domain.ProspectPending,
	})
}

func (uc *ProspectUseCase) List(ctx context.Context) ([]domain.Prospect, error) {
	return uc.repo.List(ctx)
}

func (uc *ProspectUseCase) GetByID(ctx context.Context, id uuid.UUID) (*domain.Prospect, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *ProspectUseCase) Approve(ctx context.Context, id uuid.UUID) (*domain.Prospect, error) {
	p, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Status != domain.ProspectPending {
		return nil, domain.ErrProspectNotPending
	}
	return uc.repo.UpdateStatus(ctx, id, domain.ProspectApproved, p.Notes)
}

func (uc *ProspectUseCase) Reject(ctx context.Context, id uuid.UUID, notes string) (*domain.Prospect, error) {
	p, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Status != domain.ProspectPending {
		return nil, domain.ErrProspectNotPending
	}
	return uc.repo.UpdateStatus(ctx, id, domain.ProspectRejected, notes)
}
