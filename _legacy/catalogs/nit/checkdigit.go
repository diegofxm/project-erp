// Package nit calcula el dígito de verificación módulo 11 de un NIT colombiano — el mismo
// algoritmo que usa la DIAN en el RUT. Solo aplica a identification_type_code "31" (NIT).
package nit

import (
	"errors"
	"strings"
)

var ErrInvalidNumber = errors.New("nit: el número debe contener solo dígitos")

// weights son los pesos del algoritmo, aplicados de derecha a izquierda.
// Verificado contra 6382356 → dígito 7 antes de publicar.
var weights = []int{3, 7, 13, 17, 19, 23, 29, 37, 41, 43, 47, 53, 59, 67, 71}

// ComputeCheckDigit calcula el dígito de verificación de number.
// Acepta puntos, guiones y espacios como separadores — se limpian antes de calcular.
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
