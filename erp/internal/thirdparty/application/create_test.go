package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/thirdparty/application"
	"github.com/diegofxm/erp/internal/thirdparty/domain"
)

// fakeRepo es un repositorio en memoria — suficiente para probar la lógica de fusión de roles
// de CreateUseCase sin levantar Postgres.
type fakeRepo struct {
	byID map[uuid.UUID]domain.Party
}

func newFakeRepo() *fakeRepo { return &fakeRepo{byID: map[uuid.UUID]domain.Party{}} }

func (f *fakeRepo) Save(_ context.Context, p domain.Party) (*domain.Party, error) {
	f.byID[p.ID] = p
	return &p, nil
}

func (f *fakeRepo) GetByID(_ context.Context, companyID, id uuid.UUID) (*domain.Party, error) {
	p, ok := f.byID[id]
	if !ok || p.CompanyID != companyID {
		return nil, domain.ErrPartyNotFound
	}
	return &p, nil
}

func (f *fakeRepo) GetByIdentification(_ context.Context, companyID uuid.UUID, identTypeCode, identNumber string) (*domain.Party, error) {
	for _, p := range f.byID {
		if p.CompanyID == companyID && p.IdentificationTypeCode == identTypeCode && p.IdentificationNumber == identNumber {
			pc := p
			return &pc, nil
		}
	}
	return nil, domain.ErrPartyNotFound
}

func (f *fakeRepo) List(_ context.Context, companyID uuid.UUID, role domain.Role) ([]domain.Party, error) {
	var out []domain.Party
	for _, p := range f.byID {
		if p.CompanyID != companyID {
			continue
		}
		if role == domain.RoleCustomer && !p.IsCustomer {
			continue
		}
		if role == domain.RoleSupplier && !p.IsSupplier {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeRepo) Update(_ context.Context, p domain.Party) (*domain.Party, error) {
	f.byID[p.ID] = p
	return &p, nil
}

func (f *fakeRepo) Delete(_ context.Context, companyID, id uuid.UUID) error {
	p, ok := f.byID[id]
	if !ok || p.CompanyID != companyID {
		return domain.ErrPartyNotFound
	}
	delete(f.byID, id)
	return nil
}

func baseSaveRequest() application.SaveRequest {
	return application.SaveRequest{
		IdentificationTypeCode: "31",
		IdentificationNumber:   "900999999",
		Name:                   "Acme SAS",
		TaxSchemeCode:          "ZZ",
		TaxSchemeName:          "No aplica",
	}
}

func TestCreate_NewIdentification_CreatesPartyWithRole(t *testing.T) {
	repo := newFakeRepo()
	uc := application.NewCreateUseCase(repo)
	companyID := uuid.New()

	p, err := uc.Execute(context.Background(), companyID, domain.RoleCustomer, baseSaveRequest())
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if !p.IsCustomer || p.IsSupplier {
		t.Fatalf("esperaba solo IsCustomer=true, got IsCustomer=%v IsSupplier=%v", p.IsCustomer, p.IsSupplier)
	}
}

func TestCreate_ExistingIdentificationOtherRole_MergesIntoSameParty(t *testing.T) {
	repo := newFakeRepo()
	uc := application.NewCreateUseCase(repo)
	companyID := uuid.New()

	customer, err := uc.Execute(context.Background(), companyID, domain.RoleCustomer, baseSaveRequest())
	if err != nil {
		t.Fatalf("alta de cliente falló: %v", err)
	}

	req := baseSaveRequest()
	req.PaymentTermsDays = 30
	supplier, err := uc.Execute(context.Background(), companyID, domain.RoleSupplier, req)
	if err != nil {
		t.Fatalf("alta de proveedor sobre la misma identificación falló: %v", err)
	}

	if supplier.ID != customer.ID {
		t.Fatalf("esperaba que se fusionara en el mismo tercero (id %s), se creó uno nuevo (id %s)", customer.ID, supplier.ID)
	}
	if !supplier.IsCustomer || !supplier.IsSupplier {
		t.Fatalf("esperaba ambos roles activos tras la fusión, got IsCustomer=%v IsSupplier=%v", supplier.IsCustomer, supplier.IsSupplier)
	}
	if supplier.PaymentTermsDays != 30 {
		t.Fatalf("esperaba payment_terms_days=30, got %d", supplier.PaymentTermsDays)
	}
}

func TestCreate_DuplicateSameRole_Rejected(t *testing.T) {
	repo := newFakeRepo()
	uc := application.NewCreateUseCase(repo)
	companyID := uuid.New()

	if _, err := uc.Execute(context.Background(), companyID, domain.RoleCustomer, baseSaveRequest()); err != nil {
		t.Fatalf("primera alta falló: %v", err)
	}

	_, err := uc.Execute(context.Background(), companyID, domain.RoleCustomer, baseSaveRequest())
	if !errors.Is(err, domain.ErrDuplicateCustomer) {
		t.Fatalf("esperaba ErrDuplicateCustomer, got %v", err)
	}
}

func TestCreate_SameIdentificationDifferentCompany_CreatesIndependentParty(t *testing.T) {
	repo := newFakeRepo()
	uc := application.NewCreateUseCase(repo)

	p1, err := uc.Execute(context.Background(), uuid.New(), domain.RoleCustomer, baseSaveRequest())
	if err != nil {
		t.Fatalf("alta en empresa 1 falló: %v", err)
	}
	p2, err := uc.Execute(context.Background(), uuid.New(), domain.RoleCustomer, baseSaveRequest())
	if err != nil {
		t.Fatalf("alta en empresa 2 con la misma identificación falló: %v", err)
	}
	if p1.ID == p2.ID {
		t.Fatal("esperaba terceros independientes entre empresas distintas, se reutilizó el mismo id")
	}
}
