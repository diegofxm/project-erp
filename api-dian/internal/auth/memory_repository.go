package auth

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryRepository es una implementación en memoria de Repository para tests.
type MemoryRepository struct {
	mu          sync.Mutex
	byID        map[uuid.UUID]*User
	byEmail     map[string]*User
	memberships map[uuid.UUID]map[uuid.UUID]IssuerMembership // userID -> issuerID -> membership
}

// NewMemoryRepository crea un repositorio vacío en memoria.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byID:        make(map[uuid.UUID]*User),
		byEmail:     make(map[string]*User),
		memberships: make(map[uuid.UUID]map[uuid.UUID]IssuerMembership),
	}
}

func (r *MemoryRepository) Create(_ context.Context, u User) (*User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byEmail[u.Email]; exists {
		return nil, ErrEmailAlreadyExists
	}
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	cp := u
	r.byID[u.ID] = &cp
	r.byEmail[u.Email] = &cp
	return &cp, nil
}

func (r *MemoryRepository) GetByID(_ context.Context, id uuid.UUID) (*User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.byID[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *MemoryRepository) GetByEmail(_ context.Context, email string) (*User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.byEmail[email]
	if !ok {
		return nil, ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *MemoryRepository) LinkIssuer(_ context.Context, userID, issuerID uuid.UUID, role string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.memberships[userID] == nil {
		r.memberships[userID] = make(map[uuid.UUID]IssuerMembership)
	}
	if _, exists := r.memberships[userID][issuerID]; exists {
		return nil // idempotente, mismo criterio que PostgresRepository (ON CONFLICT DO NOTHING)
	}
	r.memberships[userID][issuerID] = IssuerMembership{UserID: userID, IssuerID: issuerID, Role: role, CreatedAt: time.Now().UTC()}
	return nil
}

func (r *MemoryRepository) ListIssuerIDs(_ context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var ids []uuid.UUID
	for id := range r.memberships[userID] {
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *MemoryRepository) HasAccess(_ context.Context, userID, issuerID uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.memberships[userID][issuerID]
	return ok, nil
}
