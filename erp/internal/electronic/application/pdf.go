package application

import (
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"strings"
	"unicode"

	cofdom "github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/diegofxm/erp/internal/electronic/domain"
	reportsdomain "github.com/diegofxm/erp/internal/shared/reports/domain"
)

// GetDocumentPDFUseCase genera la representación gráfica (PDF) de un documento DIAN en memoria.
type GetDocumentPDFUseCase struct {
	docs      domain.DocumentRepository
	company   domain.CompanyPort
	numbering domain.NumberingRepository
	renderer  reportsdomain.Renderer
}

func NewGetDocumentPDFUseCase(
	docs domain.DocumentRepository,
	company domain.CompanyPort,
	numbering domain.NumberingRepository,
	renderer reportsdomain.Renderer,
) *GetDocumentPDFUseCase {
	return &GetDocumentPDFUseCase{docs: docs, company: company, numbering: numbering, renderer: renderer}
}

// GetPDF devuelve bytes del PDF. formatStr: "full_a4" (default) o "half_a4".
func (uc *GetDocumentPDFUseCase) GetPDF(ctx context.Context, companyID, docID uuid.UUID, formatStr string) ([]byte, error) {
	doc, err := uc.docs.GetByID(ctx, docID)
	if err != nil {
		return nil, err
	}
	if doc.CompanyID != companyID {
		return nil, domain.ErrDocumentNotFound
	}

	co, err := uc.company.GetCompany(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("pdf: empresa: %w", err)
	}

	nr, err := uc.numbering.GetByID(ctx, doc.NumberingRangeID)
	if err != nil {
		return nil, fmt.Errorf("pdf: rango: %w", err)
	}

	base, err := buildPDFData(doc, co, nr)
	if err != nil {
		return nil, err
	}

	var (
		templateID string
		data       map[string]any
	)

	if formatStr == "half_a4" {
		templateID = "invoice_halfpage"
		data = map[string]any{
			"Copy1": mergeSlip(base, "Original · Comercio"),
			"Copy2": mergeSlip(base, "Copia · Cliente"),
		}
	} else {
		templateID = "invoice_document"
		data = base
	}

	return uc.renderer.Render(ctx, reportsdomain.RenderRequest{
		TemplateID: templateID,
		Data:       data,
		Format:     reportsdomain.FormatPDF,
	})
}

// ── construcción de datos del template ───────────────────────────────────────

