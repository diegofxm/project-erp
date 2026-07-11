package documents

import (
	"context"
	"fmt"
	"html"
	"time"

	"github.com/beevik/etree"
	"github.com/diegofxm/apidian/internal/email"
	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/diegofxm/cofacture/builder"
	"github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/signer"
	cofzip "github.com/diegofxm/cofacture/zip"
	"github.com/google/uuid"
)

// SendDocumentEmail envía el documento ya aceptado por la DIAN al correo del cliente.
//
// El adjunto es un único ZIP que, cuando el documento fue aceptado con la migración 000013
// ya activa, contiene un AttachedDocument UBL firmado (con el XML de la factura y el
// ApplicationResponse de la DIAN embebidos), conforme a la sección 9.1 del Anexo Técnico
// 1.9. Para documentos aceptados antes de esa migración — ApplicationResponseXML vacío —
// el ZIP contiene el XML firmado crudo + el PDF (comportamiento anterior).
//
// Válido para Factura (01), Nota Crédito (91) y Nota Débito (92). Solo StatusAccepted.
func (s *Service) SendDocumentEmail(ctx context.Context, issuerID, id uuid.UUID) error {
	d, iss, err := s.loadDocumentAndIssuer(ctx, issuerID, id)
	if err != nil {
		return err
	}
	if d.Status != StatusAccepted {
		return ErrDocumentNotAccepted
	}
	if d.Customer.Email == "" {
		return ErrCustomerEmailMissing
	}

	pdfBytes, err := s.RenderDocumentPDF(ctx, issuerID, id)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s%d", d.Prefix, d.Number)
	now := time.Now()

	xmlContent, xmlFilename, err := buildAttachedDocumentXML(d, iss, filename, now)
	if err != nil {
		return fmt.Errorf("construir AttachedDocument: %w", err)
	}

	zipBytes, err := cofzip.Build([]cofzip.File{
		{Name: xmlFilename, Content: xmlContent},
		{Name: filename + ".pdf", Content: pdfBytes},
	})
	if err != nil {
		return fmt.Errorf("empacar adjuntos: %w", err)
	}

	typeName := documentTypeName(d.DianDocumentTypeCode)
	tradeName := iss.TradeName
	if tradeName == "" {
		tradeName = iss.BusinessName
	}
	// Asunto según Anexo Técnico 1.9 sección 9.1:
	// NIT; Nombre del facturador; Número del documento; Código tipo; Nombre comercial
	subject := fmt.Sprintf("%s;%s;%s;%s;%s", iss.NIT, iss.BusinessName, filename, d.DianDocumentTypeCode, tradeName)
	msg := email.Message{
		To:       d.Customer.Email,
		Subject:  subject,
		BodyText: documentEmailText(d, iss, typeName),
		BodyHTML: documentEmailHTML(d, iss, typeName),
		Attachments: []email.Attachment{
			{Filename: filename + ".zip", ContentType: "application/zip", Content: zipBytes},
		},
	}
	return s.email.Send(ctx, msg)
}

