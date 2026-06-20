// Package qr construye la URL del código QR que exige la representación gráfica de los
// documentos electrónicos DIAN (Anexo Técnico 1.9, sección 11.7.1).
package qr

const (
	habilitacionBaseURL = "https://catalogo-vpfe-hab.dian.gov.co/document/searchqr"
	produccionBaseURL   = "https://catalogo-vpfe.dian.gov.co/document/searchqr"
)

// URL construye la URL del QR a partir del CUFE (o CUDE) y el código de ambiente
// ("1" producción, "2" habilitación — el mismo valor que cbc:ProfileExecutionID). La DIAN
// usa dominios distintos por ambiente; confirmado contra dos facturas reales de producción
// y el texto del anexo técnico.
func URL(environmentCode, documentKey string) string {
	base := produccionBaseURL
	if environmentCode == "2" {
		base = habilitacionBaseURL
	}
	return base + "?documentkey=" + documentKey
}
