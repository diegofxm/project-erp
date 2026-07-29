package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	pkcs12 "software.sslmate.com/src/go-pkcs12"

	"github.com/diegofxm/erp/internal/company/domain"
)

var ErrBadCertificate = errors.New("contraseña del certificado incorrecta o archivo inválido")

type UpdateCredentialsUseCase struct {
	repo domain.Repository
}

func NewUpdateCredentialsUseCase(repo domain.Repository) *UpdateCredentialsUseCase {
	return &UpdateCredentialsUseCase{repo: repo}
}

type UpdateCredentialsRequest struct {
	SoftwareID          string
	SoftwarePIN         string
	Certificate         []byte
	CertificatePassword string
	NeSoftwareID        string
	NeSoftwarePIN       string
}

func (uc *UpdateCredentialsUseCase) Execute(ctx context.Context, id uuid.UUID, req UpdateCredentialsRequest) error {
	params := domain.CredentialsParams{
		SoftwareID:          req.SoftwareID,
		SoftwarePIN:         req.SoftwarePIN,
		Certificate:         req.Certificate,
		CertificatePassword: req.CertificatePassword,
		NeSoftwareID:        req.NeSoftwareID,
		NeSoftwarePIN:       req.NeSoftwarePIN,
	}

	if len(req.Certificate) > 0 {
		// go-pkcs12 (fork activo de x/crypto/pkcs12) soporta RC2/3DES que usan
		// las CAs colombianas (Certicámara, etc.). DecodeChain extrae el leaf
		// certificate aunque el p12 traiga una cadena completa.
		_, cert, _, err := pkcs12.DecodeChain(req.Certificate, req.CertificatePassword)
		if err != nil {
			return ErrBadCertificate
		}
		subject := cert.Subject.CommonName
		issuerCN := cert.Issuer.CommonName
		expiresAt := cert.NotAfter
		params.CertificateSubject = &subject
		params.CertificateIssuerCN = &issuerCN
		params.CertificateExpiresAt = &expiresAt
	}

	return uc.repo.UpdateCredentials(ctx, id, params)
}
