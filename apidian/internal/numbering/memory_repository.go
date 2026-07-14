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

// ReleaseIfCurrent — ver Repository.ReleaseIfCurrent. El mutex ya serializa todo el acceso al
// mapa, así que basta comparar y decrementar bajo el mismo lock que usa ClaimNext.
func (r *MemoryRepository) ReleaseIfCurrent(_ context.Context, id uuid.UUID, number int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	nr, ok := r.ranges[id]
	if !ok {
		return nil
	}
	if nr.CurrentNumber == number {
		nr.CurrentNumber--
		nr.UpdatedAt = time.Now().UTC()
	}
	return nil
}

// ClearTestSetID — ver Repository.ClearTestSetID. Mismo criterio que ReleaseIfCurrent: si el
// rango no existe, no es un error (consistente con el UPDATE de PostgresRepository, que
// simplemente no afecta ninguna fila).
func (r *MemoryRepository) ClearTestSetID(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	nr, ok := r.ranges[id]
	if !ok {
		return nil
	}
	nr.TestSetID = ""
	nr.UpdatedAt = time.Now().UTC()
	return nil
}

// Deactivate — ver Repository.Deactivate.
func (r *MemoryRepository) Deactivate(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	nr, ok := r.ranges[id]
	if !ok {
		return ErrRangeNotFound
	}
	nr.IsActive = false
	nr.UpdatedAt = time.Now().UTC()
	return nil
}

// Activate — ver Repository.Activate.
func (r *MemoryRepository) Activate(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	nr, ok := r.ranges[id]
	if !ok {
		return ErrRangeNotFound
	}
	nr.IsActive = true
	nr.UpdatedAt = time.Now().UTC()
	return nil
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
