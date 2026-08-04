package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/sales/domain"
	notificationdomain "github.com/diegofxm/erp/internal/shared/notification/domain"
	reportsdomain "github.com/diegofxm/erp/internal/shared/reports/domain"
)

// SendQuoteEmailUseCase genera el PDF de una cotización en borrador y lo envía por correo al
// cliente, dejando la cotización en estado "sent" — es la implementación real detrás del botón
// "Enviar" (antes solo cambiaba el estado sin mandar nada).
type SendQuoteEmailUseCase struct {
	quotes    domain.QuoteRepository
	customers domain.CustomerPort
	company   domain.CompanyPort
	renderer  reportsdomain.Renderer
	notifier  notificationdomain.Notifier
	baseURL   string
}

func NewSendQuoteEmailUseCase(
	quotes domain.QuoteRepository,
	customers domain.CustomerPort,
	company domain.CompanyPort,
	renderer reportsdomain.Renderer,
	notifier notificationdomain.Notifier,
	baseURL string,
) *SendQuoteEmailUseCase {
	return &SendQuoteEmailUseCase{
		quotes: quotes, customers: customers, company: company,
		renderer: renderer, notifier: notifier, baseURL: baseURL,
	}
}

// Send envía la cotización por correo. to vacío → usa el correo registrado del cliente.
// notificationdomain.Message no soporta CC todavía (ver electronic/send_email.go, mismo
// límite), así que este use case tampoco lo expone.
func (uc *SendQuoteEmailUseCase) Send(ctx context.Context, companyID, quoteID uuid.UUID, to string) (*domain.Quote, error) {
	q, err := uc.quotes.GetByID(ctx, companyID, quoteID)
	if err != nil {
		return nil, err
	}
	if q.Status != domain.QuoteStatusDraft {
		return nil, domain.ErrQuoteNotDraft
	}

	customer, err := uc.customers.GetByID(ctx, companyID, q.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("quote email: cliente: %w", err)
	}

	recipient := to
	if recipient == "" {
		recipient = customer.Email
	}
	if recipient == "" {
		return nil, fmt.Errorf("el cliente no tiene correo electrónico configurado")
	}

	co, err := uc.company.GetCompany(ctx, companyID)
	if err != nil {
		return nil, fmt.Errorf("quote email: empresa: %w", err)
	}

	pdfData := buildQuotePDFData(q, customer, co)
	pdfBytes, err := uc.renderer.Render(ctx, reportsdomain.RenderRequest{
		TemplateID: "quote_document",
		Data:       pdfData,
		Format:     reportsdomain.FormatPDF,
	})
	if err != nil {
		return nil, fmt.Errorf("quote email: generar PDF: %w", err)
	}

	docNumber := q.Number
	if docNumber == "" {
		docNumber = "borrador"
	}

	_, _, total := q.GrandTotal()

	validUntil := "sin fecha límite"
	if q.ValidUntil != nil {
		validUntil = formatQuoteDateES(*q.ValidUntil)
	}

	tradeName := co.TradeName
	if tradeName == "" {
		tradeName = co.BusinessName
	}

	msg := notificationdomain.Message{
		To:         recipient,
		Channel:    notificationdomain.ChannelEmail,
		Subject:    fmt.Sprintf("Cotización %s — %s", docNumber, tradeName),
		TemplateID: "quote_issued",
		Data: map[string]any{
			"LogoSrc":         quoteCompanyLogoURL(uc.baseURL, companyID, len(co.Logo) > 0),
			"IssuerName":      co.BusinessName,
			"IssuerTradeName": co.TradeName,
			"IssuerNIT":       co.NIT,
			"IssuerEmail":     co.Email,
			"RecipientName":   customer.Name,
			"DocumentNumber":  docNumber,
			"IssueDate":       formatQuoteDateES(q.IssueDate),
			"ValidUntil":      validUntil,
			"TotalFormatted":  formatQuotePDFCOP(total),
		},
		Attachments: []notificationdomain.Attachment{
			{
				Filename:    fmt.Sprintf("cotizacion-%s.pdf", docNumber),
				ContentType: "application/pdf",
				Content:     pdfBytes,
			},
		},
	}

	if err := uc.notifier.Send(ctx, msg); err != nil {
		return nil, fmt.Errorf("quote email: enviar: %w", err)
	}

	if err := uc.quotes.UpdateStatus(ctx, companyID, quoteID, domain.QuoteStatusSent); err != nil {
		return nil, fmt.Errorf("quote email: correo enviado pero no se pudo actualizar el estado: %w", err)
	}
	q.Status = domain.QuoteStatusSent
	return q, nil
}

func quoteCompanyLogoURL(baseURL string, companyID uuid.UUID, hasLogo bool) string {
	if baseURL == "" || !hasLogo {
		return ""
	}
	return baseURL + "/api/v1/public/companies/" + companyID.String() + "/logo"
}

var quoteMonthsES = [12]string{
	"enero", "febrero", "marzo", "abril", "mayo", "junio",
	"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
}

func formatQuoteDateES(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return fmt.Sprintf("%d de %s de %d", t.Day(), quoteMonthsES[t.Month()-1], t.Year())
}
