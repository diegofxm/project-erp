package issuers

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

// Service centraliza la lógica de negocio de emisores/tenants.
type Service struct {
	repo Repository
}

// New crea el servicio de emisores.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// RegisterIssuer valida y persiste un nuevo emisor.
func (s *Service) RegisterIssuer(ctx context.Context, iss Issuer) (*Issuer, error) {
	applyDefaults(&iss)
	if err := validateIssuer(iss); err != nil {
		return nil, err
	}
	iss.IsActive = true
	return s.repo.Create(ctx, iss)
}

// applyDefaults completa los campos del Party que la mayoría de emisores no necesita
// personalizar — valores confirmados contra una factura real autorizada por la DIAN
// (ubl21dian/soap/realsend_test.go): EntityTypeCode "1", TaxSchemeCode "ZZ" ("No aplica").
func applyDefaults(iss *Issuer) {
	if iss.EntityTypeCode == "" {
		iss.EntityTypeCode = "1"
	}
	if iss.TaxSchemeCode == "" {
		iss.TaxSchemeCode = "ZZ"
	}
	if iss.TaxSchemeName == "" {
		iss.TaxSchemeName = "No aplica"
	}
}

// GetIssuer devuelve un emisor por ID.
func (s *Service) GetIssuer(ctx context.Context, id uuid.UUID) (*Issuer, error) {
	return s.repo.GetByID(ctx, id)
}

// GetIssuerByNIT devuelve un emisor por NIT.
func (s *Service) GetIssuerByNIT(ctx context.Context, nit string) (*Issuer, error) {
	return s.repo.GetByNIT(ctx, nit)
}

func validateIssuer(iss Issuer) error {
	if strings.TrimSpace(iss.NIT) == "" {
		return ErrEmptyNIT
	}
	if strings.TrimSpace(iss.BusinessName) == "" {
		return ErrEmptyBusinessName
	}
	if strings.TrimSpace(iss.SoftwareID) == "" {
		return ErrEmptySoftwareID
	}
	if strings.TrimSpace(iss.SoftwarePIN) == "" {
		return ErrEmptySoftwarePIN
	}
	if len(iss.Certificate) == 0 {
		return ErrEmptyCertificate
	}
	switch iss.Environment {
	case EnvironmentProduccion, EnvironmentHabilitacion:
	default:
		return ErrInvalidEnvironment
	}
	return nil
}
