package issuers

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryRepository es una implementación en memoria de Repository para tests — no cifra
// nada (eso es un detalle de almacenamiento de PostgresRepository, no del dominio).
type MemoryRepository struct {
	mu    sync.Mutex
	byID  map[uuid.UUID]*Issuer
	byNIT map[string]*Issuer
}

// NewMemoryRepository crea un repositorio vacío en memoria.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byID:  make(map[uuid.UUID]*Issuer),
		byNIT: make(map[string]*Issuer),
	}
}

func (r *MemoryRepository) Create(_ context.Context, iss Issuer) (*Issuer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byNIT[iss.NIT]; exists {
		return nil, ErrNITAlreadyExists
	}
	if iss.ID == uuid.Nil {
		iss.ID = uuid.New()
	}
	now := time.Now().UTC()
	iss.CreatedAt = now
	iss.UpdatedAt = now

	cp := iss
	r.byID[iss.ID] = &cp
	r.byNIT[iss.NIT] = &cp
	return &cp, nil
}

func (r *MemoryRepository) GetByID(_ context.Context, id uuid.UUID) (*Issuer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	iss, ok := r.byID[id]
	if !ok {
		return nil, ErrIssuerNotFound
	}
	cp := *iss
	return &cp, nil
}

func (r *MemoryRepository) GetByNIT(_ context.Context, nit string) (*Issuer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	iss, ok := r.byNIT[nit]
	if !ok {
		return nil, ErrIssuerNotFound
	}
	cp := *iss
	return &cp, nil
}
