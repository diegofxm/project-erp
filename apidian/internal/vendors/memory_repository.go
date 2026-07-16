package vendors

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"
)

// MemoryRepository es una implementación en memoria de Repository para tests.
type MemoryRepository struct {
	mu      sync.Mutex
	vendors map[uuid.UUID]*Vendor
}

// NewMemoryRepository crea un repositorio vacío en memoria.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{vendors: make(map[uuid.UUID]*Vendor)}
}

func (r *MemoryRepository) Create(_ context.Context, v Vendor) (*Vendor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	now := time.Now().UTC()
	v.CreatedAt = now
	v.UpdatedAt = now

	cp := v
	r.vendors[v.ID] = &cp
	return &cp, nil
}

func (r *MemoryRepository) GetByID(_ context.Context, id uuid.UUID) (*Vendor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.vendors[id]
	if !ok {
		return nil, ErrVendorNotFound
	}
	cp := *v
	return &cp, nil
}

func (r *MemoryRepository) ListByIssuer(_ context.Context, issuerID uuid.UUID) ([]*Vendor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var out []*Vendor
	for _, v := range r.vendors {
		if v.IssuerID != issuerID {
			continue
		}
		cp := *v
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (r *MemoryRepository) Update(_ context.Context, issuerID, id uuid.UUID, party domain.Party) (*Vendor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	v, ok := r.vendors[id]
	if !ok || v.IssuerID != issuerID {
		return nil, ErrVendorNotFound
	}
	v.Party = party
	v.UpdatedAt = time.Now().UTC()
	cp := *v
	return &cp, nil
}

func (r *MemoryRepository) Delete(_ context.Context, issuerID, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	v, ok := r.vendors[id]
	if !ok || v.IssuerID != issuerID {
		return ErrVendorNotFound
	}
	delete(r.vendors, v.ID)
	return nil
}
