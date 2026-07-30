package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	cofdom "github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/electronic/domain"
)

type DocumentRepository struct {
	pool *pgxpool.Pool
}

func NewDocumentRepository(pool *pgxpool.Pool) *DocumentRepository {
	return &DocumentRepository{pool: pool}
}

// documentCols lista las columnas en orden idéntico al usado en los Scan.
const documentCols = `id, company_id, numbering_range_id, dian_document_type_code,
	prefix, number, document_key, issue_date, issue_time, currency_code,
	customer, customer_id, lines, payment_means,
	totals_line_extension_cents, totals_tax_exclusive_cents, totals_tax_inclusive_cents,
	totals_prepaid_cents, totals_payable_cents,
	billing_reference, discrepancy_response, note_type_code, note,
	qr_url, signed_xml, status, dian_track_id, dian_status_code,
	dian_status_description, dian_status_message, application_response_xml,
	supplier, operation_type_code, withholding_taxes, supplier_id,
	created_at, updated_at`

func (r *DocumentRepository) Create(ctx context.Context, d domain.Document) (*domain.Document, error) {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	now := time.Now()
	d.CreatedAt = now
	d.UpdatedAt = now

	snaps, err := marshalSnapshots(d)
	if err != nil {
		return nil, err
	}

	args := buildCreateArgs(d, snaps)
	placeholders := makePlaceholders(len(args))
	_, err = r.pool.Exec(ctx,
		`INSERT INTO electronic.documents (`+documentCols+`) VALUES (`+placeholders+`)`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("create document: %w", err)
	}
	return &d, nil
}

func (r *DocumentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Document, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+documentCols+` FROM electronic.documents WHERE id = $1`, id)
	d, err := scanDocument(row)
	if errors.Is(err, domain.ErrDocumentNotFound) {
		return nil, domain.ErrDocumentNotFound
	}
	return d, err
}

func (r *DocumentRepository) GetByDocumentKey(ctx context.Context, companyID uuid.UUID, key string) (*domain.Document, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+documentCols+` FROM electronic.documents WHERE company_id = $1 AND document_key = $2`, companyID, key)
	return scanDocument(row)
}

