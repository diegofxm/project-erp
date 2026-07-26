package accounting

import (
	"context"
	"fmt"

	"github.com/diegofxm/accounting"
	"github.com/diegofxm/apidian/internal/documents"
	"github.com/google/uuid"
)

// Client es el punto de entrada del adaptador.
// En Fase 1-2 llama al Core directamente (mismo proceso).
// En Fase 3 (NATS) solo se cambia este archivo — el mapper y el resto del dominio no se tocan.
type Client struct {
	core *accounting.Core
}

// New crea el cliente del adaptador con el Core ya inicializado.
func New(core *accounting.Core) *Client {
	return &Client{core: core}
}

// getDocRate obtiene la tasa de cambio vigente en la fecha del documento.
// Para documentos en COP retorna (0, "", nil) — el mapper interpreta rateX10000==0 como COP nativo.
func (c *Client) getDocRate(ctx context.Context, doc *documents.Document) (rateX10000 int64, currency string, err error) {
	currency = doc.CurrencyCode
	if currency == "" || currency == "COP" {
		return 0, "COP", nil
	}
	rate, err := c.core.Forex.GetRate(ctx, doc.IssueDate, currency)
	if err != nil {
		return 0, currency, fmt.Errorf("accounting client: tasa %s al %s: %w",
			currency, doc.IssueDate.Format("2006-01-02"), err)
	}
	return rate.RateX10000, currency, nil
}

// PostInvoice registra el asiento contable de una FE confirmada.
// No bloquea la confirmación de la FE si el registro falla — se loggea el error
// y se continúa (decisión MVP, ver docs/general-architecture.md sección 5).
func (c *Client) PostInvoice(ctx context.Context, doc *documents.Document, companyID uuid.UUID) error {
	rateX10000, currency, err := c.getDocRate(ctx, doc)
	if err != nil {
		return err
	}
	req, err := fromInvoice(doc, companyID, rateX10000, currency)
	if err != nil {
		return fmt.Errorf("accounting client: mapear FE: %w", err)
	}
	if _, err := c.core.Journals.Post(ctx, *req); err != nil {
		return fmt.Errorf("accounting client: registrar asiento FE: %w", err)
	}
	return nil
}

// PostSupportDocument registra el asiento contable de un DS confirmado.
// expenseAccountCode es el código PUC del gasto/costo correspondiente a la compra.
func (c *Client) PostSupportDocument(ctx context.Context, doc *documents.Document, companyID uuid.UUID, expenseAccountCode string) error {
	rateX10000, currency, err := c.getDocRate(ctx, doc)
	if err != nil {
		return err
	}
	req, err := fromSupportDocument(doc, companyID, expenseAccountCode, rateX10000, currency)
	if err != nil {
		return fmt.Errorf("accounting client: mapear DS: %w", err)
	}
	if _, err := c.core.Journals.Post(ctx, *req); err != nil {
		return fmt.Errorf("accounting client: registrar asiento DS: %w", err)
	}
	return nil
}

// PostCreditNote registra el asiento contable de una NC confirmada.
func (c *Client) PostCreditNote(ctx context.Context, doc *documents.Document, companyID uuid.UUID) error {
	rateX10000, currency, err := c.getDocRate(ctx, doc)
	if err != nil {
		return err
	}
	req, err := fromCreditNote(doc, companyID, rateX10000, currency)
	if err != nil {
		return fmt.Errorf("accounting client: mapear NC: %w", err)
	}
	if _, err := c.core.Journals.Post(ctx, *req); err != nil {
		return fmt.Errorf("accounting client: registrar asiento NC: %w", err)
	}
	return nil
}

// PostDebitNote registra el asiento contable de una ND confirmada.
func (c *Client) PostDebitNote(ctx context.Context, doc *documents.Document, companyID uuid.UUID) error {
	rateX10000, currency, err := c.getDocRate(ctx, doc)
	if err != nil {
		return err
	}
	req, err := fromDebitNote(doc, companyID, rateX10000, currency)
	if err != nil {
		return fmt.Errorf("accounting client: mapear ND: %w", err)
	}
	if _, err := c.core.Journals.Post(ctx, *req); err != nil {
		return fmt.Errorf("accounting client: registrar asiento ND: %w", err)
	}
	return nil
}

// PostAdjustmentNote registra el asiento contable de una NA (Nota de Ajuste al DS) confirmada.
// expenseAccountCode es la misma cuenta PUC que se usó en el DS original.
func (c *Client) PostAdjustmentNote(ctx context.Context, doc *documents.Document, companyID uuid.UUID, expenseAccountCode string) error {
	rateX10000, currency, err := c.getDocRate(ctx, doc)
	if err != nil {
		return err
	}
	req, err := fromAdjustmentNote(doc, companyID, expenseAccountCode, rateX10000, currency)
	if err != nil {
		return fmt.Errorf("accounting client: mapear NA: %w", err)
	}
	if _, err := c.core.Journals.Post(ctx, *req); err != nil {
		return fmt.Errorf("accounting client: registrar asiento NA: %w", err)
	}
	return nil
}
