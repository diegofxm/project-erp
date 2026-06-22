// Package soap implementa el cliente SOAP 1.2 de los webservices de recepción de la DIAN
// (WcfDianCustomerServices), incluyendo la cabecera WS-Security.
//
// Verificado contra el servidor real de habilitación (vpfe-hab.dian.gov.co): una factura
// construida con este cliente fue enviada por SendTestSetAsync y autorizada
// (StatusCode=00, "ha sido autorizada").
//
// La política publicada en el WSDL describe RequireThumbprintReference, pero esa variante
// dio "An error occurred when verifying security for the message" contra el servidor real.
// El patrón que sí funciona (igual al de implementaciones reales existentes: Chilkat,
// soap-dian en PHP) es distinto:
//
//   - wsse:BinarySecurityToken embebido + referencia directa (Direct Reference) en KeyInfo,
//     no por thumbprint.
//   - C14N exclusivo forzando una lista de namespaces inclusivos "wsa soap wcf" — sin esto,
//     la firma no coincide con lo que recalcula el servidor al verificar.
//   - TransportBinding con HttpsToken RequireClientCertificate="false" — HTTPS normal, sin
//     TLS mutuo, a pesar de que el anexo técnico (sección 7.5) sugiere lo contrario.
//   - Se firma únicamente el header wsa:To (no el Body ni el Timestamp).
//   - AlgorithmSuite Basic256Sha256Rsa15 — digest SHA-256, firma RSA-SHA256.
//   - Layout Strict — dentro de wsse:Security: Timestamp, BinarySecurityToken, Signature.
//   - Cada elemento que lleve un atributo "wsu:Id" debe declarar xmlns:wsu en sí mismo —
//     los namespaces de XML no se heredan entre hermanos. Esto causó un 400 vacío (la
//     petición se rechazaba antes de llegar al procesamiento de seguridad) hasta corregirlo.
package soap

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/beevik/etree"
	"github.com/google/uuid"
	dsig "github.com/russellhaering/goxmldsig"
)

const (
	nsSOAP12 = "http://www.w3.org/2003/05/soap-envelope"
	nsWSA    = "http://www.w3.org/2005/08/addressing"
	nsWCF    = "http://wcf.dian.colombia"
	nsWSSE   = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd"
	nsWSU    = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd"
	nsDS     = "http://www.w3.org/2000/09/xmldsig#"
	nsEC14N  = "http://www.w3.org/2001/10/xml-exc-c14n#"

	actionBase = nsWCF + "/IWcfDianCustomerServices/"

	x509v3ValueType       = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-x509-token-profile-1.0#X509v3"
	base64EncodingType    = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary"
	signatureMethodRSA256 = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
	digestMethodSHA256    = "http://www.w3.org/2001/04/xmlenc#sha256"

	// inclusiveNSPrefixList fuerza la declaración de estos prefijos en la forma canónica
	// aunque el elemento firmado no los "utilice visiblemente" en el sentido estricto del
	// algoritmo — es exactamente para qué existe el mecanismo InclusiveNamespaces de
	// EXC-C14N, y es lo que usan las implementaciones reales contra este mismo servicio.
	inclusiveNSPrefixList = "wsa soap wcf"
)

// exclusiveCanonicalizer implementa "http://www.w3.org/2001/10/xml-exc-c14n#" — el algoritmo
// que exige WS-Security, distinto del C14N inclusivo que usa la firma XAdES del documento
// (paquete signer). No comparten canonicalizador a propósito: son capas de firma distintas.
var exclusiveCanonicalizer = dsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList(inclusiveNSPrefixList)

// buildEnvelope construye el sobre SOAP 1.2 completo: WS-Addressing + WS-Security (firmando
// solo wsa:To) + el cuerpo que aporte bodyBuilder.
func (c *Client) buildEnvelope(action string, bodyBuilder func(body *etree.Element)) (*etree.Document, error) {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)

	env := doc.CreateElement("soap:Envelope")
	env.CreateAttr("xmlns:soap", nsSOAP12)
	env.CreateAttr("xmlns:wcf", nsWCF)

	header := env.CreateElement("soap:Header")

	actionEl := header.CreateElement("wsa:Action")
	actionEl.CreateAttr("xmlns:wsa", nsWSA)
	actionEl.SetText(actionBase + action)

	const toID = "_to"
	toEl := header.CreateElement("wsa:To")
	toEl.CreateAttr("xmlns:wsa", nsWSA)
	toEl.CreateAttr("xmlns:wsu", nsWSU)
	// xmlns:soap y xmlns:wcf se declaran aquí también, redundantes para el contenido de
	// wsa:To, pero necesarios para que inclusiveNSPrefixList tenga algo que retener al
	// canonicalizar este elemento de forma aislada (ver nota en exclusiveCanonicalizer).
	toEl.CreateAttr("xmlns:soap", nsSOAP12)
	toEl.CreateAttr("xmlns:wcf", nsWCF)
	toEl.CreateAttr("wsu:Id", toID)
	toEl.SetText(c.BaseURL)

	replyTo := header.CreateElement("wsa:ReplyTo")
	replyTo.CreateAttr("xmlns:wsa", nsWSA)
	replyTo.CreateElement("wsa:Address").SetText(nsWSA + "/anonymous")

	msgID := header.CreateElement("wsa:MessageID")
	msgID.CreateAttr("xmlns:wsa", nsWSA)
	msgID.SetText("urn:uuid:" + uuid.New().String())

	if err := c.appendSecurityHeader(header, toEl, toID); err != nil {
		return nil, fmt.Errorf("soap: construir wsse:Security: %w", err)
	}

	body := env.CreateElement("soap:Body")
	bodyBuilder(body)

	return doc, nil
}

