package timeutil

import "time"

// Bogota es la zona horaria de Colombia (UTC-5, sin horario de verano).
// Se inicializa al arrancar; si el sistema no tiene la base de datos de zonas
// horarias IANA (tz), cae en un offset fijo de -5h.
var Bogota *time.Location

func init() {
	loc, err := time.LoadLocation("America/Bogota")
	if err != nil {
		loc = time.FixedZone("COT", -5*60*60)
	}
	Bogota = loc
}

// Now devuelve la hora actual en la zona horaria de Colombia.
func Now() time.Time {
	return time.Now().In(Bogota)
}

// Today devuelve la fecha de hoy en Colombia (sin componente horario).
func Today() time.Time {
	n := Now()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, Bogota)
}

// In convierte cualquier time.Time a la zona horaria de Colombia.
func In(t time.Time) time.Time {
	return t.In(Bogota)
}
