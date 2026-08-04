package application_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/accounting/application"
	"github.com/diegofxm/erp/internal/accounting/domain"
)

// fakeReconciliationRepo simula la tabla reconciliation_marks — suficiente para probar que
// Mark/Unmark tratan el cruce como una relación simétrica entre las dos líneas, no una marca de
// un solo lado (ver el bug real encontrado en pruebas en vivo: marcar solo un lado dejaba la
// otra línea "atascada" como sin conciliar).
type fakeReconciliationRepo struct {
	marks map[uuid.UUID]uuid.UUID // journal_line_id -> reconciled_with
}

func newFakeReconciliationRepo() *fakeReconciliationRepo {
	return &fakeReconciliationRepo{marks: map[uuid.UUID]uuid.UUID{}}
}

func (f *fakeReconciliationRepo) Mark(_ context.Context, companyID, journalLineID uuid.UUID, reconciledWith *uuid.UUID, note string) (*domain.ReconciliationMark, error) {
	m := domain.ReconciliationMark{ID: uuid.New(), CompanyID: companyID, JournalLineID: journalLineID, ReconciledWith: reconciledWith, Note: note}
	if reconciledWith != nil {
		f.marks[journalLineID] = *reconciledWith
	} else {
		f.marks[journalLineID] = uuid.Nil
	}
	return &m, nil
}

func (f *fakeReconciliationRepo) Unmark(_ context.Context, _ uuid.UUID, journalLineID uuid.UUID) error {
	reciprocal, hasReciprocal := f.marks[journalLineID]
	delete(f.marks, journalLineID)
	if hasReciprocal {
		for lineID, target := range f.marks {
			if target == journalLineID {
				delete(f.marks, lineID)
			}
		}
		_ = reciprocal
	}
	return nil
}

func (f *fakeReconciliationRepo) ListOpenLines(_ context.Context, _ uuid.UUID, _ string) ([]domain.OpenLine, error) {
	return nil, nil
}

func (f *fakeReconciliationRepo) isMarked(lineID uuid.UUID) bool {
	_, ok := f.marks[lineID]
	return ok
}

func TestReconciliation_Mark_ClosesBothSides(t *testing.T) {
	repo := newFakeReconciliationRepo()
	uc := application.NewReconciliationUseCase(repo)
	companyID := uuid.New()
	lineA, lineB := uuid.New(), uuid.New()

	if _, err := uc.Mark(context.Background(), companyID, lineA, &lineB, "cruce"); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}

	if !repo.isMarked(lineA) {
		t.Fatal("esperaba que lineA quedara marcada")
	}
	if !repo.isMarked(lineB) {
		t.Fatal("esperaba que lineB quedara marcada también (marcado recíproco) — este es el bug real que se corrigió")
	}
}

func TestReconciliation_Unmark_ReopensBothSides(t *testing.T) {
	repo := newFakeReconciliationRepo()
	uc := application.NewReconciliationUseCase(repo)
	companyID := uuid.New()
	lineA, lineB := uuid.New(), uuid.New()

	if _, err := uc.Mark(context.Background(), companyID, lineA, &lineB, "cruce"); err != nil {
		t.Fatalf("mark falló: %v", err)
	}
	if err := uc.Unmark(context.Background(), companyID, lineA); err != nil {
		t.Fatalf("unmark falló: %v", err)
	}

	if repo.isMarked(lineA) {
		t.Fatal("esperaba que lineA quedara sin conciliar tras unmark")
	}
	if repo.isMarked(lineB) {
		t.Fatal("esperaba que lineB también quedara sin conciliar tras unmark de su contraparte")
	}
}

func TestReconciliation_MarkWithoutCounterpart_OnlyMarksSingleLine(t *testing.T) {
	repo := newFakeReconciliationRepo()
	uc := application.NewReconciliationUseCase(repo)
	companyID := uuid.New()
	lineA := uuid.New()

	if _, err := uc.Mark(context.Background(), companyID, lineA, nil, "revisada sin cruzar"); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if !repo.isMarked(lineA) {
		t.Fatal("esperaba que lineA quedara marcada")
	}
	if len(repo.marks) != 1 {
		t.Fatalf("esperaba una sola marca (sin contraparte), got %d", len(repo.marks))
	}
}
