package builder

import (
	"strings"

	"github.com/beevik/etree"
	"github.com/diegofxm/cofacture/domain"
	ubl "github.com/diegofxm/cofacture/xml"
)

// appendAccountingParty agrega AccountingSupplierParty o AccountingCustomerParty.
// showMerchantRegistration solo debe ser true para el emisor (CorporateRegistrationScheme
// es el registro ante Cámara de Comercio, no aplica al receptor). invoicePrefix se ignora
// cuando showMerchantRegistration es false.
//
// Verificado contra facturas reales: cac:CorporateRegistrationScheme/cbc:ID es el prefijo
// de la factura (no un dato propio del tercero), y cbc:Name es el número de matrícula
// mercantil — se omite si el tercero no lo tiene (ej. persona natural sin registro).
func appendAccountingParty(parent *etree.Element, node string, p domain.Party, showMerchantRegistration bool, invoicePrefix string, showPhysicalLocation bool) {
	root := parent.CreateElement("cac:" + node)
	root.CreateElement("cbc:AdditionalAccountID").SetText(p.EntityTypeCode)

	party := root.CreateElement("cac:Party")

	// cbc:IndustryClassificationCode (CIIU) va antes de PartyIdentification/PartyName en la
	// secuencia de cac:Party — confirmado contra UBL-CommonAggregateComponents-2.1.xsd. Solo
	// el emisor lo trae (showMerchantRegistration es el mismo indicador que distingue emisor
	// de receptor en esta función).
	if showMerchantRegistration && len(p.IndustryClassificationCodes) > 0 {
		party.CreateElement("cbc:IndustryClassificationCode").SetText(strings.Join(p.IndustryClassificationCodes, ";"))
	}

	if p.EntityTypeCode == "2" {
		partyID := party.CreateElement("cac:PartyIdentification").CreateElement("cbc:ID")
		setIdentificationAttrs(partyID, p.Identification)
		partyID.SetText(p.Identification.Number)
	}

	party.CreateElement("cac:PartyName").CreateElement("cbc:Name").SetText(p.Name)

	if showPhysicalLocation && p.Address.Line != "" {
		addr := party.CreateElement("cac:PhysicalLocation").CreateElement("cac:Address")
		appendAddressFields(addr, p.Address)
	}

	taxScheme := party.CreateElement("cac:PartyTaxScheme")
	taxScheme.CreateElement("cbc:RegistrationName").SetText(p.Name)
	companyID := taxScheme.CreateElement("cbc:CompanyID")
	setIdentificationAttrs(companyID, p.Identification)
	companyID.SetText(p.Identification.Number)
	if len(p.LiabilityCodes) > 0 {
		tlc := taxScheme.CreateElement("cbc:TaxLevelCode")
		tlc.CreateAttr("listName", p.TaxRegimeCode)
		tlc.SetText(strings.Join(p.LiabilityCodes, ";"))
	}
	if p.Address.Line != "" {
		regAddr := taxScheme.CreateElement("cac:RegistrationAddress")
		appendAddressFields(regAddr, p.Address)
	}
	scheme := taxScheme.CreateElement("cac:TaxScheme")
	scheme.CreateElement("cbc:ID").SetText(p.TaxSchemeCode)
	scheme.CreateElement("cbc:Name").SetText(p.TaxSchemeName)

	legal := party.CreateElement("cac:PartyLegalEntity")
	legal.CreateElement("cbc:RegistrationName").SetText(p.Name)
	legalID := legal.CreateElement("cbc:CompanyID")
	setIdentificationAttrs(legalID, p.Identification)
	legalID.SetText(p.Identification.Number)
	if showMerchantRegistration {
		crs := legal.CreateElement("cac:CorporateRegistrationScheme")
		crs.CreateElement("cbc:ID").SetText(invoicePrefix)
		if p.MerchantRegistrationNumber != nil {
			crs.CreateElement("cbc:Name").SetText(*p.MerchantRegistrationNumber)
		}
	}

	contact := party.CreateElement("cac:Contact")
	contact.CreateElement("cbc:Telephone").SetText(p.Phone)
	contact.CreateElement("cbc:ElectronicMail").SetText(p.Email)
}

func setIdentificationAttrs(el *etree.Element, id domain.Identification) {
	el.CreateAttr("schemeAgencyID", ubl.DianSchemeAgencyID)
	el.CreateAttr("schemeAgencyName", ubl.DianSchemeAgencyName)
	if id.TypeCode == ubl.IdentificationTypeNIT {
		el.CreateAttr("schemeID", id.VerificationCode)
	}
	el.CreateAttr("schemeName", id.TypeCode)
}

func appendAddressFields(addr *etree.Element, a domain.Address) {
	if a.CityCode != "" {
		addr.CreateElement("cbc:ID").SetText(a.CityCode)
		addr.CreateElement("cbc:CityName").SetText(a.CityName)
	}
	if a.PostalZone != "" {
		addr.CreateElement("cbc:PostalZone").SetText(a.PostalZone)
	}
	if a.StateCode != "" {
		addr.CreateElement("cbc:CountrySubentity").SetText(a.StateName)
		addr.CreateElement("cbc:CountrySubentityCode").SetText(a.StateCode)
	}
	addr.CreateElement("cac:AddressLine").CreateElement("cbc:Line").SetText(a.Line)
	if a.CountryCode != "" {
		country := addr.CreateElement("cac:Country")
		country.CreateElement("cbc:IdentificationCode").SetText(a.CountryCode)
		name := country.CreateElement("cbc:Name")
		name.CreateAttr("languageID", "es")
		name.SetText(a.CountryName)
	}
}
