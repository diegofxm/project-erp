package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

// ── IVA ──────────────────────────────────────────────────────────────────────────────────────

type IVAUseCase struct {
	declarations domain.IVADeclarationRepository
	journals     domain.JournalRepository
	accounts     domain.AccountRepository
}

func NewIVAUseCase(declarations domain.IVADeclarationRepository, journals domain.JournalRepository, accounts domain.AccountRepository) *IVAUseCase {
	return &IVAUseCase{declarations: declarations, journals: journals, accounts: accounts}
}

func (uc *IVAUseCase) Generate(ctx context.Context, companyID uuid.UUID, from, to time.Time, periodType string) (*domain.IVADeclaration, error) {
	trial, err := uc.journals.GetTrialBalance(ctx, companyID, from, to)
	if err != nil {
		return nil, err
	}
	// El seed real (extraído de puc.com.co) no separa IVA generado/descontable en cuentas
	// distintas — 2408 "Impuesto sobre las ventas por pagar" es la única cuenta posteable de
	// ese grupo. on_sale_confirmed la abona (generado) y on_purchase_received la carga
	// (descontable); acá separamos ambos lados del mismo movimiento.
	var generated, deductible int64
	for _, row := range trial {
		if row.AccountCode == "2408" {
			generated += row.Credit
			deductible += row.Debit
		}
	}
	previousBalance, err := uc.declarations.GetLastCarryForward(ctx, companyID, from)
	if err != nil {
		return nil, err
	}
	netIVA := generated - deductible
	total := netIVA - previousBalance
	var amountToPay, carryForward int64
	if total > 0 {
		amountToPay = total
	} else {
		carryForward = -total
	}

	pt := domain.PeriodType(periodType)
	if pt == "" {
		pt = domain.PeriodBimestral
	}
	return uc.declarations.Create(ctx, domain.IVADeclaration{
		CompanyID: companyID, PeriodStart: from, PeriodEnd: to, PeriodType: pt,
		GeneratedIVA: generated, DeductibleIVA: deductible, NetIVA: netIVA,
		PreviousBalance: previousBalance, AmountToPay: amountToPay, CarryForward: carryForward,
	})
}

func (uc *IVAUseCase) List(ctx context.Context, companyID uuid.UUID) ([]domain.IVADeclaration, error) {
	return uc.declarations.List(ctx, companyID)
}

func (uc *IVAUseCase) MarkFiled(ctx context.Context, companyID, id uuid.UUID) error {
	return uc.declarations.MarkFiled(ctx, companyID, id)
}

// ── Renta ────────────────────────────────────────────────────────────────────────────────────

type IncomeTaxUseCase struct {
	declarations domain.IncomeTaxDeclarationRepository
	journals     domain.JournalRepository
}

func NewIncomeTaxUseCase(declarations domain.IncomeTaxDeclarationRepository, journals domain.JournalRepository) *IncomeTaxUseCase {
	return &IncomeTaxUseCase{declarations: declarations, journals: journals}
}

var expenseCategoriesForTax = []string{"Gasto", "Costo", "Costo de Producción"}

func (uc *IncomeTaxUseCase) Generate(ctx context.Context, companyID uuid.UUID, year int) (*domain.IncomeTaxDeclaration, error) {
	balances, err := uc.journals.GetYearPLBalances(ctx, companyID, year)
	if err != nil {
		return nil, err
	}
	var income, expenses int64
	for _, b := range balances {
		if b.Category == "Ingreso" {
			income += -b.Balance // Balance = debit-credit; Ingreso es cuenta crédito, balance negativo = saldo real positivo
			continue
		}
		for _, c := range expenseCategoriesForTax {
			if b.Category == c {
				expenses += b.Balance
				break
			}
		}
	}
	taxableIncome := income - expenses
	if taxableIncome < 0 {
		taxableIncome = 0
	}

	rateBP, err := uc.declarations.GetRateForYear(ctx, year)
	if err != nil {
		return nil, err
	}
	taxComputed := taxableIncome * int64(rateBP) / 10000

	return uc.declarations.Create(ctx, domain.IncomeTaxDeclaration{
		CompanyID: companyID, FiscalYear: year, TaxableIncome: taxableIncome, TaxRateBP: rateBP,
		TaxComputed: taxComputed, TaxToPay: taxComputed, AmountDue: taxComputed,
	})
}

