package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/electronic/domain"
)

var ErrCompanyNotReadyForDian = errors.New("la empresa no tiene certificado y/o software_id configurados")

type GetDianRangesUseCase struct {
	company domain.CompanyPort
	fetcher domain.DianRangesFetcherPort
}

func NewGetDianRangesUseCase(company domain.CompanyPort, fetcher domain.DianRangesFetcherPort) *GetDianRangesUseCase {
	return &GetDianRangesUseCase{company: company, fetcher: fetcher}
}

func (uc *GetDianRangesUseCase) Execute(ctx context.Context, companyID uuid.UUID) ([]domain.DianRange, error) {
	c, err := uc.company.GetCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if len(c.Certificate) == 0 || c.SoftwareID == "" {
		return nil, ErrCompanyNotReadyForDian
	}
	return uc.fetcher.GetNumberingRanges(c.NIT, c.SoftwareID, c.Certificate, c.CertificatePassword, string(c.Environment))
}
