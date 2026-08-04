// Package thirdparty implementa domain.CustomerPort y domain.SupplierPort de electronic leyendo
// del módulo unificado de terceros — mismo patrón que electronic/infrastructure/company. Un solo
// Adapter satisface ambas interfaces porque la vista (domain.Party) y el mapeo son idénticos;
// solo cambia si el llamador es from_sale.go (cliente) o from_purchase.go (proveedor).
package thirdparty

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/electronic/domain"
	thirdpartydomain "github.com/diegofxm/erp/internal/thirdparty/domain"
)

// Adapter implementa domain.CustomerPort y domain.SupplierPort usando el repositorio de terceros.
type Adapter struct {
	repo thirdpartydomain.Repository
}

// New crea el adaptador.
func New(repo thirdpartydomain.Repository) *Adapter {
	return &Adapter{repo: repo}
}

var (
	_ domain.CustomerPort = (*Adapter)(nil)
	_ domain.SupplierPort = (*Adapter)(nil)
)

func (a *Adapter) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Party, error) {
	p, err := a.repo.GetByID(ctx, companyID, id)
	if err != nil {
		return nil, fmt.Errorf("thirdparty adapter: %w", err)
	}
	return &domain.Party{
		IdentificationTypeCode: p.IdentificationTypeCode,
		IdentificationNumber:   p.IdentificationNumber,
		CheckDigit:             p.CheckDigit,
		Name:                   p.Name,
		AddressLine:            p.AddressLine,
		MunicipalityCode:       p.MunicipalityCode,
		DepartmentCode:         p.DepartmentCode,
		TaxSchemeCode:          p.TaxSchemeCode,
		TaxSchemeName:          p.TaxSchemeName,
		TaxRegimeCode:          p.TaxRegimeCode,
		LiabilityCodes:         p.LiabilityCodes,
		Phone:                  p.Phone,
		Email:                  p.Email,
	}, nil
}
