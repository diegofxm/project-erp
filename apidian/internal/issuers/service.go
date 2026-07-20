package issuers

import (
	"context"
	"fmt"
	"strings"

	"github.com/diegofxm/apidian/internal/nit"
	"github.com/google/uuid"
)

// Service centraliza la lógica de negocio de emisores/tenants.
type Service struct {
	repo      Repository
	validator CertificateValidator
	parser    CertificateParser
	catalogs  CatalogPort
}

// New crea el servicio de emisores. validator y parser pueden ser nil (no valida/extrae
// metadatos del certificado) — ver CertificateValidator y CertificateParser.
func New(repo Repository, validator CertificateValidator, parser CertificateParser, catalogPort CatalogPort) *Service {
	return &Service{repo: repo, validator: validator, parser: parser, catalogs: catalogPort}
}

// RegisterIssuer valida y persiste un nuevo emisor. SoftwareID/SoftwarePIN/Certificate son
// OPCIONALES aquí a propósito — el registro solo exige los datos que la DIAN pide del emisor
// mismo (NIT, razón social, ubicación, ambiente); software y certificado se completan después,
// independientemente, vía UpdateIssuer (PUT /issuers/me), en el orden en que el usuario los
// vaya consiguiendo. documents.Service exige que estén presentes recién al confirmar un
// documento (ErrIssuerNotReadyToIssue si faltan), nunca antes — ver
// docs/apidian-architecture.md sección 9.25.
func (s *Service) RegisterIssuer(ctx context.Context, iss Issuer) (*Issuer, error) {
	applyDefaults(&iss)
	if err := s.resolveTaxSchemeName(ctx, &iss); err != nil {
		return nil, err
	}
	if err := s.validateIssuer(ctx, iss); err != nil {
		return nil, err
	}
	if err := deriveCheckDigit(&iss); err != nil {
		return nil, err
	}
	iss.IsActive = true
	return s.repo.Create(ctx, iss)
}

// resolveTaxSchemeName deriva TaxSchemeName del catálogo tax_types a partir de TaxSchemeCode
// — el cliente ya no puede mandar el nombre (ver createIssuerRequest), así ningún consumidor
// de la API puede guardar un código y un nombre que no correspondan entre sí. TaxSchemeCode
// nunca está vacío aquí: applyDefaults ya lo completó con "ZZ" si hacía falta.
func (s *Service) resolveTaxSchemeName(ctx context.Context, iss *Issuer) error {
	name, found, err := s.catalogs.GetTaxTypeName(ctx, iss.TaxSchemeCode)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: %q", ErrInvalidTaxSchemeCode, iss.TaxSchemeCode)
	}
	iss.TaxSchemeName = name
	return nil
}

// UpdateIssuerRequest son los campos que se completan gradualmente después del registro
// inicial. Cada puntero nil significa "no tocar este campo" — a diferencia de
// customers/products.Update (reemplazo completo), esta es una actualización PARCIAL a
// propósito: el usuario va cargando software/PIN/certificado en el orden en que los consiga,
// sin tener que reenviar lo que ya configuró antes.
type UpdateIssuerRequest struct {
	// FE y DS comparten el mismo software (el portal DIAN reutiliza el software de FE para DS)
	SoftwareID  *string
	SoftwarePIN *string
	// NE (Nómina Electrónica) — software independiente en el portal DIAN
	NeSoftwareID  *string
	NeSoftwarePIN *string
	Certificate         []byte  // nil = no tocar; compartido entre FE/DS/NE
	CertificatePassword *string
	Logo            []byte  // nil = no tocar
	LogoContentType *string // "png"/"jpg"/"jpeg"
}

// UpdateIssuer completa/reemplaza software/PIN/certificado de un emisor ya registrado. Un
// puntero no-nil con un valor vacío se rechaza (ErrEmptySoftwareID/ErrEmptySoftwarePIN/
// ErrEmptyCertificate) — casi siempre es un error de quien llama (omitir el campo, no
// mandarlo vacío), nunca una forma válida de "borrar" la credencial.
//
// Si esta llamada toca Certificate o CertificatePassword, y el emisor queda con AMBOS no
// vacíos después del merge, se valida que de verdad formen un .p12 legible (s.validator) —
// para fallar aquí con un error claro, no recién al confirmar un documento (ver
// docs/apidian-architecture.md sección 9.26). Si todavía falta uno de los dos (ej. se sube
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
	if req.NeSoftwareID != nil {
		if strings.TrimSpace(*req.NeSoftwareID) == "" {
			return nil, ErrEmptyNeSoftwareID
		}
		iss.NeSoftwareID = *req.NeSoftwareID
	}
	if req.NeSoftwarePIN != nil {
		if strings.TrimSpace(*req.NeSoftwarePIN) == "" {
			return nil, ErrEmptyNeSoftwarePIN
		}
		iss.NeSoftwarePIN = *req.NeSoftwarePIN
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

	if req.Logo != nil {
		if len(req.Logo) == 0 {
			return nil, ErrEmptyLogo
		}
		iss.Logo = req.Logo
	}
	if req.LogoContentType != nil {
		switch *req.LogoContentType {
		case "png", "jpg", "jpeg":
			iss.LogoContentType = *req.LogoContentType
		default:
			return nil, ErrInvalidLogoContentType
		}
	}
	result, err := s.repo.Update(ctx, *iss)
	if err != nil {
		return nil, err
	}
	s.enrichCertMetadata(result)
	return result, nil
}

