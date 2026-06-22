package documents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/diegofxm/api-dian/internal/sqlutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implementa Repository usando pgx. customer/lines/payment_means/
// billing_reference/discrepancy_response se guardan como JSONB — son snapshots de solo
// lectura, no entidades con su propia tabla (ver model.go).
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository crea el repositorio de documentos sobre PostgreSQL.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// documentColumns lista las columnas en el MISMO orden que los $N del INSERT y los campos
// del Scan — contarlas mal fue un bug real en esta misma fase (ver Fase 2.5), por eso una
// columna por línea aquí, una a una, en vez de una lista corrida.
const documentColumns = `
	id,
	issuer_id,
	numbering_range_id,
	dian_document_type_code,
	prefix,
	number,
	document_key,
	issue_date,
	issue_time,
	currency_code,
	customer,
	lines,
	payment_means,
	totals_line_extension_cents,
	totals_tax_exclusive_cents,
	totals_tax_inclusive_cents,
	totals_prepaid_cents,
	totals_payable_cents,
	billing_reference,
	discrepancy_response,
	note_type_code,
	qr_url,
	signed_xml,
	status,
	dian_track_id,
	dian_status_code,
	dian_status_description,
	dian_status_message,
	created_at,
	updated_at`

func (r *PostgresRepository) Create(ctx context.Context, d Document) (*Document, error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	now := time.Now().UTC()
	d.CreatedAt = now
	d.UpdatedAt = now

	customerJSON, err := json.Marshal(d.Customer)
	if err != nil {
		return nil, fmt.Errorf("serializar customer: %w", err)
	}
	linesJSON, err := json.Marshal(d.Lines)
	if err != nil {
		return nil, fmt.Errorf("serializar lines: %w", err)
	}
	paymentMeansJSON, err := json.Marshal(d.PaymentMeans)
	if err != nil {
		return nil, fmt.Errorf("serializar payment_means: %w", err)
	}

	// any(nil) explícito, no json.Marshal(nil) — un *T nil dentro de un []byte JSONB sería
	// el literal "null", no NULL de verdad; pgx solo trata como NULL un any(nil) genuino.
	var billingRefJSON, discrepancyJSON any
	if d.BillingReference != nil {
		b, err := json.Marshal(d.BillingReference)
		if err != nil {
			return nil, fmt.Errorf("serializar billing_reference: %w", err)
		}
		billingRefJSON = b
	}
	if d.DiscrepancyResponse != nil {
		b, err := json.Marshal(d.DiscrepancyResponse)
		if err != nil {
			return nil, fmt.Errorf("serializar discrepancy_response: %w", err)
		}
		discrepancyJSON = b
	}

	args := []any{
		d.ID,
		d.IssuerID,
		d.NumberingRangeID,
		d.DianDocumentTypeCode,
		d.Prefix,
		d.Number,
		d.DocumentKey,
		d.IssueDate,
		d.IssueTime,
		d.CurrencyCode,
		customerJSON,
		linesJSON,
		paymentMeansJSON,
		d.Totals.LineExtensionCents,
		d.Totals.TaxExclusiveCents,
		d.Totals.TaxInclusiveCents,
		d.Totals.PrepaidCents,
		d.Totals.PayableCents,
		billingRefJSON,
		discrepancyJSON,
		nullableString(d.NoteTypeCode),
		d.QRURL,
		d.SignedXML,
		string(d.Status),
		d.DianTrackID,
		d.DianStatusCode,
		d.DianStatusDescription,
		d.DianStatusMessage,
		d.CreatedAt,
		d.UpdatedAt,
	}

	_, err = r.pool.Exec(ctx, `INSERT INTO documents (`+documentColumns+`) VALUES (`+sqlutil.Placeholders(len(args))+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}
	return &d, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*Document, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+documentColumns+` FROM documents WHERE id = $1`, id)
	return scanDocument(row)
}

