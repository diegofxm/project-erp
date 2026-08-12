package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/diegofxm/erp/internal/inventory/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) GetStock(ctx context.Context, companyID, productID, warehouseID uuid.UUID) (*domain.StockEntry, error) {
	var e domain.StockEntry
	err := r.pool.QueryRow(ctx,
		`SELECT id, company_id, product_id, warehouse_id, quantity, updated_at
		 FROM inventory.stock
		 WHERE company_id=$1 AND product_id=$2 AND warehouse_id=$3`,
		companyID, productID, warehouseID,
	).Scan(&e.ID, &e.CompanyID, &e.ProductID, &e.WarehouseID, &e.Quantity, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrStockNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("obtener stock: %w", err)
	}
	return &e, nil
}

func (r *Repository) ListStock(ctx context.Context, companyID uuid.UUID) ([]domain.StockEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, company_id, product_id, warehouse_id, quantity, updated_at
		 FROM inventory.stock
		 WHERE company_id=$1 ORDER BY warehouse_id, product_id`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("listar stock: %w", err)
	}
	defer rows.Close()

	var out []domain.StockEntry
	for rows.Next() {
		var e domain.StockEntry
		if err := rows.Scan(&e.ID, &e.CompanyID, &e.ProductID, &e.WarehouseID, &e.Quantity, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("leer stock: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertStock(ctx context.Context, e domain.StockEntry) error {
	e.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO inventory.stock (id, company_id, product_id, warehouse_id, quantity, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (company_id, product_id, warehouse_id)
		 DO UPDATE SET quantity=$5, updated_at=$6`,
		e.ID, e.CompanyID, e.ProductID, e.WarehouseID, e.Quantity, e.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("actualizar stock: %w", err)
	}
	return nil
}

// querier abstrae *pgxpool.Pool/pgx.Tx — nextMovementNumber corre tanto suelto (SaveMovement)
// como dentro de una transacción (Transfer, ver abajo).
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// nextMovementNumber asigna el siguiente folio de movimiento para la empresa, tipo y año dados —
// arranca en 1 cada año, un contador por tipo (ENT-/SAL-/TRA-/AJU-). Mismo patrón que
// sales.Repository.NextSaleNumber.
func nextMovementNumber(ctx context.Context, q querier, companyID uuid.UUID, t domain.MovementType, year int) (string, error) {
	const query = `
		INSERT INTO inventory.number_counters (company_id, doc_type, year, last_seq)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (company_id, doc_type, year)
		DO UPDATE SET last_seq = inventory.number_counters.last_seq + 1
		RETURNING last_seq`
	var seq int
	if err := q.QueryRow(ctx, query, companyID, string(t), year).Scan(&seq); err != nil {
		return "", fmt.Errorf("asignar folio de movimiento: %w", err)
	}
	return fmt.Sprintf("%s-%d-%05d", t.NumberPrefix(), year, seq), nil
}

