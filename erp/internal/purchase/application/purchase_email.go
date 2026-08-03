package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/purchase/domain"
	notificationdomain "github.com/diegofxm/erp/internal/shared/notification/domain"
	reportsdomain "github.com/diegofxm/erp/internal/shared/reports/domain"
	supplierdomain "github.com/diegofxm/erp/internal/supplier/domain"
)

// SendPurchaseOrderEmailUseCase genera el PDF de una orden de compra y lo envía por correo al
// proveedor — espejo de SendQuoteEmailUseCase. A diferencia de la cotización (que pasa a "sent"
// al enviarse), la orden de compra no tiene un estado "enviada" propio: debe estar confirmada
// (o ya recibida) para poder mandarse, pero enviarla no cambia su estado.
type SendPurchaseOrderEmailUseCase struct {
	purchases domain.Repository
	suppliers supplierdomain.Repository
	company   domain.CompanyPort
	renderer  reportsdomain.Renderer
	notifier  notificationdomain.Notifier
	baseURL   string
}

func NewSendPurchaseOrderEmailUseCase(
	purchases domain.Repository,
	suppliers supplierdomain.Repository,
	company domain.CompanyPort,
	renderer reportsdomain.Renderer,
	notifier notificationdomain.Notifier,
	baseURL string,
) *SendPurchaseOrderEmailUseCase {
	return &SendPurchaseOrderEmailUseCase{
		purchases: purchases, suppliers: suppliers, company: company,
		renderer: renderer, notifier: notifier, baseURL: baseURL,
	}
}

// Send envía la orden de compra por correo. to vacío → usa el correo registrado del proveedor.
func (uc *SendPurchaseOrderEmailUseCase) Send(ctx context.Context, companyID, purchaseID uuid.UUID, to string) error {
	o, err := uc.purchases.GetByID(ctx, companyID, purchaseID)
	if err != nil {
		return err
	}
	if o.Status != domain.StatusConfirmed && o.Status != domain.StatusReceived {
		return fmt.Errorf("la orden debe estar confirmada para poder enviarse al proveedor")
	}

	supplier, err := uc.suppliers.GetByID(ctx, companyID, o.SupplierID)
	if err != nil {
		return fmt.Errorf("purchase email: proveedor: %w", err)
	}

	recipient := to
	if recipient == "" {
		recipient = supplier.Email
	}
	if recipient == "" {
		return fmt.Errorf("el proveedor no tiene correo electrónico configurado")
	}

	co, err := uc.company.GetCompany(ctx, companyID)
	if err != nil {
		return fmt.Errorf("purchase email: empresa: %w", err)
	}

	pdfData := buildPurchasePDFData(o, supplier, co)
	pdfBytes, err := uc.renderer.Render(ctx, reportsdomain.RenderRequest{
		TemplateID: "purchase_order_document",
		Data:       pdfData,
		Format:     reportsdomain.FormatPDF,
	})
	if err != nil {
		return fmt.Errorf("purchase email: generar PDF: %w", err)
	}

	docNumber := o.Number
	if docNumber == "" {
		docNumber = "borrador"
	}

	tradeName := co.TradeName
	if tradeName == "" {
		tradeName = co.BusinessName
	}

	msg := notificationdomain.Message{
		To:         recipient,
		Channel:    notificationdomain.ChannelEmail,
		Subject:    fmt.Sprintf("Orden de Compra %s — %s", docNumber, tradeName),
		TemplateID: "purchase_order_issued",
		Data: map[string]any{
			"LogoSrc":         purchaseCompanyLogoURL(uc.baseURL, companyID, len(co.Logo) > 0),
			"IssuerName":      co.BusinessName,
			"IssuerTradeName": co.TradeName,
			"IssuerNIT":       co.NIT,
			"IssuerEmail":     co.Email,
			"RecipientName":   supplier.Name,
			"DocumentNumber":  docNumber,
			"IssueDate":       formatPurchaseDateES(o.IssueDate),
			"TotalFormatted":  formatPurchasePDFCOP(o.GrandTotal()),
		},
		Attachments: []notificationdomain.Attachment{
			{
				Filename:    fmt.Sprintf("orden-compra-%s.pdf", docNumber),
				ContentType: "application/pdf",
				Content:     pdfBytes,
			},
		},
	}

	if err := uc.notifier.Send(ctx, msg); err != nil {
		return fmt.Errorf("purchase email: enviar: %w", err)
	}
	return nil
}

func purchaseCompanyLogoURL(baseURL string, companyID uuid.UUID, hasLogo bool) string {
	if baseURL == "" || !hasLogo {
		return ""
	}
	return baseURL + "/api/v1/public/companies/" + companyID.String() + "/logo"
}

var purchaseMonthsES = [12]string{
	"enero", "febrero", "marzo", "abril", "mayo", "junio",
	"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
}

func formatPurchaseDateES(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return fmt.Sprintf("%d de %s de %d", t.Day(), purchaseMonthsES[t.Month()-1], t.Year())
}
