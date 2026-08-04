package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/thirdparty/domain"
)

type DeleteUseCase struct {
	repo domain.Repository
}

func NewDeleteUseCase(repo domain.Repository) *DeleteUseCase {
	return &DeleteUseCase{repo: repo}
}

// Execute quita el rol pedido del tercero. Si no le queda ningún rol, se elimina la fila;
// mientras conserve el otro rol, la fila sigue existiendo (ej. dejó de ser proveedor pero sigue
// siendo cliente de la misma empresa).
func (uc *DeleteUseCase) Execute(ctx context.Context, companyID, id uuid.UUID, role domain.Role) error {
	p, err := uc.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return notFoundForRole(err, role)
	}
	switch role {
	case domain.RoleCustomer:
		if !p.IsCustomer {
			return domain.ErrCustomerNotFound
		}
		p.IsCustomer = false
		p.CreditLimit = nil
	case domain.RoleSupplier:
		if !p.IsSupplier {
			return domain.ErrSupplierNotFound
		}
		p.IsSupplier = false
		p.PaymentTermsDays = 0
	}
	if !p.IsCustomer && !p.IsSupplier {
		return uc.repo.Delete(ctx, companyID, id)
	}
	_, err = uc.repo.Update(ctx, *p)
	return err
}
