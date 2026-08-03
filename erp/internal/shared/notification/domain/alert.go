package domain

// Severity clasifica qué tan urgente es una alerta del sistema.
type Severity string

const (
	SeverityWarning Severity = "warning"
	SeverityDanger  Severity = "danger"
)

// Alert es un aviso generado por el sistema (certificado por vencer, rango de
// numeración agotado, etc.) — no un mensaje saliente (ver Notifier/Message en
// notifier.go, que es para envío por correo/SMS/etc.). Pensado para mostrarse
// en la campana del navbar hoy, y reutilizarse desde otros módulos más adelante
// (cualquier módulo que necesite avisar algo al usuario sin enviar un correo).
//
// ID es una clave determinística (ej. "cert_expired:2026-01-01T00:00:00Z") para
// que la misma condición produzca siempre el mismo ID — así el frontend puede
// recordar qué ya se leyó/descartó sin necesitar persistencia en el servidor.
type Alert struct {
	ID       string
	Severity Severity
	Title    string
	Message  string
	LinkTo   string
}
