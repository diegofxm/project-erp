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
	colorWhite         = &props.Color{Red: 255, Green: 255, Blue: 255}
	colorTextPrimary   = &props.Color{Red: 25, Green: 25, Blue: 25}
	colorTextSecondary = &props.Color{Red: 100, Green: 100, Blue: 100}
	colorBorder        = &props.Color{Red: 210, Green: 210, Blue: 210}
	colorAccent        = &props.Color{Red: 17, Green: 53, Blue: 93}
	colorAccentLight   = &props.Color{Red: 235, Green: 240, Blue: 248}
	colorTableHeader   = &props.Color{Red: 243, Green: 244, Blue: 246}
	colorDraft         = &props.Color{Red: 180, Green: 90, Blue: 0}
	colorDraftBg       = &props.Color{Red: 255, Green: 248, Blue: 240}
)

// InvoiceLine es una línea ya calculada (precio×cantidad, impuesto ya resuelto) — el cálculo
// real vive en documents.Service.linesFromInput, este paquete solo la muestra. Implementa
// list.Listable para que la tabla de items escale sola a varias páginas si hace falta.
type InvoiceLine struct {
	ItemCode       string
	Description    string
	UnitCode       string
	Quantity       float64
	UnitPriceCents int64
	TaxPercent     float64
	TotalCents     int64
}

func (InvoiceLine) GetHeader() core.Row {
	return row.New(7).Add(
		text.NewCol(1, "No.", headerCellProps(align.Center)),
		text.NewCol(5, "Descripción", headerCellProps(align.Left)),
		text.NewCol(1, "Cant.", headerCellProps(align.Right)),
		text.NewCol(2, "Precio Unit.", headerCellProps(align.Right)),
		text.NewCol(1, "IVA", headerCellProps(align.Right)),
		text.NewCol(2, "Total", headerCellProps(align.Right)),
	).WithStyle(&props.Cell{BackgroundColor: colorTableHeader})
}

func (l InvoiceLine) GetContent(i int) core.Row {
	tax := "—"
	if l.TaxPercent > 0 {
		tax = fmt.Sprintf("%.0f%%", l.TaxPercent)
	}
	return row.New(8).Add(
		text.NewCol(1, fmt.Sprintf("%d", i+1), bodyCellProps(align.Center)),
		text.NewCol(5, l.Description, bodyCellProps(align.Left)),
		text.NewCol(1, formatQuantity(l.Quantity), bodyCellProps(align.Right)),
		text.NewCol(2, formatCOP(l.UnitPriceCents), bodyCellProps(align.Right)),
		text.NewCol(1, tax, bodyCellProps(align.Right)),
		text.NewCol(2, formatCOP(l.TotalCents), bodyCellProps(align.Right)),
	).WithStyle(&props.Cell{BorderType: border.Bottom, BorderColor: colorBorder})
}

// InvoiceTax es el desglose de impuestos por tipo — recibido ya calculado.
type InvoiceTax struct {
	TypeName           string
	Percent            float64
	TaxableAmountCents int64
	TaxAmountCents     int64
}

// InvoiceInput es todo lo que BuildInvoicePDF necesita.
type InvoiceInput struct {
	IssuerBusinessName string
	IssuerNIT          string
	IssuerCheckDigit   string
	IssuerAddressLine  string
	IssuerCityName     string
	IssuerStateName    string
	IssuerPhone        string
	IssuerEmail        string
	IssuerLogo         []byte
	IssuerLogoExt      extension.Type

	IsDraft   bool
	Prefix    string
	Number    int64
	CUFE      string
	QRURL     string
	IssueDate string

	CurrencyCode string

	CustomerName               string
	CustomerIdentificationType string
	CustomerIdentification     string
	CustomerAddressLine        string
	CustomerPhone              string
	CustomerEmail              string

	PaymentTermName   string
	PaymentMethodName string
	PaymentDueDate    string

	Lines []InvoiceLine
	Taxes []InvoiceTax

	LineExtensionCents int64
	TaxAmountCents     int64
	PayableCents       int64

	Note string

	ResolutionNumber string
	RangePrefix      string
	RangeFrom        int64
	RangeTo          *int64
	ValidFrom        string
	ValidTo          string
}

