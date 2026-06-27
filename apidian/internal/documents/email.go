package documents

import (
	"context"
	"fmt"
	"html"

	"github.com/diegofxm/apidian/internal/email"
	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/google/uuid"
)

// SendInvoiceEmail envía la Factura ya aceptada por la DIAN al correo del cliente, con el PDF
// (representación gráfica, ver RenderInvoicePDF) y el XML firmado adjuntos — ver
// docs/apidian-architecture.md sección 9.42.
//
// Solo Factura por ahora (ErrEmailNotSupportedForDocumentType en cualquier otro caso, mismo
// alcance que RenderInvoicePDF) y solo StatusAccepted: nunca un borrador, nunca uno
// rechazado/con error de envío (no es una factura válida), nunca StatusSent (resultado de la
// DIAN todavía desconocido, podría acabar rechazado).
func (s *Service) SendInvoiceEmail(ctx context.Context, issuerID, id uuid.UUID) error {
	d, iss, err := s.loadInvoiceAndIssuer(ctx, issuerID, id, ErrEmailNotSupportedForDocumentType)
	if err != nil {
		return err
	}
	if d.Status != StatusAccepted {
		return ErrDocumentNotAccepted
	}
	if d.Customer.Email == "" {
		return ErrCustomerEmailMissing
	}

	pdfBytes, err := s.RenderInvoicePDF(ctx, issuerID, id)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s%d", d.Prefix, d.Number)
	msg := email.Message{
		To:       d.Customer.Email,
		Subject:  fmt.Sprintf("Factura electrónica %s - %s", filename, iss.BusinessName),
		BodyText: invoiceEmailText(d, iss),
		BodyHTML: invoiceEmailHTML(d, iss),
		Attachments: []email.Attachment{
			{Filename: filename + ".pdf", ContentType: "application/pdf", Content: pdfBytes},
			{Filename: filename + ".xml", ContentType: "application/xml", Content: []byte(d.SignedXML)},
		},
	}
	return s.email.Send(ctx, msg)
}

func invoiceEmailText(d *Document, iss *issuers.Issuer) string {
	return fmt.Sprintf(
		"Hola %s,\n\n"+
			"Adjuntamos tu factura electrónica No. %s%d, emitida por %s.\n\n"+
			"En este correo encontrarás:\n"+
			"- El PDF: representación gráfica de la factura.\n"+
			"- El XML: el documento electrónico firmado, válido ante la DIAN.\n\n"+
			"Este es un mensaje automático, por favor no respondas a esta dirección.\n",
		d.Customer.Name, d.Prefix, d.Number, iss.BusinessName,
	)
}

// invoiceEmailHTML escapa los campos interpolados (Customer.Name/iss.BusinessName son datos
// del cliente/snapshot, no se confía en que no traigan caracteres HTML especiales).
func invoiceEmailHTML(d *Document, iss *issuers.Issuer) string {
	return fmt.Sprintf(
		"<p>Hola %s,</p>"+
			"<p>Adjuntamos tu factura electrónica No. %s%d, emitida por %s.</p>"+
			"<p>En este correo encontrarás:</p>"+
			"<ul><li>El PDF: representación gráfica de la factura.</li>"+
			"<li>El XML: el documento electrónico firmado, válido ante la DIAN.</li></ul>"+
			"<p>Este es un mensaje automático, por favor no respondas a esta dirección.</p>",
		html.EscapeString(d.Customer.Name), html.EscapeString(d.Prefix), d.Number, html.EscapeString(iss.BusinessName),
	)
}
