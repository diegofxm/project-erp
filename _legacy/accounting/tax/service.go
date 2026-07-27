package tax

import (
	"context"
	"fmt"
	"time"

	"github.com/diegofxm/accounting/journals"
	"github.com/google/uuid"
)

const (
	// Cuentas PUC para asientos de pago de Renta (F210).
	accountRentaPagar = "240405" // De renta y complementarios — vigencia corriente (2404)
	accountRentaFavor = "135505" // Anticipo de impuestos de renta y complementarios

	// Cuentas PUC para asientos de pago de ICA (F490).
	accountIcaPagar = "2412"   // De industria y comercio (pasivo)
	accountIcaFavor = "135510" // Anticipo de impuestos de industria y comercio
)

// Service expone las operaciones tributarias: F210 Renta, F220 Certificados, F490 ICA.
type Service struct {
	repo     Repository
	journals *journals.Service
}

// NewService crea el servicio con su repositorio y el servicio de asientos.
func NewService(repo Repository, journalsSvc *journals.Service) *Service {
	return &Service{repo: repo, journals: journalsSvc}
}

// ── F210 — Renta Jurídicas ────────────────────────────────────────────────────────────────────

// GetIncomeTaxRate devuelve la tasa de renta vigente para un año.
func (s *Service) GetIncomeTaxRate(ctx context.Context, year int) (*IncomeTaxRate, error) {
	return s.repo.GetIncomeTaxRate(ctx, year)
}

// SetIncomeTaxRate registra o actualiza la tasa de renta para un año.
func (s *Service) SetIncomeTaxRate(ctx context.Context, year, rateBP int) (*IncomeTaxRate, error) {
	return s.repo.SetIncomeTaxRate(ctx, year, rateBP)
}

// ListIncomeTaxRates devuelve el historial de tasas de renta.
func (s *Service) ListIncomeTaxRates(ctx context.Context) ([]*IncomeTaxRate, error) {
	return s.repo.ListIncomeTaxRates(ctx)
}

// ComputeIncomeTax calcula la declaración de renta a partir de los datos del Estado de Resultados
// sin persistirla. Útil para previsualizar antes de confirmar.
func (s *Service) ComputeIncomeTax(ctx context.Context, req Form210Request) (*IncomeTaxDeclaration, error) {
	rate, err := s.repo.GetIncomeTaxRate(ctx, req.FiscalYear)
	if err != nil {
		return nil, fmt.Errorf("renta %d: %w", req.FiscalYear, err)
	}

	computed := req.TaxableIncome * int64(rate.RateBP) / 10_000
	taxToPay := computed - req.Discounts
	if taxToPay < 0 {
		taxToPay = 0
	}
	net := taxToPay - req.AdvancePayments

	d := &IncomeTaxDeclaration{
		CompanyID:       req.CompanyID,
		FiscalYear:      req.FiscalYear,
		TaxableIncome:   req.TaxableIncome,
		TaxRateBP:       rate.RateBP,
		TaxComputed:     computed,
		Discounts:       req.Discounts,
		TaxToPay:        taxToPay,
		AdvancePayments: req.AdvancePayments,
		Status:          StatusDraft,
	}
	if net > 0 {
		d.AmountDue = net
	} else {
		d.CarryForward = -net
	}
	return d, nil
}

