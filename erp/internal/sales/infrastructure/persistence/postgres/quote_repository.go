package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/sales/domain"
)

type QuoteRepository struct{ pool *pgxpool.Pool }

func NewQuoteRepository(pool *pgxpool.Pool) *QuoteRepository {
	return &QuoteRepository{pool: pool}
}

var _ domain.QuoteRepository = (*QuoteRepository)(nil)

func (r *QuoteRepository) Save(ctx context.Context, q domain.Quote) (*domain.Quote, error) {
	now := time.Now()
	q.CreatedAt = now
	q.UpdatedAt = now

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("iniciar transacción: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO sales.quotes (id, company_id, customer_id, number, status, issue_date, valid_until, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		q.ID, q.CompanyID, q.CustomerID, q.Number, string(q.Status),
		q.IssueDate, q.ValidUntil, q.Notes, q.CreatedAt, q.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("guardar cotización: %w", err)
	}
	for i := range q.Lines {
		l := &q.Lines[i]
		l.ID = uuid.New()
		l.QuoteID = q.ID
		_, err = tx.Exec(ctx, `
			INSERT INTO sales.quote_lines (id, quote_id, product_id, description, quantity, unit_price, tax_rate, subtotal, tax_amount, total, discount)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			l.ID, l.QuoteID, l.ProductID, l.Description,
			l.Quantity, l.UnitPrice, l.TaxRate, l.Subtotal, l.TaxAmount, l.Total, l.Discount,
		)
		if err != nil {
			return nil, fmt.Errorf("guardar línea cotización: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &q, nil
}

func (r *QuoteRepository) Update(ctx context.Context, companyID, id uuid.UUID, q domain.Quote) (*domain.Quote, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("iniciar transacción: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM sales.quotes WHERE id=$1 AND company_id=$2 FOR UPDATE`, id, companyID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrQuoteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consultar cotización: %w", err)
	}
	if status != string(domain.QuoteStatusDraft) {
		return nil, domain.ErrQuoteNotDraft
	}

	now := time.Now()
	_, err = tx.Exec(ctx,
		`UPDATE sales.quotes SET customer_id=$3, valid_until=$4, notes=$5, updated_at=$6 WHERE id=$1 AND company_id=$2`,
		id, companyID, q.CustomerID, q.ValidUntil, q.Notes, now,
	)
	if err != nil {
		return nil, fmt.Errorf("actualizar cotización: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM sales.quote_lines WHERE quote_id=$1`, id); err != nil {
		return nil, fmt.Errorf("limpiar líneas: %w", err)
	}
	for i := range q.Lines {
		l := &q.Lines[i]
		l.ID = uuid.New()
		l.QuoteID = id
		if _, err := tx.Exec(ctx, `
			INSERT INTO sales.quote_lines (id, quote_id, product_id, description, quantity, unit_price, tax_rate, subtotal, tax_amount, total, discount)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			l.ID, l.QuoteID, l.ProductID, l.Description,
			l.Quantity, l.UnitPrice, l.TaxRate, l.Subtotal, l.TaxAmount, l.Total, l.Discount,
		); err != nil {
			return nil, fmt.Errorf("guardar línea cotización: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return r.GetByID(ctx, companyID, id)
}

func (r *QuoteRepository) GetByID(ctx context.Context, companyID, id uuid.UUID) (*domain.Quote, error) {
	var q domain.Quote
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT id, company_id, customer_id, number, status, issue_date, valid_until, notes, created_at, updated_at
		FROM sales.quotes WHERE id=$1 AND company_id=$2`,
		id, companyID,
	).Scan(&q.ID, &q.CompanyID, &q.CustomerID, &q.Number, &status,
		&q.IssueDate, &q.ValidUntil, &q.Notes, &q.CreatedAt, &q.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrQuoteNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("obtener cotización: %w", err)
	}
	q.Status = domain.QuoteStatus(status)
	q.Lines, err = r.loadQuoteLines(ctx, q.ID)
	return &q, err
}

func (r *QuoteRepository) List(ctx context.Context, companyID uuid.UUID) ([]domain.Quote, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, company_id, customer_id, number, status, issue_date, valid_until, notes, created_at, updated_at
		FROM sales.quotes WHERE company_id=$1 ORDER BY created_at DESC`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar cotizaciones: %w", err)
	}
	defer rows.Close()

	var out []domain.Quote
	for rows.Next() {
		var q domain.Quote
		var status string
		if err := rows.Scan(&q.ID, &q.CompanyID, &q.CustomerID, &q.Number, &status,
			&q.IssueDate, &q.ValidUntil, &q.Notes, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, fmt.Errorf("leer cotización: %w", err)
		}
		q.Status = domain.QuoteStatus(status)
		out = append(out, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Lines, err = r.loadQuoteLines(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *QuoteRepository) UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status domain.QuoteStatus) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE sales.quotes SET status=$1, updated_at=NOW() WHERE id=$2 AND company_id=$3",
		string(status), id, companyID,
	)
	return err
}

func (r *QuoteRepository) Delete(ctx context.Context, companyID, id uuid.UUID) error {
	q, err := r.GetByID(ctx, companyID, id)
	if err != nil {
		return err
	}
	if q.Status != domain.QuoteStatusDraft {
		return domain.ErrQuoteNotDraft
	}
	_, err = r.pool.Exec(ctx, "DELETE FROM sales.quotes WHERE id=$1 AND company_id=$2", id, companyID)
	return err
}

func (r *QuoteRepository) loadQuoteLines(ctx context.Context, quoteID uuid.UUID) ([]domain.QuoteLine, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, quote_id, product_id, description, quantity, unit_price, tax_rate, subtotal, tax_amount, total, discount
		FROM sales.quote_lines WHERE quote_id=$1`,
		quoteID,
	)
	if err != nil {
		return nil, fmt.Errorf("cargar líneas cotización: %w", err)
	}
	defer rows.Close()

	var lines []domain.QuoteLine
	for rows.Next() {
		var l domain.QuoteLine
		if err := rows.Scan(&l.ID, &l.QuoteID, &l.ProductID, &l.Description,
			&l.Quantity, &l.UnitPrice, &l.TaxRate, &l.Subtotal, &l.TaxAmount, &l.Total, &l.Discount); err != nil {
			return nil, fmt.Errorf("leer línea cotización: %w", err)
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

// NextQuoteNumber asigna el siguiente consecutivo de cotización para la empresa y el año dados —
// arranca en 1 cada año, comparte la tabla con NextSaleNumber (doc_type='quote').
func (r *QuoteRepository) NextQuoteNumber(ctx context.Context, companyID uuid.UUID, year int) (int, error) {
	const q = `
		INSERT INTO sales.number_counters (company_id, doc_type, year, last_seq)
		VALUES ($1, 'quote', $2, 1)
		ON CONFLICT (company_id, doc_type, year)
		DO UPDATE SET last_seq = sales.number_counters.last_seq + 1
		RETURNING last_seq`
	var seq int
	if err := r.pool.QueryRow(ctx, q, companyID, year).Scan(&seq); err != nil {
		return 0, fmt.Errorf("asignar consecutivo de cotización: %w", err)
	}
	return seq, nil
}

// SetQuoteNumberCounter — ver Repository.SetSaleNumberCounter, mismo comportamiento para
// cotizaciones (doc_type='quote').
func (r *QuoteRepository) SetQuoteNumberCounter(ctx context.Context, companyID uuid.UUID, year, nextNumber int) (int, error) {
	const q = `
		INSERT INTO sales.number_counters (company_id, doc_type, year, last_seq)
		VALUES ($1, 'quote', $2, $3)
		ON CONFLICT (company_id, doc_type, year)
		DO UPDATE SET last_seq = EXCLUDED.last_seq
		WHERE EXCLUDED.last_seq > sales.number_counters.last_seq
		RETURNING last_seq`
	var seq int
	err := r.pool.QueryRow(ctx, q, companyID, year, nextNumber-1).Scan(&seq)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.ErrNumberCounterBackwards
	}
	if err != nil {
		return 0, fmt.Errorf("fijar consecutivo de cotización: %w", err)
	}
	return seq, nil
}
