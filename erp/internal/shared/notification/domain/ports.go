package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CertificateInfo es la vista del certificado digital que necesita notification
// para decidir si avisar por vencimiento — no necesita el certificado en sí
// (bytes/contraseña), solo su metadata.
type CertificateInfo struct {
	HasCertificate bool
	ExpiresAt      *time.Time
}

// CompanyPort lee la empresa como dueña de un certificado digital DIAN.
type CompanyPort interface {
	GetCertificateInfo(ctx context.Context, companyID uuid.UUID) (CertificateInfo, error)
}

// RangeInfo es la vista de un rango de numeración que necesita notification —
// Status ya viene resuelto ("active"|"expired"|"exhausted"|"inactive") por el
// dominio de electronic (NumberingRange.Status()), notification no reimplementa
// esa regla de negocio.
type RangeInfo struct {
	ID                   uuid.UUID
	DianDocumentTypeCode string
	Prefix               string
	Status               string
	ValidTo              time.Time
}

// NumberingPort lee los rangos de numeración vigentes de una empresa.
type NumberingPort interface {
	ListByCompany(ctx context.Context, companyID uuid.UUID) ([]RangeInfo, error)
}
