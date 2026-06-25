package api

import "github.com/diegofxm/cofacture/domain"

// Los DTOs de este archivo son el contrato JSON público de la API — deliberadamente
// independientes de los structs de dominio de cofacture (que no tienen tags json y pueden
// cambiar de forma libre, ya que cofacture es una librería interna, no un contrato HTTP).
// Solo internal/documents importa cofacture directamente (ver architecture doc sección 4.1);
// este paquete solo conoce estos DTOs y los mapea a domain.* antes de llamar al servicio.

type identificationDTO struct {
	Number           string `json:"number"`
	TypeCode         string `json:"type_code"`
	VerificationCode string `json:"verification_code,omitempty"`
}

type addressDTO struct {
	Line        string `json:"line,omitempty"`
	CityCode    string `json:"city_code,omitempty"`
	CityName    string `json:"city_name,omitempty"`
	StateCode   string `json:"state_code,omitempty"`
	StateName   string `json:"state_name,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	CountryName string `json:"country_name,omitempty"`
}

// partyDTO es el adquiriente (Customer) — pass-through puro, sin tabla propia (ver
// docs/apidian-architecture.md sección 4.2). EntityTypeCode/TaxSchemeCode/TaxSchemeName/
// LiabilityCodes son opcionales: si no se envían, documents.Service les aplica los mismos
// defaults confirmados contra la DIAN real ("2"/"ZZ"/"No aplica"/["R-99-PN"]).
type partyDTO struct {
	EntityTypeCode             string            `json:"entity_type_code,omitempty"`
	Identification             identificationDTO `json:"identification"`
	Name                       string            `json:"name"`
	Address                    *addressDTO       `json:"address,omitempty"`
	TaxSchemeCode              string            `json:"tax_scheme_code,omitempty"`
	TaxSchemeName              string            `json:"tax_scheme_name,omitempty"`
	LiabilityCodes             []string          `json:"liability_codes,omitempty"`
	TaxRegimeCode              string            `json:"tax_regime_code,omitempty"`
	Phone                      string            `json:"phone,omitempty"`
	Email                      string            `json:"email,omitempty"`
	MerchantRegistrationNumber *string           `json:"merchant_registration_number,omitempty"`
}

type taxDTO struct {
	TaxableAmountCents int64   `json:"taxable_amount_cents"`
	TaxAmountCents     int64   `json:"tax_amount_cents"`
	Percent            float64 `json:"percent"`
	TypeCode           string  `json:"type_code"`
	TypeName           string  `json:"type_name"`
}

type referencePriceDTO struct {
	PriceAmountCents int64  `json:"price_amount_cents"`
	TypeCode         string `json:"type_code"`
}

type lineDTO struct {
	Description        string             `json:"description"`
	Quantity           float64            `json:"quantity"`
	UnitCode           string             `json:"unit_code"`
	LineExtensionCents int64              `json:"line_extension_cents"`
	UnitPriceCents     int64              `json:"unit_price_cents"`
	FreeOfCharge       bool               `json:"free_of_charge,omitempty"`
	ReferencePrice     *referencePriceDTO `json:"reference_price,omitempty"`
	ItemCode           string             `json:"item_code,omitempty"`
	ItemTypeCode       string             `json:"item_type_code,omitempty"`
	ItemTypeName       string             `json:"item_type_name,omitempty"`
	ItemTypeAgencyID   string             `json:"item_type_agency_id,omitempty"`
	Taxes              []taxDTO           `json:"taxes,omitempty"`
}

type paymentMeanDTO struct {
	Code              string `json:"code"`
	PaymentMethodCode string `json:"payment_method_code"`
	DueDate           string `json:"due_date,omitempty"`
	PaymentReference  string `json:"payment_reference,omitempty"`
}

type billingReferenceDTO struct {
	Prefix    string `json:"prefix"`
	Number    string `json:"number"`
	CUFE      string `json:"cufe"`
	IssueDate string `json:"issue_date"`
}

type discrepancyResponseDTO struct {
	ReferenceID  string `json:"reference_id"`
	ResponseCode string `json:"response_code"`
	Description  string `json:"description"`
}

// totalsDTO es el inverso de domain.Totals — solo de lectura, ver totalsFromDomain.
type totalsDTO struct {
	LineExtensionCents int64 `json:"line_extension_cents"`
	TaxExclusiveCents  int64 `json:"tax_exclusive_cents"`
	TaxInclusiveCents  int64 `json:"tax_inclusive_cents"`
	PrepaidCents       int64 `json:"prepaid_cents,omitempty"`
	PayableCents       int64 `json:"payable_cents"`
}

func totalsFromDomain(t domain.Totals) totalsDTO {
	return totalsDTO{
		LineExtensionCents: t.LineExtensionCents,
		TaxExclusiveCents:  t.TaxExclusiveCents,
		TaxInclusiveCents:  t.TaxInclusiveCents,
		PrepaidCents:       t.PrepaidCents,
		PayableCents:       t.PayableCents,
	}
}

func taxFromDomain(t domain.Tax) taxDTO {
	return taxDTO{
		TaxableAmountCents: t.TaxableAmountCents,
		TaxAmountCents:     t.TaxAmountCents,
		Percent:            t.Percent,
		TypeCode:           t.TypeCode,
		TypeName:           t.TypeName,
	}
}

// lineFromDomain/linesFromDomain son el inverso de linesToDomain — no existían antes porque
// ningún endpoint devolvía las líneas de un documento ya guardado (ver
// docs/apidian-architecture.md sección 9.28: GET /documents devolvía solo metadatos, nunca
// el contenido — un frontend real sí lo necesita para mostrar algo más que un UUID).
func lineFromDomain(l domain.Line) lineDTO {
	dto := lineDTO{
		Description:        l.Description,
		Quantity:           l.Quantity,
		UnitCode:           l.UnitCode,
		LineExtensionCents: l.LineExtensionCents,
		UnitPriceCents:     l.UnitPriceCents,
		FreeOfCharge:       l.FreeOfCharge,
		ItemCode:           l.ItemCode,
		ItemTypeCode:       l.ItemTypeCode,
		ItemTypeName:       l.ItemTypeName,
		ItemTypeAgencyID:   l.ItemTypeAgencyID,
	}
	if l.ReferencePrice != nil {
		dto.ReferencePrice = &referencePriceDTO{
			PriceAmountCents: l.ReferencePrice.PriceAmountCents,
			TypeCode:         l.ReferencePrice.TypeCode,
		}
	}
	dto.Taxes = make([]taxDTO, len(l.Taxes))
	for i, t := range l.Taxes {
		dto.Taxes[i] = taxFromDomain(t)
	}
	return dto
}

func linesFromDomain(lines []domain.Line) []lineDTO {
	out := make([]lineDTO, len(lines))
	for i, l := range lines {
		out[i] = lineFromDomain(l)
	}
	return out
}

func paymentMeanFromDomain(pm domain.PaymentMean) paymentMeanDTO {
	return paymentMeanDTO{
		Code:              pm.Code,
		PaymentMethodCode: pm.PaymentMethodCode,
		DueDate:           pm.DueDate,
		PaymentReference:  pm.PaymentReference,
	}
}

func paymentMeansFromDomain(pms []domain.PaymentMean) []paymentMeanDTO {
	out := make([]paymentMeanDTO, len(pms))
	for i, pm := range pms {
		out[i] = paymentMeanFromDomain(pm)
	}
	return out
}

func (p partyDTO) toDomain() domain.Party {
	party := domain.Party{
		EntityTypeCode: p.EntityTypeCode,
		Identification: domain.Identification{
			Number:           p.Identification.Number,
			TypeCode:         p.Identification.TypeCode,
			VerificationCode: p.Identification.VerificationCode,
		},
		Name:                       p.Name,
		TaxSchemeCode:              p.TaxSchemeCode,
		TaxSchemeName:              p.TaxSchemeName,
		LiabilityCodes:             p.LiabilityCodes,
		TaxRegimeCode:              p.TaxRegimeCode,
		Phone:                      p.Phone,
		Email:                      p.Email,
		MerchantRegistrationNumber: p.MerchantRegistrationNumber,
	}
	if p.Address != nil {
		party.Address = domain.Address{
			Line:        p.Address.Line,
			CityCode:    p.Address.CityCode,
			CityName:    p.Address.CityName,
			StateCode:   p.Address.StateCode,
			StateName:   p.Address.StateName,
			CountryCode: p.Address.CountryCode,
			CountryName: p.Address.CountryName,
		}
	}
	return party
}

func linesToDomain(lines []lineDTO) []domain.Line {
	out := make([]domain.Line, len(lines))
	for i, l := range lines {
		dl := domain.Line{
			Description:        l.Description,
			Quantity:           l.Quantity,
			UnitCode:           l.UnitCode,
			LineExtensionCents: l.LineExtensionCents,
			UnitPriceCents:     l.UnitPriceCents,
			FreeOfCharge:       l.FreeOfCharge,
			ItemCode:           l.ItemCode,
			ItemTypeCode:       l.ItemTypeCode,
			ItemTypeName:       l.ItemTypeName,
			ItemTypeAgencyID:   l.ItemTypeAgencyID,
		}
		if l.ReferencePrice != nil {
			dl.ReferencePrice = &domain.ReferencePrice{
				PriceAmountCents: l.ReferencePrice.PriceAmountCents,
				TypeCode:         l.ReferencePrice.TypeCode,
			}
		}
		dl.Taxes = make([]domain.Tax, len(l.Taxes))
		for j, t := range l.Taxes {
			dl.Taxes[j] = domain.Tax{
				TaxableAmountCents: t.TaxableAmountCents,
				TaxAmountCents:     t.TaxAmountCents,
				Percent:            t.Percent,
				TypeCode:           t.TypeCode,
				TypeName:           t.TypeName,
			}
		}
		out[i] = dl
	}
	return out
}

// partyFromDomain es el inverso de partyDTO.toDomain() — lo necesita handler_customers.go
// para serializar un Customer guardado (domain.Party) de vuelta a JSON. No existía antes
// porque ningún endpoint necesitaba devolver un Party en la respuesta: invoices/credit-notes/
// debit-notes solo lo reciben (pass-through), nunca lo devuelven tal cual.
func partyFromDomain(p domain.Party) partyDTO {
	dto := partyDTO{
		EntityTypeCode: p.EntityTypeCode,
		Identification: identificationDTO{
			Number:           p.Identification.Number,
			TypeCode:         p.Identification.TypeCode,
			VerificationCode: p.Identification.VerificationCode,
		},
		Name:                       p.Name,
		TaxSchemeCode:              p.TaxSchemeCode,
		TaxSchemeName:              p.TaxSchemeName,
		LiabilityCodes:             p.LiabilityCodes,
		TaxRegimeCode:              p.TaxRegimeCode,
		Phone:                      p.Phone,
		Email:                      p.Email,
		MerchantRegistrationNumber: p.MerchantRegistrationNumber,
	}
	if p.Address != (domain.Address{}) {
		dto.Address = &addressDTO{
			Line:        p.Address.Line,
			CityCode:    p.Address.CityCode,
			CityName:    p.Address.CityName,
			StateCode:   p.Address.StateCode,
			StateName:   p.Address.StateName,
			CountryCode: p.Address.CountryCode,
			CountryName: p.Address.CountryName,
		}
	}
	return dto
}

func paymentMeansToDomain(pms []paymentMeanDTO) []domain.PaymentMean {
	out := make([]domain.PaymentMean, len(pms))
	for i, pm := range pms {
		out[i] = domain.PaymentMean{
			Code:              pm.Code,
			PaymentMethodCode: pm.PaymentMethodCode,
			DueDate:           pm.DueDate,
			PaymentReference:  pm.PaymentReference,
		}
	}
	return out
}
