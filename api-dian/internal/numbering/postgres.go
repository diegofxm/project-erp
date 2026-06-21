package numbering

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/diegofxm/api-dian/internal/cryptutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implementa Repository usando pgx. Cifra technical_key con AES-256-GCM
// (internal/cryptutil) antes de persistir.
type PostgresRepository struct {
	pool *pgxpool.Pool
	key  []byte // 32 bytes, AES-256
}

// NewPostgresRepository crea el repositorio de rangos de numeración sobre PostgreSQL.
func NewPostgresRepository(pool *pgxpool.Pool, secretsKey []byte) *PostgresRepository {
	return &PostgresRepository{pool: pool, key: secretsKey}
}

const numberingColumns = `
	id, issuer_id, dian_document_type_code, prefix, resolution_number, resolution_date,
	range_from, range_to, current_number, valid_from, valid_to, environment,
	technical_key, test_set_id, is_active, created_at, updated_at`

func (r *PostgresRepository) Create(ctx context.Context, nr NumberingRange) (*NumberingRange, error) {
	if nr.ID == uuid.Nil {
		nr.ID = uuid.New()
	}
	now := time.Now().UTC()
	nr.CreatedAt = now
	nr.UpdatedAt = now
	// El primer ClaimNext debe devolver RangeFrom — arrancar un paso atrás logra eso con la
	// misma fórmula (current_number + 1) que usan todos los claims siguientes.
	nr.CurrentNumber = nr.RangeFrom - 1

	encKey, err := cryptutil.Encrypt(r.key, []byte(nr.TechnicalKey))
	if err != nil {
		return nil, fmt.Errorf("cifrar technical_key: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO numbering_ranges (`+numberingColumns+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		nr.ID, nr.IssuerID, nr.DianDocumentTypeCode, nr.Prefix, nr.ResolutionNumber, nr.ResolutionDate,
		nr.RangeFrom, nr.RangeTo, nr.CurrentNumber, nr.ValidFrom, nr.ValidTo, string(nr.Environment),
		encKey, nr.TestSetID, nr.IsActive, nr.CreatedAt, nr.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create numbering range: %w", err)
	}
	return &nr, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*NumberingRange, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+numberingColumns+` FROM numbering_ranges WHERE id = $1`, id)
	return r.scan(row)
}

// ClaimNext reclama el siguiente número de forma atómica con un único UPDATE: Postgres
// serializa las escrituras concurrentes sobre la misma fila, así que dos llamadas
// simultáneas nunca pueden recibir el mismo número ni dejar un hueco — no hace falta
// SELECT ... FOR UPDATE explícito, el propio UPDATE ya bloquea la fila mientras se aplica.
func (r *PostgresRepository) ClaimNext(ctx context.Context, id uuid.UUID) (int64, error) {
	var claimed int64
	err := r.pool.QueryRow(ctx, `
		UPDATE numbering_ranges
		SET current_number = current_number + 1, updated_at = NOW()
		WHERE id = $1
		  AND is_active = TRUE
		  AND (range_to IS NULL OR current_number < range_to)
		RETURNING current_number`,
		id,
	).Scan(&claimed)

	if errors.Is(err, pgx.ErrNoRows) {
		// El UPDATE no afectó ninguna fila: o el rango no existe, o ya está agotado.
		// Distinguir con una consulta de solo lectura aparte (no afecta la atomicidad del
		// claim en sí, que ya falló).
		if _, getErr := r.GetByID(ctx, id); getErr != nil {
			return 0, getErr
		}
		return 0, ErrRangeExhausted
	}
	if err != nil {
		return 0, fmt.Errorf("claim next number: %w", err)
	}
	return claimed, nil
}

func (r *PostgresRepository) scan(row pgx.Row) (*NumberingRange, error) {
	var nr NumberingRange
	var env string
	var encKey []byte

	err := row.Scan(&nr.ID, &nr.IssuerID, &nr.DianDocumentTypeCode, &nr.Prefix, &nr.ResolutionNumber,
		&nr.ResolutionDate, &nr.RangeFrom, &nr.RangeTo, &nr.CurrentNumber, &nr.ValidFrom, &nr.ValidTo,
		&env, &encKey, &nr.TestSetID, &nr.IsActive, &nr.CreatedAt, &nr.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRangeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan numbering range: %w", err)
	}
	nr.Environment = Environment(env)

	key, err := cryptutil.Decrypt(r.key, encKey)
	if err != nil {
		return nil, fmt.Errorf("descifrar technical_key: %w", err)
	}
	nr.TechnicalKey = string(key)

	return &nr, nil
}
