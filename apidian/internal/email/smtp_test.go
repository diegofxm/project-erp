package email

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func sampleConfig() Config {
	return Config{
		Host:        "smtp.example.test",
		Port:        587,
		Username:    "facturacion@miempresa.test",
		Password:    "secreto",
		FromAddress: "facturacion@miempresa.test",
		FromName:    "Mi Empresa SAS",
	}
}

func renderedMsg(t *testing.T, cfg Config, msg Message) string {
	t.Helper()
	m, err := buildMsg(cfg, msg)
	if err != nil {
		t.Fatalf("buildMsg() error = %v", err)
	}
	var buf bytes.Buffer
	if _, err := m.WriteTo(&buf); err != nil {
		t.Fatalf("Msg.WriteTo() error = %v", err)
	}
	return buf.String()
}

func TestBuildMsg_TextAndHTML(t *testing.T) {
	raw := renderedMsg(t, sampleConfig(), Message{
		To:       "cliente@example.test",
		Subject:  "Tu factura electrónica",
		BodyText: "Adjuntamos tu factura.",
		BodyHTML: "<p>Adjuntamos tu factura.</p>",
	})

	for _, want := range []string{
		"To: <cliente@example.test>",
		"From: \"Mi Empresa SAS\" <facturacion@miempresa.test>",
		// go-mail codifica el asunto con acentos en quoted-printable (RFC 2047) — los
		// espacios se vuelven "_" dentro de la palabra codificada.
		"Subject: =?UTF-8?q?Tu_factura_electr",
		"Adjuntamos tu factura.",
		"<p>Adjuntamos tu factura.</p>",
		"multipart/alternative",
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("mensaje generado no contiene %q\n--- mensaje ---\n%s", want, raw)
		}
	}
}

func TestBuildMsg_OnlyText(t *testing.T) {
	raw := renderedMsg(t, sampleConfig(), Message{
		To:       "cliente@example.test",
		Subject:  "Asunto",
		BodyText: "Solo texto plano.",
	})
	if !strings.Contains(raw, "Solo texto plano.") {
		t.Errorf("mensaje generado no contiene el cuerpo de texto\n%s", raw)
	}
}

func TestBuildMsg_WithAttachment(t *testing.T) {
	raw := renderedMsg(t, sampleConfig(), Message{
		To:       "cliente@example.test",
		Subject:  "Asunto",
		BodyText: "Cuerpo",
		Attachments: []Attachment{
			{Filename: "factura.pdf", ContentType: "application/pdf", Content: []byte("%PDF-1.4 contenido falso")},
		},
	})

	for _, want := range []string{
		"factura.pdf",
		"application/pdf",
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("mensaje generado no contiene %q\n--- mensaje ---\n%s", want, raw)
		}
	}
}

func TestBuildMsg_NoFromName(t *testing.T) {
	cfg := sampleConfig()
	cfg.FromName = ""
	raw := renderedMsg(t, cfg, Message{
		To:       "cliente@example.test",
		Subject:  "Asunto",
		BodyText: "Cuerpo",
	})
	if !strings.Contains(raw, "From: <facturacion@miempresa.test>") {
		t.Errorf("mensaje generado no usó la dirección de remitente sin nombre\n%s", raw)
	}
}

func TestSMTPSender_Send_Validations(t *testing.T) {
	validMsg := Message{To: "cliente@example.test", Subject: "Asunto", BodyText: "Cuerpo"}

	tests := []struct {
		name string
		cfg  Config
		msg  Message
	}{
		{"sin host configurado", Config{}, validMsg},
		{"sin destinatario", sampleConfig(), Message{Subject: "Asunto", BodyText: "Cuerpo"}},
		{"sin asunto", sampleConfig(), Message{To: "cliente@example.test", BodyText: "Cuerpo"}},
		{"sin cuerpo", sampleConfig(), Message{To: "cliente@example.test", Subject: "Asunto"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewSMTPSender(tt.cfg).Send(context.Background(), tt.msg)
			if err == nil {
				t.Fatal("Send() esperaba un error de validación, devolvió nil")
			}
		})
	}
}
