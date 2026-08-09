package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/company/domain"
)

// MembershipReader expone qué empresas tiene vinculadas un usuario -- puerto local hacia
// security (implementado directamente por security/infrastructure/persistence/postgres.Repository,
// que ya tiene este método para el propio login). Antes, ListByUserID hacía un
// `WHERE id IN (SELECT company_id FROM security.user_companies ...)` directo desde el
// repositorio de company, violando la frontera de schema entre módulos (ver auditoría
// 2026-08-09, Fase 2 punto 15) -- ahora company solo conoce IDs de empresa, nunca el schema de
// security.
type MembershipReader interface {
	ListCompanyIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

type GetUseCase struct {
	repo    domain.Repository
	members MembershipReader
}

func NewGetUseCase(repo domain.Repository, members MembershipReader) *GetUseCase {
	return &GetUseCase{repo: repo, members: members}
}

func (uc *GetUseCase) ByID(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *GetUseCase) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Company, error) {
	return uc.repo.ListByIDs(ctx, ids)
}

func (uc *GetUseCase) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Company, error) {
	ids, err := uc.members.ListCompanyIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	return uc.repo.ListByIDs(ctx, ids)
}

func (uc *GetUseCase) UpdateBrandColor(ctx context.Context, id uuid.UUID, color string) error {
	return uc.repo.UpdateBrandColor(ctx, id, color)
}