// SaveIncomeTaxDeclaration calcula y persiste la declaración (upsert por empresa + año).
// Solo permite actualizar declaraciones en DRAFT; si ya está radicada retorna ErrAlreadyFiled.
func (s *Service) SaveIncomeTaxDeclaration(ctx context.Context, req Form210Request) (*IncomeTaxDeclaration, error) {
	existing, err := s.repo.GetIncomeTaxDeclarationByYear(ctx, req.CompanyID, req.FiscalYear)
	if err == nil && existing.Status != StatusDraft {
		return nil, ErrAlreadyFiled
	}

	d, err := s.ComputeIncomeTax(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.repo.SaveIncomeTaxDeclaration(ctx, *d)
}

// FileIncomeTaxDeclaration marca la declaración como radicada. Solo admite DRAFT.
func (s *Service) FileIncomeTaxDeclaration(ctx context.Context, id uuid.UUID) (*IncomeTaxDeclaration, error) {
	d, err := s.repo.GetIncomeTaxDeclarationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d.Status != StatusDraft {
		return nil, ErrAlreadyFiled
	}
	now := time.Now().UTC()
	if err := s.repo.UpdateIncomeTaxStatus(ctx, id, StatusFiled, nil, &now); err != nil {
		return nil, err
	}
	d.Status = StatusFiled
	d.FiledAt = &now
	return d, nil
}

// CreateIncomeTaxPaymentEntry genera el asiento de pago de renta y marca la declaración PAID.
// bankAccount es la cuenta de caja/banco a acreditar.
func (s *Service) CreateIncomeTaxPaymentEntry(ctx context.Context, id uuid.UUID, paymentDate time.Time, bankAccount string) (*journals.JournalEntry, error) {
	d, err := s.repo.GetIncomeTaxDeclarationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d.Status == StatusPaid {
		return nil, ErrAlreadyPaid
	}
	if d.AmountDue <= 0 {
		return nil, ErrNothingToPay
	}
	if bankAccount == "" {
		return nil, ErrBankAccountRequired
	}

	// DR Renta por pagar (cierra pasivo), CR banco/caja (salida).
	entry, err := s.journals.Post(ctx, journals.PostRequest{
		CompanyID:   d.CompanyID,
		Date:        paymentDate,
		Description: fmt.Sprintf("Pago impuesto de renta F210 año %d", d.FiscalYear),
		Source:      "tax_renta",
		EntryType:   journals.EntryAutomatic,
		VoucherType: "F210",
		Lines: []journals.LineRequest{
			{
				AccountCode: accountRentaPagar,
				Debit:       d.AmountDue,
				Description: fmt.Sprintf("Renta año %d", d.FiscalYear),
			},
			{
				AccountCode: bankAccount,
				Credit:      d.AmountDue,
				Description: fmt.Sprintf("Pago renta año %d", d.FiscalYear),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create renta payment entry: %w", err)
	}
	jid := entry.ID
	if err := s.repo.UpdateIncomeTaxStatus(ctx, id, StatusPaid, &jid, nil); err != nil {
		return nil, err
	}
	return entry, nil
}

// GetIncomeTaxDeclaration devuelve una declaración por UUID.
func (s *Service) GetIncomeTaxDeclaration(ctx context.Context, id uuid.UUID) (*IncomeTaxDeclaration, error) {
	return s.repo.GetIncomeTaxDeclarationByID(ctx, id)
}

// ListIncomeTaxDeclarations devuelve todas las declaraciones de renta de una empresa.
func (s *Service) ListIncomeTaxDeclarations(ctx context.Context, companyID uuid.UUID) ([]*IncomeTaxDeclaration, error) {
	return s.repo.ListIncomeTaxDeclarations(ctx, companyID)
}

// ── F220 — Certificados de Retención ─────────────────────────────────────────────────────────

// GenerateCertificates agrega las retenciones del año desde el libro mayor y las persiste como
// certificados en DRAFT (upsert). Llamar al cierre del año fiscal antes de imprimir certificados.
// La base gravable se estima como TaxWithheld × 10000 / rate_bp del primer concepto asociado.
func (s *Service) GenerateCertificates(ctx context.Context, companyID uuid.UUID, year int) ([]*WithholdingCertificate, error) {
	from := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)

	rows, err := s.repo.QueryWithholdingsByNIT(ctx, companyID, from, to)
	if err != nil {
		return nil, fmt.Errorf("generate certificates año %d: %w", year, err)
	}

	certs := make([]*WithholdingCertificate, 0, len(rows))
	for _, r := range rows {
		var gross int64
		if r.RateBP > 0 {
			gross = r.TaxWithheld * 10_000 / int64(r.RateBP)
		}
		c := WithholdingCertificate{
			CompanyID:     companyID,
			FiscalYear:    year,
			ThirdPartyNIT: r.ThirdPartyNIT,
			ConceptCode:   r.AccountCode,
			ConceptName:   r.AccountName,
			WHType:        r.WHType,
			GrossAmount:   gross,
			TaxWithheld:   r.TaxWithheld,
			Status:        CertDraft,
		}
		saved, err := s.repo.SaveCertificate(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("save certificate NIT %s: %w", r.ThirdPartyNIT, err)
		}
		certs = append(certs, saved)
	}
	return certs, nil
}

// IssueCertificate marca un certificado como emitido (ISSUED).
func (s *Service) IssueCertificate(ctx context.Context, id uuid.UUID) (*WithholdingCertificate, error) {
	c, err := s.repo.GetCertificateByID(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := s.repo.UpdateCertificateStatus(ctx, id, CertIssued, &now); err != nil {
		return nil, err
	}
	c.Status = CertIssued
	c.IssuedAt = &now
	return c, nil
}

// ListCertificates devuelve todos los certificados de una empresa para un año.
func (s *Service) ListCertificates(ctx context.Context, companyID uuid.UUID, year int) ([]*WithholdingCertificate, error) {
	return s.repo.ListCertificates(ctx, companyID, year)
}

// ── F490 — ICA por Municipio ──────────────────────────────────────────────────────────────────

// SetIcaTariff registra o actualiza la tarifa ICA para municipio + CIIU + año.
func (s *Service) SetIcaTariff(ctx context.Context, req IcaTariffRequest) (*IcaTariff, error) {
	if req.RateBP <= 0 {
		return nil, fmt.Errorf("ica tariff: rate_bp debe ser > 0")
	}
	return s.repo.SetIcaTariff(ctx, req)
}

// GetIcaTariff obtiene la tarifa ICA para un municipio, CIIU y año.
func (s *Service) GetIcaTariff(ctx context.Context, municipalityCode, ciiuCode string, year int) (*IcaTariff, error) {
	return s.repo.GetIcaTariff(ctx, municipalityCode, ciiuCode, year)
}

// ComputeIcaDeclaration calcula la declaración de ICA sin persistirla.
func (s *Service) ComputeIcaDeclaration(ctx context.Context, req IcaDeclarationRequest) (*IcaDeclaration, error) {
	year := req.PeriodStart.Year()
	tariff, err := s.repo.GetIcaTariff(ctx, req.MunicipalityCode, req.CIIUCode, year)
	if err != nil {
		return nil, fmt.Errorf("ica %s/%s %d: %w", req.MunicipalityCode, req.CIIUCode, year, err)
	}

	netBase := req.GrossRevenue - req.Deductions
	if netBase < 0 {
		netBase = 0
	}

	taxComputed := netBase * int64(tariff.RateBP) / 10_000
	surcharge := netBase * int64(tariff.SurchargeBP) / 10_000
	taxToPay := taxComputed + surcharge
	net := taxToPay - req.PreviousBalance

	d := &IcaDeclaration{
		CompanyID:        req.CompanyID,
		MunicipalityCode: req.MunicipalityCode,
		PeriodStart:      req.PeriodStart,
		PeriodEnd:        req.PeriodEnd,
		PeriodType:       req.PeriodType,
		CIIUCode:         req.CIIUCode,
		GrossRevenue:     req.GrossRevenue,
		Deductions:       req.Deductions,
		NetBase:          netBase,
		TariffBP:         tariff.RateBP,
		SurchargeBP:      tariff.SurchargeBP,
		TaxComputed:      taxComputed,
		SurchargeAmount:  surcharge,
		TaxToPay:         taxToPay,
		PreviousBalance:  req.PreviousBalance,
		Status:           StatusDraft,
	}
	if net > 0 {
		d.AmountDue = net
	} else {
		d.CarryForward = -net
	}
	return d, nil
}

// SaveIcaDeclaration calcula y persiste la declaración de ICA (upsert).
// Solo permite actualizar declaraciones en DRAFT; si ya está radicada retorna ErrAlreadyFiled.
func (s *Service) SaveIcaDeclaration(ctx context.Context, req IcaDeclarationRequest) (*IcaDeclaration, error) {
	d, err := s.ComputeIcaDeclaration(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.repo.SaveIcaDeclaration(ctx, *d)
}

// FileIcaDeclaration marca la declaración de ICA como radicada. Solo admite DRAFT.
func (s *Service) FileIcaDeclaration(ctx context.Context, id uuid.UUID) (*IcaDeclaration, error) {
	d, err := s.repo.GetIcaDeclarationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d.Status != StatusDraft {
		return nil, ErrAlreadyFiled
	}
	now := time.Now().UTC()
	if err := s.repo.UpdateIcaStatus(ctx, id, StatusFiled, nil, &now); err != nil {
		return nil, err
	}
	d.Status = StatusFiled
	d.FiledAt = &now
	return d, nil
}

// CreateIcaPaymentEntry genera el asiento de pago de ICA y marca la declaración PAID.
// bankAccount es la cuenta de caja/banco a acreditar.
func (s *Service) CreateIcaPaymentEntry(ctx context.Context, id uuid.UUID, paymentDate time.Time, bankAccount string) (*journals.JournalEntry, error) {
	d, err := s.repo.GetIcaDeclarationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d.Status == StatusPaid {
		return nil, ErrAlreadyPaid
	}
	if d.AmountDue <= 0 {
		return nil, ErrNothingToPay
	}
	if bankAccount == "" {
		return nil, ErrBankAccountRequired
	}

	label := fmt.Sprintf("ICA %s %s–%s", d.MunicipalityCode,
		d.PeriodStart.Format("2006-01-02"), d.PeriodEnd.Format("2006-01-02"))

	entry, err := s.journals.Post(ctx, journals.PostRequest{
		CompanyID:   d.CompanyID,
		Date:        paymentDate,
		Description: fmt.Sprintf("Pago %s", label),
		Source:      "tax_ica",
		EntryType:   journals.EntryAutomatic,
		VoucherType: "F490",
		Lines: []journals.LineRequest{
			{
				AccountCode: accountIcaPagar,
				Debit:       d.AmountDue,
				Description: label,
			},
			{
				AccountCode: bankAccount,
				Credit:      d.AmountDue,
				Description: fmt.Sprintf("Pago %s", label),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create ica payment entry: %w", err)
	}
	jid := entry.ID
	if err := s.repo.UpdateIcaStatus(ctx, id, StatusPaid, &jid, nil); err != nil {
		return nil, err
	}
	return entry, nil
}

// GetIcaDeclaration devuelve una declaración de ICA por UUID.
func (s *Service) GetIcaDeclaration(ctx context.Context, id uuid.UUID) (*IcaDeclaration, error) {
	return s.repo.GetIcaDeclarationByID(ctx, id)
}

// ListIcaDeclarations devuelve todas las declaraciones de ICA de una empresa.
func (s *Service) ListIcaDeclarations(ctx context.Context, companyID uuid.UUID) ([]*IcaDeclaration, error) {
	return s.repo.ListIcaDeclarations(ctx, companyID)
}
