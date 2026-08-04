package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type DeclarationStatus string

const (
	DeclDraft     DeclarationStatus = "DRAFT"
	DeclFiled     DeclarationStatus = "FILED"
	DeclPaid      DeclarationStatus = "PAID"
	DeclCorrected DeclarationStatus = "CORRECTED"
)

type PeriodType string

const (
	PeriodBimestral     PeriodType = "BIMESTRAL"
	PeriodCuatrimestral PeriodType = "CUATRIMESTRAL"
	PeriodAnual         PeriodType = "ANUAL"
)

// ── IVA (Formulario 300) ────────────────────────────────────────────────────────────────────

// IVADeclaration resume el IVA generado (ventas) vs. descontable (compras) de un periodo —
// montos en centavos, calculados a partir de los movimientos reales de 240808/240810.
type IVADeclaration struct {
	ID              uuid.UUID
	CompanyID       uuid.UUID
	PeriodStart     time.Time
	PeriodEnd       time.Time
	PeriodType      PeriodType
	GeneratedIVA    int64
	DeductibleIVA   int64
	WithheldIVA     int64
	NetIVA          int64
	PreviousBalance int64
	AmountToPay     int64
	CarryForward    int64
	Status          DeclarationStatus
	JournalID       *uuid.UUID
	FiledAt         *time.Time
	CreatedAt       time.Time
}

type IVADeclarationRepository interface {
	Create(ctx context.Context, d IVADeclaration) (*IVADeclaration, error)
	List(ctx context.Context, companyID uuid.UUID) ([]IVADeclaration, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*IVADeclaration, error)
	// GetLastCarryForward devuelve el saldo a favor de la declaración anterior a la fecha dada
	// (0 si no hay una previa) — se convierte en el previous_balance de la nueva declaración.
	GetLastCarryForward(ctx context.Context, companyID uuid.UUID, before time.Time) (int64, error)
	MarkFiled(ctx context.Context, companyID, id uuid.UUID) error
}

// ── Renta (Formulario 110) ──────────────────────────────────────────────────────────────────

type IncomeTaxDeclaration struct {
	ID              uuid.UUID
	CompanyID       uuid.UUID
	FiscalYear      int
	TaxableIncome   int64
	TaxRateBP       int
	TaxComputed     int64
	Discounts       int64
	TaxToPay        int64
	AdvancePayments int64
	AmountDue       int64
	CarryForward    int64
	Status          DeclarationStatus
	JournalID       *uuid.UUID
	FiledAt         *time.Time
	CreatedAt       time.Time
}

type IncomeTaxDeclarationRepository interface {
	Create(ctx context.Context, d IncomeTaxDeclaration) (*IncomeTaxDeclaration, error)
	List(ctx context.Context, companyID uuid.UUID) ([]IncomeTaxDeclaration, error)
	GetRateForYear(ctx context.Context, year int) (int, error) // rate_bp
	MarkFiled(ctx context.Context, companyID, id uuid.UUID) error
}

// ── ICA (industria y comercio) ──────────────────────────────────────────────────────────────

// ICATariff es la tarifa vigente para un municipio+CIIU+año — administrable manualmente, NO se
// siembra con datos ficticios porque varía por cada uno de los +1000 municipios de Colombia.
type ICATariff struct {
	ID               uuid.UUID
	MunicipalityCode string
	CIIUCode         string
	FiscalYear       int
	RateBP           int
	SurchargeBP      int
}

type ICADeclaration struct {
	ID               uuid.UUID
	CompanyID        uuid.UUID
	MunicipalityCode string
	PeriodStart      time.Time
	PeriodEnd        time.Time
	PeriodType       PeriodType
	CIIUCode         string
	GrossRevenue     int64
	Deductions       int64
	NetBase          int64
	TariffBP         int
	SurchargeBP      int
	TaxComputed      int64
	SurchargeAmount  int64
	TaxToPay         int64
	PreviousBalance  int64
	AmountDue        int64
	CarryForward     int64
	Status           DeclarationStatus
	JournalID        *uuid.UUID
	FiledAt          *time.Time
	CreatedAt        time.Time
}

var ErrICATariffNotFound = errors.New("no hay tarifa de ICA registrada para ese municipio, CIIU y año — regístrala primero")

type ICATariffRepository interface {
	Create(ctx context.Context, t ICATariff) (*ICATariff, error)
	List(ctx context.Context) ([]ICATariff, error)
	Get(ctx context.Context, municipalityCode, ciiuCode string, year int) (*ICATariff, error)
}

type ICADeclarationRepository interface {
	Create(ctx context.Context, d ICADeclaration) (*ICADeclaration, error)
	List(ctx context.Context, companyID uuid.UUID) ([]ICADeclaration, error)
	GetLastCarryForward(ctx context.Context, companyID uuid.UUID, municipalityCode string, before time.Time) (int64, error)
	MarkFiled(ctx context.Context, companyID, id uuid.UUID) error
}
