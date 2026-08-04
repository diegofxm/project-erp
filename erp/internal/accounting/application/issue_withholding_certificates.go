package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/accounting/domain"
	purchasedomain "github.com/diegofxm/erp/internal/purchase/domain"
)

// IssueWithholdingCertificatesUseCase agrega las retenciones aplicadas a un proveedor en un
// año fiscal (dato que vive en purchase/) y emite un certificado por concepto — mismo patrón
// de import directo entre módulos que AddWithholdingUseCase usa en sentido inverso.
type IssueWithholdingCertificatesUseCase struct {
	purchaseWithholdings purchasedomain.WithholdingRepository
	certificates         domain.WithholdingCertificateRepository
}

func NewIssueWithholdingCertificatesUseCase(
	purchaseWithholdings purchasedomain.WithholdingRepository,
	certificates domain.WithholdingCertificateRepository,
) *IssueWithholdingCertificatesUseCase {
	return &IssueWithholdingCertificatesUseCase{purchaseWithholdings: purchaseWithholdings, certificates: certificates}
}

type IssueCertificatesRequest struct {
	SupplierID    uuid.UUID
	ThirdPartyNIT string
	FiscalYear    int
}

func (uc *IssueWithholdingCertificatesUseCase) Execute(ctx context.Context, companyID uuid.UUID, req IssueCertificatesRequest) ([]domain.WithholdingCertificate, error) {
	summary, err := uc.purchaseWithholdings.GetWithholdingSummary(ctx, companyID, req.SupplierID, req.FiscalYear)
	if err != nil {
		return nil, fmt.Errorf("obtener resumen de retenciones: %w", err)
	}
	if len(summary) == 0 {
		return nil, fmt.Errorf("no hay retenciones aplicadas a este proveedor en %d", req.FiscalYear)
	}

	out := make([]domain.WithholdingCertificate, 0, len(summary))
	for _, s := range summary {
		c, err := uc.certificates.Create(ctx, domain.WithholdingCertificate{
			CompanyID:     companyID,
			FiscalYear:    req.FiscalYear,
			ThirdPartyNIT: req.ThirdPartyNIT,
			ConceptCode:   s.ConceptCode,
			ConceptName:   s.ConceptName,
			WHType:        "RETEFUENTE",
			GrossAmount:   s.Base,
			TaxWithheld:   s.Amount,
			Status:        "issued",
		})
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, nil
}

func (uc *IssueWithholdingCertificatesUseCase) List(ctx context.Context, companyID uuid.UUID, year int) ([]domain.WithholdingCertificate, error) {
	return uc.certificates.List(ctx, companyID, year)
}