func (r *Repository) SaveMovement(ctx context.Context, m domain.Movement) (*domain.Movement, error) {
	m.CreatedAt = time.Now()
	number, err := nextMovementNumber(ctx, r.pool, m.CompanyID, m.Type, m.CreatedAt.Year())
	if err != nil {
		return nil, err
	}
	m.Number = number
	_, err = r.pool.Exec(ctx,
		`INSERT INTO inventory.movements
		 (id, company_id, number, product_id, warehouse_id, type, quantity, reference, description, created_at, transfer_group_id, is_addition)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		m.ID, m.CompanyID, m.Number, m.ProductID, m.WarehouseID, string(m.Type),
		m.Quantity, m.Reference, m.Description, m.CreatedAt, m.TransferGroupID, m.IsAddition,
	)
	if err != nil {
		return nil, fmt.Errorf("guardar movimiento: %w", err)
	}
	return &m, nil
}

func (r *Repository) ListMovements(ctx context.Context, companyID uuid.UUID, productID *uuid.UUID) ([]domain.Movement, error) {
	var rows pgx.Rows
	var err error
	if productID != nil {
		rows, err = r.pool.Query(ctx,
			`SELECT id, company_id, number, product_id, warehouse_id, type, quantity, reference, description, created_at, transfer_group_id, is_addition
			 FROM inventory.movements
			 WHERE company_id=$1 AND product_id=$2 ORDER BY created_at DESC`,
			companyID, *productID,
		)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT id, company_id, number, product_id, warehouse_id, type, quantity, reference, description, created_at, transfer_group_id, is_addition
			 FROM inventory.movements
			 WHERE company_id=$1 ORDER BY created_at DESC`,
			companyID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("listar movimientos: %w", err)
	}
	defer rows.Close()

	var out []domain.Movement
	for rows.Next() {
		var m domain.Movement
		var mType string
		if err := rows.Scan(
			&m.ID, &m.CompanyID, &m.Number, &m.ProductID, &m.WarehouseID,
			&mType, &m.Quantity, &m.Reference, &m.Description, &m.CreatedAt, &m.TransferGroupID, &m.IsAddition,
		); err != nil {
			return nil, fmt.Errorf("leer movimiento: %w", err)
		}
		m.Type = domain.MovementType(mType)
		out = append(out, m)
	}
	return out, rows.Err()
}

