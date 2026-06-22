package numbering

import (
	"context"
	"sort"
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
	// CurrentNumber ya viene decidido por Service.RegisterRange — ver postgres.go.

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

func (r *MemoryRepository) ListByIssuer(_ context.Context, issuerID uuid.UUID, dianDocumentTypeCode string) ([]*NumberingRange, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var out []*NumberingRange
	for _, nr := range r.ranges {
		if nr.IssuerID != issuerID {
			continue
		}
		if dianDocumentTypeCode != "" && nr.DianDocumentTypeCode != dianDocumentTypeCode {
			continue
		}
		cp := *nr
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}
