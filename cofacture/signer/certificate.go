package signer

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"

	"software.sslmate.com/src/go-pkcs12"
)

// LoadPEM carga un certificado y su llave privada RSA desde bytes PEM (típicamente dos
// archivos separados: *_cert.pem y *_key.pem, extraídos de un .p12).
func LoadPEM(certPEM, keyPEM []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, errors.New("signer: no se encontró un bloque PEM de certificado")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, errors.New("signer: no se encontró un bloque PEM de llave privada")
	}
	key, err := parsePrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}

// LoadPKCS12 carga un certificado y su llave privada RSA desde un archivo .p12/.pfx, tal
// como lo entrega la autoridad certificadora.
func LoadPKCS12(data []byte, password string) (*x509.Certificate, *rsa.PrivateKey, error) {
	key, cert, err := pkcs12.Decode(data, password)
	if err != nil {
		return nil, nil, err
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, nil, errors.New("signer: la llave privada del .p12 no es RSA")
	}
	return cert, rsaKey, nil
}

func parsePrivateKey(der []byte) (*rsa.PrivateKey, error) {
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("signer: la llave privada PKCS8 no es RSA")
		}
		return rsaKey, nil
	}
	return x509.ParsePKCS1PrivateKey(der)
}
