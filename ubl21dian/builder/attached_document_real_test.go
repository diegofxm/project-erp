package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/diegofxm/ubl21dian/domain"
)

// TestBuildInvoiceAttachedDocument_RealApplicationResponse cierra el círculo de las Fases
// 1.5/1.7/1.8: usa un ApplicationResponse real (decodificado de una respuesta real de
// GetStatusZip durante el desarrollo de la Fase 1.7, guardado en docs/reference) como
// contenido de ValidationResult, y confirma que BuildInvoiceAttachedDocument lo acepta sin
// errores. Se omite por defecto — depende de un archivo que no se commitea.
func TestBuildInvoiceAttachedDocument_RealApplicationResponse(t *testing.T) {
	dir := os.Getenv("UBL21DIAN_TEST_FIXTURES_DIR")
	if dir == "" {
		t.Skip("UBL21DIAN_TEST_FIXTURES_DIR no configurado, se omite")
	}
	path := filepath.Join(dir, "outputs", "_application_response.xml")
	appResponseXML, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("no se encontró %s (correr primero la Fase 1.7/1.8 para generarlo): %v", path, err)
	}

	ad := sampleAttachedDocument()
	ad.ValidationResults = []domain.ValidationResult{
		{
			LineID:                 "1",
			DocumentID:             "SETP990068706",
			DocumentCUFE:           "853657dcf2841c55c04338b24cc4db9dfbf87042f1ce1798e53f7b1f0502d00df9bd3f371dea47b02766424976d60ba2",
			DocumentHashType:       "CUFE-SHA384",
			DocumentIssueDate:      "2026-06-20",
			ApplicationResponseXML: string(appResponseXML),
			ValidatorID:            "Unidad Especial Dirección de Impuestos y Aduanas Nacionales",
			ValidationResultCode:   "00",
			ValidationDate:         "2026-06-20",
			ValidationTime:         "07:30:00-05:00",
		},
	}

	doc, err := BuildInvoiceAttachedDocument(ad)
	if err != nil {
		t.Fatalf("BuildInvoiceAttachedDocument: %v", err)
	}
	out, err := doc.WriteToString()
	if err != nil {
		t.Fatalf("WriteToString: %v", err)
	}
	if len(out) < len(appResponseXML) {
		t.Error("la salida debería al menos contener el ApplicationResponse embebido completo")
	}
}
