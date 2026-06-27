// Package pdf construye la representación gráfica (PDF) de una Factura Electrónica — siempre
// en memoria, nunca a disco (ver docs/apidian-architecture.md sección 9.39). Deliberadamente
// no conoce los tipos de internal/documents/internal/issuers ni de cofacture: recibe un
// InvoiceInput plano, mismo principio de puertos angostos que el resto de apidian — quien
// orquesta el mapeo es documents.Service, igual que ya lo hace para construir el XML con
// cofacture.
package pdf

import (
	"fmt"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/code"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/list"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

var (
	colorTextPrimary   = &props.Color{Red: 60, Green: 60, Blue: 60}
	colorTextSecondary = &props.Color{Red: 110, Green: 110, Blue: 110}
	colorBorder        = &props.Color{Red: 200, Green: 200, Blue: 200}
	colorDraft         = &props.Color{Red: 180, Green: 90, Blue: 0}
	colorAccent        = &props.Color{Red: 30, Green: 70, Blue: 130}
)

// InvoiceLine es una línea ya calculada (precio×cantidad, impuesto ya resuelto) — el cálculo
// real vive en documents.Service.linesFromInput, este paquete solo la muestra. Implementa
// list.Listable para que la tabla de items escale sola a varias páginas si hace falta (algo
// que la versión a mano con gofpdf no tenía).
type InvoiceLine struct {
	ItemCode       string
	Description    string
	UnitCode       string
	Quantity       float64
	UnitPriceCents int64
	TaxPercent     float64 // 0 si la línea no lleva impuesto
	TotalCents     int64   // línea + su impuesto, ya sumado
}

func (InvoiceLine) GetHeader() core.Row {
	return row.New(7).Add(
		text.NewCol(1, "No.", headerCellProps(align.Center)),
		text.NewCol(2, "Código", headerCellProps(align.Left)),
		text.NewCol(4, "Descripción", headerCellProps(align.Left)),
		text.NewCol(1, "U/M", headerCellProps(align.Center)),
		text.NewCol(1, "Cant.", headerCellProps(align.Right)),
		text.NewCol(2, "Precio Unit.", headerCellProps(align.Right)),
		text.NewCol(1, "IVA", headerCellProps(align.Right)),
	).WithStyle(&props.Cell{BackgroundColor: &props.Color{Red: 240, Green: 240, Blue: 240}})
}

func (l InvoiceLine) GetContent(i int) core.Row {
	tax := "—"
	if l.TaxPercent > 0 {
		tax = fmt.Sprintf("%.0f%%", l.TaxPercent)
	}
	return row.New(8).Add(
		text.NewCol(1, fmt.Sprintf("%d", i+1), bodyCellProps(align.Center)),
		text.NewCol(2, l.ItemCode, bodyCellProps(align.Left)),
		text.NewCol(4, l.Description, bodyCellProps(align.Left)),
		text.NewCol(1, l.UnitCode, bodyCellProps(align.Center)),
		text.NewCol(1, formatQuantity(l.Quantity), bodyCellProps(align.Right)),
		text.NewCol(2, formatCOP(l.UnitPriceCents), bodyCellProps(align.Right)),
		text.NewCol(1, tax, bodyCellProps(align.Right)),
	).WithStyle(&props.Cell{BorderType: border.Bottom, BorderColor: colorBorder})
}

// InvoiceTax es el desglose de impuestos por tipo (mismo agregado que
// documents.aggregateTaxes calcula para el XML) — se recibe ya calculado, no se recalcula acá.
type InvoiceTax struct {
	TypeName           string
	Percent            float64
	TaxableAmountCents int64
	TaxAmountCents     int64
}

// InvoiceInput es todo lo que BuildInvoicePDF necesita — ningún tipo de internal/documents,
// internal/issuers ni cofacture/domain aquí, ver el comentario del paquete.
type InvoiceInput struct {
	// Emisor.
	IssuerBusinessName string
	IssuerNIT          string
	IssuerCheckDigit   string
	IssuerAddressLine  string
	IssuerCityName     string
	IssuerStateName    string
	IssuerPhone        string
	IssuerEmail        string
	IssuerLogo         []byte         // nil = sin logo
	IssuerLogoExt      extension.Type // vacío si IssuerLogo es nil

	// Estado del documento — IsDraft oculta CUFE/QR/número real y los reemplaza por avisos de
	// "borrador" (ver docs/apidian-architecture.md sección 9.39).
	IsDraft   bool
	Prefix    string
	Number    int64
	CUFE      string
	QRURL     string
	IssueDate string // ya formateada ("2026-06-26 15:30:00"), vacío si IsDraft

	CurrencyCode string

	// Cliente.
	CustomerName               string
	CustomerIdentificationType string // "Cédula de Ciudadanía"/"NIT"/... ya resuelto del catálogo
	CustomerIdentification     string
	CustomerAddressLine        string
	CustomerPhone              string
	CustomerEmail              string

	// Forma/medio de pago, ya resueltos a texto legible.
	PaymentTermName   string
	PaymentMethodName string
	PaymentDueDate    string // vacío si no es a crédito

	Lines []InvoiceLine
	Taxes []InvoiceTax

	LineExtensionCents int64
	TaxAmountCents     int64
	PayableCents       int64

	Note string

	// Pie de página — rango de numeración autorizado. RangeTo es *int64 (no int64) porque nil
	// es un valor real y distinto de "hasta el número 0" — numbering.NumberingRange.RangeTo ya
	// es nil cuando el tipo de documento no tiene un tope impuesto por la DIAN (ver
	// internal/numbering/model.go); mostrar "hasta {prefix}0" ahí sería un dato inventado.
	ResolutionNumber string
	RangePrefix      string
	RangeFrom        int64
	RangeTo          *int64
	ValidFrom        string
	ValidTo          string
}

// BuildInvoicePDF construye la representación gráfica en memoria — nunca escribe a disco,
// devuelve los bytes vía document.GetBytes() (ver docs/apidian-architecture.md sección 9.39).
func BuildInvoicePDF(in InvoiceInput) ([]byte, error) {
	cfg := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithLeftMargin(10).
		WithTopMargin(10).
		WithRightMargin(10).
		WithBottomMargin(10).
		Build()
	m := maroto.New(cfg)

	buildHeader(m, in)
	buildStatusBar(m, in)
	buildPartiesSection(m, in)
	buildItemsTable(m, in)
	buildTotals(m, in)
	buildTaxBreakdown(m, in)
	buildAmountInWords(m, in)
	buildFooter(m, in)

	document, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("pdf: generar documento: %w", err)
	}
	return document.GetBytes(), nil
}

