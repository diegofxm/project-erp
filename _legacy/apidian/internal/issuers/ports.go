package issuers

import (
	"context"
	"time"
)

// CatalogPort valida LiabilityCodes contra el catálogo liability_codes — TEXT[], sin FK
// posible contra cada elemento (mismo motivo y mismo patrón que documents.CatalogPort).
// GetTaxTypeName resuelve TaxSchemeName a partir de TaxSchemeCode — el cliente ya no puede
// mandar el nombre, el servicio lo deriva del catálogo (ver docs/apidian-architecture.md).
type CatalogPort interface {
	IsValidLiabilityCode(ctx context.Context, code string) (bool, error)
	GetTaxTypeName(ctx context.Context, code string) (name string, found bool, err error)
}

// CertificateValidator confirma que certificate+password formen un .p12 (PKCS12) válido y
// parseable. internal/issuers no importa cofacture directamente — documents es el único
// paquete de apidian que lo hace (ver docs/apidian-architecture.md sección 4.1) — por eso
// esto es un puerto angosto, inyectado desde internal/api (documents.ValidateCertificate en
// producción), mismo patrón que documents.IssuerPort/NumberingPort/CustomerPort.
//
// nil es válido: significa "no validar" — lo usan los tests de este paquete que no necesitan
// un .p12 real para probar lógica de dominio que nunca llega a esta validación (valores
// vacíos, "no encontrado", actualización parcial sin tocar el certificado, etc.).
type CertificateValidator func(certificate []byte, password string) error

// CertificateParser extrae los metadatos visibles del .p12: nombre del titular (Subject
// CommonName), nombre de la CA emisora (Issuer CommonName) y fecha de vencimiento. Se inyecta
// desde internal/api (documents.ParseCertificate) para respetar la regla de que solo documents
// importa cofacture directamente.
//
// nil es válido: significa "no extraer metadatos" — los tests de este paquete lo usan cuando
// no necesitan un .p12 real.
type CertificateParser func(certificate []byte, password string) (subject, issuerCN string, expiresAt time.Time, err error)
