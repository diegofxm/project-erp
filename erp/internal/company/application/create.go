package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/company/domain"
	"github.com/diegofxm/erp/internal/shared/nit"
)

// MembershipLinker vincula el usuario creador a la empresa recién creada, y expone si ese
// usuario es superadmin de la plataforma. Implementado por
// security/infrastructure/persistence/postgres.Repository.
type MembershipLinker interface {
	AddCompany(ctx context.Context, userID, companyID uuid.UUID, role string) error
	IsSuperAdmin(ctx context.Context, userID uuid.UUID) (bool, error)
}

// PlanAssigner asigna el plan Interno (ilimitado, sin costo, no aparece en el catálogo público) a
// la empresa que acaba de crear un superadmin — la empresa operadora de la plataforma no debe
// depender de que alguien entre luego a /admin/company a asignárselo a mano. Empresas creadas por
// un dueño normal (no superadmin) no disparan esto — a esas se les asigna un plan real después,
// manualmente, desde el panel. Implementado por company/infrastructure/saas.Adapter.
type PlanAssigner interface {
	AssignInternalPlan(ctx context.Context, companyID uuid.UUID) error
}

type CreateUseCase struct {
	repo       domain.Repository
	members    MembershipLinker
	warehouses domain.WarehouseRepository
	plans      PlanAssigner
}

func NewCreateUseCase(repo domain.Repository, members MembershipLinker, warehouses domain.WarehouseRepository, plans PlanAssigner) *CreateUseCase {
	return &CreateUseCase{repo: repo, members: members, warehouses: warehouses, plans: plans}
}

type CreateRequest struct {
	NIT                         string             `json:"nit"`
	CheckDigit                  string             `json:"check_digit"`
	BusinessName                string             `json:"business_name"`
	TradeName                   string             `json:"trade_name"`
	IdentificationTypeCode      string             `json:"identification_type_code"`
	DepartmentCode              string             `json:"department_code"`
	MunicipalityCode            string             `json:"municipality_code"`
	AddressLine                 string             `json:"address_line"`
	Email                       string             `json:"email"`
	Phone                       string             `json:"phone"`
	Environment                 domain.Environment `json:"environment"`
	EntityTypeCode              string             `json:"entity_type_code"`
	TaxSchemeCode               string             `json:"tax_scheme_code"`
	TaxSchemeName               string             `json:"tax_scheme_name"`
	LiabilityCodes              []string           `json:"liability_codes"`
	TaxRegimeCode               *string            `json:"tax_regime_code"`
	IndustryClassificationCodes []string           `json:"industry_classification_codes"`
	MerchantRegistrationNumber  *string            `json:"merchant_registration_number"`
}

func (uc *CreateUseCase) Execute(ctx context.Context, creatorID uuid.UUID, req CreateRequest) (*domain.Company, error) {
	if _, err := uc.repo.GetByNIT(ctx, req.NIT); err == nil {
		return nil, domain.ErrNITTaken
	}

	if req.Environment == "" {
		req.Environment = domain.Habilitacion
	}
	if req.EntityTypeCode == "" {
		req.EntityTypeCode = "1"
	}
	if req.TaxSchemeCode == "" {
		req.TaxSchemeCode = "ZZ"
		req.TaxSchemeName = "No aplica"
	}

	// Para NITs (tipo 31) el DV se calcula siempre — aunque el usuario mande uno incorrecto,
	// la DIAN requiere el dígito exacto o rechaza el documento.
	checkDigit := req.CheckDigit
	if req.IdentificationTypeCode == "31" {
		if dv, err := nit.ComputeCheckDigit(req.NIT); err == nil {
			checkDigit = dv
		}
	}

	c := domain.Company{
		ID:                          uuid.New(),
		NIT:                         req.NIT,
		CheckDigit:                  checkDigit,
		BusinessName:                req.BusinessName,
		TradeName:                   req.TradeName,
		IdentificationTypeCode:      req.IdentificationTypeCode,
		DepartmentCode:              req.DepartmentCode,
		MunicipalityCode:            req.MunicipalityCode,
		AddressLine:                 req.AddressLine,
		Email:                       req.Email,
		Phone:                       req.Phone,
		Environment:                 req.Environment,
		EntityTypeCode:              req.EntityTypeCode,
		TaxSchemeCode:               req.TaxSchemeCode,
		TaxSchemeName:               req.TaxSchemeName,
		LiabilityCodes:              req.LiabilityCodes,
		TaxRegimeCode:               req.TaxRegimeCode,
		IndustryClassificationCodes: req.IndustryClassificationCodes,
		MerchantRegistrationNumber:  req.MerchantRegistrationNumber,
		IsActive:                    true,
	}

	saved, err := uc.repo.Save(ctx, c)
	if err != nil {
		return nil, err
	}

	if err := uc.members.AddCompany(ctx, creatorID, saved.ID, "owner"); err != nil {
		return nil, err
	}

	// La empresa nace con una bodega "Principal" ya lista — sin esto, el primer intento de
	// ajustar inventario se encuentra con el selector de bodegas vacío (GetOrCreateDefault normalmente
	// solo se llama de forma perezosa al confirmar la primera venta/compra, ver sales.ConfirmUseCase).
	if _, err := uc.warehouses.GetOrCreateDefault(ctx, saved.ID); err != nil {
		return nil, err
	}

	// La empresa de un superadmin (la que opera la plataforma) queda con el plan Interno de una
	// vez — el resto de empresas (clientes reales) se quedan sin plan hasta que el superadmin se
	// los asigne a mano desde /admin/company.
	if isSuperAdmin, err := uc.members.IsSuperAdmin(ctx, creatorID); err == nil && isSuperAdmin && uc.plans != nil {
		if err := uc.plans.AssignInternalPlan(ctx, saved.ID); err != nil {
			return nil, fmt.Errorf("asignar plan interno: %w", err)
		}
	}

	return saved, nil
}
