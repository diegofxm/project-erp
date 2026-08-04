// Package dolarapi implementa domain.TRMFetcher consultando la TRM oficial publicada por la
// Superintendencia Financiera de Colombia, vía el espejo público https://co.dolarapi.com.
// Deliberadamente usa el endpoint /v1/trm (un valor oficial por día) y NO /v1/cotizaciones/usd
// (compra/venta comercial, con spread) — ese segundo no es la tasa que exige la norma colombiana
// para contabilizar transacciones en moneda extranjera.
package dolarapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

const trmURL = "https://co.dolarapi.com/v1/trm"

type Client struct {
	http *http.Client
}

func New() *Client {
	return &Client{http: &http.Client{Timeout: 10 * time.Second}}
}

var _ domain.TRMFetcher = (*Client)(nil)

type trmResponse struct {
	Unidad             string  `json:"unidad"`
	Nombre             string  `json:"nombre"`
	Valor              float64 `json:"valor"`
	FechaActualizacion string  `json:"fechaActualizacion"`
}

func (c *Client) FetchTRM(ctx context.Context) (float64, time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trmURL, nil)
	if err != nil {
		return 0, time.Time{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("dolarapi no disponible: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, time.Time{}, fmt.Errorf("dolarapi respondió %d", resp.StatusCode)
	}

	var out trmResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, time.Time{}, fmt.Errorf("respuesta de dolarapi inválida: %w", err)
	}
	if out.Valor <= 0 {
		return 0, time.Time{}, fmt.Errorf("dolarapi devolvió un valor inválido: %v", out.Valor)
	}

	date, err := time.Parse(time.RFC3339, out.FechaActualizacion)
	if err != nil {
		date = time.Now().UTC()
	}
	return out.Valor, date, nil
}
