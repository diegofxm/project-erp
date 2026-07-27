package documents

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"html"
	htmltmpl "html/template"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/diegofxm/apidian/internal/email"
	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/diegofxm/apidian/internal/edocuments/pdf"
	"github.com/diegofxm/cofacture/builder"
	"github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/signer"
	cofzip "github.com/diegofxm/cofacture/zip"
	"github.com/google/uuid"
)

// emailTemplateData son las variables que rellena el template email_template.html.
// BodyHTML es htmltmpl.HTML (no se re-escapa) — su contenido ya viene escapado de
// bodyTextToParas.
type emailTemplateData struct {
	IssuerName      string
	IssuerTradeName string
	IssuerNIT       string
	IssuerEmail     string
	// LogoSrc es el data URI del logo (data:image/png;base64,...). htmltmpl.URL para
	// que html/template no lo filtre como URL no segura. Vacío si no hay logo.
	LogoSrc          htmltmpl.URL
	DocumentTypeName string
	DocumentNumber   string
	IssueDate        string
	TotalFormatted   string
	BodyHTML         htmltmpl.HTML
}

//go:embed email_template.html
var emailTemplateSrc string

var emailTmpl = htmltmpl.Must(htmltmpl.New("email").Parse(emailTemplateSrc))

// resolveRecipient devuelve el nombre y correo actuales del destinatario del correo.
//
// Para DS (Documento Soporte) el destinatario es el proveedor (Vendor); para los demás
// documentos es el cliente (Customer). En ambos casos se intenta obtener el correo actual
// del catálogo (customers / vendors) usando el ID opcional guardado en el documento,
// porque el snapshot almacenado en el documento puede quedar desactualizado si el usuario
// edita el correo del cliente o proveedor después de emitir.
func (s *Service) resolveRecipient(ctx context.Context, d *Document) (name, email string, err error) {
	if d.DianDocumentTypeCode == supportDocumentDianDocType || d.DianDocumentTypeCode == adjustmentNoteDianDocType {
		// DS / NA: el destinatario es el Vendor (proveedor).
		if d.Vendor == nil {
			return "", "", ErrCustomerEmailMissing
		}
		name = d.Vendor.Name
		email = d.Vendor.Email
		if d.VendorID != nil && s.vendors != nil {
			if v, e := s.vendors.GetVendor(ctx, *d.VendorID); e == nil {
				email = v.Party.Email
				name = v.Party.Name
			}
		}
		return
	}
	// FE / NC / ND: el destinatario es el Customer.
	name = d.Customer.Name
	email = d.Customer.Email
	if d.CustomerID != nil {
		if c, e := s.customers.GetCustomer(ctx, *d.CustomerID); e == nil {
			email = c.Party.Email
			name = c.Party.Name
		}
	}
	return
}

// SendDocumentEmail envía el documento ya aceptado por la DIAN al correo del cliente
// (FE/NC/ND) o del proveedor (DS).
//
// to es el correo destino confirmado por el usuario en el modal; si está vacío se resuelve
// automáticamente desde el catálogo (customers/vendors). cc es la lista de destinatarios
// en copia; nil o vacío = sin CC.
func (s *Service) SendDocumentEmail(ctx context.Context, issuerID, id uuid.UUID, format pdf.Format, to string, cc []string) error {
	d, iss, err := s.loadDocumentAndIssuer(ctx, issuerID, id)
	if err != nil {
		return err
	}
	if d.Status != StatusAccepted {
		return ErrDocumentNotAccepted
	}
	var recipientName, recipientEmail string
	if to != "" {
		// El frontend ya resolvió y confirmó el correo (puede haber sido editado por el usuario).
		recipientEmail = to
		if (d.DianDocumentTypeCode == supportDocumentDianDocType || d.DianDocumentTypeCode == adjustmentNoteDianDocType) && d.Vendor != nil {
			recipientName = d.Vendor.Name
		} else {
			recipientName = d.Customer.Name
		}
	} else {
		recipientName, recipientEmail, err = s.resolveRecipient(ctx, d)
		if err != nil {
			return err
		}
	}
	if recipientEmail == "" {
		return ErrCustomerEmailMissing
	}

	pdfBytes, err := s.RenderDocumentPDF(ctx, issuerID, id, format)
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

	bodyPlainText := defaultEmailBody(recipientName, iss, typeName)

	issuerNIT := iss.NIT
	if iss.IdentificationTypeCode == "31" {
		issuerNIT = iss.NIT + "-" + iss.CheckDigit
	}

	data := emailTemplateData{
		IssuerName:       iss.BusinessName,
		IssuerTradeName:  iss.TradeName,
		IssuerNIT:        issuerNIT,
		IssuerEmail:      iss.Email,
		LogoSrc:          logoDataURI(iss.Logo, iss.LogoContentType),
		DocumentTypeName: capitalizeFirst(typeName),
		DocumentNumber:   filename,
		IssueDate:        formatDateES(d.IssueDate),
		TotalFormatted:   formatCentsCOP(d.Totals.PayableCents),
		BodyHTML:         bodyTextToParas(bodyPlainText),
	}

	bodyHTML, err := buildEmailHTML(data)
	if err != nil {
		return fmt.Errorf("renderizar correo HTML: %w", err)
	}
	bodyText := buildEmailText(bodyPlainText, iss.BusinessName, issuerNIT, iss.Email,
		data.DocumentTypeName, filename, data.IssueDate, data.TotalFormatted)

	msg := email.Message{
		To:       recipientEmail,
		CC:       cc,
		Subject:  subject,
		BodyText: bodyText,
		BodyHTML: bodyHTML,
		Attachments: []email.Attachment{
			{Filename: filename + ".zip", ContentType: "application/zip", Content: zipBytes},
		},
	}
	return s.email.Send(ctx, msg)
}

