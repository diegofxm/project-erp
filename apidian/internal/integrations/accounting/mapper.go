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
//	130505 Clientes nacionales    → Débito  (total con IVA; NIT del cliente)
//	413505 Ventas — Comercio       → Crédito (subtotal sin IVA)
//	240805 IVA generado por pagar  → Crédito (IVA = total − subtotal, si > 0)
func fromInvoice(doc *documents.Document, companyID uuid.UUID) (*journals.PostRequest, error) {
	total := doc.Totals.PayableCents
	subtotal := doc.Totals.LineExtensionCents
	iva := total - subtotal

	if total <= 0 {
		return nil, fmt.Errorf("accounting mapper: FE %s tiene total <= 0", doc.ID)
	}

	customerNIT := doc.Customer.Identification.Number

	lines := []journals.LineRequest{
		{
			AccountCode:   "130505",
			Debit:         total,
			ThirdPartyNIT: customerNIT,
			Description:   fmt.Sprintf("FE %s%d — cliente", doc.Prefix, doc.Number),
		},
		{
			AccountCode: "413505",
			Credit:      subtotal,
			Description: fmt.Sprintf("FE %s%d — venta", doc.Prefix, doc.Number),
		},
	}

	if iva > 0 {
		lines = append(lines, journals.LineRequest{
			AccountCode: "240805",
			Credit:      iva,
			Description: fmt.Sprintf("FE %s%d — IVA generado", doc.Prefix, doc.Number),
		})
	}

	return &journals.PostRequest{
		CompanyID:          companyID,
		Date:               doc.IssueDate,
		Description:        fmt.Sprintf("FE %s%d confirmada", doc.Prefix, doc.Number),
		Source:             "apidian",
		EntryType:          journals.EntryAutomatic,
		SourceDocumentID:   doc.ID,
		SourceDocumentType: journals.SourceFE,
		Lines:              lines,
	}, nil
}

// fromSupportDocument traduce un DS confirmado a un PostRequest contable.
// Posting rules básicas — la cuenta de gasto se recibe como parámetro porque
// depende del tipo de compra (configurar por categoría de proveedor en Fase 2).
//
//	220505 Proveedores nacionales  → Crédito (total; NIT del proveedor)
//	expenseAccountCode             → Débito  (subtotal sin IVA)
//	135530 IVA descontable         → Débito  (IVA = total − subtotal, si > 0)
func fromSupportDocument(doc *documents.Document, companyID uuid.UUID, expenseAccountCode string) (*journals.PostRequest, error) {
	total := doc.Totals.PayableCents
	subtotal := doc.Totals.LineExtensionCents
	iva := total - subtotal

	if total <= 0 {
		return nil, fmt.Errorf("accounting mapper: DS %s tiene total <= 0", doc.ID)
	}

	var vendorNIT string
	if doc.Vendor != nil {
		vendorNIT = doc.Vendor.Identification.Number
	}

	lines := []journals.LineRequest{
		{
			AccountCode:   "220505",
			Credit:        total,
			ThirdPartyNIT: vendorNIT,
			Description:   fmt.Sprintf("DS %s%d — proveedor", doc.Prefix, doc.Number),
		},
		{
			AccountCode: expenseAccountCode,
			Debit:       subtotal,
			Description: fmt.Sprintf("DS %s%d — gasto/costo", doc.Prefix, doc.Number),
		},
	}

	if iva > 0 {
		lines = append(lines, journals.LineRequest{
			AccountCode: "135530",
			Debit:       iva,
			Description: fmt.Sprintf("DS %s%d — IVA descontable", doc.Prefix, doc.Number),
		})
	}

	return &journals.PostRequest{
		CompanyID:          companyID,
		Date:               doc.IssueDate,
		Description:        fmt.Sprintf("DS %s%d confirmado", doc.Prefix, doc.Number),
		Source:             "apidian",
		EntryType:          journals.EntryAutomatic,
		SourceDocumentID:   doc.ID,
		SourceDocumentType: journals.SourceDS,
		Lines:              lines,
	}, nil
}
