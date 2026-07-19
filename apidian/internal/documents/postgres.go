package documents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/diegofxm/apidian/internal/sqlutil"
	"github.com/diegofxm/cofacture/domain"
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
	customer_id,
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
	note,
	qr_url,
	signed_xml,
	status,
	dian_track_id,
	dian_status_code,
	dian_status_description,
	dian_status_message,
	application_response_xml,
	vendor,
	operation_type_code,
	withholding_taxes,
	vendor_id,
	created_at,
	updated_at`

// Create persiste un borrador NUEVO — prefix/number/document_key/issue_date/issue_time/
// qr_url/signed_xml todavía no se conocen (ver model.go), se mandan como NULL real, nunca
// como zero-value: un número 0 sería un dato real ambiguo con "todavía no confirmado".
func (r *PostgresRepository) Create(ctx context.Context, d Document) (*Document, error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	now := time.Now().UTC()
	d.CreatedAt = now
	d.UpdatedAt = now

	customerJSON, linesJSON, paymentMeansJSON, billingRefJSON, discrepancyJSON, vendorJSON, withholdingJSON, err := marshalSnapshots(d)
	if err != nil {
		return nil, err
	}

	args := []any{
		d.ID,
		d.IssuerID,
		d.NumberingRangeID,
		d.DianDocumentTypeCode,
		nullablePrefix(d.Prefix),
		nullableNumber(d.Number),
		nullableString(d.DocumentKey),
		nullableIssueDate(d.IssueDate),
		nullableString(d.IssueTime),
		d.CurrencyCode,
		customerJSON,
		d.CustomerID,
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
		nullableString(d.Note),
		nullableString(d.QRURL),
		nullableString(d.SignedXML),
		string(d.Status),
		d.DianTrackID,
		d.DianStatusCode,
		d.DianStatusDescription,
		d.DianStatusMessage,
		nullableString(d.ApplicationResponseXML),
		vendorJSON,
		nullableString(d.OperationTypeCode),
		withholdingJSON,
		d.VendorID,
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

func (r *PostgresRepository) UpdateDianStatus(ctx context.Context, id uuid.UUID, status Status, trackID, statusCode, statusDescription, statusMessage, applicationResponseXML string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE documents
		SET status = $1, dian_track_id = $2, dian_status_code = $3, dian_status_description = $4,
		    dian_status_message = $5, application_response_xml = $6, updated_at = NOW()
		WHERE id = $7`,
		string(status), trackID, statusCode, statusDescription, statusMessage,
		nullableString(applicationResponseXML), id,
	)
	if err != nil {
		return fmt.Errorf("update dian status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDocumentNotFound
	}
	return nil
}

// UpdateDraft reemplaza los campos editables de un borrador — todo lo que NO depende de
// confirmar (ver Confirm). Acota en el propio SQL (WHERE ... AND status = 'draft'), mismo
// criterio que customers/products.Update: nunca confiar en que el llamador ya comprobó el
// estado, la query misma lo garantiza.
func (r *PostgresRepository) UpdateDraft(ctx context.Context, d Document) (*Document, error) {
	customerJSON, linesJSON, paymentMeansJSON, billingRefJSON, discrepancyJSON, vendorJSON, withholdingJSON, err := marshalSnapshots(d)
	if err != nil {
		return nil, err
	}

	tag, err := r.pool.Exec(ctx, `
		UPDATE documents SET
			numbering_range_id = $1,
			currency_code = $2,
			customer = $3,
			customer_id = $4,
			lines = $5,
			payment_means = $6,
			totals_line_extension_cents = $7,
			totals_tax_exclusive_cents = $8,
			totals_tax_inclusive_cents = $9,
			totals_prepaid_cents = $10,
			totals_payable_cents = $11,
			billing_reference = $12,
			discrepancy_response = $13,
			note_type_code = $14,
			note = $15,
			vendor = $16,
			operation_type_code = $17,
			withholding_taxes = $18,
			vendor_id = $19,
			updated_at = $20
		WHERE id = $21 AND status = 'draft'`,
		d.NumberingRangeID, d.CurrencyCode, customerJSON, d.CustomerID, linesJSON, paymentMeansJSON,
		d.Totals.LineExtensionCents, d.Totals.TaxExclusiveCents, d.Totals.TaxInclusiveCents,
		d.Totals.PrepaidCents, d.Totals.PayableCents, billingRefJSON, discrepancyJSON,
		nullableString(d.NoteTypeCode), nullableString(d.Note),
		vendorJSON, nullableString(d.OperationTypeCode), withholdingJSON,
		d.VendorID, time.Now().UTC(), d.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update draft: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrDocumentNotDraft
	}
	return r.GetByID(ctx, d.ID)
}

