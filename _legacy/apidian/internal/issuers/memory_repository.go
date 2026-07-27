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

func (r *MemoryRepository) Update(_ context.Context, iss Issuer) (*Issuer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.byID[iss.ID]
	if !ok {
		return nil, ErrIssuerNotFound
	}
	existing.SoftwareID = iss.SoftwareID
	existing.SoftwarePIN = iss.SoftwarePIN
	existing.NeSoftwareID = iss.NeSoftwareID
	existing.NeSoftwarePIN = iss.NeSoftwarePIN
	existing.Certificate = iss.Certificate
	existing.CertificatePassword = iss.CertificatePassword
	existing.Logo = iss.Logo
	existing.LogoContentType = iss.LogoContentType
	existing.UpdatedAt = time.Now().UTC()

	cp := *existing
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

func (r *MemoryRepository) UpdateProfile(_ context.Context, iss Issuer) (*Issuer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.byID[iss.ID]
	if !ok {
		return nil, ErrIssuerNotFound
	}
	existing.BusinessName = iss.BusinessName
	existing.TradeName = iss.TradeName
	existing.DepartmentCode = iss.DepartmentCode
	existing.MunicipalityCode = iss.MunicipalityCode
	existing.AddressLine = iss.AddressLine
	existing.Email = iss.Email
	existing.Phone = iss.Phone
	existing.EntityTypeCode = iss.EntityTypeCode
	existing.TaxSchemeCode = iss.TaxSchemeCode
	existing.TaxSchemeName = iss.TaxSchemeName
	existing.LiabilityCodes = iss.LiabilityCodes
	existing.TaxRegimeCode = iss.TaxRegimeCode
	existing.IndustryClassificationCodes = iss.IndustryClassificationCodes
	existing.MerchantRegistrationNumber = iss.MerchantRegistrationNumber
	existing.Environment = iss.Environment
	existing.UpdatedAt = time.Now().UTC()

	cp := *existing
	return &cp, nil
}
