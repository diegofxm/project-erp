package issuers

import (
	"time"

	"github.com/google/uuid"
)

// Environment es el ambiente DIAN en el que opera el emisor — mismos códigos que
// domain.Invoice.EnvironmentCode en cofacture ("1" producción, "2" habilitación).
type Environment string

const (
	EnvironmentProduccion   Environment = "1"
	EnvironmentHabilitacion Environment = "2"
)

// Issuer es la configuración mínima de un emisor/tenant necesaria para emitir documentos
// DIAN: identificación, ubicación, y credenciales de software/certificado.
//
// NO es un perfil de empresa completo (CRM) — eso vive en otro sistema si alguna vez se
// necesita. Ver docs/apidian-architecture.md sección 4.2 para la justificación completa.
type Issuer struct {
	ID                     uuid.UUID
	NIT                    string
	CheckDigit             string // dígito de verificación del NIT
	BusinessName           string // razón social
	TradeName              string // nombre comercial, opcional
	IdentificationTypeCode string // FK identification_types — código numérico DIAN, "31" para NIT (no "NIT")
	DepartmentCode         string // FK departments
	MunicipalityCode       string // FK municipalities
	// DepartmentName/MunicipalityName NO se guardan en la tabla issuers — vienen de un JOIN
	// contra los catálogos en cada lectura (GetByID/GetByNIT). La DIAN exige el nombre, no
	// solo el código, en cac:Address (StateName/CityName) — un rechazo real (StatusCode 99,
	// "errores en campos mandatorios") lo confirmó: el primer envío real a través del
	// orquestador completo (Fase 2.8) los mandaba vacíos. Quedan fuera de Create() a
	// propósito, para no duplicar el dato del catálogo (si la DIAN ajusta una descripción,
	// no hay que tocar issuers).
	DepartmentName   string
	MunicipalityName string
	AddressLine      string
	Email            string
	Phone            string
	Environment      Environment

	// Campos exigidos para construir el cac:AccountingSupplierParty/cac:Party del XML
	// (cofacture/domain.Party) — agregados en la Fase 2.5 al descubrir, construyendo el
	// primer Invoice real, que faltaban para reproducir exactamente la factura que la DIAN ya
	// autorizó en la Fase 1.7 (ver cofacture/soap/realsend_test.go).
	EntityTypeCode string   // domain.Party.EntityTypeCode — "1" en el caso real validado
	TaxSchemeCode  string   // FK tax_types — "ZZ" ("No aplica") en el caso real validado
	TaxSchemeName  string   // domain.Party.TaxSchemeName, acompaña a TaxSchemeCode
	LiabilityCodes []string // domain.Party.LiabilityCodes — responsabilidades fiscales, ej. "R-99-PN"
	// TaxRegimeCode es *string (no string) porque el FK a tax_regimes es opcional — un
	// catálogo sin código "no aplica" oficial (a diferencia de tax_types/ZZ), así que NULL es
	// la única forma de modelar "no aplica" sin violar el FK con una cadena vacía.
	TaxRegimeCode *string // FK tax_regimes — domain.Party.TaxRegimeCode (listName de TaxLevelCode)
	// IndustryClassificationCodes son los códigos CIIU (catálogo DANE, no DIAN — sin FK,
	// ver migración 000003_issuers). Máximo 4 (1 actividad principal + hasta 3 secundarias,
	// estructura real del RUT) — validado en Service.validateIssuer, no en el esquema.
	IndustryClassificationCodes []string
	MerchantRegistrationNumber  *string // domain.Party.MerchantRegistrationNumber, opcional

	// Credenciales DIAN. En este struct de dominio viajan en texto plano (el servicio y
	// internal/documents las necesitan así para usarlas con cofacture) — es
	// PostgresRepository quien las cifra antes de guardar y las descifra al leer (AES-256-GCM,
	// internal/cryptutil). SoftwareID no es secreto (es un identificador de registro ante la
	// DIAN, no una contraseña), por eso es el único de los cuatro que no se cifra.
	SoftwareID          string
	SoftwarePIN         string
	Certificate         []byte // contenido del .p12
	CertificatePassword string

	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
