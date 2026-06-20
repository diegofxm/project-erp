// Package signer firma documentos UBL con XAdES-EPES, tal como lo exige la DIAN.
//
// La estructura (tres ds:Reference — documento, KeyInfo y SignedProperties — más la
// política de firma fija de la DIAN) fue verificada byte a byte contra dos facturas
// electrónicas reales y aceptadas, de dos proveedores tecnológicos distintos, no inferida
// solo del anexo técnico.
package signer

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/beevik/etree"
	"github.com/google/uuid"
	dsig "github.com/russellhaering/goxmldsig"
)

const (
	digestMethodSHA256       = "http://www.w3.org/2001/04/xmlenc#sha256"
	signatureMethodRSASHA256 = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
	envelopedSignatureAlgo   = "http://www.w3.org/2000/09/xmldsig#enveloped-signature"
	signedPropertiesRefType  = "http://uri.etsi.org/01903#SignedProperties"
)

// canonicalizer implementa exactamente "http://www.w3.org/TR/2001/REC-xml-c14n-20010315"
// (C14N 1.0 inclusivo) — el algoritmo que usa la DIAN, distinto del exclusivo (xml-exc-c14n).
var canonicalizer = dsig.MakeC14N10RecCanonicalizer()

// Signer firma documentos UBL con un certificado y llave privada RSA concretos.
type Signer struct {
	cert *x509.Certificate
	key  *rsa.PrivateKey
}

// New crea un Signer a partir de un certificado y llave ya cargados (ver LoadPEM/LoadPKCS12).
func New(cert *x509.Certificate, key *rsa.PrivateKey) *Signer {
	return &Signer{cert: cert, key: key}
}

// Sign firma root insertando un ds:Signature XAdES-EPES dentro de placeholder.
//
// placeholder debe ser un ext:ExtensionContent vacío ya ubicado en su posición final dentro
// del documento (el último ext:UBLExtension de ext:UBLExtensions, tal como lo deja
// builder.BuildInvoice) — la firma necesita estar en su posición final para que la
// canonicalización inclusiva (C14N 1.0 REC) herede correctamente los namespaces del
// documento. root es el elemento raíz del documento (Invoice/CreditNote/DebitNote/
// AttachedDocument); su digest (ds:Reference URI="") se calcula antes de tocar el árbol.
//
// role es el valor de xades:ClaimedRole: "supplier" para Invoice/CreditNote/DebitNote,
// "" para la firma del AttachedDocument (verificado contra ambos casos en facturas reales).
func (s *Signer) Sign(root, placeholder *etree.Element, role string, signingTime time.Time) error {
	docDigest, err := digestValue(root)
	if err != nil {
		return fmt.Errorf("signer: digest del documento: %w", err)
	}

	id := "xmldsig-" + uuid.New().String()
	sigEl := placeholder.CreateElement("ds:Signature")
	sigEl.CreateAttr("Id", id)

	keyInfoEl := s.buildKeyInfo(sigEl, id)
	keyInfoDigest, err := digestValue(keyInfoEl)
	if err != nil {
		return fmt.Errorf("signer: digest de KeyInfo: %w", err)
	}

	signedPropsEl := s.buildSignedProperties(sigEl, id, role, signingTime)
	signedPropsDigest, err := digestValue(signedPropsEl)
	if err != nil {
		return fmt.Errorf("signer: digest de SignedProperties: %w", err)
	}

	signedInfoEl := buildSignedInfo(id, docDigest, keyInfoDigest, signedPropsDigest)
	sigEl.InsertChildAt(0, signedInfoEl) // orden de esquema: SignedInfo, SignatureValue, KeyInfo, Object

	canonSignedInfo, err := canonicalizer.Canonicalize(signedInfoEl)
	if err != nil {
		return fmt.Errorf("signer: canonicalizar SignedInfo: %w", err)
	}
	hashed := sha256.Sum256(canonSignedInfo)
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA256, hashed[:])
	if err != nil {
		return fmt.Errorf("signer: firmar SignedInfo: %w", err)
	}

	sigValueEl := etree.NewElement("ds:SignatureValue")
	sigValueEl.CreateAttr("Id", id+"-sigvalue")
	sigValueEl.SetText(base64.StdEncoding.EncodeToString(signature))
	sigEl.InsertChildAt(1, sigValueEl)

	return nil
}

func buildSignedInfo(id, docDigest, keyInfoDigest, signedPropsDigest string) *etree.Element {
	signedInfo := etree.NewElement("ds:SignedInfo")
	signedInfo.CreateElement("ds:CanonicalizationMethod").CreateAttr("Algorithm", string(canonicalizer.Algorithm()))
	signedInfo.CreateElement("ds:SignatureMethod").CreateAttr("Algorithm", signatureMethodRSASHA256)

	ref0 := signedInfo.CreateElement("ds:Reference")
	ref0.CreateAttr("Id", id+"-ref0")
	ref0.CreateAttr("URI", "")
	ref0.CreateElement("ds:Transforms").CreateElement("ds:Transform").CreateAttr("Algorithm", envelopedSignatureAlgo)
	ref0.CreateElement("ds:DigestMethod").CreateAttr("Algorithm", digestMethodSHA256)
	ref0.CreateElement("ds:DigestValue").SetText(docDigest)

	ref1 := signedInfo.CreateElement("ds:Reference")
	ref1.CreateAttr("Id", id+"-ref1")
	ref1.CreateAttr("URI", "#"+id+"-keyinfo")
	ref1.CreateElement("ds:DigestMethod").CreateAttr("Algorithm", digestMethodSHA256)
	ref1.CreateElement("ds:DigestValue").SetText(keyInfoDigest)

	ref2 := signedInfo.CreateElement("ds:Reference")
	ref2.CreateAttr("Type", signedPropertiesRefType)
	ref2.CreateAttr("URI", "#"+id+"-signedprops")
	ref2.CreateElement("ds:DigestMethod").CreateAttr("Algorithm", digestMethodSHA256)
	ref2.CreateElement("ds:DigestValue").SetText(signedPropsDigest)

	return signedInfo
}

// digestValue canonicaliza el (ya posicionado en el árbol) y retorna el SHA-256 en base64,
// listo para un ds:DigestValue.
func digestValue(el *etree.Element) (string, error) {
	canon, err := canonicalizer.Canonicalize(el)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canon)
	return base64.StdEncoding.EncodeToString(sum[:]), nil
}
