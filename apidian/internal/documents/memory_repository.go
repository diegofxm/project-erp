package documents

import (
	"context"
	"fmt"
	"sort"
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

func (r *MemoryRepository) UpdateDianStatus(_ context.Context, id uuid.UUID, status Status, trackID, statusCode, statusDescription, statusMessage, applicationResponseXML string) error {
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
	d.ApplicationResponseXML = applicationResponseXML
	d.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateDraft reemplaza los campos editables de un borrador — solo mientras Status ==
// StatusDraft, mismo criterio que PostgresRepository.UpdateDraft.
func (r *MemoryRepository) UpdateDraft(_ context.Context, d Document) (*Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.docs[d.ID]
	if !ok || existing.Status != StatusDraft {
		return nil, ErrDocumentNotDraft
	}

	d.IssuerID = existing.IssuerID
	d.DianDocumentTypeCode = existing.DianDocumentTypeCode
	d.Status = StatusDraft
	d.CreatedAt = existing.CreatedAt
	d.UpdatedAt = time.Now().UTC()
	r.docs[d.ID] = &d

	cp := d
	return &cp, nil
}

// Confirm persiste los campos que solo se conocen al confirmar — solo mientras Status ==
// StatusDraft (defensivo contra doble confirmación concurrente).
func (r *MemoryRepository) Confirm(_ context.Context, d Document) (*Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.docs[d.ID]
	if !ok || existing.Status != StatusDraft {
		return nil, ErrDocumentNotDraft
	}

	existing.Prefix = d.Prefix
	existing.Number = d.Number
	existing.DocumentKey = d.DocumentKey
	existing.IssueDate = d.IssueDate
	existing.IssueTime = d.IssueTime
	existing.QRURL = d.QRURL
	existing.SignedXML = d.SignedXML
	existing.Status = d.Status
	existing.UpdatedAt = time.Now().UTC()

	cp := *existing
	return &cp, nil
}

// Delete elimina un borrador — solo mientras Status == StatusDraft.
func (r *MemoryRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.docs[id]
	if !ok || existing.Status != StatusDraft {
		return ErrDocumentNotDraft
	}
	delete(r.docs, id)
	return nil
}

func (r *MemoryRepository) ListByIssuer(_ context.Context, issuerID uuid.UUID, filter ListFilter) ([]*Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var matched []*Document
	for _, d := range r.docs {
		if d.IssuerID != issuerID {
			continue
		}
		if filter.DianDocumentTypeCode != "" && d.DianDocumentTypeCode != filter.DianDocumentTypeCode {
			continue
		}
		if filter.Status != "" && d.Status != filter.Status {
			continue
		}
		if !filter.From.IsZero() && d.IssueDate.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && d.IssueDate.After(filter.To) {
			continue
		}
		if filter.SourceDocumentID != nil {
			src, ok := r.docs[*filter.SourceDocumentID]
			if !ok || src.IssuerID != issuerID || d.BillingReference == nil {
				continue
			}
			if d.BillingReference.Prefix != src.Prefix || d.BillingReference.Number != fmt.Sprintf("%d", src.Number) {
				continue
			}
		}
		cp := *d
		matched = append(matched, &cp)
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].CreatedAt.After(matched[j].CreatedAt)
	})

	start := filter.Offset
	if start > len(matched) {
		start = len(matched)
	}
	end := start + filter.Limit
	if end > len(matched) {
		end = len(matched)
	}
	result := matched[start:end]

	// Anota NCCount/NDCount para cada documento confirmado en el resultado.
	for _, d := range result {
		if d.Prefix == "" || d.Number == 0 {
			continue
		}
		numStr := fmt.Sprintf("%d", d.Number)
		for _, other := range r.docs {
			if other.IssuerID != issuerID || other.BillingReference == nil {
				continue
			}
			if other.BillingReference.Prefix != d.Prefix || other.BillingReference.Number != numStr {
				continue
			}
			switch other.DianDocumentTypeCode {
			case "91":
				d.NCCount++
			case "92":
				d.NDCount++
			}
		}
	}

	return result, nil
}
