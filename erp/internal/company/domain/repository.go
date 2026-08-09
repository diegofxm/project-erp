package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CredentialsParams agrupa los campos de credenciales para una actualización parcial.
// Solo se actualizan los grupos de campos cuyo "key" principal sea no vacío/nil.
type CredentialsParams struct {
	// Software FE — solo se actualiza si SoftwareID != ""
	SoftwareID  string
	SoftwarePIN string

	// Certificado — solo se actualiza si Certificate != nil
	Certificate          []byte
	CertificatePassword  string
	CertificateSubject   *string
	CertificateIssuerCN  *string
	CertificateExpiresAt *time.Time

	// Software NE — solo se actualiza si NeSoftwareID != ""
	NeSoftwareID  string
	NeSoftwarePIN string
}

type Repository interface {
	Save(ctx context.Context, c Company) (*Company, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Company, error)
	GetByNIT(ctx context.Context, nit string) (*Company, error)
	ListByIDs(ctx context.Context, ids []uuid.UUID) ([]Company, error)
	UpdateProfile(ctx context.Context, c Company) (*Company, error)
	UpdateCredentials(ctx context.Context, id uuid.UUID, p CredentialsParams) error
	ClearSoftware(ctx context.Context, id uuid.UUID) error
	ClearNeSoftware(ctx context.Context, id uuid.UUID) error
	ClearCertificate(ctx context.Context, id uuid.UUID) error
	UpdateLogo(ctx context.Context, id uuid.UUID, logo []byte, contentType string) error
	DeleteLogo(ctx context.Context, id uuid.UUID) error
	UpdateBrandColor(ctx context.Context, id uuid.UUID, color string) error
}
