package documents

import "github.com/diegofxm/cofacture/signer"

// ValidateCertificate confirma que certificate+password formen un .p12 (PKCS12) válido y
// parseable. documents es el único paquete de apidian que importa cofacture directamente
// (ver docs/apidian-architecture.md sección 4.1) — por eso esta función vive aquí, no en
// internal/issuers, aunque a quien sirve es a issuers.Service.UpdateIssuer (inyectada como
// issuers.CertificateValidator desde internal/api, ver sección 9.26).
func ValidateCertificate(certificate []byte, password string) error {
	_, _, err := signer.LoadPKCS12(certificate, password)
	return err
}
