// Package resend implementa el puerto Notifier usando la API HTTP de Resend.
// Documentación: https://resend.com/docs/api-reference/emails/send-email
package resend

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	notificationdomain "github.com/diegofxm/erp/internal/shared/notification/domain"
	emailinfra "github.com/diegofxm/erp/internal/shared/notification/infrastructure/email"
)

const apiURL = "https://api.resend.com/emails"

type Config struct {
	APIKey string
	From   string // "ERP <noreply@midominio.co>"
}

type Sender struct {
	cfg    Config
	client *http.Client
}

func New(cfg Config) *Sender {
	return &Sender{cfg: cfg, client: &http.Client{}}
}

var _ notificationdomain.Notifier = (*Sender)(nil)

func (s *Sender) Send(ctx context.Context, msg notificationdomain.Message) error {
	if msg.Channel != notificationdomain.ChannelEmail {
		return fmt.Errorf("resend: canal no soportado: %s", msg.Channel)
	}

	htmlBody, err := emailinfra.Render(msg.TemplateID, msg.Data)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"from":    s.cfg.From,
		"to":      []string{msg.To},
		"subject": msg.Subject,
		"html":    htmlBody,
	}

	if len(msg.Attachments) > 0 {
		attachments := make([]map[string]string, 0, len(msg.Attachments))
		for _, a := range msg.Attachments {
			attachments = append(attachments, map[string]string{
				"filename": a.Filename,
				"content":  base64.StdEncoding.EncodeToString(a.Content),
			})
		}
		payload["attachments"] = attachments
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("resend: crear request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("resend: enviar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return fmt.Errorf("resend: status %d: %v", resp.StatusCode, errBody)
	}
	return nil
}
