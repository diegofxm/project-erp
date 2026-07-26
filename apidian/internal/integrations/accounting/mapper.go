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
		VoucherType:        "FE",
		SourceDocumentID:   doc.ID,
		SourceDocumentType: journals.SourceFE,
		Lines:              lines,
	}, nil
}

// fromSupportDocument traduce un DS confirmado a un PostRequest contable.
// Posting rules básicas — la cuenta de gasto se recibe como parámetro porque
// depende del tipo de compra (configurar por categoría de proveedor en Fase 2).
//
// PayableCents del DS = TaxInclusiveCents (bruto, regla DIAN DSAU14). Las retenciones
// están en doc.WithholdingTaxes y se descuentan del crédito 220505; cada retención
// genera su propia línea CR (236505 Retefuente / 236540 ReteIVA / 236560 ReteICA).
//
//	expenseAccountCode             → Débito  (subtotal sin IVA)
//	135530 IVA descontable         → Débito  (IVA = TaxInclusive − subtotal, si > 0)
//	220505 Proveedores nacionales  → Crédito (bruto − retenciones; NIT del proveedor)
//	236505/236540/236560           → Crédito (retenciones, una línea por tipo, si aplica)
func fromSupportDocument(doc *documents.Document, companyID uuid.UUID, expenseAccountCode string) (*journals.PostRequest, error) {
	gross := doc.Totals.TaxInclusiveCents // subtotal + IVA, siempre == PayableCents en DS (DSAU14)
	subtotal := doc.Totals.LineExtensionCents
	iva := gross - subtotal

	if gross <= 0 {
		return nil, fmt.Errorf("accounting mapper: DS %s tiene total <= 0", doc.ID)
	}

	var vendorNIT string
	if doc.Vendor != nil {
		vendorNIT = doc.Vendor.Identification.Number
	}

	// Sumar retenciones para calcular el neto pagado al proveedor.
	var withholdingTotal int64
	for _, w := range doc.WithholdingTaxes {
		withholdingTotal += w.TaxAmountCents
	}

	lines := []journals.LineRequest{
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

	lines = append(lines, journals.LineRequest{
		AccountCode:   "220505",
		Credit:        gross - withholdingTotal,
		ThirdPartyNIT: vendorNIT,
		Description:   fmt.Sprintf("DS %s%d — proveedor (neto)", doc.Prefix, doc.Number),
	})

	for _, w := range doc.WithholdingTaxes {
		acc, ok := withholdingPayableAccount(w.TypeCode)
		if !ok || w.TaxAmountCents == 0 {
			continue
		}
		lines = append(lines, journals.LineRequest{
			AccountCode: acc,
			Credit:      w.TaxAmountCents,
			Description: fmt.Sprintf("DS %s%d — %s", doc.Prefix, doc.Number, w.TypeName),
		})
	}

	return &journals.PostRequest{
		CompanyID:          companyID,
		Date:               doc.IssueDate,
		Description:        fmt.Sprintf("DS %s%d confirmado", doc.Prefix, doc.Number),
		Source:             "apidian",
		EntryType:          journals.EntryAutomatic,
		VoucherType:        "DS",
		SourceDocumentID:   doc.ID,
		SourceDocumentType: journals.SourceDS,
		Lines:              lines,
	}, nil
}

// withholdingPayableAccount devuelve la cuenta PUC 2365XX que corresponde al tipo
// de retención según el catálogo DIAN (TypeCode del Tax en DS).
// "05" ReteIVA → 236540; "06" Retefuente → 236505; "07" ReteICA → 236560.
// Retorna ("", false) si el código no corresponde a una retención conocida.
func withholdingPayableAccount(typeCode string) (string, bool) {
	switch typeCode {
	case "05":
		return "236540", true // ReteIVA por pagar
	case "06":
		return "236505", true // Retefuente por pagar (Fase 2: cuenta específica por concepto)
	case "07":
		return "236560", true // ReteICA por pagar
	default:
		return "", false
	}
}
