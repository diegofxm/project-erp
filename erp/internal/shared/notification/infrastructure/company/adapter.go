// Package company implementa notification/domain.CompanyPort leyendo desde el
// repositorio de empresas del ERP.
package company

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	companydomain "github.com/diegofxm/erp/internal/company/domain"
	"github.com/diegofxm/erp/internal/shared/notification/domain"
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

func (a *Adapter) GetCertificateInfo(ctx context.Context, companyID uuid.UUID) (domain.CertificateInfo, error) {
	c, err := a.repo.GetByID(ctx, companyID)
	if err != nil {
		return domain.CertificateInfo{}, fmt.Errorf("company adapter: %w", err)
	}
	return domain.CertificateInfo{
		HasCertificate: len(c.Certificate) > 0,
		ExpiresAt:      c.CertificateExpiresAt,
	}, nil
}