func buildPDFData(doc *domain.Document, co *domain.CompanyInfo, nr *domain.NumberingRange) (map[string]any, error) {
	// Logo como template.URL (html/template no HTML-escapa data: URIs de este tipo)
	var logoURL template.URL
	if len(co.Logo) > 0 {
		mime := "image/png"
		if co.LogoContentType != "" {
			mime = co.LogoContentType
		}
		logoURL = template.URL("data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(co.Logo))
	}

	// QR solo en documentos confirmados
	var qr template.URL
	if doc.Status != domain.StatusDraft && doc.QRURL != "" {
		var err error
		qr, err = makeQR(doc.QRURL)
		if err != nil {
			return nil, err
		}
	}

	// Título y etiquetas por tipo de documento DIAN
	docTitle := map[string]string{
		"01": "FACTURA ELECTRÓNICA DE VENTA",
		"91": "NOTA CRÉDITO ELECTRÓNICA",
		"92": "NOTA DÉBITO ELECTRÓNICA",
		"05": "DOCUMENTO SOPORTE EN ADQUISICIONES",
		"95": "NOTA DE AJUSTE AL DOCUMENTO SOPORTE",
	}[doc.DianDocumentTypeCode]
	if docTitle == "" {
		docTitle = "DOCUMENTO ELECTRÓNICO"
	}

	hashLabel := map[string]string{
		"01": "CUFE", "91": "CUDE", "92": "CUDE", "05": "CUDS", "95": "CUDS",
	}[doc.DianDocumentTypeCode]
	if hashLabel == "" {
		hashLabel = "CUFE"
	}

	numLabel := map[string]string{
		"01": "N° de Factura",
		"91": "N° de Nota Crédito",
		"92": "N° de Nota Débito",
		"05": "N° de Doc. Soporte",
		"95": "N° de Nota de Ajuste",
	}[doc.DianDocumentTypeCode]
	if numLabel == "" {
		numLabel = "N° de Documento"
	}

	docNumber := "BORRADOR"
	if doc.Status != domain.StatusDraft {
		docNumber = fmt.Sprintf("%s%d", doc.Prefix, doc.Number)
	}

	runes := []rune(strings.ToLower(docTitle))
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}
	docTitleLower := string(runes)

	// Tercero (adquiriente o proveedor en DS/NA)
	counterparty := doc.Customer
	if doc.Vendor != nil {
		counterparty = *doc.Vendor
	}

	// Pago
	var paymentTerm, paymentMethod, dueDate string
	if len(doc.PaymentMeans) > 0 {
		pm := doc.PaymentMeans[0]
		paymentTerm = paymentTermName(pm.Code)
		paymentMethod = paymentMethodName(pm.PaymentMethodCode)
		dueDate = pm.DueDate
	}

	// Líneas e impuestos
	lines := buildPDFLines(doc.Lines)
	taxes, totalTaxCents := aggregatePDFTaxes(doc.Lines)

	// Totales
	subtotalCents := doc.Totals.LineExtensionCents
	grandTotalCents := doc.Totals.PayableCents
	if grandTotalCents == 0 {
		grandTotalCents = subtotalCents + totalTaxCents
	}

	// Texto de resolución
	var resolutionText string
	if nr.ResolutionNumber != "" {
		resolutionText = buildResolutionText(nr)
	}

	// Régimen y responsabilidades del emisor
	var taxRegime string
	if co.TaxRegimeCode != nil {
		taxRegime = "Régimen tributario: " + *co.TaxRegimeCode
	}

	// Documento afectado
	var billingRefNumber, billingRefDate, billingRefCUFE, discrepancyCode, discrepancyDesc string
	if doc.BillingReference != nil {
		billingRefNumber = doc.BillingReference.Prefix + doc.BillingReference.Number
		billingRefDate = doc.BillingReference.IssueDate
		billingRefCUFE = doc.BillingReference.CUFE
	}
	if doc.DiscrepancyResponse != nil {
		discrepancyCode = doc.DiscrepancyResponse.ResponseCode
		discrepancyDesc = doc.DiscrepancyResponse.Description
	}

	return map[string]any{
		"DocumentTitle":  docTitle,
		"DocTitleLower":  docTitleLower,
		"HashLabel":      hashLabel,
		"DocNumberLabel": numLabel,

		"BusinessName": co.BusinessName,
		"NIT":          co.NIT,
		"CheckDigit":   co.CheckDigit,
		"Address":      co.AddressLine,
		"City":         co.MunicipalityCode,
		"Phone":        co.Phone,
		"Email":        co.Email,
		"LogoDataURL":  logoURL,

		"IsDraft":   doc.Status == domain.StatusDraft,
		"IsDS":      doc.DianDocumentTypeCode == "05" || doc.DianDocumentTypeCode == "95",
		"DocNumber": docNumber,
		"CUFE":      doc.DocumentKey,
		"QR":        qr,
		"IssueDate": doc.IssueDate.Format("02/01/2006"),
		"IssueTime": func() string {
			t := doc.IssueTime
			if len(t) > 8 {
				t = t[:8] // quitar offset de zona horaria "-05:00"
			}
			return t
		}(),
		"Currency":  doc.CurrencyCode,

		"CustomerName":    counterparty.Name,
		"CustomerIDType":  idTypeName(counterparty.Identification.TypeCode),
		"CustomerID":      formatPartyID(counterparty),
		"CustomerAddress": counterparty.Address.Line,
		"CustomerPhone":   counterparty.Phone,
		"CustomerEmail":   counterparty.Email,

		"PaymentTerm":   paymentTerm,
		"PaymentMethod": paymentMethod,
		"DueDate":       dueDate,

		"Lines": lines,
		"Taxes": taxes,

		"Subtotal":      formatCOP(subtotalCents),
		"TaxTotal":      formatCOP(totalTaxCents),
		"GrandTotal":    formatCOP(grandTotalCents),
		"AmountInWords": pdfAmountInWords(grandTotalCents / 100),

		"Note":           doc.Note,
		"ResolutionText": resolutionText,

		"TaxRegime":        taxRegime,
		"Liabilities":      strings.Join(co.LiabilityCodes, " · "),
		"EconomicActivity": strings.Join(co.IndustryClassificationCodes, ", "),
		"BrandColor":       "",

		"BillingRefNumber": billingRefNumber,
		"BillingRefDate":   billingRefDate,
		"BillingRefCUFE":   billingRefCUFE,
		"DiscrepancyCode":  discrepancyCode,
		"DiscrepancyDesc":  discrepancyDesc,
	}, nil
}

