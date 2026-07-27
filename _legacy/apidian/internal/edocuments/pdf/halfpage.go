package pdf

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
)

//go:embed halfpage.html
var halfpageTmpl string

var halfPageTemplate = template.Must(template.New("halfpage").Parse(halfpageTmpl))

// halfPageData contiene las dos copias que se imprimen en una hoja A4.
type halfPageData struct {
	Copy1 halfPageSlip
	Copy2 halfPageSlip
}

// halfPageSlip promueve todos los campos de docTemplateData más el CopyLabel propio de cada copia.
type halfPageSlip struct {
	docTemplateData
	CopyLabel string
}

// BuildHalfPagePDF renderiza dos copias del documento en una sola hoja A4 separadas por línea
// de corte — Copy1 = "Original · Comercio", Copy2 = "Copia · Cliente".
func BuildHalfPagePDF(in InvoiceInput) ([]byte, error) {
	base, err := buildTemplateData(in)
	if err != nil {
		return nil, err
	}
	data := halfPageData{
		Copy1: halfPageSlip{docTemplateData: base, CopyLabel: "Original · Comercio"},
		Copy2: halfPageSlip{docTemplateData: base, CopyLabel: "Copia · Cliente"},
	}
	var buf bytes.Buffer
	if err := halfPageTemplate.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("pdf: halfpage template: %w", err)
	}
	return htmlToPDF(buf.String())
}
