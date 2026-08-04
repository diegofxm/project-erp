package application

import (
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"strings"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/purchase/domain"
	reportsdomain "github.com/diegofxm/erp/internal/shared/reports/domain"
)

// GetPurchaseOrderPDFUseCase genera la representación gráfica (PDF) de una orden de compra.
// Igual que GetQuotePDFUseCase en sales: no hay CUFE/QR/firma — es un documento comercial
// interno hacia el proveedor, no un documento electrónico DIAN.
type GetPurchaseOrderPDFUseCase struct {
	purchases domain.Repository
	suppliers domain.SupplierPort
	company   domain.CompanyPort
	renderer  reportsdomain.Renderer
}

func NewGetPurchaseOrderPDFUseCase(
	purchases domain.Repository,
	suppliers domain.SupplierPort,
	company domain.CompanyPort,
	renderer reportsdomain.Renderer,
) *GetPurchaseOrderPDFUseCase {
	return &GetPurchaseOrderPDFUseCase{purchases: purchases, suppliers: suppliers, company: company, renderer: renderer}
}

func (uc *GetPurchaseOrderPDFUseCase) GetPDF(ctx context.Context, companyID, purchaseID uuid.UUID) ([]byte, error) {
	data, err := uc.buildData(ctx, companyID, purchaseID)
	if err != nil {
		return nil, err
	}
	return uc.renderer.Render(ctx, reportsdomain.RenderRequest{
		TemplateID: "purchase_order_document",
		Data:       data,
		Format:     reportsdomain.FormatPDF,
	})
}

func (uc *GetPurchaseOrderPDFUseCase) buildData(ctx context.Context, companyID, purchaseID uuid.UUID) (map[string]any, error) {
	o, err := uc.purchases.GetByID(ctx, companyID, purchaseID)
	if err != nil {
		return nil, err
	}
	supplier, err := uc.suppliers.GetByID(ctx, companyID, o.SupplierID)
	if err != nil {
		return nil, fmt.Errorf("purchase pdf: proveedor: %w", err)
	}
	co, err := uc.company.GetCompany(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("purchase pdf: empresa: %w", err)
	}
	return buildPurchasePDFData(o, supplier, co), nil
}

func buildPurchasePDFData(o *domain.PurchaseOrder, supplier *domain.Supplier, co *domain.CompanyInfo) map[string]any {
	var logoURL template.URL
	if len(co.Logo) > 0 {
		mime := "image/png"
		if co.LogoContentType != "" {
			mime = co.LogoContentType
		}
		logoURL = template.URL("data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(co.Logo))
	}

	docNumber := o.Number
	if docNumber == "" {
		docNumber = "BORRADOR"
	}

	dueDate := "—"
	if o.DueDate != nil {
		dueDate = o.DueDate.Format("02/01/2006")
	}

	lines := make([]map[string]any, len(o.Lines))
	var subtotal, taxTotal, grandTotal float64
	for i, l := range o.Lines {
		taxRate := "—"
		if l.TaxRate > 0 {
			taxRate = fmt.Sprintf("%.0f%%", l.TaxRate)
		}
		lines[i] = map[string]any{
			"Num":         i + 1,
			"Description": l.Description,
			"Qty":         formatPurchasePDFQty(l.Quantity),
			"UnitPrice":   formatPurchasePDFCOP(l.UnitPrice),
			"TaxRate":     taxRate,
			"LineTotal":   formatPurchasePDFCOP(l.Total),
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
		"IsDraft":    o.Status == domain.StatusDraft,
		"StatusName": purchaseStatusNameES(o.Status),
		"IssueDate":  o.IssueDate.Format("02/01/2006"),
		"DueDate":    dueDate,

		"SupplierName":    supplier.Name,
		"SupplierIDType":  purchaseIDTypeName(supplier.IdentificationTypeCode),
		"SupplierID":      purchaseFormatSupplierID(supplier),
		"SupplierAddress": supplier.AddressLine,
		"SupplierPhone":   supplier.Phone,
		"SupplierEmail":   supplier.Email,

		"Lines": lines,

		"Subtotal":   formatPurchasePDFCOP(subtotal),
		"TaxTotal":   formatPurchasePDFCOP(taxTotal),
		"GrandTotal": formatPurchasePDFCOP(grandTotal),

		"Note": o.Notes,
	}
}

func purchaseStatusNameES(s domain.PurchaseStatus) string {
	switch s {
	case domain.StatusDraft:
		return "Borrador"
	case domain.StatusConfirmed:
		return "Confirmada"
	case domain.StatusReceived:
		return "Recibida"
	case domain.StatusCancelled:
		return "Cancelada"
	default:
		return string(s)
	}
}

func purchaseFormatSupplierID(s *domain.Supplier) string {
	if s.IdentificationTypeCode == "31" && s.CheckDigit != "" {
		return s.IdentificationNumber + "-" + s.CheckDigit
	}
	return s.IdentificationNumber
}

func purchaseIDTypeName(typeCode string) string {
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

func formatPurchasePDFCOP(pesos float64) string {
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

func formatPurchasePDFQty(q float64) string {
	if q == float64(int64(q)) {
		return fmt.Sprintf("%d", int64(q))
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", q), "0"), ".")
}
