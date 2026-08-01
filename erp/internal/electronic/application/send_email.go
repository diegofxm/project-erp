package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/electronic/domain"
	notificationdomain "github.com/diegofxm/erp/internal/shared/notification/domain"
	reportsdomain "github.com/diegofxm/erp/internal/shared/reports/domain"
)

// SendDocumentEmailUseCase envía por correo el PDF + XML de un documento DIAN
// al destinatario del documento (cliente para FE/NC/ND, proveedor para DS/NA).
type SendDocumentEmailUseCase struct {
	docs      domain.DocumentRepository
	company   domain.CompanyPort
	numbering domain.NumberingRepository
	renderer  reportsdomain.Renderer
	notifier  notificationdomain.Notifier
}

func NewSendDocumentEmailUseCase(
	docs domain.DocumentRepository,
	company domain.CompanyPort,
	numbering domain.NumberingRepository,
	renderer reportsdomain.Renderer,
	notifier notificationdomain.Notifier,
) *SendDocumentEmailUseCase {
	return &SendDocumentEmailUseCase{
		docs:      docs,
		company:   company,
		numbering: numbering,
		renderer:  renderer,
		notifier:  notifier,
	}
}

// Send genera el PDF y el XML firmado del documento y los envía al destinatario por correo.
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

	typeName, recipientEmail, recipientName := emailRecipient(doc)
	if recipientEmail == "" {
		return fmt.Errorf("el destinatario no tiene correo electrónico configurado")
	}

	docNumber := fmt.Sprintf("%s%d", doc.Prefix, doc.Number)

	msg := notificationdomain.Message{
		To:         recipientEmail,
		Channel:    notificationdomain.ChannelEmail,
		Subject:    typeName + " " + docNumber,
		TemplateID: "invoice_issued",
		Data: map[string]any{
			"DocumentTypeName": typeName,
			"DocumentNumber":   docNumber,
			"RecipientName":    recipientName,
			"IssueDate":        doc.IssueDate.Format("02/01/2006"),
			"CompanyName":      co.BusinessName,
			"CompanyNIT":       co.NIT + "-" + co.CheckDigit,
			"CompanyAddress":   co.AddressLine,
			"Currency":         doc.CurrencyCode,
			"Total":            formatCOP(doc.Totals.PayableCents),
		},
		Attachments: []notificationdomain.Attachment{
			{
				Filename:    docNumber + ".pdf",
				ContentType: "application/pdf",
				Content:     pdfBytes,
			},
			{
				Filename:    docNumber + ".xml",
				ContentType: "application/xml",
				Content:     []byte(doc.SignedXML),
			},
		},
	}

	return uc.notifier.Send(ctx, msg)
}

// emailRecipient devuelve nombre del tipo de documento, correo y nombre del destinatario.
// Para FE/NC/ND el destinatario es el cliente; para DS/NA es el proveedor.
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
