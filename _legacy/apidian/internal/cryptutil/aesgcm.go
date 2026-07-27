// Package cryptutil cifra secretos en reposo (credenciales DIAN: PIN, certificado, clave
// técnica) con AES-256-GCM. Lo usan internal/issuers e internal/numbering — vive aparte de
// los dos para que ninguno dependa del otro (ver docs/apidian-architecture.md sección 4.1),
// y para no duplicar código de seguridad que después haya que mantener en dos sitios.
package cryptutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// Encrypt cifra plaintext con AES-256-GCM. key debe ser de 32 bytes. El nonce va al inicio
// del resultado (patrón estándar de GCM), no se guarda por separado.
func Encrypt(key, plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt revierte Encrypt. Devuelve nil si ciphertext está vacío (simétrico con Encrypt).
func Decrypt(key, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("cryptutil: ciphertext más corto que el nonce esperado")
	}
	nonce, data := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, data, nil)
}
