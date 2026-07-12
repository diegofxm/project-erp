package email

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	gomail "github.com/wneessen/go-mail"
)

// Config son las credenciales/parámetros del servidor SMTP. Se usa SMTP (no API de correo)
// porque la facturación electrónica requiere adjuntar archivos binarios (ZIP con XML + PDF);
// solo cambia la config entre entornos, no el código.
type Config struct {
	Host        string
	Port        int
	Username    string
	Password    string
	FromAddress string
	FromName    string
}

// SMTPSender envía Message por SMTP. Sin interfaz propia todavía — el día que un consumidor
// real lo necesite (ej. documents al enviar la factura al cliente), define su propio puerto
// angosto satisfecho por *SMTPSender estructuralmente, mismo patrón ya usado en todo el
// proyecto (ver documents.IssuerPort/CatalogPort).
type SMTPSender struct {
	cfg Config
}

func NewSMTPSender(cfg Config) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

// Send valida y envía el mensaje. Las validaciones son tempranas y con mensajes claros — no se
// deja que fallen como un error críptico de red o de la librería SMTP.
func (s *SMTPSender) Send(ctx context.Context, msg Message) error {
	if s.cfg.Host == "" {
		return errors.New("email: SMTP no configurado (falta SMTP_HOST)")
	}
	if msg.To == "" {
		return errors.New("email: el destinatario (To) es obligatorio")
	}
	if msg.Subject == "" {
		return errors.New("email: el asunto (Subject) es obligatorio")
	}
	if msg.BodyText == "" && msg.BodyHTML == "" {
		return errors.New("email: el mensaje debe tener al menos un cuerpo (texto u HTML)")
	}

	m, err := buildMsg(s.cfg, msg)
	if err != nil {
		return fmt.Errorf("email: armando el mensaje: %w", err)
	}

	client, err := gomail.NewClient(s.cfg.Host,
		gomail.WithPort(s.cfg.Port),
		gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover),
		gomail.WithUsername(s.cfg.Username),
		gomail.WithPassword(s.cfg.Password),
		gomail.WithTLSPolicy(gomail.TLSMandatory),
	)
	if err != nil {
		return fmt.Errorf("email: configurando el cliente SMTP: %w", err)
	}

	if err := client.DialAndSendWithContext(ctx, m); err != nil {
		return fmt.Errorf("email: enviando a %s: %w", msg.To, err)
	}
	return nil
}

// buildMsg traduce nuestro Message (independiente de la librería) a *gomail.Msg — separado de
// Send para poder probarlo sin red (ver smtp_test.go).
func buildMsg(cfg Config, msg Message) (*gomail.Msg, error) {
	m := gomail.NewMsg()

	if cfg.FromName != "" {
		if err := m.FromFormat(cfg.FromName, cfg.FromAddress); err != nil {
			return nil, err
		}
	} else if err := m.From(cfg.FromAddress); err != nil {
		return nil, err
	}
	if err := m.To(msg.To); err != nil {
		return nil, err
	}
	m.Subject(msg.Subject)

	switch {
	case msg.BodyText != "" && msg.BodyHTML != "":
		m.SetBodyString(gomail.TypeTextPlain, msg.BodyText)
		m.AddAlternativeString(gomail.TypeTextHTML, msg.BodyHTML)
	case msg.BodyHTML != "":
		m.SetBodyString(gomail.TypeTextHTML, msg.BodyHTML)
	default:
		m.SetBodyString(gomail.TypeTextPlain, msg.BodyText)
	}

	for _, a := range msg.Attachments {
		err := m.AttachReader(a.Filename, bytes.NewReader(a.Content),
			gomail.WithFileContentType(gomail.ContentType(a.ContentType)))
		if err != nil {
			return nil, err
		}
	}

	return m, nil
}