// UpdateProfileRequest contiene los campos de perfil editables después del registro. Los
// secretos (software/cert) NO están aquí — se modifican exclusivamente por UpdateIssuer.
// Todos los campos de texto son requeridos en la llamada (no punteros): el handler del API
// siempre los envía completos desde el formulario — nil no ocurre en la práctica.
type UpdateProfileRequest struct {
	BusinessName                string
	TradeName                   string      // "" = borrar nombre comercial
	DepartmentCode              string
	MunicipalityCode            string
	AddressLine                 string
	Email                       string
	Phone                       string      // "" = borrar teléfono
	EntityTypeCode              string
	TaxSchemeCode               string
	LiabilityCodes              []string    // slice vacío = borrar todos
	TaxRegimeCode               *string     // nil = borrar; "code" = asignar
	IndustryClassificationCodes []string
	MerchantRegistrationNumber  *string     // nil = borrar; "XYZ" = asignar
	Environment                 Environment // "1" producción, "2" habilitación
}

// UpdateProfile actualiza los campos de perfil del emisor — razón social, dirección, datos
// fiscales y ambiente DIAN. NIT/tipo_identificación son inmutables. Los secretos (software/cert)
// no se tocan. TaxSchemeName se re-deriva del catálogo a partir de TaxSchemeCode para que nombre
// y código nunca queden desincronizados.
func (s *Service) UpdateProfile(ctx context.Context, id uuid.UUID, req UpdateProfileRequest) (*Issuer, error) {
	if strings.TrimSpace(req.BusinessName) == "" {
		return nil, ErrEmptyBusinessName
	}
	if len(req.IndustryClassificationCodes) > 4 {
		return nil, ErrTooManyIndustryClassificationCodes
	}
	for _, code := range req.LiabilityCodes {
		ok, err := s.catalogs.IsValidLiabilityCode(ctx, code)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrInvalidLiabilityCode, code)
		}
	}

	iss, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	iss.BusinessName = req.BusinessName
	iss.TradeName = req.TradeName
	iss.DepartmentCode = req.DepartmentCode
	iss.MunicipalityCode = req.MunicipalityCode
	iss.AddressLine = req.AddressLine
	iss.Email = req.Email
	iss.Phone = req.Phone
	iss.EntityTypeCode = req.EntityTypeCode
	iss.TaxSchemeCode = req.TaxSchemeCode
	iss.LiabilityCodes = req.LiabilityCodes
	iss.TaxRegimeCode = req.TaxRegimeCode
	iss.IndustryClassificationCodes = req.IndustryClassificationCodes
	iss.MerchantRegistrationNumber = req.MerchantRegistrationNumber
	if req.Environment == EnvironmentProduccion || req.Environment == EnvironmentHabilitacion {
		iss.Environment = req.Environment
	}

	if err := s.resolveTaxSchemeName(ctx, iss); err != nil {
		return nil, err
	}

	result, err := s.repo.UpdateProfile(ctx, *iss)
	if err != nil {
		return nil, err
	}
	s.enrichCertMetadata(result)
	return result, nil
}

// DeleteLogo quita el logo del emisor — UpdateIssuer no puede expresar "bórralo": ahí
// Logo nil ya significa "no tocar" (ver UpdateIssuerRequest), así que hace falta esta
// operación separada en vez de forzar un caso especial dentro de ese merge parcial.
func (s *Service) DeleteLogo(ctx context.Context, id uuid.UUID) (*Issuer, error) {
	iss, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	iss.Logo = nil
	iss.LogoContentType = ""
	result, err := s.repo.Update(ctx, *iss)
	if err != nil {
		return nil, err
	}
	s.enrichCertMetadata(result)
	return result, nil
}

// ClearSoftware borra software_id y software_pin del emisor (FE). El certificado no se toca.
func (s *Service) ClearSoftware(ctx context.Context, id uuid.UUID) (*Issuer, error) {
	iss, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	iss.SoftwareID = ""
	iss.SoftwarePIN = ""
	result, err := s.repo.Update(ctx, *iss)
	if err != nil {
		return nil, err
	}
	s.enrichCertMetadata(result)
	return result, nil
}

