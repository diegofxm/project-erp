package documents_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"

	"software.sslmate.com/src/go-pkcs12"
)

const testP12Password = "clave-de-prueba"

// selfSignedTestP12 genera un certificado autofirmado + llave RSA y lo empaqueta como bytes
// PKCS12 — equivalente de prueba al .p12 real, para que signer.LoadPKCS12 (llamado dentro de
// documents.Service.IssueInvoice) lo acepte sin depender de credenciales reales.
func selfSignedTestP12() ([]byte, string) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "apidian test"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		panic(err)
	}

	p12, err := pkcs12.Modern.Encode(key, cert, nil, testP12Password)
	if err != nil {
		panic(err)
	}
	return p12, testP12Password
}
