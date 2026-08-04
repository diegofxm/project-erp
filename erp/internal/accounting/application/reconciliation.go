package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/accounting/domain"
)

// ReconciliationUseCase cruza manualmente líneas de asiento de una misma cuenta que se cancelan
// entre sí (ej. cartera: el débito de la factura contra el crédito del pago) — distinto de
// ManageBankUseCase.Reconcile, que cruza contra un extracto bancario externo.
type ReconciliationUseCase struct {
	repo domain.ReconciliationRepository
}

func NewReconciliationUseCase(repo domain.ReconciliationRepository) *ReconciliationUseCase {
	return &ReconciliationUseCase{repo: repo}
}

// Mark cruza journalLineID contra reconciledWith. Cuando reconciledWith no es nil, marca AMBOS
// lados del cruce (cada línea referenciando a la otra) — si solo se marcara un lado, la otra
// línea seguiría apareciendo como "sin conciliar" en ListOpenLines, que filtra por si existe una
// fila en reconciliation_marks con esa línea como journal_line_id, no como reconciled_with.
func (uc *ReconciliationUseCase) Mark(ctx context.Context, companyID, journalLineID uuid.UUID, reconciledWith *uuid.UUID, note string) (*domain.ReconciliationMark, error) {
	mark, err := uc.repo.Mark(ctx, companyID, journalLineID, reconciledWith, note)
	if err != nil {
		return nil, err
	}
	if reconciledWith != nil {
		if _, err := uc.repo.Mark(ctx, companyID, *reconciledWith, &journalLineID, note); err != nil {
			return nil, err
		}
	}
	return mark, nil
}

func (uc *ReconciliationUseCase) Unmark(ctx context.Context, companyID, journalLineID uuid.UUID) error {
	return uc.repo.Unmark(ctx, companyID, journalLineID)
}

func (uc *ReconciliationUseCase) OpenLines(ctx context.Context, companyID uuid.UUID, accountCode string) ([]domain.OpenLine, error) {
	return uc.repo.ListOpenLines(ctx, companyID, accountCode)
}