// BuildInvoicePDF construye la representación gráfica en memoria.
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
	buildFinancials(m, in)
	buildFooter(m, in)

	document, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("pdf: generar documento: %w", err)
	}
	return document.GetBytes(), nil
}

// buildHeader genera la cabecera: banda azul con tipo de documento, luego logo | datos del
// emisor | número y fecha de la factura.
func buildHeader(m core.Maroto, in InvoiceInput) {
	// Banda de título.
	m.AddRows(row.New(8).Add(
		col.New(12).Add(
			text.New("FACTURA ELECTRÓNICA DE VENTA", props.Text{
				Top: 2, Size: 9, Align: align.Center,
				Style: fontstyle.Bold, Color: colorWhite,
			}),
		).WithStyle(&props.Cell{BackgroundColor: colorAccent}),
	))

	// Logo del emisor.
	logoCol := col.New(2)
	if len(in.IssuerLogo) > 0 {
		logoCol = image.NewFromBytesCol(2, in.IssuerLogo, in.IssuerLogoExt,
			props.Rect{Center: true, Percent: 85})
	}

	// Datos del emisor.
	issuerCol := col.New(7).Add(
		text.New(in.IssuerBusinessName, infoLineProps(1.5, fontstyle.Bold, 10, colorTextPrimary)),
		text.New(fmt.Sprintf("NIT: %s-%s", in.IssuerNIT, in.IssuerCheckDigit),
			infoLineProps(6, fontstyle.Bold, 8.5, colorAccent)),
		text.New(in.IssuerAddressLine,
			infoLineProps(10.5, fontstyle.Normal, 7.5, colorTextSecondary)),
		text.New(fmt.Sprintf("%s, %s", in.IssuerCityName, in.IssuerStateName),
			infoLineProps(14, fontstyle.Normal, 7.5, colorTextSecondary)),
		text.New(fmt.Sprintf("Tel: %s   %s", in.IssuerPhone, in.IssuerEmail),
			infoLineProps(17.5, fontstyle.Normal, 7.5, colorTextSecondary)),
	)

	// Número de factura (prominente) y fecha.
	invoiceNumber := "BORRADOR"
	numberColor := colorDraft
	if !in.IsDraft {
		invoiceNumber = fmt.Sprintf("%s%d", in.Prefix, in.Number)
		numberColor = colorAccent
	}
	invoiceDetailCol := col.New(3).Add(
		text.New("No.", props.Text{
			Top: 2, Size: 7.5, Align: align.Right, Color: colorTextSecondary,
		}),
		text.New(invoiceNumber, props.Text{
			Top: 6, Size: 12, Align: align.Right,
			Style: fontstyle.Bold, Color: numberColor,
		}),
		text.New(orDash(in.IssueDate), props.Text{
			Top: 17, Size: 7.5, Align: align.Right, Color: colorTextSecondary,
		}),
	).WithStyle(&props.Cell{BorderType: border.Full, BorderColor: colorBorder})

	m.AddRows(row.New(26).Add(logoCol, issuerCol, invoiceDetailCol))
}

// buildStatusBar muestra el CUFE (factura confirmada) o un aviso de borrador.
func buildStatusBar(m core.Maroto, in InvoiceInput) {
	if in.IsDraft {
		m.AddRows(row.New(7).Add(
			col.New(12).Add(text.New("BORRADOR — PENDIENTE DE CONFIRMACIÓN ANTE LA DIAN", props.Text{
				Top: 2, Size: 8, Align: align.Center,
				Style: fontstyle.Bold, Color: colorDraft,
			})).WithStyle(&props.Cell{
				BackgroundColor: colorDraftBg,
				BorderType:      border.Full,
				BorderColor:     colorDraft,
			}),
		))
		return
	}
	m.AddRows(row.New(6).Add(
		col.New(12).Add(text.New("CUFE: "+in.CUFE, props.Text{
			Top: 1.5, Size: 6.5, Align: align.Left, Color: colorTextSecondary,
		})).WithStyle(&props.Cell{
			BackgroundColor: colorAccentLight,
			BorderType:      border.Full,
			BorderColor:     colorBorder,
		}),
	))
}

