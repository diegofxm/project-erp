package domain

import "time"

// Bogota es el huso horario de Colombia: UTC-5 fijo, sin horario de verano. Se usa
// time.FixedZone en vez de time.LoadLocation("America/Bogota") para no depender de que el
// sistema (o el binario) tenga cargada la base de datos de zonas horarias IANA.
var Bogota = time.FixedZone("America/Bogota", -5*60*60)
