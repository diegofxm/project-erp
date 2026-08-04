// Package thirdparty implementa domain.CustomerPort de sales leyendo del módulo unificado de
// terceros — mismo patrón que sales/infrastructure/company.
package thirdparty

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/sales/domain"
	thirdpartydomain "github.com/diegofxm/erp/internal/thirdparty/domain"
)

// Adapter implementa domain.CustomerPort usando el repositorio de terceros.
type Adapter struct {
	repo thirdpartydomain.Repository
}

// New crea el adaptador.
func New(repo thirdpartydomain.Repository) *Adapter {
	return &Adapter{repo: repo}
}

var _ domain.CustomerPort = (*Adapter)(nil)

func (a *Adapter) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Customer, error) {
	p, err := a.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, fmt.Errorf("thirdparty adapter: %w", err)
	}
	return &domain.Customer{
		Name:                   p.Name,
		IdentificationTypeCode: p.IdentificationTypeCode,
		IdentificationNumber:   p.IdentificationNumber,
		CheckDigit:             p.CheckDigit,
		AddressLine:            p.AddressLine,
		Phone:                  p.Phone,
		Email:                  p.Email,
		CreditLimit:            p.CreditLimit,
	}, nil
}
