package documents

import (
	"context"

	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/diegofxm/apidian/internal/numbering"
	"github.com/diegofxm/apidian/internal/pdf"
	"github.com/google/uuid"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
)

// RenderDocumentPDF construye la representación gráfica en PDF de cualquier documento DIAN
// (Factura, Nota Crédito o Nota Débito) — borrador o ya confirmado — siempre en memoria, nunca
// a disco (ver docs/apidian-architecture.md sección 9.39/9.49).
func (s *Service) RenderDocumentPDF(ctx context.Context, issuerID, id uuid.UUID) ([]byte, error) {
	d, iss, err := s.loadDocumentAndIssuer(ctx, issuerID, id)
	if err != nil {
		return nil, err
	}
	nr, err := s.numbering.GetRange(ctx, d.NumberingRangeID)
	if err != nil {
		return nil, err
	}
	input, err := s.invoicePDFInput(ctx, d, iss, nr)
	if err != nil {
		return nil, err
	}
	return pdf.BuildInvoicePDF(input)
}

// loadDocumentAndIssuer carga un documento, valida que pertenezca al emisor y carga el emisor.
// Acepta los tres tipos de documento DIAN (01, 91, 92).
func (s *Service) loadDocumentAndIssuer(ctx context.Context, issuerID, id uuid.UUID) (*Document, *issuers.Issuer, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if d.IssuerID != issuerID {
		return nil, nil, ErrDocumentNotFound
	}
	iss, err := s.issuers.GetIssuer(ctx, issuerID)
	if err != nil {
		return nil, nil, err
	}
	return d, iss, nil
}

func (s *Service) invoicePDFInput(ctx context.Context, d *Document, iss *issuers.Issuer, nr *numbering.NumberingRange) (pdf.InvoiceInput, error) {
	identificationTypeName, err := s.resolveCatalogName(ctx, s.catalogs.GetIdentificationTypeName, d.Customer.Identification.TypeCode)
	if err != nil {
		return pdf.InvoiceInput{}, err
	}

	var paymentTermName, paymentMethodName, dueDate string
	if len(d.PaymentMeans) > 0 {
		pm := d.PaymentMeans[0]
		if paymentTermName, err = s.resolveCatalogName(ctx, s.catalogs.GetPaymentTermName, pm.Code); err != nil {
			return pdf.InvoiceInput{}, err
		}
		if paymentMethodName, err = s.resolveCatalogName(ctx, s.catalogs.GetPaymentMethodName, pm.PaymentMethodCode); err != nil {
			return pdf.InvoiceInput{}, err
		}
		dueDate = pm.DueDate
	}

	lines := make([]pdf.InvoiceLine, len(d.Lines))
	for i, l := range d.Lines {
		var percent float64
		var taxCents int64
		if len(l.Taxes) > 0 {
			percent = l.Taxes[0].Percent
			taxCents = l.Taxes[0].TaxAmountCents
		}
		lines[i] = pdf.InvoiceLine{
			ItemCode:       l.ItemCode,
			Description:    l.Description,
			UnitCode:       l.UnitCode,
			Quantity:       l.Quantity,
			UnitPriceCents: l.UnitPriceCents,
			TaxPercent:     percent,
			TotalCents:     l.LineExtensionCents + taxCents,
		}
	}

	aggregated := aggregateTaxes(d.Lines)
	taxes := make([]pdf.InvoiceTax, len(aggregated))
	for i, t := range aggregated {
		taxes[i] = pdf.InvoiceTax{
			TypeName:           t.TypeName,
			Percent:            t.Percent,
			TaxableAmountCents: t.TaxableAmountCents,
			TaxAmountCents:     t.TaxAmountCents,
		}
	}

	var issueDate string
	if !d.IssueDate.IsZero() {
		issueDate = d.IssueDate.Format("2006-01-02") + " " + d.IssueTime
	}

	// Título y etiqueta de hash varían según el tipo de documento.
	var documentTitle, hashLabel string
	switch d.DianDocumentTypeCode {
	case creditNoteDianDocumentType:
		documentTitle = "NOTA CRÉDITO DE VENTA"
		hashLabel = "CUDE"
	case debitNoteDianDocumentType:
		documentTitle = "NOTA DÉBITO DE VENTA"
		hashLabel = "CUDE"
	default:
		documentTitle = "FACTURA ELECTRÓNICA DE VENTA"
		hashLabel = "CUFE"
	}

	return pdf.InvoiceInput{
		IssuerBusinessName: iss.BusinessName,
		IssuerNIT:          iss.NIT,
		IssuerCheckDigit:   iss.CheckDigit,
		IssuerAddressLine:  iss.AddressLine,
		IssuerCityName:     iss.MunicipalityName,
		IssuerStateName:    iss.DepartmentName,
		IssuerPhone:        iss.Phone,
		IssuerEmail:        iss.Email,
		IssuerLogo:         iss.Logo,
		IssuerLogoExt:      extension.Type(iss.LogoContentType),

		IsDraft:      d.Status == StatusDraft,
		Prefix:       d.Prefix,
		Number:       d.Number,
		CUFE:         d.DocumentKey,
		QRURL:        d.QRURL,
		IssueDate:    issueDate,
		CurrencyCode: d.CurrencyCode,

		CustomerName:               d.Customer.Name,
		CustomerIdentificationType: identificationTypeName,
		CustomerIdentification:     d.Customer.Identification.Number,
		CustomerAddressLine:        d.Customer.Address.Line,
		CustomerPhone:              d.Customer.Phone,
		CustomerEmail:              d.Customer.Email,

		PaymentTermName:   paymentTermName,
		PaymentMethodName: paymentMethodName,
		PaymentDueDate:    dueDate,

		Lines: lines,
		Taxes: taxes,

		LineExtensionCents: d.Totals.LineExtensionCents,
		TaxAmountCents:     d.Totals.TaxInclusiveCents - d.Totals.LineExtensionCents,
		PayableCents:       d.Totals.PayableCents,

		Note: d.Note,

		DocumentTitle: documentTitle,
		HashLabel:     hashLabel,

		ResolutionNumber: nr.ResolutionNumber,
		RangePrefix:      nr.Prefix,
		RangeFrom:        nr.RangeFrom,
		RangeTo:          nr.RangeTo,
		ValidFrom:        nr.ValidFrom.Format("2006-01-02"),
		ValidTo:          nr.ValidTo.Format("2006-01-02"),
	}, nil
}

// resolveCatalogName envuelve los GetXName de CatalogPort (todos found=false sin error si no
// existe, ver Repository en internal/catalogs) — para el PDF no tiene sentido fallar si un
// código quedó huérfano del catálogo, simplemente se muestra el código tal cual en vez del
// nombre legible.
func (s *Service) resolveCatalogName(ctx context.Context, get func(context.Context, string) (string, bool, error), code string) (string, error) {
	if code == "" {
		return "", nil
	}
	name, found, err := get(ctx, code)
	if err != nil {
		return "", err
	}
	if !found {
		return code, nil
	}
	return name, nil
}
