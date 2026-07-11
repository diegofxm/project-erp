// Package pdf construye la representación gráfica (PDF) de una Factura Electrónica — siempre
// en memoria, nunca a disco (ver docs/apidian-architecture.md sección 9.39). Deliberadamente
// no conoce los tipos de internal/documents/internal/issuers ni de cofacture: recibe un
// InvoiceInput plano, mismo principio de puertos angostos que el resto de apidian — quien
// orquesta el mapeo es documents.Service, igual que ya lo hace para construir el XML con
// cofacture.
package pdf

import (
	"fmt"
	"strings"
	"unicode"

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
	colorTextPrimary   = &props.Color{Red: 20, Green: 20, Blue: 20}
	colorTextSecondary = &props.Color{Red: 90, Green: 90, Blue: 90}
	colorBorder        = &props.Color{Red: 160, Green: 160, Blue: 160}
	colorSectionBand   = &props.Color{Red: 40, Green: 40, Blue: 40}
	colorTableHeader   = &props.Color{Red: 215, Green: 215, Blue: 215}
	colorStripe        = &props.Color{Red: 248, Green: 248, Blue: 248}
	colorAccent        = &props.Color{Red: 17, Green: 53, Blue: 93}
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
	return row.New(5).Add(
		text.NewCol(1, "No.", headerCellProps(align.Center)),
		text.NewCol(5, "Descripción", headerCellProps(align.Left)),
		text.NewCol(1, "Cant.", headerCellProps(align.Right)),
		text.NewCol(2, "Precio Unit.", headerCellProps(align.Right)),
		text.NewCol(1, "IVA", headerCellProps(align.Right)),
		text.NewCol(2, "Total", headerCellProps(align.Right)),
	).WithStyle(&props.Cell{BackgroundColor: colorTableHeader, BorderType: border.Full, BorderColor: colorBorder})
}

