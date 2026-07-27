package domain

import "context"

type Format string

const (
	FormatHTML  Format = "html"
	FormatPDF   Format = "pdf"
	FormatExcel Format = "excel"
	FormatCSV   Format = "csv"
)

// RenderRequest agrupa lo que necesita el motor para generar un documento.
type RenderRequest struct {
	TemplateID string         // e.g. "payroll/payslip", "sales/invoice", "electronic/invoice_graphic"
	Data       map[string]any // variables del template
	Format     Format
}

// Renderer es el puerto del motor de documentos.
// Los módulos de negocio solo conocen esta interfaz.
type Renderer interface {
	Render(ctx context.Context, req RenderRequest) ([]byte, error)
}
