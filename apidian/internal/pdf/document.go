// Package pdf construye la representación gráfica (PDF) de una Factura Electrónica — siempre
// en memoria, nunca a disco (ver docs/apidian-architecture.md sección 9.39). Usa chromedp para
// renderizar el template HTML a PDF mediante Chrome headless; el template document.html es el
// archivo editable para ajustar el diseño sin recompilar.
package pdf

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"strings"
	"time"
	"unicode"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	qrcode "github.com/skip2/go-qrcode"
)

//go:embed document.html
var documentTmpl string

var docTemplate = template.Must(template.New("document").Parse(documentTmpl))

// InvoiceLine es una línea ya calculada (precio×cantidad, impuesto ya resuelto) — el cálculo
// real vive en documents.Service.linesFromInput, este paquete solo la muestra.
type InvoiceLine struct {
	ItemCode       string
	Description    string
	UnitCode       string
	Quantity       float64
	UnitPriceCents int64
	TaxPercent     float64
	TotalCents     int64
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
	IssuerLogoExt      string // "png", "jpg", etc.

	IsDraft      bool
	Prefix       string
	Number       int64
	CUFE         string
	QRURL        string // URL que va en sts:QRCode del XML; para FE/NC/ND es el mismo contenido del QR
	QRContent    string // para DS: texto multi-línea que se codifica en la imagen QR (distinto de QRURL)
	IssueDate    string

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

	// DocumentTitle es el título superior (ej. "FACTURA ELECTRÓNICA DE VENTA").
	// Vacío → usa ese valor por defecto.
	DocumentTitle string
	// HashLabel es la etiqueta del código de unicidad: "CUFE" para Factura, "CUDE" para NC/ND.
	// Vacío → "CUFE".
	HashLabel string
	// IsDS indica que el documento es un Documento Soporte (tipo 05). Activa la marca
	// "SIN VALOR FISCAL" y cambia la etiqueta "Adquiriente" por "Proveedor" en el PDF.
	IsDS bool

	// Información fiscal del emisor para el pie del documento (por norma — Resolución DIAN 042).
	IssuerTaxRegime        string // ej. "Responsable de IVA"
	IssuerLiabilities      string // ej. "O-13 – Gran Contribuyente · O-15 – Autorretenedor"
	IssuerEconomicActivity string // ej. "6201, 9901" (códigos CIIU sin descripción)

	ResolutionNumber string
	RangePrefix      string
	RangeFrom        int64
	RangeTo          *int64
	ValidFrom        string
	ValidTo          string
}

// chromePath almacena la ruta al binario Chrome/Chromium configurada en arranque.
// Vacío = chromedp busca en PATH automáticamente (funciona en local; falla en VPS sin Chrome).
var chromePath string

// SetChromePath fija la ruta al binario Chrome/Chromium para el generador de PDF.
// Debe llamarse una sola vez al arrancar el servidor (antes de cualquier petición de PDF).
// Vacío restablece la búsqueda automática en PATH.
func SetChromePath(path string) {
	chromePath = path
}

// BuildInvoicePDF construye la representación gráfica en memoria usando chromedp.
func BuildInvoicePDF(in InvoiceInput) ([]byte, error) {
	data, err := buildTemplateData(in)
	if err != nil {
		return nil, err
	}
	html, err := renderHTML(data)
	if err != nil {
		return nil, err
	}
	return htmlToPDF(html)
}

// ── datos del template ────────────────────────────────────────────────────────

type docTemplateData struct {
	DocumentTitle  string
	DocTitleLower  string
	HashLabel      string
	DocNumberLabel string

	BusinessName string
	NIT          string
	CheckDigit   string
	Address      string
	City         string
	Phone        string
	Email        string
	LogoDataURL  template.URL

	IsDraft bool
	IsDS    bool
	DocNumber string
	CUFE      string
	QR        template.URL
	IssueDate string
	Currency  string

	CustomerName    string
	CustomerIDType  string
	CustomerID      string
	CustomerAddress string
	CustomerPhone   string
	CustomerEmail   string

	PaymentTerm   string
	PaymentMethod string
	DueDate       string

	Lines []docLineData
	Taxes []docTaxData

	Subtotal      string
	TaxTotal      string
	GrandTotal    string
	AmountInWords string

	Note           string
	ResolutionText string

	TaxRegime        string
	Liabilities      string
	EconomicActivity string
}

type docLineData struct {
	Num         int
	Code        string
	Description string
	Qty         string
	UnitPrice   string
	TaxRate     string
	LineTotal   string
}

type docTaxData struct {
	Name      string
	Rate      string
	TaxBase   string
	TaxAmount string
}

