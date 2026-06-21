package documents

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryRepository es una implementación en memoria de Repository para tests.
type MemoryRepository struct {
	mu   sync.Mutex
	docs map[uuid.UUID]*Document
}

// NewMemoryRepository crea un repositorio vacío en memoria.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{docs: make(map[uuid.UUID]*Document)}
}

func (r *MemoryRepository) Create(_ context.Context, d Document) (*Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	now := time.Now().UTC()
	d.CreatedAt = now
	d.UpdatedAt = now

	cp := d
	r.docs[d.ID] = &cp
	return &cp, nil
}

func (r *MemoryRepository) GetByID(_ context.Context, id uuid.UUID) (*Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.docs[id]
	if !ok {
		return nil, ErrDocumentNotFound
	}
	cp := *d
	return &cp, nil
}

func (r *MemoryRepository) UpdateDianStatus(_ context.Context, id uuid.UUID, status Status, trackID, statusCode, statusDescription, statusMessage string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.docs[id]
	if !ok {
		return ErrDocumentNotFound
	}
	d.Status = status
	d.DianTrackID = trackID
	d.DianStatusCode = statusCode
	d.DianStatusDescription = statusDescription
	d.DianStatusMessage = statusMessage
	d.UpdatedAt = time.Now().UTC()
	return nil
}