func mergeSlip(base map[string]any, label string) map[string]any {
	m := make(map[string]any, len(base)+1)
	for k, v := range base {
		m[k] = v
	}
	m["CopyLabel"] = label
	return m
}

func buildPDFLines(lines []cofdom.Line) []map[string]any {
	out := make([]map[string]any, len(lines))
	for i, l := range lines {
		taxRate := "—"
		if len(l.Taxes) > 0 && l.Taxes[0].Percent > 0 {
			taxRate = fmt.Sprintf("%.0f%%", l.Taxes[0].Percent)
		}
		out[i] = map[string]any{
			"Num":         i + 1,
			"Code":        l.ItemCode,
			"Description": l.Description,
			"Qty":         formatQty(l.Quantity),
			"UnitPrice":   formatCOP(l.UnitPriceCents),
			"TaxRate":     taxRate,
			"LineTotal":   formatCOP(l.LineExtensionCents),
		}
	}
	return out
}

func aggregatePDFTaxes(lines []cofdom.Line) ([]map[string]any, int64) {
	type key struct {
		code string
		pct  float64
	}
	type agg struct {
		name   string
		base   int64
		amount int64
	}

	var keys []key
	m := map[key]*agg{}
	for _, line := range lines {
		for _, t := range line.Taxes {
			k := key{t.TypeCode, t.Percent}
			if _, ok := m[k]; !ok {
				keys = append(keys, k)
				m[k] = &agg{name: t.TypeName}
			}
			m[k].base += t.TaxableAmountCents
			m[k].amount += t.TaxAmountCents
		}
	}

	var totalTax int64
	result := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		a := m[k]
		totalTax += a.amount
		name := a.name
		if name == "" {
			name = k.code
		}
		rate := "—"
		if k.pct > 0 {
			rate = fmt.Sprintf("%.0f%%", k.pct)
		}
		result = append(result, map[string]any{
			"Name":      name,
			"Rate":      rate,
			"TaxBase":   formatCOP(a.base),
			"TaxAmount": formatCOP(a.amount),
		})
	}
	return result, totalTax
}

// ── helpers de formato ────────────────────────────────────────────────────────

func makeQR(content string) (template.URL, error) {
	png, err := qrcode.Encode(content, qrcode.Medium, 180)
	if err != nil {
		return "", fmt.Errorf("pdf: QR: %w", err)
	}
	return template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(png)), nil
}

func formatCOP(cents int64) string {
	pesos := cents / 100
	neg := pesos < 0
	if neg {
		pesos = -pesos
	}
	s := fmt.Sprintf("%d", pesos)
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, c)
	}
	r := "$ " + string(out)
	if neg {
		r = "-" + r
	}
	return r
}

func formatQty(q float64) string {
	if q == float64(int64(q)) {
		return fmt.Sprintf("%d", int64(q))
	}
	return fmt.Sprintf("%.2f", q)
}

func formatPartyID(p cofdom.Party) string {
	if p.Identification.TypeCode == "31" && p.Identification.VerificationCode != "" {
		return p.Identification.Number + "-" + p.Identification.VerificationCode
	}
	return p.Identification.Number
}

func idTypeName(typeCode string) string {
	switch typeCode {
	case "13":
		return "CC"
	case "31":
		return "NIT"
	case "22":
		return "CE"
	case "41":
		return "Pasaporte"
	case "12":
		return "Tarjeta de identidad"
	case "11":
		return "Registro civil"
	case "91":
		return "NUIP"
	default:
		return typeCode
	}
}

