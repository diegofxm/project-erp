package cartera

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service expone el análisis de cartera: aging, extracto por cliente y conciliación.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetAgingReport genera el reporte de antigüedad de cartera a la fecha asOf.
// Usa el algoritmo FIFO: los créditos (pagos) se aplican primero a los débitos
// (facturas) más antiguos. Los débitos con saldo remanente se clasifican por antigüedad.
// accountPrefixes: códigos PUC a analizar (nil usa DefaultAccountPrefixes).
func (s *Service) GetAgingReport(ctx context.Context, companyID uuid.UUID, asOf time.Time, accountPrefixes []string) (*AgingReport, error) {
	if len(accountPrefixes) == 0 {
		accountPrefixes = DefaultAccountPrefixes
	}

	movements, err := s.repo.GetMovements(ctx, companyID, asOf, accountPrefixes)
	if err != nil {
		return nil, fmt.Errorf("aging report: %w", err)
	}

	// Agrupar movimientos por NIT en el orden en que llegaron (ya ordenados por NIT + fecha).
	grouped := groupByNIT(movements)

	report := &AgingReport{
		AsOf:            asOf,
		AccountPrefixes: accountPrefixes,
		Totals:          make(map[string]int64),
	}

	for _, nit := range orderedKeys(grouped) {
		ca := applyFIFO(nit, grouped[nit], asOf)
		if ca.Total <= 0 {
			continue
		}
		for label, amount := range ca.Buckets {
			report.Totals[label] += amount
		}
		report.GrandTotal += ca.Total
		report.Customers = append(report.Customers, ca)
	}

	return report, nil
}

// GetCustomerStatement devuelve el extracto de cuenta de un cliente con saldo acumulado.
// Útil para enviarle al cliente como base de conciliación.
func (s *Service) GetCustomerStatement(ctx context.Context, companyID uuid.UUID, nit string, from, to time.Time, accountPrefixes []string) (*CustomerStatement, error) {
	if len(accountPrefixes) == 0 {
		accountPrefixes = DefaultAccountPrefixes
	}

	movements, err := s.repo.GetNITMovements(ctx, companyID, nit, accountPrefixes, from, to)
	if err != nil {
		return nil, fmt.Errorf("customer statement: %w", err)
	}

	stmt := &CustomerStatement{
		ThirdPartyNIT: nit,
		From:          from,
		To:            to,
	}

	var running int64
	for _, m := range movements {
		running += m.Debit - m.Credit
		stmt.Lines = append(stmt.Lines, &StatementLine{
			LineID:      m.LineID,
			JournalID:   m.JournalID,
			Date:        m.Date,
			Description: m.Description,
			Debit:       m.Debit,
			Credit:      m.Credit,
			RunningBal:  running,
			Reconciled:  m.Reconciled,
		})
	}
	stmt.OpenBalance = running
	return stmt, nil
}

// GetOpenItems devuelve los cargos abiertos (no totalmente pagados) de un cliente
// a una fecha de corte, clasificados por antigüedad. Útil para cobro de cartera.
func (s *Service) GetOpenItems(ctx context.Context, companyID uuid.UUID, nit string, asOf time.Time, accountPrefixes []string) ([]*OpenItem, error) {
	if len(accountPrefixes) == 0 {
		accountPrefixes = DefaultAccountPrefixes
	}

	movements, err := s.repo.GetMovements(ctx, companyID, asOf, accountPrefixes)
	if err != nil {
		return nil, fmt.Errorf("open items: %w", err)
	}

	var nitMovements []*Movement
	for _, m := range movements {
		if m.ThirdPartyNIT == nit {
			nitMovements = append(nitMovements, m)
		}
	}

	ca := applyFIFO(nit, nitMovements, asOf)
	return ca.OpenItems, nil
}

// Reconcile empareja una línea de débito (factura) con una de crédito (pago).
// Ambas líneas quedan marcadas en la tabla reconciliation_marks.
func (s *Service) Reconcile(ctx context.Context, companyID, debitLineID, creditLineID uuid.UUID, note string) error {
	if debitLineID == creditLineID {
		return ErrSameLineReconciliation
	}

	// Marcar el débito (factura) contra el crédito (pago).
	_, err := s.repo.MarkReconciled(ctx, ReconciliationMark{
		CompanyID:      companyID,
		JournalLineID:  debitLineID,
		ReconciledWith: &creditLineID,
		Note:           note,
	})
	if err != nil {
		return fmt.Errorf("reconcile debit: %w", err)
	}

	// Marcar el crédito (pago) contra el débito (factura).
	_, err = s.repo.MarkReconciled(ctx, ReconciliationMark{
		CompanyID:      companyID,
		JournalLineID:  creditLineID,
		ReconciledWith: &debitLineID,
		Note:           note,
	})
	if err != nil {
		// Rollback la primera marca si la segunda falla.
		_ = s.repo.UnmarkReconciled(ctx, debitLineID)
		return fmt.Errorf("reconcile credit: %w", err)
	}

	return nil
}

