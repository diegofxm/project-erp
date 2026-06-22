package signer

import (
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/beevik/etree"
)

// Política de firma de la DIAN. Es única y fija para todos los documentos electrónicos —
// confirmado contra dos facturas reales de proveedores distintos (2023 y 2026), mismo hash.
const (
	PolicyURL           = "https://facturaelectronica.dian.gov.co/politicadefirma/v2/politicadefirmav2.pdf"
	PolicyDescription   = "Política de firma para facturas electrónicas de la República de Colombia."
	PolicyHashSHA256B64 = "dMoMvtcG5aIzgYo0tIsSQeVJBDnUnfSOfBpxXrmor0Y="
)

// buildKeyInfo agrega ds:KeyInfo (con el certificado X.509 en DER+base64) como hijo de
// parent y lo retorna. parent debe ya estar insertado en su posición final dentro del
// documento para que la canonicalización herede el contexto de namespaces correcto.
func (s *Signer) buildKeyInfo(parent *etree.Element, id string) *etree.Element {
	keyInfo := parent.CreateElement("ds:KeyInfo")
	keyInfo.CreateAttr("Id", id+"-keyinfo")
	cert := keyInfo.CreateElement("ds:X509Data").CreateElement("ds:X509Certificate")
	cert.SetText(base64.StdEncoding.EncodeToString(s.cert.Raw))
	return keyInfo
}

// buildSignedProperties agrega ds:Object/xades:QualifyingProperties/xades:SignedProperties
// como hijo de parent y retorna el elemento SignedProperties. role es el valor de
// xades:ClaimedRole ("supplier" para Invoice/CreditNote/DebitNote, "" para AttachedDocument).
func (s *Signer) buildSignedProperties(parent *etree.Element, id, role string, signingTime time.Time) *etree.Element {
	object := parent.CreateElement("ds:Object")
	qualifying := object.CreateElement("xades:QualifyingProperties")
	qualifying.CreateAttr("Target", "#"+id)

	signedProps := qualifying.CreateElement("xades:SignedProperties")
	signedProps.CreateAttr("Id", id+"-signedprops")

	ssp := signedProps.CreateElement("xades:SignedSignatureProperties")
	ssp.CreateElement("xades:SigningTime").SetText(signingTime.Format(time.RFC3339))

	certDigest := sha256.Sum256(s.cert.Raw)
	cert := ssp.CreateElement("xades:SigningCertificate").CreateElement("xades:Cert")
	digest := cert.CreateElement("xades:CertDigest")
	digest.CreateElement("ds:DigestMethod").CreateAttr("Algorithm", digestMethodSHA256)
	digest.CreateElement("ds:DigestValue").SetText(base64.StdEncoding.EncodeToString(certDigest[:]))
	issuerSerial := cert.CreateElement("xades:IssuerSerial")
	issuerSerial.CreateElement("ds:X509IssuerName").SetText(s.cert.Issuer.String())
	issuerSerial.CreateElement("ds:X509SerialNumber").SetText(s.cert.SerialNumber.String())

	signaturePolicyID := ssp.CreateElement("xades:SignaturePolicyIdentifier").CreateElement("xades:SignaturePolicyId")
	sigPolicyID := signaturePolicyID.CreateElement("xades:SigPolicyId")
	sigPolicyID.CreateElement("xades:Identifier").SetText(PolicyURL)
	sigPolicyID.CreateElement("xades:Description").SetText(PolicyDescription)
	policyHash := signaturePolicyID.CreateElement("xades:SigPolicyHash")
	policyHash.CreateElement("ds:DigestMethod").CreateAttr("Algorithm", digestMethodSHA256)
	policyHash.CreateElement("ds:DigestValue").SetText(PolicyHashSHA256B64)

	if role != "" {
		claimedRoles := ssp.CreateElement("xades:SignerRole").CreateElement("xades:ClaimedRoles")
		claimedRoles.CreateElement("xades:ClaimedRole").SetText(role)
	}

	return signedProps
}
