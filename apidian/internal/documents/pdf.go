package documents

import (
	"context"
	"strings"

	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/diegofxm/apidian/internal/numbering"
	"github.com/diegofxm/apidian/internal/pdf"
	"github.com/google/uuid"
)

// RenderDocumentPDF construye la representación gráfica en PDF de cualquier documento DIAN
// (Factura, Nota Crédito o Nota Débito) — borrador o ya confirmado — siempre en memoria, nunca
// a disco (ver docs/apidian-architecture.md sección 9.39/9.49).
func (s *Service) RenderDocumentPDF(ctx context.Context, issuerID, id uuid.UUID, format pdf.Format) ([]byte, error) {
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
	return pdf.BuildPDF(format, input)
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
	var paymentTermName, paymentMethodName, dueDate string
	var err error
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
	var taxes []pdf.InvoiceTax
	for _, t := range aggregated {
		if t.TaxAmountCents == 0 {
			continue // IVA 0% requerido en el XML (FAU04) pero no tiene valor visual en el PDF
		}
		taxes = append(taxes, pdf.InvoiceTax{
			TypeName:           t.TypeName,
			Percent:            t.Percent,
			TaxableAmountCents: t.TaxableAmountCents,
			TaxAmountCents:     t.TaxAmountCents,
		})
	}

	var issueDate string
	if !d.IssueDate.IsZero() {
		// IssueTime se almacena como "HH:MM:SS-07:00" — para el PDF se muestra solo la hora
		// sin el offset de zona horaria, que el usuario final no necesita ver.
		t := d.IssueTime
		if len(t) > 8 {
			t = t[:8]
		}
		issueDate = d.IssueDate.Format("2006-01-02") + " " + t
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

	// Información fiscal del emisor para el pie del PDF (Resolución DIAN 042).
	var taxRegimeName string
	if iss.TaxRegimeCode != nil {
		taxRegimeName, err = s.resolveCatalogName(ctx, s.catalogs.GetTaxRegimeName, *iss.TaxRegimeCode)
		if err != nil {
			return pdf.InvoiceInput{}, err
		}
	}
	var liabilityParts []string
	for _, code := range iss.LiabilityCodes {
		name, _, lerr := s.catalogs.GetLiabilityCodeName(ctx, code)
		if lerr != nil {
			return pdf.InvoiceInput{}, lerr
		}
		if name != "" {
			liabilityParts = append(liabilityParts, code+" – "+name)
		} else {
			liabilityParts = append(liabilityParts, code)
		}
	}
	var econParts []string
	for _, code := range iss.IndustryClassificationCodes {
		desc, _, derr := s.catalogs.GetCiiuDescription(ctx, code)
		if derr != nil {
			return pdf.InvoiceInput{}, derr
		}
		if desc != "" {
			econParts = append(econParts, code+" – "+desc)
		} else {
			econParts = append(econParts, code)
		}
	}
	econActivity := strings.Join(econParts, ", ")

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
		IssuerLogoExt:      iss.LogoContentType,

		IsDraft:      d.Status == StatusDraft,
		Prefix:       d.Prefix,
		Number:       d.Number,
		CUFE:         d.DocumentKey,
		QRURL:        d.QRURL,
		IssueDate:    issueDate,
		CurrencyCode: d.CurrencyCode,

		CustomerName:               d.Customer.Name,
		CustomerIdentificationType: identTypeAbbrev(d.Customer.Identification.TypeCode),
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

		IssuerTaxRegime:        taxRegimeName,
		IssuerLiabilities:      strings.Join(liabilityParts, " · "),
		IssuerEconomicActivity: econActivity,

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
// identTypeAbbrev convierte el código DIAN de tipo de identificación a la sigla usada en el PDF.
// Fuente: seed/identification_types.csv. Código desconocido → se devuelve tal cual.
func identTypeAbbrev(code string) string {
	switch code {
	case "11":
		return "RC"
	case "12":
		return "TI"
	case "13":
		return "CC"
	case "21":
		return "TE"
	case "22":
		return "CE"
	case "31":
		return "NIT"
	case "41":
		return "PA"
	case "42":
		return "DIE"
	case "47":
		return "NIT Ext."
	case "50":
		return "NIT DIAN"
	case "91":
		return "NUIP"
	default:
		return code
	}
}

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
