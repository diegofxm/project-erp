package issuers

// CertificateValidator confirma que certificate+password formen un .p12 (PKCS12) válido y
// parseable. internal/issuers no importa cofacture directamente — documents es el único
// paquete de api-dian que lo hace (ver docs/api-dian-architecture.md sección 4.1) — por eso
// esto es un puerto angosto, inyectado desde internal/api (documents.ValidateCertificate en
// producción), mismo patrón que documents.IssuerPort/NumberingPort/CustomerPort.
//
// nil es válido: significa "no validar" — lo usan los tests de este paquete que no necesitan
// un .p12 real para probar lógica de dominio que nunca llega a esta validación (valores
// vacíos, "no encontrado", actualización parcial sin tocar el certificado, etc.).
type CertificateValidator func(certificate []byte, password string) error