func (l InvoiceLine) GetContent(i int) core.Row {
	tax := "—"
	if l.TaxPercent > 0 {
		tax = fmt.Sprintf("%.0f%%", l.TaxPercent)
	}
	bg := colorWhite
	if i%2 == 1 {
		bg = colorStripe
	}
	return row.New(5).Add(
		text.NewCol(1, fmt.Sprintf("%d", i+1), bodyCellProps(align.Center)),
		text.NewCol(5, l.Description, bodyCellProps(align.Left)),
		text.NewCol(1, formatQuantity(l.Quantity), bodyCellProps(align.Right)),
		text.NewCol(2, formatCOP(l.UnitPriceCents), bodyCellProps(align.Right)),
		text.NewCol(1, tax, bodyCellProps(align.Right)),
		text.NewCol(2, formatCOP(l.TotalCents), props.Text{
			Top: 1, Size: 7, Align: align.Right, Style: fontstyle.Bold, Color: colorTextPrimary,
		}),
	).WithStyle(&props.Cell{BackgroundColor: bg, BorderType: border.Full, BorderColor: colorBorder})
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

	// DocumentTitle es el título de la banda superior (ej. "FACTURA ELECTRÓNICA DE VENTA").
	// Vacío → se usa ese valor por defecto.
	DocumentTitle string
	// HashLabel es la etiqueta del código de unicidad: "CUFE" para Factura, "CUDE" para NC/ND.
	// Vacío → "CUFE".
	HashLabel string

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
		WithTopMargin(8).
		WithRightMargin(10).
		WithBottomMargin(8).
		Build()
	m := maroto.New(cfg)

	buildHeader(m, in)
	buildTitleBand(m, in)
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

// buildHeader: logo enmarcado (izquierda) | datos del emisor (centro) | número prominente + QR (derecha).
// El QR se ubica en la esquina superior derecha, junto al número, para que sea fácil de escanear
// en el documento impreso — inspirado en el Bill of Lading donde el código de barras aparece
// inmediatamente debajo del número de referencia (sección 9.39/9.49).
func buildHeader(m core.Maroto, in InvoiceInput) {
	// Logo — siempre enmarcado en un recuadro, con imagen si la hay.
	var logoCol core.Col
	if len(in.IssuerLogo) > 0 {
		logoCol = image.NewFromBytesCol(2, in.IssuerLogo, in.IssuerLogoExt,
			props.Rect{Center: true, Percent: 85})
	} else {
		logoCol = col.New(2)
	}
	logoCol = logoCol.WithStyle(&props.Cell{BorderType: border.Full, BorderColor: colorBorder})

	// Datos del emisor.
	issuerCol := col.New(7).Add(
		text.New(in.IssuerBusinessName, props.Text{
			Top: 1.5, Size: 9, Style: fontstyle.Bold, Color: colorTextPrimary,
		}),
		text.New(fmt.Sprintf("NIT: %s-%s", in.IssuerNIT, in.IssuerCheckDigit), props.Text{
			Top: 7, Size: 8, Style: fontstyle.Bold, Color: colorAccent,
		}),
		text.New(in.IssuerAddressLine, props.Text{
			Top: 11.5, Size: 7, Color: colorTextSecondary,
		}),
		text.New(fmt.Sprintf("%s, %s", in.IssuerCityName, in.IssuerStateName), props.Text{
			Top: 15.5, Size: 7, Color: colorTextSecondary,
		}),
		text.New(fmt.Sprintf("Tel: %s   %s", in.IssuerPhone, in.IssuerEmail), props.Text{
			Top: 19.5, Size: 7, Color: colorTextSecondary,
		}),
	)

	// Número prominente arriba a la derecha + QR abajo (como el número + código de barras del BOL).
	invoiceNumber := "BORRADOR"
	numberColor := colorDraft
	if !in.IsDraft {
		invoiceNumber = fmt.Sprintf("%s%d", in.Prefix, in.Number)
		numberColor = colorAccent
	}

	numberItems := []core.Component{
		text.New("No.", props.Text{
			Top: 1, Size: 6.5, Align: align.Right, Color: colorTextSecondary,
		}),
		text.New(invoiceNumber, props.Text{
			Top: 4.5, Size: 12, Align: align.Right, Style: fontstyle.Bold, Color: numberColor,
		}),
		text.New(orDash(in.IssueDate), props.Text{
			Top: 15, Size: 7, Align: align.Right, Color: colorTextSecondary,
		}),
	}
	if !in.IsDraft && in.QRURL != "" {
		numberItems = append(numberItems,
			code.NewQr(in.QRURL, props.Rect{Top: 19, Center: true, Percent: 85}))
	} else if in.IsDraft {
		numberItems = append(numberItems, text.New("BORRADOR", props.Text{
			Top: 21, Size: 7.5, Align: align.Center, Style: fontstyle.Bold, Color: colorDraft,
		}))
	}
	numberCol := col.New(3).Add(numberItems...).
		WithStyle(&props.Cell{BorderType: border.Full, BorderColor: colorBorder})

	m.AddRows(row.New(38).Add(logoCol, issuerCol, numberCol))
}

// buildTitleBand: banda oscura con el tipo de documento + barra de CUFE (o aviso de borrador).
func buildTitleBand(m core.Maroto, in InvoiceInput) {
	docTitle := in.DocumentTitle
	if docTitle == "" {
		docTitle = "FACTURA ELECTRÓNICA DE VENTA"
	}

	m.AddRows(row.New(5).Add(
		col.New(12).Add(text.New(docTitle, props.Text{
			Top: 0.8, Size: 8, Align: align.Center, Style: fontstyle.Bold, Color: colorWhite,
		})).WithStyle(&props.Cell{BackgroundColor: colorSectionBand}),
	))

	if in.IsDraft {
		m.AddRows(row.New(5).Add(
			col.New(12).Add(text.New("BORRADOR — PENDIENTE DE CONFIRMACIÓN ANTE LA DIAN", props.Text{
				Top: 1, Size: 7.5, Align: align.Center, Style: fontstyle.Bold, Color: colorDraft,
			})).WithStyle(&props.Cell{
				BackgroundColor: colorDraftBg, BorderType: border.Full, BorderColor: colorDraft,
			}),
		))
		return
	}

	hashLabel := in.HashLabel
	if hashLabel == "" {
		hashLabel = "CUFE"
	}
	m.AddRows(row.New(5).Add(
		col.New(12).Add(text.New(hashLabel+": "+in.CUFE, props.Text{
			Top: 1, Size: 6, Align: align.Left, Color: colorTextSecondary,
		})).WithStyle(&props.Cell{BorderType: border.Bottom, BorderColor: colorBorder}),
	))
}

// buildPartiesSection: bandas oscuras de sección + datos del cliente y condiciones de pago.
func buildPartiesSection(m core.Maroto, in InvoiceInput) {
	m.AddRows(row.New(4).Add(
		col.New(6).Add(text.New("DATOS DEL CLIENTE", props.Text{
			Top: 0.8, Size: 7, Style: fontstyle.Bold, Color: colorWhite,
		})).WithStyle(&props.Cell{BackgroundColor: colorSectionBand}),
		col.New(6).Add(text.New("CONDICIONES DE PAGO", props.Text{
			Top: 0.8, Size: 7, Style: fontstyle.Bold, Color: colorWhite,
		})).WithStyle(&props.Cell{BackgroundColor: colorSectionBand}),
	))

	customerCol := col.New(6).Add(
		text.New(in.CustomerName, props.Text{
			Top: 1.5, Size: 8, Style: fontstyle.Bold, Color: colorTextPrimary,
		}),
		text.New(fmt.Sprintf("%s: %s", in.CustomerIdentificationType, in.CustomerIdentification), props.Text{
			Top: 6, Size: 7, Color: colorTextSecondary,
		}),
		text.New(orDash(in.CustomerAddressLine), props.Text{
			Top: 9.5, Size: 7, Color: colorTextSecondary,
		}),
		text.New(fmt.Sprintf("Tel: %s   %s", orDash(in.CustomerPhone), orDash(in.CustomerEmail)), props.Text{
			Top: 13, Size: 7, Color: colorTextSecondary,
		}),
	).WithStyle(&props.Cell{BorderType: border.Full, BorderColor: colorBorder})

	paymentItems := []core.Component{
		text.New("Forma: "+orDash(in.PaymentTermName), props.Text{
			Top: 1.5, Size: 7, Color: colorTextSecondary,
		}),
		text.New("Medio: "+orDash(in.PaymentMethodName), props.Text{
			Top: 5, Size: 7, Color: colorTextSecondary,
		}),
		text.New("Moneda: "+in.CurrencyCode, props.Text{
			Top: 8.5, Size: 7, Color: colorTextSecondary,
		}),
	}
	if in.PaymentDueDate != "" {
		paymentItems = append(paymentItems, text.New("Vencimiento: "+in.PaymentDueDate, props.Text{
			Top: 12, Size: 7, Color: colorTextSecondary,
		}))
	}
	paymentCol := col.New(6).Add(paymentItems...).
		WithStyle(&props.Cell{BorderType: border.Full, BorderColor: colorBorder})

	m.AddRows(row.New(18).Add(customerCol, paymentCol))
}

func buildItemsTable(m core.Maroto, in InvoiceInput) {
	m.AddRows(row.New(4).Add(
		col.New(12).Add(text.New("DETALLE DE PRODUCTOS / SERVICIOS", props.Text{
			Top: 0.8, Size: 7, Style: fontstyle.Bold, Color: colorWhite,
		})).WithStyle(&props.Cell{BackgroundColor: colorSectionBand}),
	))

	rows, err := list.Build[InvoiceLine](in.Lines)
	if err != nil {
		m.AddRows(row.New(6).Add(
			text.NewCol(12, "No fue posible construir la tabla de ítems",
				props.Text{Color: colorDraft})))
		return
	}
	m.AddRows(rows...)
}

// buildFinancials: tabla de impuestos (si hay) + subtotal + impuestos + total con banda oscura.
func buildFinancials(m core.Maroto, in InvoiceInput) {
	if len(in.Taxes) > 0 {
		m.AddRows(row.New(4).Add(
			text.NewCol(3, "Impuesto", headerCellProps(align.Left)),
			text.NewCol(2, "Tarifa", headerCellProps(align.Right)),
			text.NewCol(4, "Base Gravable", headerCellProps(align.Right)),
			text.NewCol(3, "Valor Impuesto", headerCellProps(align.Right)),
		).WithStyle(&props.Cell{BackgroundColor: colorTableHeader, BorderType: border.Full, BorderColor: colorBorder}))
		for _, t := range in.Taxes {
			m.AddRows(row.New(5).Add(
				text.NewCol(3, t.TypeName, bodyCellProps(align.Left)),
				text.NewCol(2, fmt.Sprintf("%.0f%%", t.Percent), bodyCellProps(align.Right)),
				text.NewCol(4, formatCOP(t.TaxableAmountCents), bodyCellProps(align.Right)),
				text.NewCol(3, formatCOP(t.TaxAmountCents), bodyCellProps(align.Right)),
			).WithStyle(&props.Cell{BorderType: border.Full, BorderColor: colorBorder}))
		}
	}

	m.AddRows(row.New(5).Add(
		col.New(8),
		text.NewCol(2, "Subtotal:", props.Text{Top: 1, Size: 7.5, Align: align.Right, Color: colorTextSecondary}).
			WithStyle(&props.Cell{BorderType: border.Bottom, BorderColor: colorBorder}),
		text.NewCol(2, formatCOP(in.LineExtensionCents), props.Text{Top: 1, Size: 7.5, Align: align.Right, Color: colorTextPrimary}).
			WithStyle(&props.Cell{BorderType: border.Bottom, BorderColor: colorBorder}),
	))

	m.AddRows(row.New(5).Add(
		col.New(8),
		text.NewCol(2, "Impuestos:", props.Text{Top: 1, Size: 7.5, Align: align.Right, Color: colorTextSecondary}).
			WithStyle(&props.Cell{BorderType: border.Bottom, BorderColor: colorBorder}),
		text.NewCol(2, formatCOP(in.TaxAmountCents), props.Text{Top: 1, Size: 7.5, Align: align.Right, Color: colorTextPrimary}).
			WithStyle(&props.Cell{BorderType: border.Bottom, BorderColor: colorBorder}),
	))

	m.AddRows(row.New(7).Add(
		col.New(8),
		text.NewCol(2, "TOTAL A PAGAR:", props.Text{
			Top: 1.5, Size: 8, Align: align.Right, Style: fontstyle.Bold, Color: colorWhite,
		}).WithStyle(&props.Cell{BackgroundColor: colorSectionBand}),
		text.NewCol(2, formatCOP(in.PayableCents), props.Text{
			Top: 1.5, Size: 8, Align: align.Right, Style: fontstyle.Bold, Color: colorWhite,
		}).WithStyle(&props.Cell{BackgroundColor: colorSectionBand}),
	))
}

// buildFooter: valor en letras, nota opcional, texto de autorización de numeración DIAN.
// El QR va en la cabecera (junto al número), no aquí — ver buildHeader.
func buildFooter(m core.Maroto, in InvoiceInput) {
	words := AmountInWords(in.PayableCents / 100)
	m.AddRows(row.New(5).Add(
		col.New(12).Add(text.New("Son: "+words, props.Text{
			Top: 1, Size: 7, Style: fontstyle.Bold, Color: colorTextPrimary,
		})).WithStyle(&props.Cell{
			BackgroundColor: colorTableHeader,
			BorderType:      border.Full,
			BorderColor:     colorBorder,
		}),
	))

	if in.Note != "" {
		m.AddRows(row.New(5).Add(
			text.NewCol(12, "Nota: "+in.Note, props.Text{
				Top: 1, Size: 7, Color: colorTextSecondary,
			}),
		))
	}

	m.AddRows(row.New(3))

	docTitle := in.DocumentTitle
	if docTitle == "" {
		docTitle = "FACTURA ELECTRÓNICA DE VENTA"
	}
	runes := []rune(strings.ToLower(docTitle))
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}
	legalItems := []core.Component{
		text.New("Representación gráfica de "+string(runes)+".", props.Text{
			Top: 1, Size: 6.5, Color: colorTextSecondary,
		}),
	}
	if in.ResolutionNumber != "" {
		legalItems = append(legalItems, text.New(resolutionDisclaimer(in), props.Text{
			Top: 6, Size: 6.5, Color: colorTextSecondary,
		}))
	}
	m.AddRows(row.New(12).Add(col.New(12).Add(legalItems...)))
}

// resolutionDisclaimer construye el texto de autorización DIAN. RangeTo nil omite "hasta ..."
// (rango sin tope).
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
	return props.Text{Top: 1, Size: 7, Align: a, Style: fontstyle.Bold, Color: colorTextPrimary}
}

func bodyCellProps(a align.Type) props.Text {
	return props.Text{Top: 1, Size: 7, Align: a, Color: colorTextPrimary}
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
