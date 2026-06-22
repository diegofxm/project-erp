// Package securitycode calcula sts:SoftwareSecurityCode según la sección 11.8 del Anexo
// Técnico 1.9. No es un CUFE ni un CUDE — es la huella del software autorizado por la DIAN,
// aplica por igual a Invoice/CreditNote/DebitNote, y por eso tiene su propio paquete en vez
// de vivir dentro de cufe o cude.
package securitycode

import (
	"crypto/sha512"
	"encoding/hex"
)

// Compute calcula sts:SoftwareSecurityCode.
//
// softwareID y pin son datos sensibles asignados por la DIAN al activar el software; nunca
// deben loguearse ni persistirse en texto plano fuera de donde ya estén protegidos.
// documentID es el cbc:ID del documento (Invoice/CreditNote/DebitNote/ApplicationResponse),
// es decir Prefijo+Número.
func Compute(softwareID, pin, documentID string) string {
	sum := sha512.Sum384([]byte(softwareID + pin + documentID))
	return hex.EncodeToString(sum[:])
}