func buildHeader(m core.Maroto, in InvoiceInput) {
	logoCol := col.New(2)
	if len(in.IssuerLogo) > 0 {
		logoCol = image.NewFromBytesCol(2, in.IssuerLogo, in.IssuerLogoExt, props.Rect{Center: true, Percent: 90})
	}

	infoCol := col.New(7).Add(
		text.New(in.IssuerBusinessName, infoLineProps(0, fontstyle.Bold, 10, colorTextPrimary)),
		text.New(fmt.Sprintf("NIT: %s-%s", in.IssuerNIT, in.IssuerCheckDigit), infoLineProps(4, fontstyle.Bold, 9.5, colorTextPrimary)),
		text.New(in.IssuerAddressLine, infoLineProps(8, fontstyle.Normal, 8.5, colorTextSecondary)),
		text.New(fmt.Sprintf("%s, %s", in.IssuerCityName, in.IssuerStateName), infoLineProps(11.5, fontstyle.Normal, 8.5, colorTextSecondary)),
		text.New(fmt.Sprintf("Tel: %s   %s", in.IssuerPhone, in.IssuerEmail), infoLineProps(15, fontstyle.Normal, 8.5, colorTextSecondary)),
	)

	qrCol := col.New(3).WithStyle(&props.Cell{BorderType: border.Full, BorderColor: colorBorder})
	if !in.IsDraft && in.QRURL != "" {
		qrCol = qrCol.Add(code.NewQr(in.QRURL, props.Rect{Center: true, Percent: 90}))
	} else {
		qrCol = qrCol.Add(text.New("QR disponible al confirmar", props.Text{
			Top: 10, Size: 7.5, Align: align.Center, Color: colorTextSecondary,
		}))
	}

	m.AddRows(row.New(28).Add(logoCol, infoCol, qrCol))
}