// Unreconcile elimina el emparejamiento de ambas líneas involucradas.
func (s *Service) Unreconcile(ctx context.Context, journalLineID uuid.UUID) error {
	if err := s.repo.UnmarkReconciled(ctx, journalLineID); err != nil {
		return err
	}
	return nil
}

// ─── helpers internos ────────────────────────────────────────────────────────

// applyFIFO aplica el algoritmo FIFO a los movimientos de un cliente y clasifica
// los saldos remanentes en los rangos de antigüedad estándar.
func applyFIFO(nit string, movements []*Movement, asOf time.Time) *CustomerAging {
	type openSlot struct {
		lineID      uuid.UUID
		journalID   uuid.UUID
		date        time.Time
		description string
		original    int64
		remaining   int64
	}

	var slots []openSlot
	for _, m := range movements {
		if m.Debit > 0 {
			slots = append(slots, openSlot{
				lineID:      m.LineID,
				journalID:   m.JournalID,
				date:        m.Date,
				description: m.Description,
				original:    m.Debit,
				remaining:   m.Debit,
			})
		} else if m.Credit > 0 {
			// Aplicar el pago a los slots más antiguos primero.
			toApply := m.Credit
			for i := range slots {
				if toApply <= 0 {
					break
				}
				apply := toApply
				if apply > slots[i].remaining {
					apply = slots[i].remaining
				}
				slots[i].remaining -= apply
				toApply -= apply
			}
		}
	}

	ca := &CustomerAging{
		ThirdPartyNIT: nit,
		Buckets:       make(map[string]int64),
	}

	for _, slot := range slots {
		if slot.remaining <= 0 {
			continue
		}
		days := int(asOf.Sub(slot.date).Hours() / 24)
		bucket := classifyBucket(days)

		ca.Buckets[bucket] += slot.remaining
		ca.Total += slot.remaining
		ca.OpenItems = append(ca.OpenItems, &OpenItem{
			LineID:      slot.lineID,
			JournalID:   slot.journalID,
			Date:        slot.date,
			Description: slot.description,
			Original:    slot.original,
			Remaining:   slot.remaining,
			DaysOld:     days,
			BucketLabel: bucket,
		})
	}

	return ca
}

func classifyBucket(days int) string {
	for _, b := range StandardBuckets {
		if days >= b.DaysMin && (b.DaysMax == -1 || days <= b.DaysMax) {
			return b.Label
		}
	}
	return "Más de 360 días"
}

func groupByNIT(movements []*Movement) map[string][]*Movement {
	out := make(map[string][]*Movement)
	for _, m := range movements {
		out[m.ThirdPartyNIT] = append(out[m.ThirdPartyNIT], m)
	}
	return out
}

// orderedKeys devuelve las claves del mapa en el orden en que fueron insertadas.
// Como groupByNIT preserva el orden de llegada de movements (ya ordenados por NIT),
// iterando el slice de movements podemos reconstruir el orden correcto.
func orderedKeys(m map[string][]*Movement) []string {
	seen := make(map[string]bool)
	var keys []string
	for nit := range m {
		if !seen[nit] {
			seen[nit] = true
			keys = append(keys, nit)
		}
	}
	// Ordenar alfabéticamente para output determinístico.
	for i := 0; i < len(keys)-1; i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

// ProvisionsEstimate calcula la provisión de cartera sugerida según los rangos
// del Art. 145 del ET colombiano. Es una estimación orientativa, no un asiento.
func ProvisionsEstimate(report *AgingReport) map[string]int64 {
	rates := map[string]float64{
		"Corriente":       0.00,
		"31-60 días":      0.05,
		"61-90 días":      0.10,
		"91-180 días":     0.15,
		"181-360 días":    0.25,
		"Más de 360 días": 0.50,
	}
	provisions := make(map[string]int64)
	for label, amount := range report.Totals {
		if rate, ok := rates[label]; ok && rate > 0 {
			provisions[label] = int64(float64(amount) * rate)
		}
	}
	return provisions
}

