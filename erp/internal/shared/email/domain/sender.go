package domain

import "context"

// Message representa un correo electrónico.
type Message struct {
	To      []string
	CC      []string
	Subject string
	HTML    string
	Text    string
}

// Sender es el puerto de envío de correo.
// Las implementaciones concretas (SMTP, Resend, SES) satisfacen esta interfaz.
type Sender interface {
	Send(ctx context.Context, m Message) error
}
