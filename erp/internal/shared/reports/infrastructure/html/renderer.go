// Package html implementa el Renderer del dominio usando html/template + embed.
// Renderiza cualquier template como HTML. Los demás generadores (pdf, excel, csv)
// reciben el HTML producido aquí.
//
// TemplateID sigue la convención "modulo/nombre" (sin extensión), p.ej.:
//
//	"payslip"           → templates/payslip.html
//	"invoice_graphic"   → templates/invoice_graphic.html
//	"journal_entry"     → templates/journal_entry.html
package html

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"strings"

	"github.com/diegofxm/erp/internal/shared/reports/domain"
)

//go:embed templates
var templateFS embed.FS

// Renderer implementa domain.Renderer devolviendo HTML.
type Renderer struct {
	tmpl *template.Template
}

// NewRenderer carga todos los .html de templates/ de forma recursiva.
// Los templates se registran con el nombre relativo sin extensión, e.g. "payslip".
func NewRenderer() (*Renderer, error) {
	root := template.New("").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	})

	err := fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, werr error) error {
		if werr != nil || d.IsDir() || !strings.HasSuffix(path, ".html") {
			return werr
		}
		raw, err := templateFS.ReadFile(path)
		if err != nil {
			return err
		}
		// nombre: quita "templates/" y ".html" → "payslip", "invoice_graphic", etc.
		name := strings.TrimSuffix(strings.TrimPrefix(path, "templates/"), ".html")
		if _, err := root.New(name).Parse(string(raw)); err != nil {
			return fmt.Errorf("parsear %s: %w", path, err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reports/html: cargar templates: %w", err)
	}
	return &Renderer{tmpl: root}, nil
}

var _ domain.Renderer = (*Renderer)(nil)

func (r *Renderer) Render(_ context.Context, req domain.RenderRequest) ([]byte, error) {
	if req.Format != domain.FormatHTML {
		return nil, fmt.Errorf("reports/html: formato no soportado: %s", req.Format)
	}
	out, err := r.RenderToString(req.TemplateID, req.Data)
	if err != nil {
		return nil, err
	}
	return []byte(out), nil
}

// RenderToString devuelve HTML a otros generadores (pdf, etc.) sin pasar por el dominio.
func (r *Renderer) RenderToString(templateID string, data map[string]any) (string, error) {
	var buf bytes.Buffer
	if err := r.tmpl.ExecuteTemplate(&buf, templateID, data); err != nil {
		return "", fmt.Errorf("reports/html: render %q: %w", templateID, err)
	}
	return buf.String(), nil
}
