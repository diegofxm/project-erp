// Package nit calcula el dígito de verificación módulo 11 de un NIT colombiano — el mismo
// algoritmo que usa la DIAN en el RUT. Solo aplica a identification_type_code "31" (NIT), ver
// domain.Identification.VerificationCode en cofacture/domain/types.go.
package nit

import (
	"errors"
	"strings"
)

// ErrInvalidNumber: el número no quedó vacío tras limpiar separadores, pero contiene algo no
// numérico — un NIT real siempre es solo dígitos.
var ErrInvalidNumber = errors.New("nit: el número debe contener solo dígitos")

// weights son los pesos del algoritmo, aplicados de derecha a izquierda (posición 1 = dígito
// menos significativo). Verificado contra un NIT real con dígito de verificación conocido
// (6382356 -> 7) antes de confiar en la tabla — no es una tabla copiada sin probar.
var weights = []int{3, 7, 13, 17, 19, 23, 29, 37, 41, 43, 47, 53, 59, 67, 71}

// ComputeCheckDigit calcula el dígito de verificación de number (puede traer puntos/guiones,
// se limpian antes de calcular). Devuelve ErrInvalidNumber si, tras limpiar, queda vacío o con
// caracteres no numéricos, o si tiene más dígitos que pesos definidos (un NIT real nunca llega
// a 15 dígitos).
func ComputeCheckDigit(number string) (string, error) {
	digits := strings.Map(func(r rune) rune {
		if r == '.' || r == '-' || r == ' ' {
			return -1
		}
		return r
	}, number)
	if digits == "" || len(digits) > len(weights) {
		return "", ErrInvalidNumber
	}

	sum := 0
	for i := 0; i < len(digits); i++ {
		d := digits[len(digits)-1-i]
		if d < '0' || d > '9' {
			return "", ErrInvalidNumber
		}
		sum += int(d-'0') * weights[i]
	}

	remainder := sum % 11
	if remainder == 0 || remainder == 1 {
		return itoa(remainder), nil
	}
	return itoa(11 - remainder), nil
}

func itoa(n int) string {
	return string(rune('0' + n))
}
