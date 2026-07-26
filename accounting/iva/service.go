package iva

import (
	"context"
	"fmt"
	"time"

	"github.com/diegofxm/accounting/journals"
	"github.com/google/uuid"
)

// defaultIVAAccount es la cuenta PUC que concentra el IVA por pagar neto.
// La empresa puede pasar otra cuenta en CreatePaymentEntry.
const defaultIVAAccount = "240805"

// Service expone las operaciones del ciclo de IVA: consulta, declaración y pago.
type Service struct {
	repo        Repository
	journalsSvc *journals.Service
}

func NewService(repo Repository, journalsSvc *journals.Service) *Service {
	return &Service{repo: repo, journalsSvc: journalsSvc}
}

// GenerateForm300 construye la vista preliminar del Formulario 300 para el rango de fechas
// indicado, leyendo directamente del libro mayor. previousBalance es el saldo a favor
// del período anterior (0 si es el primer período o si no hay saldo).
func (s *Service) GenerateForm300(ctx context.Context, companyID uuid.UUID, from, to time.Time, periodType TaxPeriodType, previousBalance int64) (*Form300, error) {
	ivaLines, err := s.repo.QueryIVAMovements(ctx, companyID, from, to)
	if err != nil {
		return nil, fmt.Errorf("form300: %w", err)
	}

	reteLines, err := s.repo.QueryReteIVA(ctx, companyID, from, to)
	if err != nil {
		return nil, fmt.Errorf("form300 reteiva: %w", err)
	}

	form := &Form300{
		CompanyID:    companyID,
		PeriodStart:  from,
		PeriodEnd:    to,
		PeriodType:   periodType,
		IVALines:     ivaLines,
		ReteIVALines: reteLines,
		PreviousBalance: previousBalance,
	}

	for _, l := range ivaLines {
		form.TotalGenerated += l.Generated
		form.TotalDeductible += l.Deductible
	}
	for _, l := range reteLines {
		form.TotalWithheld += l.Withheld
	}

	// Net = generado − descontable − reteiva a favor
	form.NetIVA = form.TotalGenerated - form.TotalDeductible - form.TotalWithheld

	net := form.NetIVA - previousBalance
	if net > 0 {
		form.ToPay = net
		form.CarryForward = 0
	} else {
		form.ToPay = 0
		form.CarryForward = -net // saldo a favor para el siguiente período
	}

	return form, nil
}

// SaveDeclaration guarda una declaración en estado DRAFT a partir del Form300 generado.
// Si ya existe una declaración para el mismo período, la actualiza.
func (s *Service) SaveDeclaration(ctx context.Context, form *Form300) (*IVADeclaration, error) {
	decl := IVADeclaration{
		CompanyID:       form.CompanyID,
		PeriodStart:     form.PeriodStart,
		PeriodEnd:       form.PeriodEnd,
		PeriodType:      form.PeriodType,
		GeneratedIVA:    form.TotalGenerated,
		DeductibleIVA:   form.TotalDeductible,
		WithheldIVA:     form.TotalWithheld,
		NetIVA:          form.NetIVA,
		PreviousBalance: form.PreviousBalance,
		ToPay:           form.ToPay,
		CarryForward:    form.CarryForward,
		Status:          StatusDraft,
	}
	return s.repo.Save(ctx, decl)
}

// FileDeclaration marca la declaración como radicada ante la DIAN.
// Solo permite radicar declaraciones en estado DRAFT.
func (s *Service) FileDeclaration(ctx context.Context, id uuid.UUID) (*IVADeclaration, error) {
	decl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if decl.Status != StatusDraft {
		return nil, ErrDeclarationAlreadyFiled
	}
	now := time.Now().UTC()
	if err := s.repo.UpdateStatus(ctx, id, StatusFiled, nil, &now); err != nil {
		return nil, err
	}
	decl.Status = StatusFiled
	decl.FiledAt = &now
	return decl, nil
}

// CreatePaymentEntry genera el asiento contable de pago de IVA y marca la declaración
// como PAID. ivaAccount es la cuenta 2408xx a debitar (default: 240805).
// bankAccount es la cuenta de caja/banco a acreditar.
func (s *Service) CreatePaymentEntry(ctx context.Context, id uuid.UUID, paymentDate time.Time, bankAccount, ivaAccount, voucherType string) (*journals.JournalEntry, error) {
	decl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if decl.Status == StatusPaid {
		return nil, ErrDeclarationAlreadyPaid
	}
	if decl.ToPay <= 0 {
		return nil, ErrNothingToPay
	}
	if bankAccount == "" {
		return nil, ErrBankAccountRequired
	}
	if ivaAccount == "" {
		ivaAccount = defaultIVAAccount
	}

	// DR IVA por pagar (cierra el pasivo), CR banco/caja (salida de caja).
	entry, err := s.journalsSvc.Post(ctx, journals.PostRequest{
		CompanyID:   decl.CompanyID,
		Date:        paymentDate,
		Description: fmt.Sprintf("Pago IVA F300 %s – %s", decl.PeriodStart.Format("2006-01-02"), decl.PeriodEnd.Format("2006-01-02")),
		Source:      "iva_payment",
		EntryType:   journals.EntryAutomatic,
		VoucherType: voucherType,
		Lines: []journals.LineRequest{
			{
				AccountCode: ivaAccount,
				Debit:       decl.ToPay,
				Description: fmt.Sprintf("IVA declaración %s", decl.ID),
			},
			{
				AccountCode: bankAccount,
				Credit:      decl.ToPay,
				Description: fmt.Sprintf("Pago IVA declaración %s", decl.ID),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create iva payment entry: %w", err)
	}

	jid := entry.ID
	if err := s.repo.UpdateStatus(ctx, id, StatusPaid, &jid, nil); err != nil {
		return nil, err
	}
	return entry, nil
}

// GetDeclaration devuelve una declaración por su UUID.
func (s *Service) GetDeclaration(ctx context.Context, id uuid.UUID) (*IVADeclaration, error) {
	return s.repo.GetByID(ctx, id)
}

// ListDeclarations devuelve todas las declaraciones de una empresa, la más reciente primero.
func (s *Service) ListDeclarations(ctx context.Context, companyID uuid.UUID) ([]*IVADeclaration, error) {
	return s.repo.ListByCompany(ctx, companyID)
}
