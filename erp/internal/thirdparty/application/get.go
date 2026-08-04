package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/thirdparty/domain"
)

type GetUseCase struct {
	repo domain.Repository
}

func NewGetUseCase(repo domain.Repository) *GetUseCase {
	return &GetUseCase{repo: repo}
}

// ByID exige que el tercero tenga el rol pedido — un tercero que solo es Proveedor no aparece al
// consultarlo desde el catálogo de Clientes, aunque exista la fila.
func (uc *GetUseCase) ByID(ctx context.Context, companyID, id uuid.UUID, role domain.Role) (*domain.Party, error) {
	p, err := uc.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, notFoundForRole(err, role)
	}
	if role == domain.RoleCustomer && !p.IsCustomer {
		return nil, domain.ErrCustomerNotFound
	}
	if role == domain.RoleSupplier && !p.IsSupplier {
		return nil, domain.ErrSupplierNotFound
	}
	return p, nil
}

func (uc *GetUseCase) List(ctx context.Context, companyID uuid.UUID, role domain.Role) ([]domain.Party, error) {
	return uc.repo.List(ctx, companyID, role)
}