func (c *Client) appendSecurityHeader(header, toEl *etree.Element, toID string) error {
	sec := header.CreateElement("wsse:Security")
	sec.CreateAttr("xmlns:wsse", nsWSSE)
	sec.CreateAttr("soap:mustUnderstand", "1")

	now := time.Now().UTC()
	ts := sec.CreateElement("wsu:Timestamp")
	ts.CreateAttr("xmlns:wsu", nsWSU)
	ts.CreateAttr("wsu:Id", "_ts")
	ts.CreateElement("wsu:Created").SetText(now.Format("2006-01-02T15:04:05.000Z"))
	ts.CreateElement("wsu:Expires").SetText(now.Add(5 * time.Minute).Format("2006-01-02T15:04:05.000Z"))

	tokenID := "X509-" + uuid.New().String()
	bst := sec.CreateElement("wsse:BinarySecurityToken")
	// xmlns:wsu debe declararse aquí: no se hereda del Timestamp hermano (los namespaces
	// declarados en XML solo bajan a descendientes, no se comparten entre hermanos). Sin
	// esto, "wsu:Id" usa un prefijo no declarado y un parser estricto de namespaces (como
	// el de WCF) rechaza el documento antes de procesar la seguridad — exactamente el 400
	// vacío que devolvía el servidor real.
	bst.CreateAttr("xmlns:wsu", nsWSU)
	bst.CreateAttr("wsu:Id", tokenID)
	bst.CreateAttr("EncodingType", base64EncodingType)
	bst.CreateAttr("ValueType", x509v3ValueType)
	bst.SetText(base64.StdEncoding.EncodeToString(c.Cert.Raw))

	return c.appendSignature(sec, toEl, toID, tokenID)
}

// appendSignature firma únicamente toEl (el header wsa:To, vía Reference URI="#"+toID).
func (c *Client) appendSignature(sec, toEl *etree.Element, toID, tokenID string) error {
	canonTo, err := exclusiveCanonicalizer.Canonicalize(toEl)
	if err != nil {
		return err
	}
	digestTo := sha256.Sum256(canonTo)

	sigEl := sec.CreateElement("ds:Signature")
	sigEl.CreateAttr("xmlns:ds", nsDS)

	signedInfo := sigEl.CreateElement("ds:SignedInfo")
	// El canonicalizador exclusivo (a diferencia del inclusivo que usa el paquete signer
	// para XAdES) no hereda namespaces de ancestros: trata el elemento canonicalizado como
	// si fuera la raíz. xmlns:ds/soap/wcf deben estar declarados dentro de este subárbol.
	signedInfo.CreateAttr("xmlns:ds", nsDS)
	signedInfo.CreateAttr("xmlns:soap", nsSOAP12)
	signedInfo.CreateAttr("xmlns:wcf", nsWCF)

	canonMethod := signedInfo.CreateElement("ds:CanonicalizationMethod")
	canonMethod.CreateAttr("Algorithm", nsEC14N)
	appendInclusiveNamespaces(canonMethod)

	signedInfo.CreateElement("ds:SignatureMethod").CreateAttr("Algorithm", signatureMethodRSA256)

	ref := signedInfo.CreateElement("ds:Reference")
	ref.CreateAttr("URI", "#"+toID)
	transform := ref.CreateElement("ds:Transforms").CreateElement("ds:Transform")
	transform.CreateAttr("Algorithm", nsEC14N)
	appendInclusiveNamespaces(transform)
	ref.CreateElement("ds:DigestMethod").CreateAttr("Algorithm", digestMethodSHA256)
	ref.CreateElement("ds:DigestValue").SetText(base64.StdEncoding.EncodeToString(digestTo[:]))

	canonSignedInfo, err := exclusiveCanonicalizer.Canonicalize(signedInfo)
	if err != nil {
		return err
	}
	hashed := sha256.Sum256(canonSignedInfo)
	signature, err := rsa.SignPKCS1v15(rand.Reader, c.Key, crypto.SHA256, hashed[:])
	if err != nil {
		return err
	}
	sigEl.CreateElement("ds:SignatureValue").SetText(base64.StdEncoding.EncodeToString(signature))

	keyInfo := sigEl.CreateElement("ds:KeyInfo")
	str := keyInfo.CreateElement("wsse:SecurityTokenReference")
	str.CreateAttr("xmlns:wsse", nsWSSE)
	tokenRef := str.CreateElement("wsse:Reference")
	tokenRef.CreateAttr("URI", "#"+tokenID)
	tokenRef.CreateAttr("ValueType", x509v3ValueType)

	return nil
}

func appendInclusiveNamespaces(transformOrMethod *etree.Element) {
	ec := transformOrMethod.CreateElement("ec:InclusiveNamespaces")
	ec.CreateAttr("xmlns:ec", nsEC14N)
	ec.CreateAttr("PrefixList", inclusiveNSPrefixList)
}