func paymentTermName(code string) string {
	switch code {
	case "1":
		return "Contado"
	case "2":
		return "Crédito"
	default:
		return code
	}
}

func paymentMethodName(code string) string {
	switch code {
	case "10":
		return "Efectivo"
	case "42":
		return "Transferencia bancaria"
	case "20":
		return "Cheque"
	case "48":
		return "Tarjeta crédito"
	case "49":
		return "Tarjeta débito"
	case "1":
		return "Instrumento no definido"
	case "ZZZ":
		return "Otro"
	default:
		return code
	}
}

func buildResolutionText(nr *domain.NumberingRange) string {
	rangeClause := fmt.Sprintf("con prefijo %s, desde %d", nr.Prefix, nr.RangeFrom)
	if nr.RangeTo != nil {
		rangeClause += fmt.Sprintf(" hasta %d", *nr.RangeTo)
	}
	return fmt.Sprintf(
		"Autorización de numeración No. %s vigente desde %s hasta %s. Rango autorizado %s.",
		nr.ResolutionNumber,
		nr.ValidFrom.Format("2006-01-02"),
		nr.ValidTo.Format("2006-01-02"),
		rangeClause,
	)
}

// ── monto en letras (español colombiano) ─────────────────────────────────────

func pdfAmountInWords(pesos int64) string {
	if pesos == 0 {
		return "CERO PESOS M/CTE"
	}
	return strings.ToUpper(strings.TrimSpace(pdfNumberToWords(pesos))) + " PESOS M/CTE"
}

var (
	pdfUnidades     = []string{"", "uno", "dos", "tres", "cuatro", "cinco", "seis", "siete", "ocho", "nueve"}
	pdfDec10_19     = []string{"diez", "once", "doce", "trece", "catorce", "quince", "dieciséis", "diecisiete", "dieciocho", "diecinueve"}
	pdfDecenas      = []string{"", "", "veinte", "treinta", "cuarenta", "cincuenta", "sesenta", "setenta", "ochenta", "noventa"}
	pdfCentenas     = []string{"", "ciento", "doscientos", "trescientos", "cuatrocientos", "quinientos", "seiscientos", "setecientos", "ochocientos", "novecientos"}
)

func pdfNumberToWords(n int64) string {
	if n == 0 {
		return "cero"
	}
	if n < 0 {
		return "menos " + pdfNumberToWords(-n)
	}
	switch {
	case n < 1000:
		return pdfHundreds(n)
	case n < 1_000_000:
		return pdfGroup(n, 1000, "mil", "mil")
	case n < 1_000_000_000:
		return pdfGroup(n, 1_000_000, "un millón", "millones")
	default:
		return pdfGroup(n, 1_000_000_000, "mil millones", "mil millones")
	}
}

func pdfGroup(n, unit int64, singular, label string) string {
	count, rest := n/unit, n%unit
	var prefix string
	switch {
	case unit == 1000 && count == 1:
		prefix = "mil"
	case unit == 1000:
		prefix = pdfHundreds(count) + " " + label
	case count == 1:
		prefix = singular
	default:
		prefix = pdfNumberToWords(count) + " " + label
	}
	if rest == 0 {
		return prefix
	}
	if unit == 1000 {
		return prefix + " " + pdfHundreds(rest)
	}
	return prefix + " " + pdfNumberToWords(rest)
}

func pdfHundreds(n int64) string {
	if n == 100 {
		return "cien"
	}
	h, rest := n/100, n%100
	var parts []string
	if h > 0 {
		parts = append(parts, pdfCentenas[h])
	}
	if rest > 0 {
		parts = append(parts, pdfTens(rest))
	}
	return strings.Join(parts, " ")
}

func pdfTens(n int64) string {
	if n < 10 {
		return pdfUnidades[n]
	}
	if n < 20 {
		return pdfDec10_19[n-10]
	}
	d, u := n/10, n%10
	if u == 0 {
		return pdfDecenas[d]
	}
	if d == 2 {
		return "veinti" + pdfUnidades[u]
	}
	return pdfDecenas[d] + " y " + pdfUnidades[u]
}
