// Package thirdparty implementa domain.SupplierPort de purchase leyendo del módulo unificado de
// terceros — mismo patrón que purchase/infrastructure/company.
package thirdparty

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/purchase/domain"
	thirdpartydomain "github.com/diegofxm/erp/internal/thirdparty/domain"
)

// Adapter implementa domain.SupplierPort usando el repositorio de terceros.
type Adapter struct {
	repo thirdpartydomain.Repository
}

// New crea el adaptador.
func New(repo thirdpartydomain.Repository) *Adapter {
	return &Adapter{repo: repo}
}

var _ domain.SupplierPort = (*Adapter)(nil)

func (a *Adapter) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Supplier, error) {
	p, err := a.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, fmt.Errorf("thirdparty adapter: %w", err)
	}
	return &domain.Supplier{
		Name:                   p.Name,
		IdentificationTypeCode: p.IdentificationTypeCode,
		IdentificationNumber:   p.IdentificationNumber,
		CheckDigit:             p.CheckDigit,
		AddressLine:            p.AddressLine,
		Phone:                  p.Phone,
		Email:                  p.Email,
	}, nil
}
