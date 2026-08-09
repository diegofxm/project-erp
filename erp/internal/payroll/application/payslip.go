package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/payroll/domain"
	"github.com/diegofxm/erp/internal/shared/events"
)

type PayslipUseCase struct {
	payslips  domain.PayslipRepository
	contracts domain.ContractRepository
	employees domain.EmployeeRepository
	bus       *events.Bus
	audit     events.AuditLogger
}

func NewPayslipUseCase(
	payslips domain.PayslipRepository,
	contracts domain.ContractRepository,
	employees domain.EmployeeRepository,
	bus *events.Bus,
	audit events.AuditLogger,
) *PayslipUseCase {
	return &PayslipUseCase{payslips: payslips, contracts: contracts, employees: employees, bus: bus, audit: audit}
}

// Generate crea una liquidación en estado draft calculando todos los conceptos de ley.
func (uc *PayslipUseCase) Generate(ctx context.Context, companyID uuid.UUID, in domain.CreatePayslipInput) (*domain.Payslip, error) {
	emp, err := uc.employees.GetByID(ctx, in.EmployeeID)
	if err != nil {
		return nil, err
	}
	if emp.CompanyID != companyID {
		return nil, domain.ErrEmployeeNotFound
	}

	contract, err := uc.contracts.GetByID(ctx, in.ContractID)
	if err != nil {
		return nil, err
	}
	if contract.CompanyID != companyID || contract.EmployeeID != in.EmployeeID {
		return nil, domain.ErrContractNotFound
	}

	smmlv, err := uc.payslips.GetSMMLV(ctx, in.PeriodYear)
	if err != nil {
		return nil, err
	}
	arlRate, err := uc.payslips.GetARLRate(ctx, in.PeriodYear, string(contract.RiskClass))
	if err != nil {
		return nil, err
	}

	if in.WorkedDays <= 0 {
		in.WorkedDays = 30
	}
	in.CompanyID = companyID

	result := domain.Calculate(domain.CalcParams{
		Contract:   contract,
		WorkedDays: in.WorkedDays,
		SMMLVCents: smmlv,
		ARLRateBP:  arlRate,
	})

	return uc.payslips.Create(ctx, in, result)
}

func (uc *PayslipUseCase) Get(ctx context.Context, companyID, id uuid.UUID) (*domain.Payslip, error) {
	ps, err := uc.payslips.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ps.CompanyID != companyID {
		return nil, domain.ErrPayslipNotFound
	}
	return ps, nil
}

func (uc *PayslipUseCase) List(ctx context.Context, companyID uuid.UUID, year, month int) ([]*domain.Payslip, error) {
	return uc.payslips.ListByCompany(ctx, companyID, year, month)
}

func (uc *PayslipUseCase) Approve(ctx context.Context, companyID, id uuid.UUID) error {
	ps, err := uc.payslips.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if ps.CompanyID != companyID {
		return domain.ErrPayslipNotFound
	}
	if ps.Status != domain.PayslipDraft {
		return domain.ErrPayslipNotDraft
	}
	if err := uc.payslips.UpdateStatus(ctx, id, domain.PayslipApproved); err != nil {
		return err
	}
	events.PublishAndAudit(ctx, uc.bus, domain.PayrollGenerated{
		PayslipID:          ps.ID,
		CompanyID:          ps.CompanyID,
		EmployeeID:         ps.EmployeeID,
		PeriodYear:         ps.PeriodYear,
		PeriodMonth:        ps.PeriodMonth,
		TotalEarnedCents:   ps.TotalEarnedCents,
		TotalDeductedCents: ps.TotalDeductedCents,
		NetPayCents:        ps.NetPayCents,
	}, uc.audit, ps.CompanyID, "payroll.approved_side_effect_failed", "payslip", ps.ID)
	return nil
}

func (uc *PayslipUseCase) MarkPaid(ctx context.Context, companyID, id uuid.UUID) error {
	ps, err := uc.payslips.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if ps.CompanyID != companyID {
		return domain.ErrPayslipNotFound
	}
	if ps.Status != domain.PayslipApproved {
		return domain.ErrPayslipNotApproved
	}
	return uc.payslips.UpdateStatus(ctx, id, domain.PayslipPaid)
}

func (uc *PayslipUseCase) Void(ctx context.Context, companyID, id uuid.UUID) error {
	ps, err := uc.payslips.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if ps.CompanyID != companyID {
		return domain.ErrPayslipNotFound
	}
	if ps.Status == domain.PayslipPaid {
		return domain.ErrPayslipNotDraft
	}
	return uc.payslips.UpdateStatus(ctx, id, domain.PayslipVoided)
}
