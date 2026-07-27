package customers

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
	mu        sync.Mutex
	customers map[uuid.UUID]*Customer
}

// NewMemoryRepository crea un repositorio vacío en memoria.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{customers: make(map[uuid.UUID]*Customer)}
}

func (r *MemoryRepository) Create(_ context.Context, c Customer) (*Customer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now

	cp := c
	r.customers[c.ID] = &cp
	return &cp, nil
}

func (r *MemoryRepository) GetByID(_ context.Context, id uuid.UUID) (*Customer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.customers[id]
	if !ok {
		return nil, ErrCustomerNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *MemoryRepository) ListByIssuer(_ context.Context, issuerID uuid.UUID) ([]*Customer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var out []*Customer
	for _, c := range r.customers {
		if c.IssuerID != issuerID {
			continue
		}
		cp := *c
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (r *MemoryRepository) Update(_ context.Context, issuerID, id uuid.UUID, party domain.Party) (*Customer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.customers[id]
	if !ok || c.IssuerID != issuerID {
		return nil, ErrCustomerNotFound
	}
	c.Party = party
	c.UpdatedAt = time.Now().UTC()
	cp := *c
	return &cp, nil
}

func (r *MemoryRepository) Delete(_ context.Context, issuerID, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.customers[id]
	if !ok || c.IssuerID != issuerID {
		return ErrCustomerNotFound
	}
	delete(r.customers, c.ID)
	return nil
}
