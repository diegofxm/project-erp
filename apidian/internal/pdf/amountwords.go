package pdf

import "strings"

var unidades = []string{"", "uno", "dos", "tres", "cuatro", "cinco", "seis", "siete", "ocho", "nueve"}
var especiales10a19 = []string{"diez", "once", "doce", "trece", "catorce", "quince", "dieciséis", "diecisiete", "dieciocho", "diecinueve"}
var decenas = []string{"", "", "veinte", "treinta", "cuarenta", "cincuenta", "sesenta", "setenta", "ochenta", "noventa"}
var centenas = []string{"", "ciento", "doscientos", "trescientos", "cuatrocientos", "quinientos", "seiscientos", "setecientos", "ochocientos", "novecientos"}

// AmountInWords convierte un monto en pesos colombianos (entero, sin centavos — la DIAN no
// usa decimales de peso) a su representación en letras para el "Son: ___ PESOS" de la
// representación gráfica. Solo se usa para montos no negativos — un total negativo no tiene
// sentido en una factura.
func AmountInWords(pesos int64) string {
	if pesos == 0 {
		return "CERO PESOS"
	}
	return strings.ToUpper(strings.TrimSpace(numberToWords(pesos))) + " PESOS"
}

func numberToWords(n int64) string {
	if n == 0 {
		return "cero"
	}
	if n < 0 {
		return "menos " + numberToWords(-n)
	}

	switch {
	case n < 1000:
		return hundreds(n)
	case n < 1_000_000:
		return groupOf(n, 1000, "mil", "mil")
	case n < 1_000_000_000:
		return groupOf(n, 1_000_000, "un millón", "millones")
	default:
		return groupOf(n, 1_000_000_000, "mil millones", "mil millones")
	}
}

// groupOf descompone n en cuanto*unit + resto, con singular/plural propios del español
// ("mil" nunca lleva "un" adelante; "un millón"/"N millones" sí cambian).
func groupOf(n, unit int64, singular, label string) string {
	count := n / unit
	rest := n % unit

	var prefix string
	switch {
	case unit == 1000:
		if count == 1 {
			prefix = "mil"
		} else {
			prefix = hundreds(count) + " " + label
		}
	case count == 1:
		prefix = singular
	default:
		prefix = numberToWords(count) + " " + label
	}

	if rest == 0 {
		return prefix
	}
	if unit == 1000 {
		return prefix + " " + hundreds(rest)
	}
	return prefix + " " + numberToWords(rest)
}

func hundreds(n int64) string {
	if n == 0 {
		return ""
	}
	if n == 100 {
		return "cien"
	}
	h := n / 100
	rest := n % 100
	var parts []string
	if h > 0 {
		parts = append(parts, centenas[h])
	}
	if rest > 0 {
		parts = append(parts, tens(rest))
	}
	return strings.Join(parts, " ")
}

func tens(n int64) string {
	if n < 10 {
		return unidades[n]
	}
	if n < 20 {
		return especiales10a19[n-10]
	}
	d := n / 10
	u := n % 10
	if u == 0 {
		return decenas[d]
	}
	if d == 2 {
		return "veinti" + unidades[u]
	}
	return decenas[d] + " y " + unidades[u]
}
