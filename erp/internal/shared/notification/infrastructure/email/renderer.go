// Package email contiene el renderer de templates y las implementaciones de envío de correo.
package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

var tmpl = template.Must(template.ParseFS(templateFS, "templates/*.html"))

// Render renderiza un template por ID con los datos dados y devuelve HTML.
// templateID es el nombre del archivo sin extensión: "welcome", "invoice_issued", etc.
func Render(templateID string, data any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, templateID+".html", data); err != nil {
		return "", fmt.Errorf("email: render template %q: %w", templateID, err)
	}
	return buf.String(), nil
}
