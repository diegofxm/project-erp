package documents

import (
	"context"

	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/diegofxm/apidian/internal/numbering"
	"github.com/diegofxm/apidian/internal/pdf"
	"github.com/google/uuid"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
)

// RenderInvoicePDF construye la representación gráfica en PDF de una Factura — borrador o ya
// confirmada — siempre en memoria, nunca a disco (ver docs/apidian-architecture.md sección
// 9.39). Mismo rol orquestador que ya cumple este Service para el XML (vía cofacture): junta
// lo que internal/pdf necesita y le delega la construcción real, sin que ese paquete conozca
// Document/Issuer/NumberingRange.
//
// Solo Invoice por ahora (ErrPDFNotSupportedForDocumentType en cualquier otro caso) — Nota
// Crédito/Nota Débito se agregan cuando el ciclo de Factura esté probado de punta a punta.
func (s *Service) RenderInvoicePDF(ctx context.Context, issuerID, id uuid.UUID) ([]byte, error) {
	d, iss, err := s.loadInvoiceAndIssuer(ctx, issuerID, id, ErrPDFNotSupportedForDocumentType)
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

// loadInvoiceAndIssuer carga un documento, valida que pertenezca al emisor y que sea Factura
// (única restricción que comparten PDF y correo, ver secciones 9.39/9.42), y carga el emisor.
// notSupportedErr es el error específico del llamador para "este tipo de documento no soporta
// esto todavía" — PDF y correo tienen su propio mensaje aunque el chequeo sea idéntico.
func (s *Service) loadInvoiceAndIssuer(ctx context.Context, issuerID, id uuid.UUID, notSupportedErr error) (*Document, *issuers.Issuer, error) {
	d, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if d.IssuerID != issuerID {
		return nil, nil, ErrDocumentNotFound
	}
	if d.DianDocumentTypeCode != invoiceDianDocumentType {
		return nil, nil, notSupportedErr
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
