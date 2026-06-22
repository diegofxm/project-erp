package builder

import (
	"errors"

	"github.com/beevik/etree"
	"github.com/diegofxm/cofacture/domain"
	ubl "github.com/diegofxm/cofacture/xml"
)

// SignaturePlaceholder ubica el ext:ExtensionContent vacío reservado para la firma XAdES
// (el último ext:UBLExtension de ext:UBLExtensions) en un documento ya construido por
// BuildInvoice o equivalente. El paquete signer lo usa como punto de inserción.
func SignaturePlaceholder(doc *etree.Document) (*etree.Element, error) {
	extensions := doc.FindElement("//ext:UBLExtensions")
	if extensions == nil {
		return nil, errors.New("builder: el documento no tiene ext:UBLExtensions")
	}
	children := extensions.SelectElements("ext:UBLExtension")
	if len(children) == 0 {
		return nil, errors.New("builder: ext:UBLExtensions no tiene ningún ext:UBLExtension")
	}
	last := children[len(children)-1]
	content := last.SelectElement("ext:ExtensionContent")
	if content == nil {
		return nil, errors.New("builder: el último ext:UBLExtension no tiene ext:ExtensionContent")
	}
	if len(content.ChildElements()) > 0 {
		return nil, errors.New("builder: el placeholder de firma ya tiene contenido")
	}
	return content, nil
}

// appendUBLExtensions agrega ext:UBLExtensions con las extensiones DIAN (sts:DianExtensions)
// y una segunda ext:UBLExtension vacía, reservada para la firma XAdES que agrega el
// paquete signer en un paso posterior del pipeline.
func appendUBLExtensions(parent *etree.Element, inv domain.Invoice) {
	extensions := parent.CreateElement("ext:UBLExtensions")

	dianExt := extensions.CreateElement("ext:UBLExtension").
		CreateElement("ext:ExtensionContent").
		CreateElement("sts:DianExtensions")

	if inv.DocumentTypeCode == "01" {
		appendInvoiceControl(dianExt, inv.NumberingRange)
	}

	source := dianExt.CreateElement("sts:InvoiceSource").CreateElement("cbc:IdentificationCode")
	source.CreateAttr("listAgencyID", "6")
	source.CreateAttr("listAgencyName", "United Nations Economic Commission for Europe")
	source.CreateAttr("listSchemeURI", "urn:oasis:names:specification:ubl:codelist:gc:CountryIdentificationCode-2.1")
	source.SetText("CO")

	swProvider := dianExt.CreateElement("sts:SoftwareProvider")
	providerID := swProvider.CreateElement("sts:ProviderID")
	providerID.CreateAttr("schemeAgencyID", ubl.DianSchemeAgencyID)
	providerID.CreateAttr("schemeAgencyName", ubl.DianSchemeAgencyName)
	if inv.SoftwareProvider.ProviderIdentification.TypeCode == ubl.IdentificationTypeNIT {
		providerID.CreateAttr("schemeID", inv.SoftwareProvider.ProviderIdentification.VerificationCode)
	}
	providerID.CreateAttr("schemeName", inv.SoftwareProvider.ProviderIdentification.TypeCode)
	providerID.SetText(inv.SoftwareProvider.ProviderIdentification.Number)

	softwareID := swProvider.CreateElement("sts:SoftwareID")
	softwareID.CreateAttr("schemeAgencyID", ubl.DianSchemeAgencyID)
	softwareID.CreateAttr("schemeAgencyName", ubl.DianSchemeAgencyName)
	softwareID.SetText(inv.SoftwareProvider.SoftwareID)

	secCode := dianExt.CreateElement("sts:SoftwareSecurityCode")
	secCode.CreateAttr("schemeAgencyID", ubl.DianSchemeAgencyID)
	secCode.CreateAttr("schemeAgencyName", ubl.DianSchemeAgencyName)
	secCode.SetText(inv.SoftwareSecurityCode)

	authProviderID := dianExt.CreateElement("sts:AuthorizationProvider").CreateElement("sts:AuthorizationProviderID")
	authProviderID.CreateAttr("schemeAgencyID", ubl.DianSchemeAgencyID)
	authProviderID.CreateAttr("schemeAgencyName", ubl.DianSchemeAgencyName)
	authProviderID.CreateAttr("schemeID", "4")
	authProviderID.CreateAttr("schemeName", "31")
	authProviderID.SetText(ubl.DianAuthorizationProviderID)

	dianExt.CreateElement("sts:QRCode").SetText(inv.QRURL)

	// Reservado para la firma XAdES (signer.Sign la completa más adelante).
	extensions.CreateElement("ext:UBLExtension").CreateElement("ext:ExtensionContent")
}

func appendInvoiceControl(parent *etree.Element, nr domain.NumberingRange) {
	control := parent.CreateElement("sts:InvoiceControl")
	control.CreateElement("sts:InvoiceAuthorization").SetText(nr.AuthorizedCode)

	period := control.CreateElement("sts:AuthorizationPeriod")
	period.CreateElement("cbc:StartDate").SetText(nr.StartDate)
	period.CreateElement("cbc:EndDate").SetText(nr.EndDate)

	authorized := control.CreateElement("sts:AuthorizedInvoices")
	if nr.Prefix != "" {
		authorized.CreateElement("sts:Prefix").SetText(nr.Prefix)
	}
	authorized.CreateElement("sts:From").SetText(nr.StartNumber)
	authorized.CreateElement("sts:To").SetText(nr.EndNumber)
}
