package products

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryRepository es una implementación en memoria de Repository para tests.
type MemoryRepository struct {
	mu       sync.Mutex
	products map[uuid.UUID]*Product
}

// NewMemoryRepository crea un repositorio vacío en memoria.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{products: make(map[uuid.UUID]*Product)}
}

func (r *MemoryRepository) Create(_ context.Context, p Product) (*Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.checkDuplicateItemCode(p.IssuerID, p.ItemCode, uuid.Nil); err != nil {
		return nil, err
	}

	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	cp := p
	r.products[p.ID] = &cp
	return &cp, nil
}

// checkDuplicateItemCode replica en memoria el índice único parcial (issuer_id, item_code) de
// la migración 000011 — excludeID se pasa en Update para no chocar contra el propio producto
// que se está actualizando.
func (r *MemoryRepository) checkDuplicateItemCode(issuerID uuid.UUID, itemCode string, excludeID uuid.UUID) error {
	if itemCode == "" {
		return nil
	}
	for _, p := range r.products {
		if p.ID != excludeID && p.IssuerID == issuerID && p.ItemCode == itemCode {
			return ErrDuplicateItemCode
		}
	}
	return nil
}

func (r *MemoryRepository) GetByID(_ context.Context, id uuid.UUID) (*Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.products[id]
	if !ok {
		return nil, ErrProductNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *MemoryRepository) ListByIssuer(_ context.Context, issuerID uuid.UUID) ([]*Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var out []*Product
	for _, p := range r.products {
		if p.IssuerID != issuerID {
			continue
		}
		cp := *p
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (r *MemoryRepository) Update(_ context.Context, issuerID, id uuid.UUID, p Product) (*Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.products[id]
	if !ok || existing.IssuerID != issuerID {
		return nil, ErrProductNotFound
	}
	if err := r.checkDuplicateItemCode(issuerID, p.ItemCode, id); err != nil {
		return nil, err
	}

	p.ID = existing.ID
	p.IssuerID = existing.IssuerID
	p.CreatedAt = existing.CreatedAt
	p.UpdatedAt = time.Now().UTC()
	r.products[id] = &p

	cp := p
	return &cp, nil
}

func (r *MemoryRepository) Delete(_ context.Context, issuerID, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.products[id]
	if !ok || p.IssuerID != issuerID {
		return ErrProductNotFound
	}
	delete(r.products, id)
	return nil
}