// DeleteMovement elimina un movimiento (o el par completo si es un transfer, mismo
// transfer_group_id) y revierte su efecto sobre el stock -- todo en una transacción con
// bloqueo de fila (FOR UPDATE) sobre el stock afectado, mismo patrón que Transfer().
func (r *Repository) DeleteMovement(ctx context.Context, companyID, id uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciar transacción: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var groupID *uuid.UUID
	err = tx.QueryRow(ctx,
		`SELECT transfer_group_id FROM inventory.movements WHERE id=$1 AND company_id=$2`,
		id, companyID,
	).Scan(&groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrMovementNotFound
	}
	if err != nil {
		return fmt.Errorf("consultar movimiento: %w", err)
	}

	var rows pgx.Rows
	if groupID != nil {
		rows, err = tx.Query(ctx,
			`SELECT id, product_id, warehouse_id, quantity, is_addition FROM inventory.movements WHERE company_id=$1 AND transfer_group_id=$2`,
			companyID, *groupID,
		)
	} else {
		rows, err = tx.Query(ctx,
			`SELECT id, product_id, warehouse_id, quantity, is_addition FROM inventory.movements WHERE id=$1 AND company_id=$2`,
			id, companyID,
		)
	}
	if err != nil {
		return fmt.Errorf("consultar movimientos a revertir: %w", err)
	}
	type toReverse struct {
		id          uuid.UUID
		productID   uuid.UUID
		warehouseID uuid.UUID
		quantity    float64
		isAddition  bool
	}
	var list []toReverse
	for rows.Next() {
		var t toReverse
		if err := rows.Scan(&t.id, &t.productID, &t.warehouseID, &t.quantity, &t.isAddition); err != nil {
			rows.Close()
			return fmt.Errorf("leer movimiento: %w", err)
		}
		list = append(list, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, t := range list {
		var current float64
		err := tx.QueryRow(ctx,
			`SELECT quantity FROM inventory.stock WHERE company_id=$1 AND product_id=$2 AND warehouse_id=$3 FOR UPDATE`,
			companyID, t.productID, t.warehouseID,
		).Scan(&current)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("consultar stock: %w", err)
		}
		// Revertir: si el movimiento sumó, ahora hay que restar, y viceversa.
		var newQty float64
		if t.isAddition {
			newQty = current - t.quantity
		} else {
			newQty = current + t.quantity
		}
		if newQty < 0 {
			return domain.ErrDeleteWouldMakeStockNegative
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO inventory.stock (id, company_id, product_id, warehouse_id, quantity, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6)
			 ON CONFLICT (company_id, product_id, warehouse_id) DO UPDATE SET quantity=$5, updated_at=$6`,
			uuid.New(), companyID, t.productID, t.warehouseID, newQty, time.Now(),
		); err != nil {
			return fmt.Errorf("actualizar stock: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM inventory.movements WHERE id=$1`, t.id); err != nil {
			return fmt.Errorf("eliminar movimiento: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// Transfer traslada `quantity` de `fromWarehouseID` a `toWarehouseID` en una sola transacción:
// valida stock suficiente en origen, genera los dos movimientos enlazados por un
// TransferGroupID compartido, y actualiza ambos saldos de stock.
func (r *Repository) Transfer(ctx context.Context, companyID, productID, fromWarehouseID, toWarehouseID uuid.UUID, quantity float64, reference, description string) (*domain.Movement, *domain.Movement, error) {
	if quantity <= 0 {
		return nil, nil, fmt.Errorf("la cantidad a trasladar debe ser mayor a cero")
	}
	if fromWarehouseID == toWarehouseID {
		return nil, nil, fmt.Errorf("la bodega de origen y destino no pueden ser la misma")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("iniciar transacción: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var fromQty float64
	err = tx.QueryRow(ctx,
		`SELECT quantity FROM inventory.stock WHERE company_id=$1 AND product_id=$2 AND warehouse_id=$3 FOR UPDATE`,
		companyID, productID, fromWarehouseID,
	).Scan(&fromQty)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, fmt.Errorf("consultar stock de origen: %w", err)
	}
	if fromQty < quantity {
		return nil, nil, domain.ErrInsufficientStock
	}

	groupID := uuid.New()
	now := time.Now()

	fromMovement := domain.Movement{
		ID: uuid.New(), CompanyID: companyID, ProductID: productID, WarehouseID: fromWarehouseID,
		Type: domain.MovementTransfer, Quantity: quantity, Reference: reference, Description: description,
		TransferGroupID: &groupID, CreatedAt: now, IsAddition: false,
	}
	toMovement := domain.Movement{
		ID: uuid.New(), CompanyID: companyID, ProductID: productID, WarehouseID: toWarehouseID,
		Type: domain.MovementTransfer, Quantity: quantity, Reference: reference, Description: description,
		TransferGroupID: &groupID, CreatedAt: now, IsAddition: true,
	}

	for _, m := range []*domain.Movement{&fromMovement, &toMovement} {
		number, err := nextMovementNumber(ctx, tx, companyID, domain.MovementTransfer, now.Year())
		if err != nil {
			return nil, nil, err
		}
		m.Number = number
		if _, err := tx.Exec(ctx,
			`INSERT INTO inventory.movements
			 (id, company_id, number, product_id, warehouse_id, type, quantity, reference, description, created_at, transfer_group_id, is_addition)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			m.ID, m.CompanyID, m.Number, m.ProductID, m.WarehouseID, string(m.Type),
			m.Quantity, m.Reference, m.Description, m.CreatedAt, m.TransferGroupID, m.IsAddition,
		); err != nil {
			return nil, nil, fmt.Errorf("guardar movimiento de traslado: %w", err)
		}
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO inventory.stock (id, company_id, product_id, warehouse_id, quantity, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (company_id, product_id, warehouse_id) DO UPDATE SET quantity = inventory.stock.quantity - $5, updated_at=$6`,
		uuid.New(), companyID, productID, fromWarehouseID, quantity, now,
	); err != nil {
		return nil, nil, fmt.Errorf("descontar stock de origen: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO inventory.stock (id, company_id, product_id, warehouse_id, quantity, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (company_id, product_id, warehouse_id) DO UPDATE SET quantity = inventory.stock.quantity + $5, updated_at=$6`,
		uuid.New(), companyID, productID, toWarehouseID, quantity, now,
	); err != nil {
		return nil, nil, fmt.Errorf("acreditar stock de destino: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit: %w", err)
	}
	return &fromMovement, &toMovement, nil
}
