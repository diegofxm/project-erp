package pdf

// Format identifica el layout de la representación gráfica. El valor por defecto
// es FullA4 (A4 completo, una factura por hoja). HalfA4 imprime dos copias en una
// sola hoja A4 separadas por línea de corte — útil para negocios que entregan
// original + copia física.
type Format string

const (
	FormatFullA4  Format = "full_a4"
	FormatHalfA4  Format = "half_a4"
)

// ParseFormat convierte un string externo (query param, body JSON) al tipo Format.
// Devuelve FormatFullA4 para cualquier valor desconocido o vacío.
func ParseFormat(s string) Format {
	switch Format(s) {
	case FormatHalfA4:
		return FormatHalfA4
	default:
		return FormatFullA4
	}
}

// BuildPDF despacha al builder correcto según el formato solicitado.
func BuildPDF(format Format, in InvoiceInput) ([]byte, error) {
	if format == FormatHalfA4 {
		return BuildHalfPagePDF(in)
	}
	return BuildInvoicePDF(in)
}
