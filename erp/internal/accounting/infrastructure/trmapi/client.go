// Package trmapi implementa domain.TRMFetcher consultando el servicio propio de TRM
// (https://github.com/.../trm-colombia, desplegado en co-trm.vercel.app), que a su vez consulta
// el Web Service SOAP oficial de la Superintendencia Financiera de Colombia. Reemplaza al
// espejo público co.dolarapi.com — mismo criterio de fondo: expone la TRM oficial (un valor por
// día), no la tasa comercial de compra/venta con spread.
package trmapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: 15 * time.Second}}
}

var _ domain.TRMFetcher = (*Client)(nil)

type trmResponse struct {
	Data struct {
		Unit         string  `json:"unit"`
		ValidityFrom string  `json:"validityFrom"`
		ValidityTo   string  `json:"validityTo"`
		Value        float64 `json:"value"`
		Success      bool    `json:"success"`
	} `json:"data"`
	Source      string `json:"source"`
	Description string `json:"description"`
}

// FetchTRM pide la TRM de una fecha exacta (YYYY-MM-DD) — nunca se omite el parámetro, así el
// día que aplica siempre es el que decidimos nosotros (hora Colombia), sin depender de en qué
// huso horario interprete "hoy" el servicio remoto.
func (c *Client) FetchTRM(ctx context.Context, date time.Time) (float64, string, error) {
	if c.baseURL == "" {
		return 0, "", fmt.Errorf("TRM_API_URL no configurada")
	}
	url := fmt.Sprintf("%s/trm?date=%s", c.baseURL, date.Format("2006-01-02"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("servicio TRM no disponible: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("servicio TRM respondió %d", resp.StatusCode)
	}

	var out trmResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, "", fmt.Errorf("respuesta del servicio TRM inválida: %w", err)
	}
	if !out.Data.Success || out.Data.Value <= 0 {
		return 0, "", fmt.Errorf("servicio TRM devolvió un valor inválido para %s", date.Format("2006-01-02"))
	}
	return out.Data.Value, out.Description, nil
}
