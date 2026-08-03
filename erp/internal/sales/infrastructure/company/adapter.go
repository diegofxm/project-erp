// Package company implementa domain.CompanyPort de sales leyendo del repositorio de empresas
// del ERP — mismo patrón que electronic/infrastructure/company.
package company

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	companydomain "github.com/diegofxm/erp/internal/company/domain"
	"github.com/diegofxm/erp/internal/sales/domain"
)

// Adapter implementa domain.CompanyPort usando el repositorio de empresas.
type Adapter struct {
	repo companydomain.Repository
}

// New crea el adaptador.
func New(repo companydomain.Repository) *Adapter {
	return &Adapter{repo: repo}
}

var _ domain.CompanyPort = (*Adapter)(nil)

func (a *Adapter) GetCompany(ctx context.Context, id uuid.UUID) (*domain.CompanyInfo, error) {
	c, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("company adapter: %w", err)
	}
	return &domain.CompanyInfo{
		BusinessName:     c.BusinessName,
		TradeName:        c.TradeName,
		NIT:              c.NIT,
		CheckDigit:       c.CheckDigit,
		AddressLine:      c.AddressLine,
		MunicipalityCode: c.MunicipalityCode,
		Phone:            c.Phone,
		Email:            c.Email,
		Logo:             c.Logo,
		LogoContentType:  c.LogoContentType,
	}, nil
}
