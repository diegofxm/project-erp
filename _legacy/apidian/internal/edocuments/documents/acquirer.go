package documents

import (
	"context"
	"fmt"

	"github.com/diegofxm/cofacture/signer"
	"github.com/diegofxm/cofacture/soap"
	"github.com/google/uuid"
)

// AcquirerInfo es el resultado de VerifyAcquirer — Found en false (con ReceiverName/
// ReceiverEmail vacíos) es el resultado normal y esperado para la mayoría de cédulas, no un
// error: GetAcquirer solo consulta el registro de intercambio/notificación de la DIAN, no un
// RUT completo (ver docs/apidian-architecture.md sección 9.41).
type AcquirerInfo struct {
	Found         bool
	ReceiverName  string
	ReceiverEmail string
}

// VerifyAcquirer es una ayuda OPCIONAL al capturar un NIT — nunca bloqueante, nunca se llama
// automáticamente. Exige que el emisor ya tenga certificado configurado (mismo
// ErrIssuerNotReadyToIssue que ya exige claimAndLoadCert antes de confirmar un documento) para
// poder firmar la petición WS-Security — sin eso no hay forma de llamar a la DIAN.
func (s *Service) VerifyAcquirer(ctx context.Context, issuerID uuid.UUID, identificationTypeCode, identificationNumber string) (*AcquirerInfo, error) {
	iss, err := s.issuers.GetIssuer(ctx, issuerID)
	if err != nil {
		return nil, err
	}
	if len(iss.Certificate) == 0 {
		return nil, ErrIssuerNotReadyToIssue
	}

	cert, key, err := signer.LoadPKCS12(iss.Certificate, iss.CertificatePassword)
	if err != nil {
		return nil, fmt.Errorf("cargar certificado del emisor: %w", err)
	}

	// HabilitacionURL fijo a propósito, igual que claimAndLoadCert/sendAndUpdate — el proyecto
	// no tiene todavía selección producción/habilitación en ningún lado, no se agrega aquí
	// tampoco.
	client := soap.New(soap.HabilitacionURL, cert, key)
	resp, err := client.GetAcquirer(identificationTypeCode, identificationNumber)
	if err != nil {
		return nil, fmt.Errorf("consultar GetAcquirer: %w", err)
	}

	return &AcquirerInfo{
		Found:         resp.ReceiverName != "" || resp.ReceiverEmail != "",
		ReceiverName:  resp.ReceiverName,
		ReceiverEmail: resp.ReceiverEmail,
	}, nil
}
