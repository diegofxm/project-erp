package cryptutil

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func randomKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	key := randomKey(t)
	plaintext := []byte("clave-tecnica-secreta-de-la-dian")

	ciphertext, err := Encrypt(key, plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	got, err := Decrypt(key, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestEncrypt_DifferentNoncePerCall(t *testing.T) {
	key := randomKey(t)
	plaintext := []byte("mismo contenido")

	a, err := Encrypt(key, plaintext)
	require.NoError(t, err)
	b, err := Encrypt(key, plaintext)
	require.NoError(t, err)

	assert.False(t, bytes.Equal(a, b), "el mismo texto plano no debería cifrar igual dos veces (nonce distinto)")
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	ciphertext, err := Encrypt(randomKey(t), []byte("secreto"))
	require.NoError(t, err)

	_, err = Decrypt(randomKey(t), ciphertext)
	assert.Error(t, err)
}

func TestEncryptDecrypt_EmptyIsNil(t *testing.T) {
	ciphertext, err := Encrypt(randomKey(t), nil)
	require.NoError(t, err)
	assert.Nil(t, ciphertext)

	plaintext, err := Decrypt(randomKey(t), nil)
	require.NoError(t, err)
	assert.Nil(t, plaintext)
}
