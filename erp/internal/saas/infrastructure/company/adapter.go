// Package company implementa saas/domain.CompanyPort y CompanyProvisioningPort leyendo/creando
// empresas vía el módulo company del ERP — mismo patrón que sales/infrastructure/company y
// electronic/infrastructure/company.
package company

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	companyapp "github.com/diegofxm/erp/internal/company/application"
	companydomain "github.com/diegofxm/erp/internal/company/domain"
	"github.com/diegofxm/erp/internal/saas/domain"
)

type Adapter struct {
	repo     companydomain.Repository
	createUC *companyapp.CreateUseCase
}

func New(repo companydomain.Repository, createUC *companyapp.CreateUseCase) *Adapter {
	return &Adapter{repo: repo, createUC: createUC}
}

var (
	_ domain.CompanyPort             = (*Adapter)(nil)
	_ domain.CompanyProvisioningPort = (*Adapter)(nil)
)

func (a *Adapter) GetCompany(ctx context.Context, id uuid.UUID) (*domain.CompanyInfo, error) {
	c, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("saas company adapter: %w", err)
	}
	return &domain.CompanyInfo{
		ID:           c.ID,
		BusinessName: c.BusinessName,
		TradeName:    c.TradeName,
		NIT:          c.NIT,
	}, nil
}

// CreateCompanyForOwner crea la primera empresa de un dueño nuevo (ver saas/application/prospect.go)
// — reusa company.CreateUseCase, que ya resuelve defaults fiscales (régimen, ambiente, tipo de
// entidad), calcula el dígito de verificación para NIT, vincula al creador como "owner", y crea la
// bodega por defecto (ver company.CreateUseCase.Execute).
func (a *Adapter) CreateCompanyForOwner(ctx context.Context, ownerID uuid.UUID, businessName, nit, contactEmail string) (uuid.UUID, error) {
	c, err := a.createUC.Execute(ctx, ownerID, companyapp.CreateRequest{
		NIT:                    nit,
		BusinessName:           businessName,
		IdentificationTypeCode: "31", // NIT — lo que trae un prospecto que solicitó acceso
		Email:                  contactEmail,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("crear empresa del nuevo dueño: %w", err)
	}
	return c.ID, nil
}
