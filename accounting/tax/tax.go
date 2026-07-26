package tax

import (
	"time"

	"github.com/google/uuid"
)

// DeclarationStatus ciclo de vida de una declaración tributaria (renta e ICA).
type DeclarationStatus string

const (
	StatusDraft     DeclarationStatus = "DRAFT"
	StatusFiled     DeclarationStatus = "FILED"
	StatusPaid      DeclarationStatus = "PAID"
	StatusCorrected DeclarationStatus = "CORRECTED"
)

// CertificateStatus ciclo de vida de un certificado de retención (F220).
type CertificateStatus string

const (
	CertDraft     CertificateStatus = "DRAFT"
	CertIssued    CertificateStatus = "ISSUED"
	CertCorrected CertificateStatus = "CORRECTED"
)

// IcaPeriodType frecuencia de declaración de ICA.
type IcaPeriodType string

const (
	PeriodBimestral     IcaPeriodType = "BIMESTRAL"
	PeriodCuatrimestral IcaPeriodType = "CUATRIMESTRAL"
	PeriodAnual         IcaPeriodType = "ANUAL"
)

// ── F210 — Renta Jurídicas ───────────────────────────────────────────────────────────────────

// IncomeTaxRate almacena la tasa de impuesto de renta por año fiscal.
// Permite ajustar la tasa cuando la ley cambia sin modificar código.
type IncomeTaxRate struct {
	Year      int
	RateBP    int // tasa en basis points (3500 = 35%)
	CreatedAt time.Time
}

// IncomeTaxDeclaration es la declaración anual de renta (F210).
type IncomeTaxDeclaration struct {
	ID              uuid.UUID
	CompanyID       uuid.UUID
	FiscalYear      int
	TaxableIncome   int64 // renta líquida gravable en centavos
	TaxRateBP       int   // snapshot de la tasa al momento del cálculo
	TaxComputed     int64 // TaxableIncome * TaxRateBP / 10000
	Discounts       int64 // descuentos tributarios
	TaxToPay        int64 // TaxComputed − Discounts
	AdvancePayments int64 // anticipos + retenciones a favor (135505)
	AmountDue       int64 // max(0, TaxToPay − AdvancePayments)
	CarryForward    int64 // max(0, AdvancePayments − TaxToPay) — saldo a favor
	Status          DeclarationStatus
	JournalID       uuid.UUID  // asiento de pago (uuid.Nil hasta PAID)
	FiledAt         *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Form210Request contiene los datos de entrada para calcular la declaración de renta.
// TaxableIncome, Discounts y AdvancePayments vienen del Estado de Resultados del período.
type Form210Request struct {
	CompanyID       uuid.UUID
	FiscalYear      int
	TaxableIncome   int64 // renta líquida gravable: el caller ingresa este valor
	Discounts       int64 // descuentos tributarios (inversiones, etc.)
	AdvancePayments int64 // anticipos + retenciones en la fuente a favor
}

// ── F220 — Certificados de Retención en la Fuente ─────────────────────────────────────────────

// WithholdingCertificate es el certificado de retención emitido a un proveedor al cierre del año.
// concept_code es el account_code de la cuenta de retención del mayor.
type WithholdingCertificate struct {
	ID             uuid.UUID
	CompanyID      uuid.UUID
	FiscalYear     int
	ThirdPartyNIT  string
	ConceptCode    string            // account_code de la cuenta de retención (ej. "236540")
	ConceptName    string            // nombre descriptivo de la cuenta
	WHType         string            // "RETEFUENTE", "RETEIVA" o "RETEICA"
	GrossAmount    int64             // base gravable estimada (TaxWithheld * 10000 / rate_bp)
	TaxWithheld    int64             // valor retenido (suma de créditos en el período)
	Status         CertificateStatus
	IssuedAt       *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// WHByAccount es la fila de agregación del libro mayor por NIT + cuenta, usada para generar F220.
type WHByAccount struct {
	ThirdPartyNIT string
	AccountCode   string
	AccountName   string
	WHType        string
	TaxWithheld   int64
	RateBP        int // primer rate_bp en withholding_concepts para estimar base gravable
}

// ── F490 — ICA por Municipio ──────────────────────────────────────────────────────────────────

// IcaTariff almacena la tarifa de ICA para un municipio + CIIU + año.
// municipality_code y ciiu_code se almacenan sin FK para respetar el aislamiento de módulos.
type IcaTariff struct {
	ID               uuid.UUID
	MunicipalityCode string // código DANE sin FK a public.municipalities
	CIIUCode         string // código CIIU sin FK a public.ciiu_codes
	FiscalYear       int
	RateBP           int // tarifa en basis points (1000 = 10‰)
	SurchargeBP      int // sobretasa en basis points (0 si no aplica)
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// IcaTariffRequest es la entrada para registrar o actualizar una tarifa ICA.
type IcaTariffRequest struct {
	MunicipalityCode string
	CIIUCode         string
	FiscalYear       int
	RateBP           int
	SurchargeBP      int
}

// IcaDeclaration es la declaración de ICA por municipio para un período.
type IcaDeclaration struct {
	ID               uuid.UUID
	CompanyID        uuid.UUID
	MunicipalityCode string
	PeriodStart      time.Time
	PeriodEnd        time.Time
	PeriodType       IcaPeriodType
	CIIUCode         string
	GrossRevenue     int64 // ingresos operacionales del municipio en el período
	Deductions       int64 // deducciones permitidas (exportaciones, etc.)
	NetBase          int64 // GrossRevenue − Deductions
	TariffBP         int   // snapshot de la tarifa al calcular
	SurchargeBP      int   // snapshot de la sobretasa al calcular
	TaxComputed      int64 // NetBase * TariffBP / 10000
	SurchargeAmount  int64 // NetBase * SurchargeBP / 10000
	TaxToPay         int64 // TaxComputed + SurchargeAmount
	PreviousBalance  int64 // saldo a favor del período anterior
	AmountDue        int64 // max(0, TaxToPay − PreviousBalance)
	CarryForward     int64 // max(0, PreviousBalance − TaxToPay)
	Status           DeclarationStatus
	JournalID        uuid.UUID
	FiledAt          *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// IcaDeclarationRequest es la entrada para generar y persistir una declaración de ICA.
type IcaDeclarationRequest struct {
	CompanyID        uuid.UUID
	MunicipalityCode string
	PeriodStart      time.Time
	PeriodEnd        time.Time
	PeriodType       IcaPeriodType
	CIIUCode         string
	GrossRevenue     int64
	Deductions       int64
	PreviousBalance  int64
}