func buildStatusBar(m core.Maroto, in InvoiceInput) {
	if in.IsDraft {
		m.AddRows(row.New(6).Add(
			col.New(12).Add(text.New("BORRADOR — PENDIENTE DE CONFIRMACIÓN ANTE LA DIAN", props.Text{
				Top: 1.5, Size: 8, Align: align.Center, Style: fontstyle.Bold, Color: colorDraft,
			})).WithStyle(&props.Cell{BorderType: border.Full, BorderColor: colorDraft}),
		))
		return
	}
	m.AddRows(row.New(6).Add(
		col.New(12).Add(text.New("CUFE: "+in.CUFE, props.Text{
			Top: 1.5, Size: 6.5, Align: align.Left, Color: colorTextSecondary,
		})).WithStyle(&props.Cell{BorderType: border.Full, BorderColor: colorBorder}),
	))
}

func buildPartiesSection(m core.Maroto, in InvoiceInput) {
	clientCol := col.New(6).Add(
		text.New("Cliente: "+in.CustomerName, infoLineProps(1, fontstyle.Bold, 9, colorTextPrimary)),
		text.New(fmt.Sprintf("%s: %s", in.CustomerIdentificationType, in.CustomerIdentification), infoLineProps(5, fontstyle.Normal, 8.5, colorTextSecondary)),
		text.New("Dirección: "+in.CustomerAddressLine, infoLineProps(9, fontstyle.Normal, 8.5, colorTextSecondary)),
		text.New(fmt.Sprintf("Tel: %s   %s", in.CustomerPhone, in.CustomerEmail), infoLineProps(13, fontstyle.Normal, 8.5, colorTextSecondary)),
	).WithStyle(&props.Cell{BorderType: border.Full, BorderColor: colorBorder})

	invoiceNumber := "BORRADOR"
	numberColor := colorDraft
	if !in.IsDraft {
		invoiceNumber = fmt.Sprintf("%s%d", in.Prefix, in.Number)
		numberColor = colorAccent
	}

	invoiceCol := col.New(6).Add(
		text.New("Factura Electrónica de Venta No.", infoLineProps(1, fontstyle.Bold, 9, colorTextPrimary)),
		text.New(invoiceNumber, infoLineProps(5, fontstyle.Bold, 11, numberColor)),
		text.New("Fecha de emisión: "+orDash(in.IssueDate), infoLineProps(10, fontstyle.Normal, 8.5, colorTextSecondary)),
		text.New("Forma de pago: "+orDash(in.PaymentTermName)+"   Medio: "+orDash(in.PaymentMethodName), infoLineProps(14, fontstyle.Normal, 8.5, colorTextSecondary)),
		text.New("Moneda: "+in.CurrencyCode, infoLineProps(18, fontstyle.Normal, 8.5, colorTextSecondary)),
	).WithStyle(&props.Cell{BorderType: border.Full, BorderColor: colorBorder})

	m.AddRows(row.New(23).Add(clientCol, invoiceCol))
	m.AddRows(row.New(2))
}

func buildItemsTable(m core.Maroto, in InvoiceInput) {
	rows, err := list.Build[InvoiceLine](in.Lines)
	if err != nil {
		// list.Build solo falla si Lines está vacío de una forma que rompe el cálculo de
		// cabecera — documents.Service ya garantiza al menos una línea antes de llegar aquí,
		// así que esto es defensivo, no se espera que ocurra en producción.
		m.AddRows(row.New(6).Add(text.NewCol(12, "No fue posible construir la tabla de ítems", props.Text{Color: colorDraft})))
		return
	}
	m.AddRows(rows...)
	m.AddRows(row.New(3))
}

func buildTotals(m core.Maroto, in InvoiceInput) {
	cell := func(label, value string) core.Col {
		return col.New(4).Add(
			text.New(label, props.Text{Top: 1.5, Size: 7.5, Align: align.Center, Color: colorTextSecondary}),
			text.New(value, props.Text{Top: 6, Size: 9.5, Align: align.Center, Style: fontstyle.Bold, Color: colorTextPrimary}),
		).WithStyle(&props.Cell{BorderType: border.Full, BorderColor: colorBorder})
	}
	m.AddRows(row.New(13).Add(
		cell("Subtotal", formatCOP(in.LineExtensionCents)),
		cell("Impuestos", formatCOP(in.TaxAmountCents)),
		cell("Total a pagar", formatCOP(in.PayableCents)),
	))
	m.AddRows(row.New(2))
}

