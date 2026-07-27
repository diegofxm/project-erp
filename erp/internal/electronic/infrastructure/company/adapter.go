// Package company implementa CompanyPort leyendo desde el repositorio de empresas del ERP.
package company

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	companydomain "github.com/diegofxm/erp/internal/company/domain"
	"github.com/diegofxm/erp/internal/electronic/domain"
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
	return mapToInfo(c), nil
}

func mapToInfo(c *companydomain.Company) *domain.CompanyInfo {
	info := &domain.CompanyInfo{
		ID:                          c.ID,
		NIT:                         c.NIT,
		CheckDigit:                  c.CheckDigit,
		BusinessName:                c.BusinessName,
		IdentificationTypeCode:      c.IdentificationTypeCode,
		DepartmentCode:              c.DepartmentCode,
		MunicipalityCode:            c.MunicipalityCode,
		AddressLine:                 c.AddressLine,
		Email:                       c.Email,
		Phone:                       c.Phone,
		Environment:                 domain.Environment(c.Environment),
		EntityTypeCode:              c.EntityTypeCode,
		TaxSchemeCode:               c.TaxSchemeCode,
		TaxSchemeName:               c.TaxSchemeName,
		LiabilityCodes:              c.LiabilityCodes,
		TaxRegimeCode:               c.TaxRegimeCode,
		IndustryClassificationCodes: c.IndustryClassificationCodes,
		MerchantRegistrationNumber:  c.MerchantRegistrationNumber,
		SoftwareID:                  c.SoftwareID,
		SoftwarePIN:                 c.SoftwarePIN,
		Certificate:                 c.Certificate,
		CertificatePassword:         c.CertificatePassword,
	}
	return info
}
