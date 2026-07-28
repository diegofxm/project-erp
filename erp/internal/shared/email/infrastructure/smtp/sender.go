// Package smtp implementa email.Sender usando net/smtp estándar con TLS.
package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	emaildomain "github.com/diegofxm/erp/internal/shared/email/domain"
)

// Sender envía correos vía SMTP con TLS.
type Sender struct {
	host string
	port string
	user string
	pass string
	from string
}

func New(host, port, user, pass, from string) *Sender {
	return &Sender{host: host, port: port, user: user, pass: pass, from: from}
}

var _ emaildomain.Sender = (*Sender)(nil)

func (s *Sender) Send(_ context.Context, m emaildomain.Message) error {
	addr := net.JoinHostPort(s.host, s.port)
	auth := smtp.PlainAuth("", s.user, s.pass, s.host)
	body := buildRaw(s.from, m)

	// Intentar TLS primero; fallback a STARTTLS vía SendMail
	tlsCfg := &tls.Config{ServerName: s.host}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return smtp.SendMail(addr, auth, s.from, m.To, []byte(body))
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("smtp: nuevo cliente: %w", err)
	}
	defer c.Close()

	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("smtp: auth: %w", err)
	}
	if err := c.Mail(s.from); err != nil {
		return fmt.Errorf("smtp: MAIL FROM: %w", err)
	}
	for _, to := range m.To {
		if err := c.Rcpt(to); err != nil {
			return fmt.Errorf("smtp: RCPT TO %s: %w", to, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp: DATA: %w", err)
	}
	if _, err := fmt.Fprint(w, body); err != nil {
		return fmt.Errorf("smtp: escribir mensaje: %w", err)
	}
	return w.Close()
}

func buildRaw(from string, m emaildomain.Message) string {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + strings.Join(m.To, ", ") + "\r\n")
	if len(m.CC) > 0 {
		b.WriteString("Cc: " + strings.Join(m.CC, ", ") + "\r\n")
	}
	b.WriteString("Subject: " + m.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	if m.HTML != "" {
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		b.WriteString(m.HTML)
	} else {
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		b.WriteString(m.Text)
	}
	return b.String()
}