func (r *DocumentRepository) UpdateDraft(ctx context.Context, d domain.Document) (*domain.Document, error) {
	snaps, err := marshalSnapshots(d)
	if err != nil {
		return nil, err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE electronic.documents SET
			numbering_range_id = $1, currency_code = $2,
			customer = $3, customer_id = $4, lines = $5, payment_means = $6,
			totals_line_extension_cents = $7, totals_tax_exclusive_cents = $8,
			totals_tax_inclusive_cents = $9, totals_prepaid_cents = $10, totals_payable_cents = $11,
			billing_reference = $12, discrepancy_response = $13, note_type_code = $14, note = $15,
			supplier = $16, operation_type_code = $17, withholding_taxes = $18, supplier_id = $19,
			updated_at = $20
		WHERE id = $21 AND status = 'draft'`,
		d.NumberingRangeID, d.CurrencyCode,
		snaps.customer, d.CustomerID, snaps.lines, snaps.paymentMeans,
		d.Totals.LineExtensionCents, d.Totals.TaxExclusiveCents,
		d.Totals.TaxInclusiveCents, d.Totals.PrepaidCents, d.Totals.PayableCents,
		snaps.billingRef, snaps.discrepancy, nullableStr(d.NoteTypeCode), nullableStr(d.Note),
		snaps.supplier, nullableStr(d.OperationTypeCode), snaps.withholding, d.SupplierID,
		time.Now(), d.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("update draft: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, domain.ErrDocumentNotDraft
	}
	return r.GetByID(ctx, d.ID)
}

func (r *DocumentRepository) Confirm(ctx context.Context, d domain.Document) (*domain.Document, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE electronic.documents SET
			prefix = $1, number = $2, document_key = $3,
			issue_date = $4, issue_time = $5, qr_url = $6, signed_xml = $7,
			status = $8, updated_at = $9
		WHERE id = $10 AND status = 'draft'`,
		d.Prefix, d.Number, d.DocumentKey,
		d.IssueDate, d.IssueTime, d.QRURL, d.SignedXML,
		string(d.Status), time.Now(), d.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("confirm document: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, domain.ErrDocumentNotDraft
	}
	return r.GetByID(ctx, d.ID)
}

func (r *DocumentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM electronic.documents WHERE id = $1 AND status = 'draft'`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDocumentNotDraft
	}
	return nil
}

func (r *DocumentRepository) UpdateDianStatus(ctx context.Context, id uuid.UUID, status domain.Status, trackID, statusCode, statusDescription, statusMessage, applicationResponseXML string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE electronic.documents
		SET status = $1, dian_track_id = $2, dian_status_code = $3,
		    dian_status_description = $4, dian_status_message = $5,
		    application_response_xml = $6, updated_at = NOW()
		WHERE id = $7`,
		string(status), trackID, statusCode, statusDescription, statusMessage,
		nullableStr(applicationResponseXML), id,
	)
	if err != nil {
		return fmt.Errorf("update dian status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDocumentNotFound
	}
	return nil
}

func (r *DocumentRepository) ListByCompany(ctx context.Context, companyID uuid.UUID, filter domain.ListFilter) ([]*domain.Document, error) {
	q := `SELECT ` + documentCols + ` FROM electronic.documents WHERE company_id = $1`
	args := []any{companyID}

	if filter.DianDocumentTypeCode != "" {
		args = append(args, filter.DianDocumentTypeCode)
		q += fmt.Sprintf(" AND dian_document_type_code = $%d", len(args))
	}
	if filter.Status != "" {
		args = append(args, string(filter.Status))
		q += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if !filter.From.IsZero() {
		args = append(args, filter.From)
		q += fmt.Sprintf(" AND issue_date >= $%d", len(args))
	}
	if !filter.To.IsZero() {
		args = append(args, filter.To)
		q += fmt.Sprintf(" AND issue_date <= $%d", len(args))
	}
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		n := len(args)
		q += fmt.Sprintf(`
			AND (customer->>'Name' ILIKE $%[1]d OR supplier->>'Name' ILIKE $%[1]d
			     OR customer->'Identification'->>'Number' ILIKE $%[1]d
			     OR CONCAT(COALESCE(prefix,''), COALESCE(number::text,'')) ILIKE $%[1]d)`, n)
	}

	args = append(args, filter.Limit, filter.Offset)
	q += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	var out []*domain.Document
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

	// Segunda query: contar NC/ND/NA por documento referenciado (billing_reference).
	// Una sola round-trip, no N+1 independiente del tamaño de página.
	if len(out) > 0 {
		cntRows, err := r.pool.Query(ctx, `
			SELECT
				COALESCE(billing_reference->>'prefix', ''),
				billing_reference->>'number',
				SUM(CASE WHEN dian_document_type_code = '91' THEN 1 ELSE 0 END)::int,
				SUM(CASE WHEN dian_document_type_code = '92' THEN 1 ELSE 0 END)::int,
				SUM(CASE WHEN dian_document_type_code = '95' THEN 1 ELSE 0 END)::int
			FROM electronic.documents
			WHERE company_id = $1 AND billing_reference IS NOT NULL
			GROUP BY 1, 2`, companyID)
		if err == nil {
			defer cntRows.Close()
			type countKey struct{ prefix, number string }
			type countVal struct{ nc, nd, na int }
			counts := map[countKey]countVal{}
			for cntRows.Next() {
				var k countKey
				var v countVal
				if err := cntRows.Scan(&k.prefix, &k.number, &v.nc, &v.nd, &v.na); err == nil {
					counts[k] = v
				}
			}
			for _, d := range out {
				if d.Number == 0 {
					continue
				}
				k := countKey{prefix: d.Prefix, number: fmt.Sprintf("%d", d.Number)}
				if c, ok := counts[k]; ok {
					d.NCCount = c.nc
					d.NDCount = c.nd
					d.NACount = c.na
				}
			}
		}
	}

	return out, nil
}

func (r *DocumentRepository) GetRelatedNotes(ctx context.Context, companyID uuid.UUID, prefix string, number int64) ([]domain.RelatedNote, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, dian_document_type_code,
		       COALESCE(prefix, ''), COALESCE(number, 0),
		       totals_payable_cents, status, issue_date
		FROM electronic.documents
		WHERE company_id = $1
		  AND billing_reference->>'prefix' = $2
		  AND billing_reference->>'number' = $3
		ORDER BY created_at ASC`,
		companyID, prefix, fmt.Sprintf("%d", number))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RelatedNote
	for rows.Next() {
		var n domain.RelatedNote
		var status string
		var issueDate *time.Time
		if err := rows.Scan(&n.ID, &n.DianDocumentTypeCode, &n.Prefix, &n.Number,
			&n.PayableCents, &status, &issueDate); err != nil {
			return nil, err
		}
		n.Status = domain.Status(status)
		n.IssueDate = issueDate
		out = append(out, n)
	}
	return out, rows.Err()
}

// ── helpers internos ──────────────────────────────────────────────────────────────────────

type snapshots struct {
	customer     []byte
	lines        []byte
	paymentMeans []byte
	billingRef   any
	discrepancy  any
	supplier     any
	withholding  any
}

func marshalSnapshots(d domain.Document) (snapshots, error) {
	var s snapshots
	var err error
	if s.customer, err = json.Marshal(d.Customer); err != nil {
		return s, fmt.Errorf("serializar customer: %w", err)
	}
	if s.lines, err = json.Marshal(d.Lines); err != nil {
		return s, fmt.Errorf("serializar lines: %w", err)
	}
	if s.paymentMeans, err = json.Marshal(d.PaymentMeans); err != nil {
		return s, fmt.Errorf("serializar payment_means: %w", err)
	}
	if d.BillingReference != nil {
		b, err := json.Marshal(d.BillingReference)
		if err != nil {
			return s, fmt.Errorf("serializar billing_reference: %w", err)
		}
		s.billingRef = b
	}
	if d.DiscrepancyResponse != nil {
		b, err := json.Marshal(d.DiscrepancyResponse)
		if err != nil {
			return s, fmt.Errorf("serializar discrepancy_response: %w", err)
		}
		s.discrepancy = b
	}
	if d.Supplier != nil {
		b, err := json.Marshal(d.Supplier)
		if err != nil {
			return s, fmt.Errorf("serializar supplier: %w", err)
		}
		s.supplier = b
	}
	if len(d.WithholdingTaxes) > 0 {
		b, err := json.Marshal(d.WithholdingTaxes)
		if err != nil {
			return s, fmt.Errorf("serializar withholding_taxes: %w", err)
		}
		s.withholding = b
	}
	return s, nil
}

func buildCreateArgs(d domain.Document, s snapshots) []any {
	return []any{
		d.ID, d.CompanyID, d.NumberingRangeID, d.DianDocumentTypeCode,
		nullableStr(d.Prefix), nullableInt(d.Number), nullableStr(d.DocumentKey),
		nullableTime(d.IssueDate), nullableStr(d.IssueTime),
		d.CurrencyCode,
		s.customer, d.CustomerID, s.lines, s.paymentMeans,
		d.Totals.LineExtensionCents, d.Totals.TaxExclusiveCents,
		d.Totals.TaxInclusiveCents, d.Totals.PrepaidCents, d.Totals.PayableCents,
		s.billingRef, s.discrepancy, nullableStr(d.NoteTypeCode), nullableStr(d.Note),
		nullableStr(d.QRURL), nullableStr(d.SignedXML),
		string(d.Status), d.DianTrackID, d.DianStatusCode,
		d.DianStatusDescription, d.DianStatusMessage, nullableStr(d.ApplicationResponseXML),
		s.supplier, nullableStr(d.OperationTypeCode), s.withholding, d.SupplierID,
		d.CreatedAt, d.UpdatedAt,
	}
}

func scanDocument(row pgx.Row) (*domain.Document, error) {
	var d domain.Document
	var status string
	var customerJSON, linesJSON, paymentMeansJSON []byte
	var billingRefJSON, discrepancyJSON, supplierJSON, withholdingJSON []byte
	var prefix, documentKey, issueTime, qrURL, signedXML, noteTypeCode, note *string
	var operationTypeCode, dianTrackID, dianStatusCode, dianStatusDesc, dianStatusMsg *string
	var applicationResponseXML *string
	var number *int64
	var issueDate *time.Time
	var supplierID *uuid.UUID

	err := row.Scan(
		&d.ID, &d.CompanyID, &d.NumberingRangeID, &d.DianDocumentTypeCode,
		&prefix, &number, &documentKey, &issueDate, &issueTime, &d.CurrencyCode,
		&customerJSON, &d.CustomerID, &linesJSON, &paymentMeansJSON,
		&d.Totals.LineExtensionCents, &d.Totals.TaxExclusiveCents,
		&d.Totals.TaxInclusiveCents, &d.Totals.PrepaidCents, &d.Totals.PayableCents,
		&billingRefJSON, &discrepancyJSON, &noteTypeCode, &note,
		&qrURL, &signedXML, &status, &dianTrackID, &dianStatusCode,
		&dianStatusDesc, &dianStatusMsg, &applicationResponseXML,
		&supplierJSON, &operationTypeCode, &withholdingJSON, &supplierID,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrDocumentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan document: %w", err)
	}

	d.Status = domain.Status(status)
	d.Prefix = derefStr(prefix)
	d.DocumentKey = derefStr(documentKey)
	d.IssueTime = derefStr(issueTime)
	d.QRURL = derefStr(qrURL)
	d.SignedXML = derefStr(signedXML)
	d.NoteTypeCode = derefStr(noteTypeCode)
	d.Note = derefStr(note)
	d.OperationTypeCode = derefStr(operationTypeCode)
	d.DianTrackID = derefStr(dianTrackID)
	d.DianStatusCode = derefStr(dianStatusCode)
	d.DianStatusDescription = derefStr(dianStatusDesc)
	d.DianStatusMessage = derefStr(dianStatusMsg)
	d.ApplicationResponseXML = derefStr(applicationResponseXML)
	d.SupplierID = supplierID
	if number != nil {
		d.Number = *number
	}
	if issueDate != nil {
		d.IssueDate = *issueDate
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
	if len(supplierJSON) > 0 {
		var v cofdom.Party
		if err := json.Unmarshal(supplierJSON, &v); err != nil {
			return nil, fmt.Errorf("deserializar supplier: %w", err)
		}
		d.Supplier = &v
	}
	if len(withholdingJSON) > 0 {
		if err := json.Unmarshal(withholdingJSON, &d.WithholdingTaxes); err != nil {
			return nil, fmt.Errorf("deserializar withholding_taxes: %w", err)
		}
	}
	return &d, nil
}

func makePlaceholders(n int) string {
	s := ""
	for i := 1; i <= n; i++ {
		if i > 1 {
			s += ","
		}
		s += fmt.Sprintf("$%d", i)
	}
	return s
}

func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullableInt(n int64) any {
	if n == 0 {
		return nil
	}
	return n
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