// buildAttachedDocumentXML construye el XML que va dentro del ZIP del correo al cliente.
//
// Cuando hay ApplicationResponseXML guardado, produce un AttachedDocument UBL firmado
// conforme al Anexo Técnico 1.9 sección 9.1. Sin ApplicationResponseXML (documentos
// aceptados antes de la migración 000013), devuelve el SignedXML crudo como fallback.
func buildAttachedDocumentXML(d *Document, iss *issuers.Issuer, filename string, now time.Time) (xmlBytes []byte, xmlFilename string, err error) {
	if d.ApplicationResponseXML == "" {
		return []byte(d.SignedXML), filename + ".xml", nil
	}

	cert, key, err := signer.LoadPKCS12(iss.Certificate, iss.CertificatePassword)
	if err != nil {
		return nil, "", fmt.Errorf("cargar certificado del emisor: %w", err)
	}

	taxRegime := ""
	if iss.TaxRegimeCode != nil {
		taxRegime = *iss.TaxRegimeCode
	}

	hashType := "CUFE-SHA384"
	if d.DianDocumentTypeCode != invoiceDianDocumentType {
		hashType = "CUDE-SHA384"
	}

	ad := domain.AttachedDocument{
		EnvironmentCode:  string(iss.Environment),
		ID:               filename,
		IssueDate:        now.Format("2006-01-02"),
		IssueTime:        now.Format("15:04:05-07:00"),
		ParentDocumentID: filename,
		Sender: domain.AttachedPartyInfo{
			Name:           iss.BusinessName,
			Identification: domain.Identification{TypeCode: iss.IdentificationTypeCode, Number: iss.NIT},
			TaxRegimeCode:  taxRegime,
			LiabilityCodes: iss.LiabilityCodes,
			TaxSchemeCode:  iss.TaxSchemeCode,
			TaxSchemeName:  iss.TaxSchemeName,
		},
		Receiver: domain.AttachedPartyInfo{
			Name:           d.Customer.Name,
			Identification: domain.Identification{TypeCode: d.Customer.Identification.TypeCode, Number: d.Customer.Identification.Number},
			TaxRegimeCode:  d.Customer.TaxRegimeCode,
			LiabilityCodes: d.Customer.LiabilityCodes,
			TaxSchemeCode:  d.Customer.TaxSchemeCode,
			TaxSchemeName:  d.Customer.TaxSchemeName,
		},
		AttachmentXML: d.SignedXML,
		ValidationResults: []domain.ValidationResult{
			{
				LineID:                 "1",
				DocumentID:             filename,
				DocumentCUFE:           d.DocumentKey,
				DocumentHashType:       hashType,
				DocumentIssueDate:      d.IssueDate.Format("2006-01-02"),
				ApplicationResponseXML: d.ApplicationResponseXML,
				ValidatorID:            "Unidad Especial Dirección de Impuestos y Aduanas Nacionales",
				ValidationResultCode:   d.DianStatusCode,
				ValidationDate:         now.Format("2006-01-02"),
				ValidationTime:         now.Format("15:04:05-07:00"),
			},
		},
	}

	var doc *etree.Document
	switch d.DianDocumentTypeCode {
	case creditNoteDianDocumentType:
		doc, err = builder.BuildCreditNoteAttachedDocument(ad)
	case debitNoteDianDocumentType:
		doc, err = builder.BuildDebitNoteAttachedDocument(ad)
	default:
		doc, err = builder.BuildInvoiceAttachedDocument(ad)
	}
	if err != nil {
		return nil, "", err
	}

	root := doc.Root()
	placeholder := root.FindElement("./ext:UBLExtensions/ext:UBLExtension/ext:ExtensionContent")
	if placeholder == nil {
		return nil, "", fmt.Errorf("elemento placeholder de firma no encontrado en AttachedDocument")
	}
	sg := signer.New(cert, key)
	if err := sg.Sign(root, placeholder, "", now); err != nil {
		return nil, "", fmt.Errorf("firmar AttachedDocument: %w", err)
	}

	xmlBytes, err = doc.WriteToBytes()
	if err != nil {
		return nil, "", fmt.Errorf("serializar AttachedDocument: %w", err)
	}
	return xmlBytes, filename + "ad.xml", nil
}

// documentTypeName devuelve el nombre en minúsculas del tipo de documento para usar en emails.
func documentTypeName(typeCode string) string {
	switch typeCode {
	case creditNoteDianDocumentType:
		return "nota crédito"
	case debitNoteDianDocumentType:
		return "nota débito"
	default:
		return "factura electrónica"
	}
}

func documentEmailText(d *Document, iss *issuers.Issuer, typeName string) string {
	return fmt.Sprintf(
		"Hola %s,\n\n"+
			"Adjuntamos tu %s No. %s%d, emitida por %s.\n\n"+
			"El archivo adjunto (ZIP) contiene:\n"+
			"- El XML: el documento electrónico firmado, válido ante la DIAN.\n"+
			"- El PDF: representación gráfica del documento.\n\n"+
			"Este es un mensaje automático, por favor no respondas a esta dirección.\n",
		d.Customer.Name, typeName, d.Prefix, d.Number, iss.BusinessName,
	)
}

// documentEmailHTML escapa los campos interpolados (Customer.Name/iss.BusinessName son datos
// del cliente/snapshot, no se confía en que no traigan caracteres HTML especiales).
func documentEmailHTML(d *Document, iss *issuers.Issuer, typeName string) string {
	return fmt.Sprintf(
		"<p>Hola %s,</p>"+
			"<p>Adjuntamos tu %s No. %s%d, emitida por %s.</p>"+
			"<p>El archivo adjunto (ZIP) contiene:</p>"+
			"<ul><li>El XML: el documento electrónico firmado, válido ante la DIAN.</li>"+
			"<li>El PDF: representación gráfica del documento.</li></ul>"+
			"<p>Este es un mensaje automático, por favor no respondas a esta dirección.</p>",
		html.EscapeString(d.Customer.Name), typeName, html.EscapeString(d.Prefix), d.Number, html.EscapeString(iss.BusinessName),
	)
}