// Confirm persiste los campos que solo se conocen al confirmar — el único UPDATE que mueve un
// documento fuera de StatusDraft, por eso acota WHERE status = 'draft' (ErrDocumentNotDraft
// si ya no lo es, ej. una doble confirmación concurrente).
func (r *PostgresRepository) Confirm(ctx context.Context, d Document) (*Document, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE documents SET
			prefix = $1,
			number = $2,
			document_key = $3,
			issue_date = $4,
			issue_time = $5,
			qr_url = $6,
			signed_xml = $7,
			status = $8,
			updated_at = $9
		WHERE id = $10 AND status = 'draft'`,
		d.Prefix, d.Number, d.DocumentKey, d.IssueDate, d.IssueTime, d.QRURL, d.SignedXML,
		string(d.Status), time.Now().UTC(), d.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("confirm document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrDocumentNotDraft
	}
	return r.GetByID(ctx, d.ID)
}

// Delete elimina un borrador — acota WHERE status = 'draft', mismo criterio que UpdateDraft.
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM documents WHERE id = $1 AND status = 'draft'`, id)
	if err != nil {
		return fmt.Errorf("delete draft: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrDocumentNotDraft
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
	if filter.SourceDocumentID != nil {
		// Filtra NC/ND cuya billing_reference apunta a la factura con ese ID. El
		// subquery garantiza que la factura de origen pertenezca al mismo emisor ($1).
		args = append(args, *filter.SourceDocumentID)
		query += fmt.Sprintf(`
			AND (billing_reference->>'prefix', billing_reference->>'number') = (
				SELECT prefix, number::text FROM documents WHERE id = $%d AND issuer_id = $1
			)`, len(args))
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
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Cuenta NC/ND que referencian cada documento confirmado en este resultado.
	// Una sola query sobre billing_reference — nunca N+1 independiente del tamaño de la página.
	if len(out) > 0 {
		type noteCounts struct{ nc, nd int }
		counts := make(map[string]noteCounts)

		noteRows, err := r.pool.Query(ctx, `
			SELECT
				billing_reference->>'prefix'  AS ref_prefix,
				billing_reference->>'number'  AS ref_number,
				SUM(CASE WHEN dian_document_type_code = '91' THEN 1 ELSE 0 END)::int AS nc_count,
				SUM(CASE WHEN dian_document_type_code = '92' THEN 1 ELSE 0 END)::int AS nd_count
			FROM documents
			WHERE issuer_id = $1 AND billing_reference IS NOT NULL
			GROUP BY ref_prefix, ref_number`, issuerID)
		if err == nil {
			for noteRows.Next() {
				var prefix, number string
				var nc, nd int
				if noteRows.Scan(&prefix, &number, &nc, &nd) == nil {
					counts[prefix+"|"+number] = noteCounts{nc: nc, nd: nd}
				}
			}
			_ = noteRows.Err()
			noteRows.Close()
			for _, d := range out {
				if d.Prefix != "" && d.Number != 0 {
					if c, ok := counts[d.Prefix+"|"+fmt.Sprintf("%d", d.Number)]; ok {
						d.NCCount = c.nc
						d.NDCount = c.nd
					}
				}
			}
		}
	}

	return out, nil
}

func scanDocument(row pgx.Row) (*Document, error) {
	var d Document
	var status string
	var customerJSON, linesJSON, paymentMeansJSON, billingRefJSON, discrepancyJSON []byte
	var vendorJSON, withholdingJSON []byte
	var dianTrackID, dianStatusCode, dianStatusDescription, dianStatusMessage, noteTypeCode, note *string
	var applicationResponseXML *string
	var prefix, documentKey, issueTime, qrURL, signedXML *string
	var operationTypeCode *string
	var number *int64
	var issueDate *time.Time
	var vendorID *uuid.UUID

	err := row.Scan(
		&d.ID,
		&d.IssuerID,
		&d.NumberingRangeID,
		&d.DianDocumentTypeCode,
		&prefix,
		&number,
		&documentKey,
		&issueDate,
		&issueTime,
		&d.CurrencyCode,
		&customerJSON,
		&d.CustomerID,
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
		&note,
		&qrURL,
		&signedXML,
		&status,
		&dianTrackID,
		&dianStatusCode,
		&dianStatusDescription,
		&dianStatusMessage,
		&applicationResponseXML,
		&vendorJSON,
		&operationTypeCode,
		&withholdingJSON,
		&vendorID,
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
	d.Prefix = deref(prefix)
	d.DocumentKey = deref(documentKey)
	d.IssueTime = deref(issueTime)
	d.QRURL = deref(qrURL)
	d.SignedXML = deref(signedXML)
	d.NoteTypeCode = deref(noteTypeCode)
	d.Note = deref(note)
	if number != nil {
		d.Number = *number
	}
	if issueDate != nil {
		d.IssueDate = *issueDate
	}
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
	d.ApplicationResponseXML = deref(applicationResponseXML)

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
	if len(vendorJSON) > 0 {
		var v domain.Party
		if err := json.Unmarshal(vendorJSON, &v); err != nil {
			return nil, fmt.Errorf("deserializar vendor: %w", err)
		}
		d.Vendor = &v
	}
	if operationTypeCode != nil {
		d.OperationTypeCode = *operationTypeCode
	}
	d.VendorID = vendorID
	if len(withholdingJSON) > 0 {
		if err := json.Unmarshal(withholdingJSON, &d.WithholdingTaxes); err != nil {
			return nil, fmt.Errorf("deserializar withholding_taxes: %w", err)
		}
	}

	return &d, nil
}

// marshalSnapshots serializa los campos JSONB compartidos por Create/UpdateDraft.
//
// any(nil) explícito, no json.Marshal(nil) — un *T nil dentro de un []byte JSONB sería el
// literal "null", no NULL de verdad; pgx solo trata como NULL un any(nil) genuino.
func marshalSnapshots(d Document) (customerJSON, linesJSON, paymentMeansJSON []byte, billingRefJSON, discrepancyJSON, vendorJSON, withholdingJSON any, err error) {
	customerJSON, err = json.Marshal(d.Customer)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("serializar customer: %w", err)
	}
	linesJSON, err = json.Marshal(d.Lines)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("serializar lines: %w", err)
	}
	paymentMeansJSON, err = json.Marshal(d.PaymentMeans)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("serializar payment_means: %w", err)
	}
	if d.BillingReference != nil {
		b, err := json.Marshal(d.BillingReference)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("serializar billing_reference: %w", err)
		}
		billingRefJSON = b
	}
	if d.DiscrepancyResponse != nil {
		b, err := json.Marshal(d.DiscrepancyResponse)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("serializar discrepancy_response: %w", err)
		}
		discrepancyJSON = b
	}
	if d.Vendor != nil {
		b, err := json.Marshal(d.Vendor)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("serializar vendor: %w", err)
		}
		vendorJSON = b
	}
	if len(d.WithholdingTaxes) > 0 {
		b, err := json.Marshal(d.WithholdingTaxes)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("serializar withholding_taxes: %w", err)
		}
		withholdingJSON = b
	}
	return customerJSON, linesJSON, paymentMeansJSON, billingRefJSON, discrepancyJSON, vendorJSON, withholdingJSON, nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullablePrefix(s string) any { return nullableString(s) }

