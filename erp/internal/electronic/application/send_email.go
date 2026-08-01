package application

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/electronic/domain"
	notificationdomain "github.com/diegofxm/erp/internal/shared/notification/domain"
	reportsdomain "github.com/diegofxm/erp/internal/shared/reports/domain"
)

// SendDocumentEmailUseCase envía por correo el ZIP (AttachedDocument XML + PDF) de un
// documento DIAN al destinatario: cliente para FE/NC/ND, proveedor para DS/NA.
type SendDocumentEmailUseCase struct {
	docs      domain.DocumentRepository
	company   domain.CompanyPort
	numbering domain.NumberingRepository
	renderer  reportsdomain.Renderer
	notifier  notificationdomain.Notifier
	emailZip  domain.EmailZipPort
	baseURL   string // ej. "https://erp.cofacture.co" — para construir la URL pública del logo
}

func NewSendDocumentEmailUseCase(
	docs domain.DocumentRepository,
	company domain.CompanyPort,
	numbering domain.NumberingRepository,
	renderer reportsdomain.Renderer,
	notifier notificationdomain.Notifier,
	emailZip domain.EmailZipPort,
	baseURL string,
) *SendDocumentEmailUseCase {
	return &SendDocumentEmailUseCase{
		docs:      docs,
		company:   company,
		numbering: numbering,
		renderer:  renderer,
		notifier:  notifier,
		emailZip:  emailZip,
		baseURL:   baseURL,
	}
}

// Send genera el ZIP adjunto (AttachedDocument + PDF) y lo envía al destinatario.
func (uc *SendDocumentEmailUseCase) Send(ctx context.Context, companyID, docID uuid.UUID) error {
	doc, err := uc.docs.GetByID(ctx, docID)
	if err != nil {
		return err
	}
	if doc.CompanyID != companyID {
		return domain.ErrDocumentNotFound
	}
	if doc.SignedXML == "" {
		return fmt.Errorf("el documento no tiene XML firmado — confirma el documento antes de enviar el correo")
	}

	co, err := uc.company.GetCompany(ctx, companyID)
	if err != nil {
		return fmt.Errorf("email: empresa: %w", err)
	}

	nr, err := uc.numbering.GetByID(ctx, doc.NumberingRangeID)
	if err != nil {
		return fmt.Errorf("email: rango: %w", err)
	}

	pdfData, err := buildPDFData(doc, co, nr)
	if err != nil {
		return fmt.Errorf("email: datos PDF: %w", err)
	}
	pdfBytes, err := uc.renderer.Render(ctx, reportsdomain.RenderRequest{
		TemplateID: "invoice_document",
		Data:       pdfData,
		Format:     reportsdomain.FormatPDF,
	})
	if err != nil {
		return fmt.Errorf("email: generar PDF: %w", err)
	}

	docNumber := fmt.Sprintf("%s%d", doc.Prefix, doc.Number)

	now := time.Now()
	zipBytes, err := uc.emailZip.BuildEmailZip(doc, co, docNumber, pdfBytes, now)
	if err != nil {
		return fmt.Errorf("email: construir ZIP adjunto: %w", err)
	}

	typeName, recipientEmail, recipientName := emailRecipient(doc)
	if recipientEmail == "" {
		return fmt.Errorf("el destinatario no tiene correo electrónico configurado")
	}

	// Asunto según Anexo Técnico 1.9 §9.1:
	// NIT; Nombre del facturador; Número del documento; Código tipo; Nombre comercial
	tradeName := co.TradeName
	if tradeName == "" {
		tradeName = co.BusinessName
	}
	subject := fmt.Sprintf("%s;%s;%s;%s;%s", co.NIT, co.BusinessName, docNumber, doc.DianDocumentTypeCode, tradeName)

	issuerNIT := co.NIT
	if co.IdentificationTypeCode == "31" {
		issuerNIT = co.NIT + "-" + co.CheckDigit
	}

	msg := notificationdomain.Message{
		To:         recipientEmail,
		Channel:    notificationdomain.ChannelEmail,
		Subject:    subject,
		TemplateID: "invoice_issued",
		Data: map[string]any{
			"LogoSrc":          companyLogoURL(uc.baseURL, companyID, len(co.Logo) > 0),
			"IssuerName":       co.BusinessName,
			"IssuerTradeName":  co.TradeName,
			"IssuerNIT":        issuerNIT,
			"IssuerEmail":      co.Email,
			"RecipientName":    recipientName,
			"DocumentTypeName": typeName,
			"DocumentNumber":   docNumber,
			"IssueDate":        formatDateES(doc.IssueDate),
			"TotalFormatted":   formatCentsCOP(doc.Totals.PayableCents),
		},
		Attachments: []notificationdomain.Attachment{
			{
				Filename:    docNumber + ".zip",
				ContentType: "application/zip",
				Content:     zipBytes,
			},
		},
	}

	return uc.notifier.Send(ctx, msg)
}

// companyLogoURL devuelve la URL pública del logo de la empresa, o "" si no hay logo
// o no está configurado el baseURL del servidor.
func companyLogoURL(baseURL string, companyID uuid.UUID, hasLogo bool) string {
	if baseURL == "" || !hasLogo {
		return ""
	}
	return baseURL + "/api/v1/public/companies/" + companyID.String() + "/logo"
}

// emailRecipient devuelve tipo de documento, correo y nombre del destinatario.
// FE/NC/ND → cliente; DS/NA → proveedor.
func emailRecipient(doc *domain.Document) (typeName, email, name string) {
	switch doc.DianDocumentTypeCode {
	case dianFE:
		return "Factura Electrónica de Venta", doc.Customer.Email, doc.Customer.Name
	case dianNC:
		return "Nota Crédito Electrónica", doc.Customer.Email, doc.Customer.Name
	case dianND:
		return "Nota Débito Electrónica", doc.Customer.Email, doc.Customer.Name
	case dianDS:
		if doc.Supplier != nil {
			return "Documento Soporte en Adquisiciones", doc.Supplier.Email, doc.Supplier.Name
		}
		return "Documento Soporte en Adquisiciones", "", ""
	case dianNA:
		if doc.Supplier != nil {
			return "Nota de Ajuste al Documento Soporte", doc.Supplier.Email, doc.Supplier.Name
		}
		return "Nota de Ajuste al Documento Soporte", "", ""
	default:
		return "Documento Electrónico DIAN", doc.Customer.Email, doc.Customer.Name
	}
}

var monthsES = [12]string{
	"enero", "febrero", "marzo", "abril", "mayo", "junio",
	"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
}

func formatDateES(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return fmt.Sprintf("%d de %s de %d", t.Day(), monthsES[t.Month()-1], t.Year())
}

func formatCentsCOP(cents int64) string {
	pesos := cents / 100
	s := strconv.FormatInt(pesos, 10)
	n := len(s)
	var result []byte
	for i := 0; i < n; i++ {
		if i > 0 && (n-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, s[i])
	}
	return "$ " + string(result)
}
