package tax

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository define el contrato de persistencia del módulo tax.
type Repository interface {

	// ── F210 Renta ──────────────────────────────────────────────────────────────────────────
	GetIncomeTaxRate(ctx context.Context, year int) (*IncomeTaxRate, error)
	SetIncomeTaxRate(ctx context.Context, year, rateBP int) (*IncomeTaxRate, error)
	ListIncomeTaxRates(ctx context.Context) ([]*IncomeTaxRate, error)

	SaveIncomeTaxDeclaration(ctx context.Context, d IncomeTaxDeclaration) (*IncomeTaxDeclaration, error)
	GetIncomeTaxDeclarationByID(ctx context.Context, id uuid.UUID) (*IncomeTaxDeclaration, error)
	GetIncomeTaxDeclarationByYear(ctx context.Context, companyID uuid.UUID, year int) (*IncomeTaxDeclaration, error)
	ListIncomeTaxDeclarations(ctx context.Context, companyID uuid.UUID) ([]*IncomeTaxDeclaration, error)
	UpdateIncomeTaxStatus(ctx context.Context, id uuid.UUID, status DeclarationStatus, journalID *uuid.UUID, filedAt *time.Time) error

	// ── F220 Certificados de Retención ──────────────────────────────────────────────────────
	// QueryWithholdingsByNIT agrega los créditos a cuentas de retención del libro mayor,
	// agrupando por NIT del tercero y account_code para generar los certificados anuales.
	QueryWithholdingsByNIT(ctx context.Context, companyID uuid.UUID, from, to time.Time) ([]*WHByAccount, error)

	SaveCertificate(ctx context.Context, c WithholdingCertificate) (*WithholdingCertificate, error)
	GetCertificateByID(ctx context.Context, id uuid.UUID) (*WithholdingCertificate, error)
	ListCertificates(ctx context.Context, companyID uuid.UUID, year int) ([]*WithholdingCertificate, error)
	UpdateCertificateStatus(ctx context.Context, id uuid.UUID, status CertificateStatus, issuedAt *time.Time) error

	// ── F490 ICA por Municipio ───────────────────────────────────────────────────────────────
	SetIcaTariff(ctx context.Context, req IcaTariffRequest) (*IcaTariff, error)
	GetIcaTariff(ctx context.Context, municipalityCode, ciiuCode string, year int) (*IcaTariff, error)

	SaveIcaDeclaration(ctx context.Context, d IcaDeclaration) (*IcaDeclaration, error)
	GetIcaDeclarationByID(ctx context.Context, id uuid.UUID) (*IcaDeclaration, error)
	ListIcaDeclarations(ctx context.Context, companyID uuid.UUID) ([]*IcaDeclaration, error)
	UpdateIcaStatus(ctx context.Context, id uuid.UUID, status DeclarationStatus, journalID *uuid.UUID, filedAt *time.Time) error
}