func (uc *IncomeTaxUseCase) List(ctx context.Context, companyID uuid.UUID) ([]domain.IncomeTaxDeclaration, error) {
	return uc.declarations.List(ctx, companyID)
}

func (uc *IncomeTaxUseCase) MarkFiled(ctx context.Context, companyID, id uuid.UUID) error {
	return uc.declarations.MarkFiled(ctx, companyID, id)
}

// ── ICA ──────────────────────────────────────────────────────────────────────────────────────

type ICAUseCase struct {
	declarations domain.ICADeclarationRepository
	tariffs      domain.ICATariffRepository
	journals     domain.JournalRepository
}

func NewICAUseCase(declarations domain.ICADeclarationRepository, tariffs domain.ICATariffRepository, journals domain.JournalRepository) *ICAUseCase {
	return &ICAUseCase{declarations: declarations, tariffs: tariffs, journals: journals}
}

func (uc *ICAUseCase) SetTariff(ctx context.Context, municipalityCode, ciiuCode string, year, rateBP, surchargeBP int) (*domain.ICATariff, error) {
	if rateBP <= 0 {
		return nil, fmt.Errorf("la tarifa debe ser mayor a cero")
	}
	return uc.tariffs.Create(ctx, domain.ICATariff{MunicipalityCode: municipalityCode, CIIUCode: ciiuCode, FiscalYear: year, RateBP: rateBP, SurchargeBP: surchargeBP})
}

func (uc *ICAUseCase) ListTariffs(ctx context.Context) ([]domain.ICATariff, error) {
	return uc.tariffs.List(ctx)
}

type GenerateICARequest struct {
	MunicipalityCode string
	CIIUCode         string
	PeriodStart      time.Time
	PeriodEnd        time.Time
	PeriodType       string
	DeductionsCents  int64
}

func (uc *ICAUseCase) Generate(ctx context.Context, companyID uuid.UUID, req GenerateICARequest) (*domain.ICADeclaration, error) {
	tariff, err := uc.tariffs.Get(ctx, req.MunicipalityCode, req.CIIUCode, req.PeriodStart.Year())
	if err != nil {
		return nil, err
	}
	grossRevenue, err := uc.journals.GetIncomeInPeriod(ctx, companyID, req.PeriodStart, req.PeriodEnd)
	if err != nil {
		return nil, err
	}
	netBase := grossRevenue - req.DeductionsCents
	if netBase < 0 {
		netBase = 0
	}
	taxComputed := netBase * int64(tariff.RateBP) / 10000
	surcharge := taxComputed * int64(tariff.SurchargeBP) / 10000
	taxToPay := taxComputed + surcharge

	previousBalance, err := uc.declarations.GetLastCarryForward(ctx, companyID, req.MunicipalityCode, req.PeriodStart)
	if err != nil {
		return nil, err
	}
	total := taxToPay - previousBalance
	var amountDue, carryForward int64
	if total > 0 {
		amountDue = total
	} else {
		carryForward = -total
	}

	pt := domain.PeriodType(req.PeriodType)
	if pt == "" {
		pt = domain.PeriodBimestral
	}
	return uc.declarations.Create(ctx, domain.ICADeclaration{
		CompanyID: companyID, MunicipalityCode: req.MunicipalityCode, PeriodStart: req.PeriodStart, PeriodEnd: req.PeriodEnd,
		PeriodType: pt, CIIUCode: req.CIIUCode, GrossRevenue: grossRevenue, Deductions: req.DeductionsCents, NetBase: netBase,
		TariffBP: tariff.RateBP, SurchargeBP: tariff.SurchargeBP, TaxComputed: taxComputed, SurchargeAmount: surcharge,
		TaxToPay: taxToPay, PreviousBalance: previousBalance, AmountDue: amountDue, CarryForward: carryForward,
	})
}

func (uc *ICAUseCase) List(ctx context.Context, companyID uuid.UUID) ([]domain.ICADeclaration, error) {
	return uc.declarations.List(ctx, companyID)
}

func (uc *ICAUseCase) MarkFiled(ctx context.Context, companyID, id uuid.UUID) error {
	return uc.declarations.MarkFiled(ctx, companyID, id)
}
