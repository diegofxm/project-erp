package payslips

import (
	"context"
	"fmt"

	"github.com/diegofxm/payroll/contracts"
	"github.com/google/uuid"
)

// Service orquesta el cálculo y la persistencia de liquidaciones de nómina.
type Service struct {
	repo      Repository
	contracts *contracts.Service
}

func NewService(repo Repository, contractsSvc *contracts.Service) *Service {
	return &Service{repo: repo, contracts: contractsSvc}
}

// Generate calcula y persiste la liquidación de un empleado para el período dado.
// Si ya existe una liquidación para ese período retorna error (constraint UNIQUE).
func (s *Service) Generate(ctx context.Context, in CreateInput) (*Payslip, error) {
	contract, err := s.contracts.Get(ctx, in.ContractID)
	if err != nil {
		return nil, fmt.Errorf("payslips generate: contrato: %w", err)
	}

	smmlv, err := s.repo.GetSMMLV(ctx, in.PeriodYear)
	if err != nil {
		return nil, fmt.Errorf("payslips generate: smmlv: %w", err)
	}

	arlRate, err := s.repo.GetARLRate(ctx, in.PeriodYear, string(contract.RiskClass))
	if err != nil {
		return nil, fmt.Errorf("payslips generate: arl: %w", err)
	}

	result := Calculate(CalcParams{
		Contract:   contract,
		WorkedDays: in.WorkedDays,
		SMMLVCents: smmlv,
		ARLRateBP:  arlRate,
	})

	return s.repo.Create(ctx, in, result)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Payslip, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, companyID uuid.UUID, year, month int) ([]*Payslip, error) {
	return s.repo.List(ctx, companyID, year, month)
}

func (s *Service) Approve(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, id, StatusApproved)
}

func (s *Service) Void(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateStatus(ctx, id, StatusVoided)
}
