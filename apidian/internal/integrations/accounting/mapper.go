package accounting

import (
	"fmt"

	"github.com/diegofxm/accounting/journals"
	"github.com/diegofxm/apidian/internal/documents"
	"github.com/google/uuid"
)

// fromInvoice traduce una FE confirmada a un PostRequest contable.
// Posting rules hardcodeadas para MVP (Fase 2 → tabla posting_rules configurable).
//
//	130505 Clientes nacionales    → Débito  (total con IVA)
//	413505 Ventas — Comercio       → Crédito (subtotal)
//	240805 IVA generado por pagar  → Crédito (IVA = total − subtotal)
func fromInvoice(doc *documents.Document, companyID uuid.UUID) (*journals.PostRequest, error) {
	totalCOP := float64(doc.Totals.PayableCents) / 100
	subtotalCOP := float64(doc.Totals.LineExtensionCents) / 100
	ivaCOP := totalCOP - subtotalCOP

	if totalCOP <= 0 {
		return nil, fmt.Errorf("accounting mapper: FE %s tiene total <= 0", doc.ID)
	}

	lines := []journals.LineRequest{
		{
			AccountCode: "130505",
			Debit:       totalCOP,
			Description: fmt.Sprintf("FE %s%d — cliente", doc.Prefix, doc.Number),
		},
		{
			AccountCode: "413505",
			Credit:      subtotalCOP,
			Description: fmt.Sprintf("FE %s%d — venta", doc.Prefix, doc.Number),
		},
	}

	if ivaCOP > 0.01 {
		lines = append(lines, journals.LineRequest{
			AccountCode: "240805",
			Credit:      ivaCOP,
			Description: fmt.Sprintf("FE %s%d — IVA generado", doc.Prefix, doc.Number),
		})
	}

	return &journals.PostRequest{
		CompanyID:   companyID,
		Date:        doc.IssueDate,
		Description: fmt.Sprintf("FE %s%d confirmada", doc.Prefix, doc.Number),
		Source:      "apidian",
		EntryType:   journals.EntryAutomatic,
		Lines:       lines,
	}, nil
}

// fromSupportDocument traduce un DS confirmado a un PostRequest contable.
// Posting rules básicas — la cuenta de gasto se recibe como parámetro porque
// depende del tipo de compra (configurar por categoría de proveedor en Fase 2).
//
//	220505 Proveedores nacionales  → Crédito (total)
//	expenseAccountCode             → Débito  (base sin IVA)
//	135530 IVA descontable         → Débito  (IVA, si > 0)
func fromSupportDocument(doc *documents.Document, companyID uuid.UUID, expenseAccountCode string) (*journals.PostRequest, error) {
	totalCOP := float64(doc.Totals.PayableCents) / 100
	subtotalCOP := float64(doc.Totals.LineExtensionCents) / 100
	ivaCOP := totalCOP - subtotalCOP

	if totalCOP <= 0 {
		return nil, fmt.Errorf("accounting mapper: DS %s tiene total <= 0", doc.ID)
	}

	lines := []journals.LineRequest{
		{
			AccountCode: "220505",
			Credit:      totalCOP,
			Description: fmt.Sprintf("DS %s%d — proveedor", doc.Prefix, doc.Number),
		},
		{
			AccountCode: expenseAccountCode,
			Debit:       subtotalCOP,
			Description: fmt.Sprintf("DS %s%d — gasto/costo", doc.Prefix, doc.Number),
		},
	}

	if ivaCOP > 0.01 {
		lines = append(lines, journals.LineRequest{
			AccountCode: "135530",
			Debit:       ivaCOP,
			Description: fmt.Sprintf("DS %s%d — IVA descontable", doc.Prefix, doc.Number),
		})
	}

	return &journals.PostRequest{
		CompanyID:   companyID,
		Date:        doc.IssueDate,
		Description: fmt.Sprintf("DS %s%d confirmado", doc.Prefix, doc.Number),
		Source:      "apidian",
		EntryType:   journals.EntryAutomatic,
		Lines:       lines,
	}, nil
}
