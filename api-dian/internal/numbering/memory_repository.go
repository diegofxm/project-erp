package numbering

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryRepository es una implementación en memoria de Repository para tests. ClaimNext usa
// un mutex para preservar la misma garantía de atomicidad que PostgresRepository logra con
// un UPDATE de una sola fila.
type MemoryRepository struct {
	mu     sync.Mutex
	ranges map[uuid.UUID]*NumberingRange
}

// NewMemoryRepository crea un repositorio vacío en memoria.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{ranges: make(map[uuid.UUID]*NumberingRange)}
}

func (r *MemoryRepository) Create(_ context.Context, nr NumberingRange) (*NumberingRange, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if nr.ID == uuid.Nil {
		nr.ID = uuid.New()
	}
	now := time.Now().UTC()
	nr.CreatedAt = now
	nr.UpdatedAt = now
	nr.CurrentNumber = nr.RangeFrom - 1

	cp := nr
	r.ranges[nr.ID] = &cp
	return &cp, nil
}

func (r *MemoryRepository) GetByID(_ context.Context, id uuid.UUID) (*NumberingRange, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	nr, ok := r.ranges[id]
	if !ok {
		return nil, ErrRangeNotFound
	}
	cp := *nr
	return &cp, nil
}

func (r *MemoryRepository) ClaimNext(_ context.Context, id uuid.UUID) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	nr, ok := r.ranges[id]
	if !ok {
		return 0, ErrRangeNotFound
	}
	if !nr.IsActive {
		return 0, ErrRangeExhausted
	}
	if nr.RangeTo != nil && nr.CurrentNumber >= *nr.RangeTo {
		return 0, ErrRangeExhausted
	}

	nr.CurrentNumber++
	nr.UpdatedAt = time.Now().UTC()
	return nr.CurrentNumber, nil
}
