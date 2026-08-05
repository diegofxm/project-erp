// Package company implementa saas/domain.CompanyPort leyendo del repositorio de empresas del
// ERP — mismo patrón que sales/infrastructure/company y electronic/infrastructure/company.
package company

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	companydomain "github.com/diegofxm/erp/internal/company/domain"
	"github.com/diegofxm/erp/internal/saas/domain"
)

type Adapter struct {
	repo companydomain.Repository
}

func New(repo companydomain.Repository) *Adapter {
	return &Adapter{repo: repo}
}

var _ domain.CompanyPort = (*Adapter)(nil)

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
