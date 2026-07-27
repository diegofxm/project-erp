// Package smtp implementa el puerto Notifier para correo saliente vía SMTP estándar.
// Compatible con cualquier servidor SMTP: Mox, Postfix, Gmail relay, etc.
package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

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

// Compile-time check
var _ notificationdomain.Notifier = (*Sender)(nil)

func (s *Sender) Send(_ context.Context, msg notificationdomain.Message) error {
	if msg.Channel != notificationdomain.ChannelEmail {
		return fmt.Errorf("smtp: canal no soportado: %s", msg.Channel)
	}

	body, err := emailinfra.Render(msg.TemplateID, msg.Data)
	if err != nil {
		return err
	}

	raw := buildMIME(s.cfg.From, msg.To, msg.Subject, body)

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	if s.cfg.TLS {
		return sendTLS(addr, s.cfg.Host, auth, s.cfg.From, msg.To, raw)
	}
	return smtp.SendMail(addr, auth, s.cfg.From, []string{msg.To}, raw)
}

func sendTLS(addr, host string, auth smtp.Auth, from, to string, body []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer c.Close()
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("smtp write body: %w", err)
	}
	return w.Close()
}

func buildMIME(from, to, subject, htmlBody string) []byte {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(htmlBody)
	return []byte(b.String())
}

// localHostname devuelve el hostname local para el saludo EHLO si smtp.SendMail lo necesita.
func localHostname() string {
	h, _ := net.LookupAddr("127.0.0.1")
	if len(h) > 0 {
		return strings.TrimSuffix(h[0], ".")
	}
	return "localhost"
}

var _ = localHostname // silenciar linter si no se usa
