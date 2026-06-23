package issuers

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Service centraliza la lógica de negocio de emisores/tenants.
type Service struct {
	repo      Repository
	validator CertificateValidator
}

// New crea el servicio de emisores. validator puede ser nil (no valida certificados) — ver
// CertificateValidator.
func New(repo Repository, validator CertificateValidator) *Service {
	return &Service{repo: repo, validator: validator}
}

// RegisterIssuer valida y persiste un nuevo emisor. SoftwareID/SoftwarePIN/Certificate son
// OPCIONALES aquí a propósito — el registro solo exige los datos que la DIAN pide del emisor
// mismo (NIT, razón social, ubicación, ambiente); software y certificado se completan después,
// independientemente, vía UpdateIssuer (PUT /issuers/me), en el orden en que el usuario los
// vaya consiguiendo. documents.Service exige que estén presentes recién al confirmar un
// documento (ErrIssuerNotReadyToIssue si faltan), nunca antes — ver
// docs/api-dian-architecture.md sección 9.25.
func (s *Service) RegisterIssuer(ctx context.Context, iss Issuer) (*Issuer, error) {
	applyDefaults(&iss)
	if err := validateIssuer(iss); err != nil {
		return nil, err
	}
	iss.IsActive = true
	return s.repo.Create(ctx, iss)
}

// UpdateIssuerRequest son los campos que se completan gradualmente después del registro
// inicial. Cada puntero nil significa "no tocar este campo" — a diferencia de
// customers/products.Update (reemplazo completo), esta es una actualización PARCIAL a
// propósito: el usuario va cargando software/PIN/certificado en el orden en que los consiga,
// sin tener que reenviar lo que ya configuró antes.
type UpdateIssuerRequest struct {
	SoftwareID          *string
	SoftwarePIN         *string
	Certificate         []byte // nil = no tocar
	CertificatePassword *string
}

// UpdateIssuer completa/reemplaza software/PIN/certificado de un emisor ya registrado. Un
// puntero no-nil con un valor vacío se rechaza (ErrEmptySoftwareID/ErrEmptySoftwarePIN/
// ErrEmptyCertificate) — casi siempre es un error de quien llama (omitir el campo, no
// mandarlo vacío), nunca una forma válida de "borrar" la credencial.
//
// Si esta llamada toca Certificate o CertificatePassword, y el emisor queda con AMBOS no
// vacíos después del merge, se valida que de verdad formen un .p12 legible (s.validator) —
// para fallar aquí con un error claro, no recién al confirmar un documento (ver
// docs/api-dian-architecture.md sección 9.26). Si todavía falta uno de los dos (ej. se sube
// el certificado pero la contraseña se va a completar después), la validación se omite a
// propósito: no sería justo rechazar una combinación que el usuario nunca pretendió que
// fuera completa todavía.
func (s *Service) UpdateIssuer(ctx context.Context, id uuid.UUID, req UpdateIssuerRequest) (*Issuer, error) {
	iss, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	touchedCertificate := req.Certificate != nil || req.CertificatePassword != nil

	if req.SoftwareID != nil {
		if strings.TrimSpace(*req.SoftwareID) == "" {
			return nil, ErrEmptySoftwareID
		}
		iss.SoftwareID = *req.SoftwareID
	}
	if req.SoftwarePIN != nil {
		if strings.TrimSpace(*req.SoftwarePIN) == "" {
			return nil, ErrEmptySoftwarePIN
		}
		iss.SoftwarePIN = *req.SoftwarePIN
	}
	if req.Certificate != nil {
		if len(req.Certificate) == 0 {
			return nil, ErrEmptyCertificate
		}
		iss.Certificate = req.Certificate
	}
	if req.CertificatePassword != nil {
		iss.CertificatePassword = *req.CertificatePassword
	}

	if touchedCertificate && s.validator != nil && len(iss.Certificate) > 0 && iss.CertificatePassword != "" {
		if err := s.validator(iss.Certificate, iss.CertificatePassword); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidCertificate, err)
		}
	}

	return s.repo.Update(ctx, *iss)
}

// applyDefaults completa los campos del Party que la mayoría de emisores no necesita
// personalizar — valores confirmados contra una factura real autorizada por la DIAN
// (cofacture/soap/realsend_test.go): TaxSchemeCode "ZZ" ("No aplica").
//
// EntityTypeCode y LiabilityCodes se agregaron el 2026-06-23 tras un rechazo real (StatusCode
// 99, "errores en campos mandatorios") al registrar un emisor persona natural (identificado
// por cédula, "13"): el default fijo "1" (Persona Jurídica) quedaba contradicho por una
// identificación personal, y LiabilityCodes se mandaba vacío del todo — a diferencia de
// applyCustomerDefaults (documents/service.go), que ya respaldaba al adquiriente con
// "R-99-PN" pero nunca se replicó para el emisor. Ver docs/api-dian-architecture.md sección
// 9.29.
func applyDefaults(iss *Issuer) {
	if iss.EntityTypeCode == "" {
		iss.EntityTypeCode = defaultEntityTypeCode(iss.IdentificationTypeCode)
	}
	if iss.TaxSchemeCode == "" {
		iss.TaxSchemeCode = "ZZ"
	}
	if iss.TaxSchemeName == "" {
		iss.TaxSchemeName = "No aplica"
	}
	if len(iss.LiabilityCodes) == 0 {
		iss.LiabilityCodes = []string{"R-99-PN"}
	}
}

// defaultEntityTypeCode deriva "1" (Persona Jurídica y asimiladas) o "2" (Persona Natural y
// asimiladas) del tipo de identificación — "31"/"47"/"50" son los únicos tipos de
// identificación tributaria (NIT, NIT de otro país, NIT de la DIAN); el resto
// (cédula/tarjeta de identidad/extranjería/pasaporte/NUIP) son identificaciones personales.
// La DIAN rechaza la combinación "Persona Jurídica" + identificación personal.
func defaultEntityTypeCode(identificationTypeCode string) string {
	switch identificationTypeCode {
	case "31", "47", "50":
		return "1"
	default:
		return "2"
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

// validateIssuer solo exige los datos que la DIAN pide del emisor mismo — NO exige
// SoftwareID/SoftwarePIN/Certificate, esos se completan después (ver RegisterIssuer).
func validateIssuer(iss Issuer) error {
	if strings.TrimSpace(iss.NIT) == "" {
		return ErrEmptyNIT
	}
	if strings.TrimSpace(iss.BusinessName) == "" {
		return ErrEmptyBusinessName
	}
	switch iss.Environment {
	case EnvironmentProduccion, EnvironmentHabilitacion:
	default:
		return ErrInvalidEnvironment
	}
	return nil
}
