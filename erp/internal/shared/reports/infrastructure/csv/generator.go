// Package csv implementa domain.Renderer produciendo CSV usando la stdlib.
package csv

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"

	"github.com/diegofxm/erp/internal/shared/reports/domain"
)

// Generator implementa domain.Renderer produciendo CSV.
// req.Data debe contener:
//   - "headers" []string   cabeceras de columna
//   - "rows"    [][]string filas de datos (todos strings)
type Generator struct{}

func New() *Generator { return &Generator{} }

var _ domain.Renderer = (*Generator)(nil)

func (g *Generator) Render(_ context.Context, req domain.RenderRequest) ([]byte, error) {
	if req.Format != domain.FormatCSV {
		return nil, fmt.Errorf("reports/csv: formato no soportado: %s", req.Format)
	}

	var buf bytes.Buffer
	// BOM UTF-8 para que Excel abra el CSV con tildes correctamente
	buf.WriteString("\xEF\xBB\xBF")

	w := csv.NewWriter(&buf)

	if headers, ok := req.Data["headers"].([]string); ok {
		if err := w.Write(headers); err != nil {
			return nil, fmt.Errorf("reports/csv: escribir cabeceras: %w", err)
		}
	}

	if rows, ok := req.Data["rows"].([][]string); ok {
		for _, row := range rows {
			if err := w.Write(row); err != nil {
				return nil, fmt.Errorf("reports/csv: escribir fila: %w", err)
			}
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("reports/csv: flush: %w", err)
	}
	return buf.Bytes(), nil
}
