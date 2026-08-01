package domain

import "context"

type Channel string

const (
	ChannelEmail    Channel = "email"
	ChannelSMS      Channel = "sms"
	ChannelWhatsApp Channel = "whatsapp"
	ChannelPush     Channel = "push"
)

// Attachment es un archivo adjunto (PDF, XML, ZIP) que se incluye en un Message de email.
type Attachment struct {
	Filename    string
	ContentType string // "application/pdf", "application/xml", "application/zip"
	Content     []byte
}

// Message es lo único que los módulos de negocio conocen del sistema de notificaciones.
type Message struct {
	To          string         // email, teléfono, device token
	Channel     Channel
	Subject     string         // solo para email
	TemplateID  string         // "welcome", "invoice_issued", "payslip", "password_reset"...
	Data        map[string]any // variables del template
	Attachments []Attachment   // adjuntos opcionales (PDF + XML para facturas)
}

// Notifier es el puerto. Los módulos de negocio solo conocen esta interfaz.
// Infrastructure la implementa por canal y proveedor.
type Notifier interface {
	Send(ctx context.Context, msg Message) error
}