// nullableNumber: un Number == 0 en Go significa "todavía sin confirmar" (ver model.go) — se
// manda como NULL real, nunca como el entero 0, que sería un dato ambiguo con un consecutivo
// real (aunque RangeFrom > 0 siempre lo impide en la práctica, NULL es inequívoco).
func nullableNumber(n int64) any {
	if n == 0 {
		return nil
	}
	return n
}

func nullableIssueDate(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// GetBillingStats ejecuta tres queries SQL para construir las métricas del dashboard.
// Todas las queries filtran por issuer_id y usan la zona horaria de Bogotá para los
// cortes de mes/año — mismo criterio que issue_date se graba en hora local Colombia.
func (r *PostgresRepository) GetBillingStats(ctx context.Context, issuerID uuid.UUID) (*BillingStats, error) {
	stats := &BillingStats{}

	// ── 1. Stats por período: mes actual, mes anterior, acumulado anual ──────────────────
	err := r.pool.QueryRow(ctx, `
		SELECT
			-- Mes actual
			COALESCE(SUM(totals_payable_cents) FILTER (
				WHERE issue_date >= date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
				  AND status = 'accepted'), 0),
			COUNT(*) FILTER (
				WHERE issue_date >= date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
				  AND status != 'draft'),
			COUNT(*) FILTER (
				WHERE issue_date >= date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
				  AND status = 'accepted'),
			COUNT(*) FILTER (
				WHERE issue_date >= date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
				  AND status IN ('rejected', 'send_error')),
			-- Borradores: no tienen issue_date, se cuentan sin filtro de fecha
			COUNT(*) FILTER (WHERE status = 'draft'),
			-- Mes anterior
			COALESCE(SUM(totals_payable_cents) FILTER (
				WHERE issue_date >= (date_trunc('month', now() AT TIME ZONE 'America/Bogota') - INTERVAL '1 month')::date
				  AND issue_date <  date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
				  AND status = 'accepted'), 0),
			COUNT(*) FILTER (
				WHERE issue_date >= (date_trunc('month', now() AT TIME ZONE 'America/Bogota') - INTERVAL '1 month')::date
				  AND issue_date <  date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
				  AND status != 'draft'),
			COUNT(*) FILTER (
				WHERE issue_date >= (date_trunc('month', now() AT TIME ZONE 'America/Bogota') - INTERVAL '1 month')::date
				  AND issue_date <  date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
				  AND status = 'accepted'),
			COUNT(*) FILTER (
				WHERE issue_date >= (date_trunc('month', now() AT TIME ZONE 'America/Bogota') - INTERVAL '1 month')::date
				  AND issue_date <  date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
				  AND status IN ('rejected', 'send_error')),
			-- Acumulado año
			COALESCE(SUM(totals_payable_cents) FILTER (
				WHERE issue_date >= date_trunc('year', now() AT TIME ZONE 'America/Bogota')::date
				  AND status = 'accepted'), 0),
			COUNT(*) FILTER (
				WHERE issue_date >= date_trunc('year', now() AT TIME ZONE 'America/Bogota')::date
				  AND status != 'draft'),
			COUNT(*) FILTER (
				WHERE issue_date >= date_trunc('year', now() AT TIME ZONE 'America/Bogota')::date
				  AND status = 'accepted')
		FROM documents
		WHERE issuer_id = $1`,
		issuerID,
	).Scan(
		&stats.CurrentMonth.RevenueCents,
		&stats.CurrentMonth.DocumentCount,
		&stats.CurrentMonth.AcceptedCount,
		&stats.CurrentMonth.RejectedCount,
		&stats.CurrentMonth.DraftCount,
		&stats.PreviousMonth.RevenueCents,
		&stats.PreviousMonth.DocumentCount,
		&stats.PreviousMonth.AcceptedCount,
		&stats.PreviousMonth.RejectedCount,
		&stats.YTD.RevenueCents,
		&stats.YTD.DocumentCount,
		&stats.YTD.AcceptedCount,
	)
	if err != nil {
		return nil, fmt.Errorf("billing stats periods: %w", err)
	}

	// ── 2. Desglose por tipo de documento (mes actual) ───────────────────────────────────
	typeRows, err := r.pool.Query(ctx, `
		SELECT
			dian_document_type_code,
			COUNT(*) FILTER (WHERE status != 'draft'),
			COALESCE(SUM(totals_payable_cents) FILTER (WHERE status = 'accepted'), 0)
		FROM documents
		WHERE issuer_id = $1
		  AND issue_date >= date_trunc('month', now() AT TIME ZONE 'America/Bogota')::date
		GROUP BY dian_document_type_code
		ORDER BY COUNT(*) FILTER (WHERE status != 'draft') DESC`,
		issuerID,
	)
	if err != nil {
		return nil, fmt.Errorf("billing stats by_type: %w", err)
	}
	defer typeRows.Close()

	typeNames := map[string]string{
		"01": "Factura Electrónica",
		"91": "Nota Crédito",
		"92": "Nota Débito",
		"05": "Documento Soporte",
	}
	for typeRows.Next() {
		var ts TypeStats
		if err := typeRows.Scan(&ts.TypeCode, &ts.Count, &ts.RevenueCents); err != nil {
			return nil, fmt.Errorf("billing stats by_type scan: %w", err)
		}
		if name, ok := typeNames[ts.TypeCode]; ok {
			ts.TypeName = name
		} else {
			ts.TypeName = ts.TypeCode
		}
		stats.ByType = append(stats.ByType, ts)
	}
	if err := typeRows.Err(); err != nil {
		return nil, fmt.Errorf("billing stats by_type rows: %w", err)
	}
	if stats.ByType == nil {
		stats.ByType = []TypeStats{}
	}

	// ── 3. Serie mensual: últimos 12 meses ──────────────────────────────────────────────
	seriesRows, err := r.pool.Query(ctx, `
		SELECT
			to_char(date_trunc('month', issue_date), 'YYYY-MM'),
			COUNT(*) FILTER (WHERE status != 'draft'),
			COUNT(*) FILTER (WHERE status = 'accepted'),
			COALESCE(SUM(totals_payable_cents) FILTER (WHERE status = 'accepted'), 0)
		FROM documents
		WHERE issuer_id = $1
		  AND issue_date >= (date_trunc('month', now() AT TIME ZONE 'America/Bogota') - INTERVAL '11 months')::date
		  AND status != 'draft'
		GROUP BY date_trunc('month', issue_date)
		ORDER BY date_trunc('month', issue_date)`,
		issuerID,
	)
	if err != nil {
		return nil, fmt.Errorf("billing stats series: %w", err)
	}
	defer seriesRows.Close()

	for seriesRows.Next() {
		var ms MonthSeries
		if err := seriesRows.Scan(&ms.Month, &ms.Count, &ms.AcceptedCount, &ms.RevenueCents); err != nil {
			return nil, fmt.Errorf("billing stats series scan: %w", err)
		}
		stats.Series = append(stats.Series, ms)
	}
	if err := seriesRows.Err(); err != nil {
		return nil, fmt.Errorf("billing stats series rows: %w", err)
	}
	if stats.Series == nil {
		stats.Series = []MonthSeries{}
	}

	return stats, nil
}
