package application

import (
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"strings"

	"github.com/google/uuid"

	customerdomain "github.com/diegofxm/erp/internal/customer/domain"
	"github.com/diegofxm/erp/internal/sales/domain"
	reportsdomain "github.com/diegofxm/erp/internal/shared/reports/domain"
)

// GetQuotePDFUseCase genera la representación gráfica (PDF) de una cotización.
// A diferencia de electronic, no hay CUFE/QR/firma — es un documento comercial interno,
// no un documento electrónico DIAN.
type GetQuotePDFUseCase struct {
	quotes    domain.QuoteRepository
	customers customerdomain.Repository
	company   domain.CompanyPort
	renderer  reportsdomain.Renderer
}

func NewGetQuotePDFUseCase(
	quotes domain.QuoteRepository,
	customers customerdomain.Repository,
	company domain.CompanyPort,
	renderer reportsdomain.Renderer,
) *GetQuotePDFUseCase {
	return &GetQuotePDFUseCase{quotes: quotes, customers: customers, company: company, renderer: renderer}
}

func (uc *GetQuotePDFUseCase) GetPDF(ctx context.Context, companyID, quoteID uuid.UUID) ([]byte, error) {
	data, err := uc.buildData(ctx, companyID, quoteID)
	if err != nil {
		return nil, err
	}
	return uc.renderer.Render(ctx, reportsdomain.RenderRequest{
		TemplateID: "quote_document",
		Data:       data,
		Format:     reportsdomain.FormatPDF,
	})
}

func (uc *GetQuotePDFUseCase) buildData(ctx context.Context, companyID, quoteID uuid.UUID) (map[string]any, error) {
	q, err := uc.quotes.GetByID(ctx, companyID, quoteID)
	if err != nil {
		return nil, err
	}
	customer, err := uc.customers.GetByID(ctx, companyID, q.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("quote pdf: cliente: %w", err)
	}
	co, err := uc.company.GetCompany(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("quote pdf: empresa: %w", err)
	}
	return buildQuotePDFData(q, customer, co), nil
}

func buildQuotePDFData(q *domain.Quote, customer *customerdomain.Customer, co *domain.CompanyInfo) map[string]any {
	var logoURL template.URL
	if len(co.Logo) > 0 {
		mime := "image/png"
		if co.LogoContentType != "" {
			mime = co.LogoContentType
		}
		logoURL = template.URL("data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(co.Logo))
	}

	docNumber := q.Number
	if docNumber == "" {
		docNumber = "BORRADOR"
	}

	validUntil := "—"
	if q.ValidUntil != nil {
		validUntil = q.ValidUntil.Format("02/01/2006")
	}

	lines := make([]map[string]any, len(q.Lines))
	var subtotal, taxTotal, grandTotal float64
	for i, l := range q.Lines {
		taxRate := "—"
		if l.TaxRate > 0 {
			taxRate = fmt.Sprintf("%.0f%%", l.TaxRate)
		}
		lines[i] = map[string]any{
			"Num":         i + 1,
			"Description": l.Description,
			"Qty":         formatQuotePDFQty(l.Quantity),
			"UnitPrice":   formatQuotePDFCOP(l.UnitPrice),
			"TaxRate":     taxRate,
			"LineTotal":   formatQuotePDFCOP(l.Total),
		}
		subtotal += l.Subtotal
		taxTotal += l.TaxAmount
		grandTotal += l.Total
	}

	return map[string]any{
		"BusinessName": co.BusinessName,
		"NIT":          co.NIT,
		"CheckDigit":   co.CheckDigit,
		"Address":      co.AddressLine,
		"City":         co.MunicipalityCode,
		"Phone":        co.Phone,
		"Email":        co.Email,
		"LogoDataURL":  logoURL,

		"DocNumber":  docNumber,
		"IsDraft":    q.Status == domain.QuoteStatusDraft,
		"StatusName": quoteStatusNameES(q.Status),
		"IssueDate":  q.IssueDate.Format("02/01/2006"),
		"ValidUntil": validUntil,

		"CustomerName":    customer.Name,
		"CustomerIDType":  quoteIDTypeName(customer.IdentificationTypeCode),
		"CustomerID":      quoteFormatCustomerID(customer),
		"CustomerAddress": customer.AddressLine,
		"CustomerPhone":   customer.Phone,
		"CustomerEmail":   customer.Email,

		"Lines": lines,

		"Subtotal":   formatQuotePDFCOP(subtotal),
		"TaxTotal":   formatQuotePDFCOP(taxTotal),
		"GrandTotal": formatQuotePDFCOP(grandTotal),

		"Note": q.Notes,
	}
}

func quoteStatusNameES(s domain.QuoteStatus) string {
	switch s {
	case domain.QuoteStatusDraft:
		return "Borrador"
	case domain.QuoteStatusSent:
		return "Enviada"
	case domain.QuoteStatusAccepted:
		return "Aceptada"
	case domain.QuoteStatusRejected:
		return "Rechazada"
	case domain.QuoteStatusExpired:
		return "Vencida"
	default:
		return string(s)
	}
}

func quoteFormatCustomerID(c *customerdomain.Customer) string {
	if c.IdentificationTypeCode == "31" && c.CheckDigit != "" {
		return c.IdentificationNumber + "-" + c.CheckDigit
	}
	return c.IdentificationNumber
}

func quoteIDTypeName(typeCode string) string {
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

func formatQuotePDFCOP(pesos float64) string {
	neg := pesos < 0
	if neg {
		pesos = -pesos
	}
	s := fmt.Sprintf("%.0f", pesos)
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

func formatQuotePDFQty(q float64) string {
	if q == float64(int64(q)) {
		return fmt.Sprintf("%d", int64(q))
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", q), "0"), ".")
}