func buildTemplateData(in InvoiceInput) (docTemplateData, error) {
	var logoURL template.URL
	if len(in.IssuerLogo) > 0 {
		mime := "image/png"
		if ext := strings.ToLower(in.IssuerLogoExt); ext != "" && ext != "png" {
			mime = "image/" + ext
		}
		logoURL = template.URL("data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(in.IssuerLogo))
	}

	var qrURL template.URL
	if !in.IsDraft && in.QRURL != "" {
		qrSource := in.QRURL
		if in.QRContent != "" {
			qrSource = in.QRContent // DS: imagen QR codifica el bloque multi-línea, no la URL
		}
		var err error
		qrURL, err = makeQR(qrSource)
		if err != nil {
			return docTemplateData{}, fmt.Errorf("pdf: QR: %w", err)
		}
	}

	docTitle := in.DocumentTitle
	if docTitle == "" {
		docTitle = "FACTURA ELECTRÓNICA DE VENTA"
	}
	hashLabel := in.HashLabel
	if hashLabel == "" {
		hashLabel = "CUFE"
	}

	docNumber := "BORRADOR"
	if !in.IsDraft {
		docNumber = fmt.Sprintf("%s%d", in.Prefix, in.Number)
	}

	city := in.IssuerCityName
	if in.IssuerStateName != "" && in.IssuerStateName != in.IssuerCityName {
		city = in.IssuerCityName + ", " + in.IssuerStateName
	}

	runes := []rune(strings.ToLower(docTitle))
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}
	docTitleLower := string(runes)

	var numLabel string
	switch {
	case strings.Contains(docTitle, "CRÉDITO"):
		numLabel = "N° de Nota Crédito"
	case strings.Contains(docTitle, "DÉBITO"):
		numLabel = "N° de Nota Débito"
	case strings.Contains(docTitle, "SOPORTE"):
		numLabel = "N° de Doc. Soporte"
	default:
		numLabel = "N° de Factura"
	}

	lines := make([]docLineData, len(in.Lines))
	for i, l := range in.Lines {
		tax := "—"
		if l.TaxPercent > 0 {
			tax = fmt.Sprintf("%.0f%%", l.TaxPercent)
		}
		lines[i] = docLineData{
			Num:         i + 1,
			Code:        l.ItemCode,
			Description: l.Description,
			Qty:         formatQuantity(l.Quantity),
			UnitPrice:   formatCOP(l.UnitPriceCents),
			TaxRate:     tax,
			LineTotal:   formatCOP(l.TotalCents),
		}
	}

	taxes := make([]docTaxData, len(in.Taxes))
	for i, t := range in.Taxes {
		taxes[i] = docTaxData{
			Name:      t.TypeName,
			Rate:      fmt.Sprintf("%.0f%%", t.Percent),
			TaxBase:   formatCOP(t.TaxableAmountCents),
			TaxAmount: formatCOP(t.TaxAmountCents),
		}
	}

	var resolutionText string
	if in.ResolutionNumber != "" {
		resolutionText = resolutionDisclaimer(in)
	}

	return docTemplateData{
		DocumentTitle:  docTitle,
		DocTitleLower:  docTitleLower,
		HashLabel:      hashLabel,
		DocNumberLabel: numLabel,

		BusinessName: in.IssuerBusinessName,
		NIT:          in.IssuerNIT,
		CheckDigit:   in.IssuerCheckDigit,
		Address:      in.IssuerAddressLine,
		City:         city,
		Phone:        in.IssuerPhone,
		Email:        in.IssuerEmail,
		LogoDataURL:  logoURL,

		IsDraft:   in.IsDraft,
		IsDS:      in.IsDS,
		DocNumber: docNumber,
		CUFE:      in.CUFE,
		QR:        qrURL,
		IssueDate: in.IssueDate,
		Currency:  in.CurrencyCode,

		CustomerName:    in.CustomerName,
		CustomerIDType:  in.CustomerIdentificationType,
		CustomerID:      in.CustomerIdentification,
		CustomerAddress: in.CustomerAddressLine,
		CustomerPhone:   in.CustomerPhone,
		CustomerEmail:   in.CustomerEmail,

		PaymentTerm:   in.PaymentTermName,
		PaymentMethod: in.PaymentMethodName,
		DueDate:       in.PaymentDueDate,

		Lines: lines,
		Taxes: taxes,

		Subtotal:      formatCOP(in.LineExtensionCents),
		TaxTotal:      formatCOP(in.TaxAmountCents),
		GrandTotal:    formatCOP(in.PayableCents),
		AmountInWords: AmountInWords(in.PayableCents / 100),

		Note:           in.Note,
		ResolutionText: resolutionText,

		TaxRegime:        in.IssuerTaxRegime,
		Liabilities:      in.IssuerLiabilities,
		EconomicActivity: in.IssuerEconomicActivity,
	}, nil
}

func renderHTML(data docTemplateData) (string, error) {
	var buf bytes.Buffer
	if err := docTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("pdf: ejecutar template: %w", err)
	}
	return buf.String(), nil
}

func htmlToPDF(html string) ([]byte, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("disable-web-security", true),
	)
	if chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	dataURL := "data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(html))

	var pdfBuf []byte
	err := chromedp.Run(ctx,
		chromedp.Navigate(dataURL),
		chromedp.WaitReady("body"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuf, _, err = page.PrintToPDF().
				WithPaperWidth(8.27).
				WithPaperHeight(11.69).
				WithMarginTop(0).
				WithMarginBottom(0).
				WithMarginLeft(0).
				WithMarginRight(0).
				WithPrintBackground(true).
				Do(ctx)
			return err
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("pdf: chromedp: %w", err)
	}
	return pdfBuf, nil
}

func makeQR(content string) (template.URL, error) {
	if content == "" {
		return "", nil
	}
	png, err := qrcode.Encode(content, qrcode.Medium, 180)
	if err != nil {
		return "", fmt.Errorf("pdf: QR: %w", err)
	}
	return template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(png)), nil
}

// ── helpers de formato ────────────────────────────────────────────────────────

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