// buildEmailHTML ejecuta email_template.html con los datos del documento.
func buildEmailHTML(data emailTemplateData) (string, error) {
	var buf bytes.Buffer
	if err := emailTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// buildEmailText genera la versión texto plano del correo (fallback para clientes sin HTML).
func buildEmailText(bodyText, issuerName, issuerNIT, issuerEmail, docTypeName, docNumber, issueDate, total string) string {
	sep := strings.Repeat("─", 40)
	return fmt.Sprintf("%s\n%s\n\n%s\n\n%s\n%s: %s\nFecha: %s\nTotal: %s\n%s\n\nAdjunto (ZIP):\n• XML – documento electrónico firmado, válido ante la DIAN\n• PDF – representación gráfica del documento\n\nMensaje automático – por favor no respondas a este correo.\nNIT %s • %s\n",
		issuerName, sep,
		bodyText,
		sep, docTypeName, docNumber, issueDate, total, sep,
		issuerNIT, issuerEmail,
	)
}

// defaultEmailBody retorna el cuerpo predeterminado cuando el emisor no configuró una
// plantilla personalizada. Es breve — el template HTML ya muestra los datos del documento.
func defaultEmailBody(recipientName string, iss *issuers.Issuer, typeName string) string {
	return fmt.Sprintf("Hola %s,\n\n%s te hace llegar tu %s. Encuéntrala en el archivo ZIP adjunto a este correo.",
		recipientName, iss.BusinessName, typeName)
}

// bodyTextToParas convierte texto plano en párrafos HTML seguros para inyectar en el
// template. Escapa caracteres especiales y convierte saltos de línea simples en <br>.
func bodyTextToParas(text string) htmltmpl.HTML {
	paragraphs := strings.Split(text, "\n\n")
	var b strings.Builder
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		escaped := html.EscapeString(p)
		escaped = strings.ReplaceAll(escaped, "\n", "<br>")
		b.WriteString(`<p style="margin:0 0 14px">`)
		b.WriteString(escaped)
		b.WriteString("</p>")
	}
	return htmltmpl.HTML(b.String())
}

// formatCentsCOP formatea un valor en centavos como peso colombiano: "$ 1.234.567"
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

var monthsES = [12]string{
	"enero", "febrero", "marzo", "abril", "mayo", "junio",
	"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
}

// formatDateES formatea una fecha como "13 de julio de 2026".
func formatDateES(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return fmt.Sprintf("%d de %s de %d", t.Day(), monthsES[t.Month()-1], t.Year())
}

// logoDataURI construye un data URI para incrustar el logo inline en el correo HTML.
// Devuelve htmltmpl.URL vacío si no hay logo — el template lo omite con {{if .LogoSrc}}.
func logoDataURI(logo []byte, contentType string) htmltmpl.URL {
	if len(logo) == 0 || contentType == "" {
		return ""
	}
	mime := "image/jpeg"
	if contentType == "png" {
		mime = "image/png"
	}
	return htmltmpl.URL("data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(logo))
}

// capitalizeFirst pone en mayúscula la primera letra de s.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// documentTypeName devuelve el nombre en minúsculas del tipo de documento para usar en emails.
func documentTypeName(typeCode string) string {
	switch typeCode {
	case creditNoteDianDocumentType:
		return "nota crédito"
	case debitNoteDianDocumentType:
		return "nota débito"
	case supportDocumentDianDocType:
		return "documento soporte"
	case adjustmentNoteDianDocType:
		return "nota de ajuste"
	default:
		return "factura electrónica"
	}
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

	hashType := invoiceHashType
	switch d.DianDocumentTypeCode {
	case creditNoteDianDocumentType, debitNoteDianDocumentType:
		hashType = creditNoteHashType
	case supportDocumentDianDocType, adjustmentNoteDianDocType:
		hashType = supportDocumentHashType
	}

	// Para DS/NA el receptor del AttachedDocument es el proveedor (Vendor); para el resto es el cliente.
	var receiver domain.Party
	if (d.DianDocumentTypeCode == supportDocumentDianDocType || d.DianDocumentTypeCode == adjustmentNoteDianDocType) && d.Vendor != nil {
		receiver = *d.Vendor
	} else {
		receiver = d.Customer
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
			Name:           receiver.Name,
			Identification: domain.Identification{TypeCode: receiver.Identification.TypeCode, Number: receiver.Identification.Number},
			TaxRegimeCode:  receiver.TaxRegimeCode,
			LiabilityCodes: receiver.LiabilityCodes,
			TaxSchemeCode:  receiver.TaxSchemeCode,
			TaxSchemeName:  receiver.TaxSchemeName,
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
	default: // Invoice (01), Documento Soporte (05) y Nota de Ajuste (95) usan la misma estructura UBL
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
