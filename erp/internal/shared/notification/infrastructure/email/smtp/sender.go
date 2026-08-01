// Package smtp implementa el puerto Notifier para correo saliente vía SMTP estándar.
// Compatible con cualquier servidor SMTP: Mox, Postfix, Gmail relay, etc.
// Usa go-mail para manejar adjuntos y MIME multiparte correctamente.
package smtp

import (
	"bytes"
	"context"
	"fmt"

	gomail "github.com/wneessen/go-mail"

	notificationdomain "github.com/diegofxm/erp/internal/shared/notification/domain"
	emailinfra "github.com/diegofxm/erp/internal/shared/notification/infrastructure/email"
)

// Config contiene los parámetros de conexión SMTP.
type Config struct {
	Host     string // e.g. "mail.midominio.co" (Mox) o "smtp.gmail.com"
	Port     int    // 587 STARTTLS, 465 SMTPS, 25 plain
	Username string
	Password string
	From     string // "ERP <noreply@midominio.co>"
	TLS      bool   // true → SMTPS (puerto 465); false → STARTTLS (puerto 587)
}

type Sender struct{ cfg Config }

func New(cfg Config) *Sender { return &Sender{cfg: cfg} }

var _ notificationdomain.Notifier = (*Sender)(nil)

func (s *Sender) Send(ctx context.Context, msg notificationdomain.Message) error {
	if msg.Channel != notificationdomain.ChannelEmail {
		return fmt.Errorf("smtp: canal no soportado: %s", msg.Channel)
	}

	htmlBody, err := emailinfra.Render(msg.TemplateID, msg.Data)
	if err != nil {
		return err
	}

	m := gomail.NewMsg()
	if err := m.From(s.cfg.From); err != nil {
		return fmt.Errorf("smtp: From: %w", err)
	}
	if err := m.To(msg.To); err != nil {
		return fmt.Errorf("smtp: To: %w", err)
	}
	m.Subject(msg.Subject)
	m.SetBodyString(gomail.TypeTextHTML, htmlBody)

	for _, a := range msg.Attachments {
		if err := m.AttachReader(a.Filename, bytes.NewReader(a.Content),
			gomail.WithFileContentType(gomail.ContentType(a.ContentType)),
		); err != nil {
			return fmt.Errorf("smtp: adjunto %q: %w", a.Filename, err)
		}
	}

	tlsPolicy := gomail.TLSOpportunistic
	if s.cfg.TLS {
		tlsPolicy = gomail.TLSMandatory
	}

	client, err := gomail.NewClient(s.cfg.Host,
		gomail.WithPort(s.cfg.Port),
		gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover),
		gomail.WithUsername(s.cfg.Username),
		gomail.WithPassword(s.cfg.Password),
		gomail.WithTLSPolicy(tlsPolicy),
	)
	if err != nil {
		return fmt.Errorf("smtp: configurando cliente: %w", err)
	}

	if err := client.DialAndSendWithContext(ctx, m); err != nil {
		return fmt.Errorf("smtp: enviando a %s: %w", msg.To, err)
	}
	return nil
}
