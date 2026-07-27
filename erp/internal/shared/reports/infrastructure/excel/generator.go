// Package excel implementa domain.Renderer produciendo archivos .xlsx usando excelize.
package excel

import (
	"context"
	"fmt"

	"github.com/xuri/excelize/v2"

	"github.com/diegofxm/erp/internal/shared/reports/domain"
)

// Generator implementa domain.Renderer produciendo Excel.
// req.Data debe contener:
//   - "sheet"   string           nombre de la hoja (default "Hoja1")
//   - "headers" []string         cabeceras de columna
//   - "rows"    [][]any          filas de datos
//   - "title"   string           título en A1 (opcional)
type Generator struct{}

func New() *Generator { return &Generator{} }

var _ domain.Renderer = (*Generator)(nil)

func (g *Generator) Render(_ context.Context, req domain.RenderRequest) ([]byte, error) {
	if req.Format != domain.FormatExcel {
		return nil, fmt.Errorf("reports/excel: formato no soportado: %s", req.Format)
	}

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Hoja1"
	if s, ok := req.Data["sheet"].(string); ok && s != "" {
		sheetName = s
	}
	f.SetSheetName("Sheet1", sheetName)

	row := 1

	// Título opcional
	if title, ok := req.Data["title"].(string); ok && title != "" {
		f.SetCellValue(sheetName, cell(1, row), title)
		titleStyle, _ := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Size: 14},
		})
		f.SetCellStyle(sheetName, cell(1, row), cell(1, row), titleStyle)
		row++
		row++ // línea vacía
	}

	// Cabeceras
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1A56DB"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	if headers, ok := req.Data["headers"].([]string); ok {
		for col, h := range headers {
			c := cell(col+1, row)
			f.SetCellValue(sheetName, c, h)
			f.SetCellStyle(sheetName, c, c, headerStyle)
		}
		row++
	}

	// Filas de datos
	if rows, ok := req.Data["rows"].([][]any); ok {
		altStyle, _ := f.NewStyle(&excelize.Style{
			Fill: excelize.Fill{Type: "pattern", Color: []string{"F9FAFB"}, Pattern: 1},
		})
		for i, dataRow := range rows {
			for col, val := range dataRow {
				c := cell(col+1, row)
				f.SetCellValue(sheetName, c, val)
				if i%2 == 1 {
					f.SetCellStyle(sheetName, c, c, altStyle)
				}
			}
			row++
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("reports/excel: escribir buffer: %w", err)
	}
	return buf.Bytes(), nil
}

// cell convierte col (1-indexed) y fila a coordenada Excel (e.g. col=1, row=1 → "A1").
func cell(col, row int) string {
	name, _ := excelize.ColumnNumberToName(col)
	return fmt.Sprintf("%s%d", name, row)
}