func buildTaxBreakdown(m core.Maroto, in InvoiceInput) {
	if len(in.Taxes) == 0 {
		return
	}
	m.AddRows(row.New(6).Add(
		text.NewCol(4, "Impuesto", headerCellProps(align.Left)),
		text.NewCol(2, "Tarifa", headerCellProps(align.Right)),
		text.NewCol(3, "Base", headerCellProps(align.Right)),
		text.NewCol(3, "Valor", headerCellProps(align.Right)),
	).WithStyle(&props.Cell{BackgroundColor: &props.Color{Red: 240, Green: 240, Blue: 240}}))
	for _, t := range in.Taxes {
		m.AddRows(row.New(6).Add(
			text.NewCol(4, t.TypeName, bodyCellProps(align.Left)),
			text.NewCol(2, fmt.Sprintf("%.0f%%", t.Percent), bodyCellProps(align.Right)),
			text.NewCol(3, formatCOP(t.TaxableAmountCents), bodyCellProps(align.Right)),
			text.NewCol(3, formatCOP(t.TaxAmountCents), bodyCellProps(align.Right)),
		).WithStyle(&props.Cell{BorderType: border.Bottom, BorderColor: colorBorder}))
	}
	m.AddRows(row.New(2))
}

func buildAmountInWords(m core.Maroto, in InvoiceInput) {
	words := AmountInWords(in.PayableCents / 100)
	m.AddRows(text.NewRow(6, "Son: "+words, props.Text{Size: 8, Style: fontstyle.Bold, Color: colorTextPrimary}))
	if in.Note != "" {
		m.AddRows(text.NewRow(6, "Nota: "+in.Note, props.Text{Size: 8, Color: colorTextSecondary}))
	}
	m.AddRows(row.New(3))
}

func buildFooter(m core.Maroto, in InvoiceInput) {
	m.AddRows(text.NewRow(5, "Representación gráfica de Factura Electrónica de Venta.", props.Text{
		Size: 7, Color: colorTextSecondary,
	}))
	if in.ResolutionNumber != "" {
		m.AddRows(text.NewRow(8, resolutionDisclaimer(in), props.Text{Size: 7, Color: colorTextSecondary}))
	}
}

// resolutionDisclaimer es el texto de autorización del pie — separado de buildFooter para
// poder probarlo sin inspeccionar bytes de PDF. RangeTo nil (rango sin tope, ver el comentario
// en InvoiceInput) omite la cláusula "hasta ..." en vez de inventar un "0".
func resolutionDisclaimer(in InvoiceInput) string {
	rangeClause := fmt.Sprintf("desde %s%d", in.RangePrefix, in.RangeFrom)
	if in.RangeTo != nil {
		rangeClause += fmt.Sprintf(" hasta %s%d", in.RangePrefix, *in.RangeTo)
	}
	return fmt.Sprintf(
		"Autorización de numeración No. %s vigente desde %s hasta %s. Rango autorizado %s.",
		in.ResolutionNumber, in.ValidFrom, in.ValidTo, rangeClause,
	)
}

// ── helpers de formato ──────────────────────────────────────────────────────────────────────

func headerCellProps(a align.Type) props.Text {
	return props.Text{Top: 2, Size: 7.5, Align: a, Style: fontstyle.Bold, Color: colorTextPrimary}
}

func bodyCellProps(a align.Type) props.Text {
	return props.Text{Top: 2, Size: 7.5, Align: a, Color: colorTextPrimary}
}

func infoLineProps(top float64, style fontstyle.Type, size float64, color *props.Color) props.Text {
	return props.Text{Top: top, Size: size, Style: style, Align: align.Left, Color: color}
}

func formatCOP(cents int64) string {
	pesos := cents / 100
	s := fmt.Sprintf("%d", pesos)
	// Separador de miles "." (convención colombiana) — sin librería de formato de moneda
	// aparte, el monto siempre es un entero de pesos (sin centavos, ver model de Totals).
	neg := false
	if pesos < 0 {
		neg = true
		s = s[1:]
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, c)
	}
	result := "$ " + string(out)
	if neg {
		result = "-" + result
	}
	return result
}

func formatQuantity(q float64) string {
	if q == float64(int64(q)) {
		return fmt.Sprintf("%d", int64(q))
	}
	return fmt.Sprintf("%.2f", q)
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
