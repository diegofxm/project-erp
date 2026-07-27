// Package sqlutil tiene utilidades genéricas para construir SQL a mano con pgx — sin lógica
// de negocio, solo para no repetir (y volver a desalinear, como ya pasó dos veces en la Fase
// 2.5/2.6) el conteo de columnas vs. placeholders $N en un INSERT.
package sqlutil

import "strconv"

// Placeholders genera "$1,$2,...,$n" — el número de placeholders SIEMPRE coincide con el
// número de argumentos porque se deriva de len(args), no se escribe a mano.
func Placeholders(n int) string {
	out := make([]byte, 0, n*4)
	for i := 1; i <= n; i++ {
		if i > 1 {
			out = append(out, ',')
		}
		out = append(out, '$')
		out = append(out, []byte(strconv.Itoa(i))...)
	}
	return string(out)
}