// buildPartiesSection muestra datos del adquiriente y condiciones de pago en dos columnas.
func buildPartiesSection(m core.Maroto, in InvoiceInput) {
	m.AddRows(row.New(2))

	customerItems := []core.Component{
		text.New("CLIENTE", props.Text{
			Top: 1.5, Size: 6.5, Style: fontstyle.Bold, Color: colorAccent,
		}),
		text.New(in.CustomerName, infoLineProps(5.5, fontstyle.Bold, 8.5, colorTextPrimary)),
		text.New(fmt.Sprintf("%s: %s", in.CustomerIdentificationType, in.CustomerIdentification),
			infoLineProps(10, fontstyle.Normal, 7.5, colorTextSecondary)),
		text.New("Dir: "+orDash(in.CustomerAddressLine),
			infoLineProps(13.5, fontstyle.Normal, 7.5, colorTextSecondary)),
		text.New(fmt.Sprintf("Tel: %s   %s", orDash(in.CustomerPhone), orDash(in.CustomerEmail)),
			infoLineProps(17, fontstyle.Normal, 7.5, colorTextSecondary)),
	}
	customerCol := col.New(6).Add(customerItems...).
		WithStyle(&props.Cell{BorderType: border.Full, BorderColor: colorBorder})

	paymentItems := []core.Component{
		text.New("CONDICIONES DE PAGO", props.Text{
			Top: 1.5, Size: 6.5, Style: fontstyle.Bold, Color: colorAccent,
		}),
		text.New("Forma: "+orDash(in.PaymentTermName),
			infoLineProps(5.5, fontstyle.Normal, 7.5, colorTextSecondary)),
		text.New("Medio: "+orDash(in.PaymentMethodName),
			infoLineProps(9, fontstyle.Normal, 7.5, colorTextSecondary)),
		text.New("Moneda: "+in.CurrencyCode,
			infoLineProps(12.5, fontstyle.Normal, 7.5, colorTextSecondary)),
	}
	if in.PaymentDueDate != "" {
		paymentItems = append(paymentItems,
			text.New("Vencimiento: "+in.PaymentDueDate,
				infoLineProps(16, fontstyle.Normal, 7.5, colorTextSecondary)))
	}
	paymentCol := col.New(6).Add(paymentItems...).
		WithStyle(&props.Cell{BorderType: border.Full, BorderColor: colorBorder})

	m.AddRows(row.New(23).Add(customerCol, paymentCol))
	m.AddRows(row.New(2))
}

func buildItemsTable(m core.Maroto, in InvoiceInput) {
	rows, err := list.Build[InvoiceLine](in.Lines)
	if err != nil {
		m.AddRows(row.New(6).Add(
			text.NewCol(12, "No fue posible construir la tabla de ítems",
				props.Text{Color: colorDraft})))
		return
	}
	m.AddRows(rows...)
	m.AddRows(row.New(3))
}