func (r *PostgresRepository) UpdateDianStatus(ctx context.Context, id uuid.UUID, status Status, trackID, statusCode, statusDescription, statusMessage string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE documents
		SET status = $1, dian_track_id = $2, dian_status_code = $3, dian_status_description = $4,
		    dian_status_message = $5, updated_at = NOW()
		WHERE id = $6`,
		string(status), trackID, statusCode, statusDescription, statusMessage, id,
	)
	if err != nil {
		return fmt.Errorf("update dian status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDocumentNotFound
	}
	return nil
}

func (r *PostgresRepository) ListByIssuer(ctx context.Context, issuerID uuid.UUID, filter ListFilter) ([]*Document, error) {
	query := `SELECT ` + documentColumns + ` FROM documents WHERE issuer_id = $1`
	args := []any{issuerID}

	if filter.DianDocumentTypeCode != "" {
		args = append(args, filter.DianDocumentTypeCode)
		query += fmt.Sprintf(" AND dian_document_type_code = $%d", len(args))
	}
	if filter.Status != "" {
		args = append(args, string(filter.Status))
		query += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if !filter.From.IsZero() {
		args = append(args, filter.From)
		query += fmt.Sprintf(" AND issue_date >= $%d", len(args))
	}
	if !filter.To.IsZero() {
		args = append(args, filter.To)
		query += fmt.Sprintf(" AND issue_date <= $%d", len(args))
	}

	args = append(args, filter.Limit, filter.Offset)
	query += fmt.Sprintf(" ORDER BY issue_date DESC, created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	var out []*Document
	for rows.Next() {
		d, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func scanDocument(row pgx.Row) (*Document, error) {
	var d Document
	var status string
	var customerJSON, linesJSON, paymentMeansJSON, billingRefJSON, discrepancyJSON []byte
	var dianTrackID, dianStatusCode, dianStatusDescription, dianStatusMessage, noteTypeCode *string

	err := row.Scan(
		&d.ID,
		&d.IssuerID,
		&d.NumberingRangeID,
		&d.DianDocumentTypeCode,
		&d.Prefix,
		&d.Number,
		&d.DocumentKey,
		&d.IssueDate,
		&d.IssueTime,
		&d.CurrencyCode,
		&customerJSON,
		&linesJSON,
		&paymentMeansJSON,
		&d.Totals.LineExtensionCents,
		&d.Totals.TaxExclusiveCents,
		&d.Totals.TaxInclusiveCents,
		&d.Totals.PrepaidCents,
		&d.Totals.PayableCents,
		&billingRefJSON,
		&discrepancyJSON,
		&noteTypeCode,
		&d.QRURL,
		&d.SignedXML,
		&status,
		&dianTrackID,
		&dianStatusCode,
		&dianStatusDescription,
		&dianStatusMessage,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDocumentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan document: %w", err)
	}
	d.Status = Status(status)
	if dianTrackID != nil {
		d.DianTrackID = *dianTrackID
	}
	if dianStatusCode != nil {
		d.DianStatusCode = *dianStatusCode
	}
	if dianStatusDescription != nil {
		d.DianStatusDescription = *dianStatusDescription
	}
	if dianStatusMessage != nil {
		d.DianStatusMessage = *dianStatusMessage
	}
	if noteTypeCode != nil {
		d.NoteTypeCode = *noteTypeCode
	}

	if err := json.Unmarshal(customerJSON, &d.Customer); err != nil {
		return nil, fmt.Errorf("deserializar customer: %w", err)
	}
	if err := json.Unmarshal(linesJSON, &d.Lines); err != nil {
		return nil, fmt.Errorf("deserializar lines: %w", err)
	}
	if len(paymentMeansJSON) > 0 {
		if err := json.Unmarshal(paymentMeansJSON, &d.PaymentMeans); err != nil {
			return nil, fmt.Errorf("deserializar payment_means: %w", err)
		}
	}
	if len(billingRefJSON) > 0 {
		if err := json.Unmarshal(billingRefJSON, &d.BillingReference); err != nil {
			return nil, fmt.Errorf("deserializar billing_reference: %w", err)
		}
	}
	if len(discrepancyJSON) > 0 {
		if err := json.Unmarshal(discrepancyJSON, &d.DiscrepancyResponse); err != nil {
			return nil, fmt.Errorf("deserializar discrepancy_response: %w", err)
		}
	}

	return &d, nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