// ClearNeSoftware borra ne_software_id y ne_software_pin del emisor (NE). No toca FE ni certificado.
func (s *Service) ClearNeSoftware(ctx context.Context, id uuid.UUID) (*Issuer, error) {
	iss, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	iss.NeSoftwareID = ""
	iss.NeSoftwarePIN = ""
	result, err := s.repo.Update(ctx, *iss)
	if err != nil {
		return nil, err
	}
	s.enrichCertMetadata(result)
	return result, nil
}

// ClearCertificate borra el certificado y su contraseña. El software no se toca.
func (s *Service) ClearCertificate(ctx context.Context, id uuid.UUID) (*Issuer, error) {
	iss, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	iss.Certificate = nil
	iss.CertificatePassword = ""
	result, err := s.repo.Update(ctx, *iss)
	if err != nil {
		return nil, err
	}
	// No hay certificado — los campos derivados quedan vacíos (zero values).
	return result, nil
}

// enrichCertMetadata popula CertificateSubject/CertificateIssuerCN/CertificateExpiresAt en
// memoria a partir del .p12 ya descifrado. Los metadatos no se guardan en la base de datos —
// el certificado cifrado YA está almacenado; parsearlo aquí garantiza que los metadatos
// siempre sean frescos y evita tener que sincronizar columnas derivadas.
func (s *Service) enrichCertMetadata(iss *Issuer) {
	if s.parser == nil || len(iss.Certificate) == 0 || iss.CertificatePassword == "" {
		return
	}
	subject, issuerCN, expiresAt, err := s.parser(iss.Certificate, iss.CertificatePassword)
	if err != nil {
		return
	}
	iss.CertificateSubject = subject
	iss.CertificateIssuerCN = issuerCN
	iss.CertificateExpiresAt = &expiresAt
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
// "R-99-PN" pero nunca se replicó para el emisor. Ver docs/apidian-architecture.md sección
// 9.29.
func applyDefaults(iss *Issuer) {
	if iss.EntityTypeCode == "" {
		iss.EntityTypeCode = defaultEntityTypeCode(iss.IdentificationTypeCode)
	}
	if iss.TaxSchemeCode == "" {
		iss.TaxSchemeCode = "ZZ"
	}
	if len(iss.LiabilityCodes) == 0 {
		iss.LiabilityCodes = []string{"R-99-PN"}
	}
}

// deriveCheckDigit calcula el dígito de verificación módulo 11 cuando la identificación es un
// NIT ("31" — ver domain.Identification.VerificationCode en cofacture/domain/types.go, cuyo
// propio comentario dice que solo aplica a ese tipo, no a "47"/"50" aunque
// defaultEntityTypeCode los trate igual para efectos de persona jurídica). El cliente ya no
// puede mandar un dígito que no corresponda al NIT, se deriva igual que TaxSchemeName.
func deriveCheckDigit(iss *Issuer) error {
	if iss.IdentificationTypeCode != "31" {
		return nil
	}
	digit, err := nit.ComputeCheckDigit(iss.NIT)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidIdentificationNumber, err)
	}
	iss.CheckDigit = digit
	return nil
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

// GetIssuer devuelve un emisor por ID con metadatos del certificado ya derivados en memoria.
func (s *Service) GetIssuer(ctx context.Context, id uuid.UUID) (*Issuer, error) {
	iss, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.enrichCertMetadata(iss)
	return iss, nil
}

// GetIssuerByNIT devuelve un emisor por NIT con metadatos del certificado ya derivados.
func (s *Service) GetIssuerByNIT(ctx context.Context, nit string) (*Issuer, error) {
	iss, err := s.repo.GetByNIT(ctx, nit)
	if err != nil {
		return nil, err
	}
	s.enrichCertMetadata(iss)
	return iss, nil
}

// validateIssuer solo exige los datos que la DIAN pide del emisor mismo — NO exige
// SoftwareID/SoftwarePIN/Certificate, esos se completan después (ver RegisterIssuer). Es un
// método de Service (no función libre) porque valida LiabilityCodes contra el catálogo en
// Postgres — liability_codes es TEXT[], sin FK posible contra cada elemento (ver CatalogPort
// en ports.go).
func (s *Service) validateIssuer(ctx context.Context, iss Issuer) error {
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
	if len(iss.IndustryClassificationCodes) > 4 {
		return ErrTooManyIndustryClassificationCodes
	}
	for _, code := range iss.LiabilityCodes {
		ok, err := s.catalogs.IsValidLiabilityCode(ctx, code)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: %q", ErrInvalidLiabilityCode, code)
		}
	}
	return nil
}