// buildFinancials muestra el desglose de impuestos y los totales con Total a Pagar destacado.
func buildFinancials(m core.Maroto, in InvoiceInput) {
	// Tabla de impuestos (si hay).
	if len(in.Taxes) > 0 {
		m.AddRows(row.New(6).Add(
			text.NewCol(3, "Impuesto", headerCellProps(align.Left)),
			text.NewCol(2, "Tarifa", headerCellProps(align.Right)),
			text.NewCol(4, "Base Gravable", headerCellProps(align.Right)),
			text.NewCol(3, "Valor Impuesto", headerCellProps(align.Right)),
		).WithStyle(&props.Cell{BackgroundColor: colorTableHeader}))
		for _, t := range in.Taxes {
			m.AddRows(row.New(6).Add(
				text.NewCol(3, t.TypeName, bodyCellProps(align.Left)),
				text.NewCol(2, fmt.Sprintf("%.0f%%", t.Percent), bodyCellProps(align.Right)),
				text.NewCol(4, formatCOP(t.TaxableAmountCents), bodyCellProps(align.Right)),
				text.NewCol(3, formatCOP(t.TaxAmountCents), bodyCellProps(align.Right)),
			).WithStyle(&props.Cell{BorderType: border.Bottom, BorderColor: colorBorder}))
		}
		m.AddRows(row.New(3))
	}

	// Subtotal.
	m.AddRows(row.New(7).Add(
		col.New(8),
		text.NewCol(2, "Subtotal", props.Text{
			Top: 1.5, Size: 8, Align: align.Right, Color: colorTextSecondary,
		}).WithStyle(&props.Cell{BorderType: border.Bottom, BorderColor: colorBorder}),
		text.NewCol(2, formatCOP(in.LineExtensionCents), props.Text{
			Top: 1.5, Size: 8, Align: align.Right, Color: colorTextPrimary,
		}).WithStyle(&props.Cell{BorderType: border.Bottom, BorderColor: colorBorder}),
	))

	// Impuestos.
	m.AddRows(row.New(7).Add(
		col.New(8),
		text.NewCol(2, "Impuestos", props.Text{
			Top: 1.5, Size: 8, Align: align.Right, Color: colorTextSecondary,
		}).WithStyle(&props.Cell{BorderType: border.Bottom, BorderColor: colorBorder}),
		text.NewCol(2, formatCOP(in.TaxAmountCents), props.Text{
			Top: 1.5, Size: 8, Align: align.Right, Color: colorTextPrimary,
		}).WithStyle(&props.Cell{BorderType: border.Bottom, BorderColor: colorBorder}),
	))

	// Total a Pagar — fila destacada con fondo accent.
	m.AddRows(row.New(10).Add(
		col.New(8),
		text.NewCol(2, "TOTAL A PAGAR", props.Text{
			Top: 2.5, Size: 8.5, Align: align.Right,
			Style: fontstyle.Bold, Color: colorWhite,
		}).WithStyle(&props.Cell{BackgroundColor: colorAccent}),
		text.NewCol(2, formatCOP(in.PayableCents), props.Text{
			Top: 2.5, Size: 8.5, Align: align.Right,
			Style: fontstyle.Bold, Color: colorWhite,
		}).WithStyle(&props.Cell{BackgroundColor: colorAccent}),
	))
}

// buildFooter muestra la cantidad en letras, nota opcional, y al pie el QR junto al texto de
// autorización de numeración DIAN — el QR va aquí (no en la cabecera) para que quede a mano
// al escanear el documento físico impreso.
func buildFooter(m core.Maroto, in InvoiceInput) {
	m.AddRows(row.New(3))

	// Valor en letras con fondo suave.
	words := AmountInWords(in.PayableCents / 100)
	m.AddRows(row.New(6).Add(
		col.New(12).Add(
			text.New("Son: "+words, props.Text{
				Top: 1.5, Size: 7.5, Style: fontstyle.Bold, Color: colorTextPrimary,
			}),
		).WithStyle(&props.Cell{
			BackgroundColor: colorAccentLight,
			BorderType:      border.Full,
			BorderColor:     colorBorder,
		}),
	))

	if in.Note != "" {
		m.AddRows(row.New(2))
		m.AddRows(row.New(6).Add(
			text.NewCol(12, "Nota: "+in.Note, props.Text{
				Top: 1.5, Size: 7.5, Color: colorTextSecondary,
			}),
		))
	}

	m.AddRows(row.New(5))

	// QR + texto legal.
	qrCol := col.New(3)
	if !in.IsDraft && in.QRURL != "" {
		qrCol = qrCol.Add(code.NewQr(in.QRURL, props.Rect{Center: true, Percent: 95}))
	}

	legalItems := []core.Component{
		text.New("Representación gráfica de Factura Electrónica de Venta.", props.Text{
			Top: 1, Size: 7, Color: colorTextSecondary,
		}),
	}
	if in.ResolutionNumber != "" {
		legalItems = append(legalItems,
			text.New(resolutionDisclaimer(in), props.Text{
				Top: 6, Size: 7, Color: colorTextSecondary,
			}))
	}

	m.AddRows(row.New(25).Add(
		qrCol,
		col.New(9).Add(legalItems...),
	))
}

// resolutionDisclaimer construye el texto de autorización. RangeTo nil omite la cláusula
// "hasta ..." (rango sin tope, ver InvoiceInput.RangeTo).
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
