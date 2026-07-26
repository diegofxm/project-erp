package accounting

import (
	"fmt"

	"github.com/diegofxm/accounting/journals"
	"github.com/diegofxm/apidian/internal/documents"
	"github.com/google/uuid"
)

// ── Helpers de conversión de moneda ─────────────────────────────────────────────────────────

// toCOP convierte centavos en moneda extranjera a centavos COP usando rateX10000.
// Si rateX10000 == 0 (documento en COP) devuelve el valor sin cambios.
func toCOP(foreignCents, rateX10000 int64) int64 {
	if rateX10000 == 0 {
		return foreignCents
	}
	return foreignCents * rateX10000 / 10_000
}

// fAmt devuelve el monto en moneda extranjera para poblar JournalLine.ForeignAmount.
// Devuelve 0 cuando el documento es en COP (currency vacío o "COP").
func fAmt(foreignCents int64, currency string) int64 {
	if currency == "" || currency == "COP" {
		return 0
	}
	return foreignCents
}

// fCur devuelve el código de moneda para JournalLine.ForeignCurrency.
// Devuelve "" cuando el documento es en COP.
func fCur(currency string) string {
	if currency == "COP" {
		return ""
	}
	return currency
}

// ── Mappers por tipo de documento ────────────────────────────────────────────────────────────

// fromInvoice traduce una FE confirmada a un PostRequest contable.
// Posting rules hardcodeadas para MVP (Fase 2 → tabla posting_rules configurable).
//
//	130505 Clientes nacionales    → Débito  (total con IVA; NIT del cliente)
//	413505 Ventas — Comercio       → Crédito (subtotal sin IVA)
//	240805 IVA generado por pagar  → Crédito (IVA = total − subtotal, si > 0)
//
// rateX10000: TRM × 10 000 si el documento es en moneda extranjera; 0 para COP.
// currency:   código ISO 4217 ("USD", "EUR") o "" para COP.
func fromInvoice(doc *documents.Document, companyID uuid.UUID, rateX10000 int64, currency string) (*journals.PostRequest, error) {
	rawTotal := doc.Totals.PayableCents
	rawSubtotal := doc.Totals.LineExtensionCents

	total := toCOP(rawTotal, rateX10000)
	subtotal := toCOP(rawSubtotal, rateX10000)
	iva := total - subtotal

	if total <= 0 {
		return nil, fmt.Errorf("accounting mapper: FE %s tiene total <= 0", doc.ID)
	}

	customerNIT := doc.Customer.Identification.Number
	cur := fCur(currency)

	lines := []journals.LineRequest{
		{
			AccountCode:     "130505",
			Debit:           total,
			ThirdPartyNIT:   customerNIT,
			Description:     fmt.Sprintf("FE %s%d — cliente", doc.Prefix, doc.Number),
			ForeignAmount:   fAmt(rawTotal, currency),
			ForeignCurrency: cur,
		},
		{
			AccountCode:     "413505",
			Credit:          subtotal,
			Description:     fmt.Sprintf("FE %s%d — venta", doc.Prefix, doc.Number),
			ForeignAmount:   fAmt(rawSubtotal, currency),
			ForeignCurrency: cur,
		},
	}

	if iva > 0 {
		rawIVA := rawTotal - rawSubtotal
		lines = append(lines, journals.LineRequest{
			AccountCode:     "240805",
			Credit:          iva,
			Description:     fmt.Sprintf("FE %s%d — IVA generado", doc.Prefix, doc.Number),
			ForeignAmount:   fAmt(rawIVA, currency),
			ForeignCurrency: cur,
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
func fromSupportDocument(doc *documents.Document, companyID uuid.UUID, expenseAccountCode string, rateX10000 int64, currency string) (*journals.PostRequest, error) {
	rawGross := doc.Totals.TaxInclusiveCents
	rawSubtotal := doc.Totals.LineExtensionCents

	gross := toCOP(rawGross, rateX10000)
	subtotal := toCOP(rawSubtotal, rateX10000)
	iva := gross - subtotal

	if gross <= 0 {
		return nil, fmt.Errorf("accounting mapper: DS %s tiene total <= 0", doc.ID)
	}

	var vendorNIT string
	if doc.Vendor != nil {
		vendorNIT = doc.Vendor.Identification.Number
	}
	cur := fCur(currency)

	// Sumar retenciones (en COP) para calcular el neto pagado al proveedor.
	var withholdingTotal int64
	for _, w := range doc.WithholdingTaxes {
		withholdingTotal += toCOP(w.TaxAmountCents, rateX10000)
	}

	lines := []journals.LineRequest{
		{
			AccountCode:     expenseAccountCode,
			Debit:           subtotal,
			Description:     fmt.Sprintf("DS %s%d — gasto/costo", doc.Prefix, doc.Number),
			ForeignAmount:   fAmt(rawSubtotal, currency),
			ForeignCurrency: cur,
		},
	}

	if iva > 0 {
		rawIVA := rawGross - rawSubtotal
		lines = append(lines, journals.LineRequest{
			AccountCode:     "135530",
			Debit:           iva,
			Description:     fmt.Sprintf("DS %s%d — IVA descontable", doc.Prefix, doc.Number),
			ForeignAmount:   fAmt(rawIVA, currency),
			ForeignCurrency: cur,
		})
	}

	lines = append(lines, journals.LineRequest{
		AccountCode:     "220505",
		Credit:          gross - withholdingTotal,
		ThirdPartyNIT:   vendorNIT,
		Description:     fmt.Sprintf("DS %s%d — proveedor (neto)", doc.Prefix, doc.Number),
		ForeignAmount:   fAmt(rawGross, currency),
		ForeignCurrency: cur,
	})

	for _, w := range doc.WithholdingTaxes {
		acc, ok := withholdingPayableAccount(w.TypeCode)
		if !ok || w.TaxAmountCents == 0 {
			continue
		}
		lines = append(lines, journals.LineRequest{
			AccountCode: acc,
			Credit:      toCOP(w.TaxAmountCents, rateX10000),
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

// fromCreditNote traduce una NC confirmada a un PostRequest contable.
// La NC reduce la cuenta por cobrar y revierte ventas + IVA del período.
//
//	413505 Ventas — Comercio       → Débito  (subtotal de la nota)
//	240805 IVA generado por pagar  → Débito  (IVA de la nota, si > 0)
//	130505 Clientes nacionales     → Crédito (total con IVA; NIT del cliente)
func fromCreditNote(doc *documents.Document, companyID uuid.UUID, rateX10000 int64, currency string) (*journals.PostRequest, error) {
	rawTotal := doc.Totals.PayableCents
	rawSubtotal := doc.Totals.LineExtensionCents

	total := toCOP(rawTotal, rateX10000)
	subtotal := toCOP(rawSubtotal, rateX10000)
	iva := total - subtotal

	if total <= 0 {
		return nil, fmt.Errorf("accounting mapper: NC %s tiene total <= 0", doc.ID)
	}

	customerNIT := doc.Customer.Identification.Number
	cur := fCur(currency)

	lines := []journals.LineRequest{
		{
			AccountCode:     "413505",
			Debit:           subtotal,
			Description:     fmt.Sprintf("NC %s%d — reverso venta", doc.Prefix, doc.Number),
			ForeignAmount:   fAmt(rawSubtotal, currency),
			ForeignCurrency: cur,
		},
	}
	if iva > 0 {
		rawIVA := rawTotal - rawSubtotal
		lines = append(lines, journals.LineRequest{
			AccountCode:     "240805",
			Debit:           iva,
			Description:     fmt.Sprintf("NC %s%d — reverso IVA", doc.Prefix, doc.Number),
			ForeignAmount:   fAmt(rawIVA, currency),
			ForeignCurrency: cur,
		})
	}
	lines = append(lines, journals.LineRequest{
		AccountCode:     "130505",
		Credit:          total,
		ThirdPartyNIT:   customerNIT,
		Description:     fmt.Sprintf("NC %s%d — cliente", doc.Prefix, doc.Number),
		ForeignAmount:   fAmt(rawTotal, currency),
		ForeignCurrency: cur,
	})

	return &journals.PostRequest{
		CompanyID:          companyID,
		Date:               doc.IssueDate,
		Description:        fmt.Sprintf("NC %s%d confirmada", doc.Prefix, doc.Number),
		Source:             "apidian",
		EntryType:          journals.EntryAutomatic,
		VoucherType:        "NC",
		SourceDocumentID:   doc.ID,
		SourceDocumentType: journals.SourceNC,
		Lines:              lines,
	}, nil
}

// fromDebitNote traduce una ND confirmada a un PostRequest contable.
// La ND aumenta la cuenta por cobrar (cargo adicional al cliente).
//
//	130505 Clientes nacionales     → Débito  (total con IVA; NIT del cliente)
//	413505 Ventas — Comercio       → Crédito (subtotal de la nota)
//	240805 IVA generado por pagar  → Crédito (IVA de la nota, si > 0)
func fromDebitNote(doc *documents.Document, companyID uuid.UUID, rateX10000 int64, currency string) (*journals.PostRequest, error) {
	rawTotal := doc.Totals.PayableCents
	rawSubtotal := doc.Totals.LineExtensionCents

	total := toCOP(rawTotal, rateX10000)
	subtotal := toCOP(rawSubtotal, rateX10000)
	iva := total - subtotal

	if total <= 0 {
		return nil, fmt.Errorf("accounting mapper: ND %s tiene total <= 0", doc.ID)
	}

	customerNIT := doc.Customer.Identification.Number
	cur := fCur(currency)

	lines := []journals.LineRequest{
		{
			AccountCode:     "130505",
			Debit:           total,
			ThirdPartyNIT:   customerNIT,
			Description:     fmt.Sprintf("ND %s%d — cliente", doc.Prefix, doc.Number),
			ForeignAmount:   fAmt(rawTotal, currency),
			ForeignCurrency: cur,
		},
		{
			AccountCode:     "413505",
			Credit:          subtotal,
			Description:     fmt.Sprintf("ND %s%d — cargo adicional", doc.Prefix, doc.Number),
			ForeignAmount:   fAmt(rawSubtotal, currency),
			ForeignCurrency: cur,
		},
	}
	if iva > 0 {
		rawIVA := rawTotal - rawSubtotal
		lines = append(lines, journals.LineRequest{
			AccountCode:     "240805",
			Credit:          iva,
			Description:     fmt.Sprintf("ND %s%d — IVA generado", doc.Prefix, doc.Number),
			ForeignAmount:   fAmt(rawIVA, currency),
			ForeignCurrency: cur,
		})
	}

	return &journals.PostRequest{
		CompanyID:          companyID,
		Date:               doc.IssueDate,
		Description:        fmt.Sprintf("ND %s%d confirmada", doc.Prefix, doc.Number),
		Source:             "apidian",
		EntryType:          journals.EntryAutomatic,
		VoucherType:        "ND",
		SourceDocumentID:   doc.ID,
		SourceDocumentType: journals.SourceND,
		Lines:              lines,
	}, nil
}

// fromAdjustmentNote traduce una NA (Nota de Ajuste al DS) confirmada a un PostRequest contable.
// La NA revierte total o parcialmente el asiento del DS original.
// Es el espejo exacto de fromSupportDocument: los débitos pasan a crédito y viceversa.
//
//	220505 Proveedores nacionales  → Débito  (bruto − retenciones; NIT del proveedor)
//	236505/236540/236560           → Débito  (reverso retenciones, si las había)
//	expenseAccountCode             → Crédito (subtotal sin IVA)
//	135530 IVA descontable         → Crédito (IVA, si > 0)
func fromAdjustmentNote(doc *documents.Document, companyID uuid.UUID, expenseAccountCode string, rateX10000 int64, currency string) (*journals.PostRequest, error) {
	rawGross := doc.Totals.TaxInclusiveCents
	rawSubtotal := doc.Totals.LineExtensionCents

	gross := toCOP(rawGross, rateX10000)
	subtotal := toCOP(rawSubtotal, rateX10000)
	iva := gross - subtotal

	if gross <= 0 {
		return nil, fmt.Errorf("accounting mapper: NA %s tiene total <= 0", doc.ID)
	}

	var vendorNIT string
	if doc.Vendor != nil {
		vendorNIT = doc.Vendor.Identification.Number
	}
	cur := fCur(currency)

	var withholdingTotal int64
	for _, w := range doc.WithholdingTaxes {
		withholdingTotal += toCOP(w.TaxAmountCents, rateX10000)
	}

	lines := []journals.LineRequest{
		{
			AccountCode:     "220505",
			Debit:           gross - withholdingTotal,
			ThirdPartyNIT:   vendorNIT,
			Description:     fmt.Sprintf("NA %s%d — reverso proveedor", doc.Prefix, doc.Number),
			ForeignAmount:   fAmt(rawGross, currency),
			ForeignCurrency: cur,
		},
	}

	for _, w := range doc.WithholdingTaxes {
		acc, ok := withholdingPayableAccount(w.TypeCode)
		if !ok || w.TaxAmountCents == 0 {
			continue
		}
		lines = append(lines, journals.LineRequest{
			AccountCode: acc,
			Debit:       toCOP(w.TaxAmountCents, rateX10000),
			Description: fmt.Sprintf("NA %s%d — reverso %s", doc.Prefix, doc.Number, w.TypeName),
		})
	}

	lines = append(lines, journals.LineRequest{
		AccountCode:     expenseAccountCode,
		Credit:          subtotal,
		Description:     fmt.Sprintf("NA %s%d — reverso gasto/costo", doc.Prefix, doc.Number),
		ForeignAmount:   fAmt(rawSubtotal, currency),
		ForeignCurrency: cur,
	})

	if iva > 0 {
		rawIVA := rawGross - rawSubtotal
		lines = append(lines, journals.LineRequest{
			AccountCode:     "135530",
			Credit:          iva,
			Description:     fmt.Sprintf("NA %s%d — reverso IVA", doc.Prefix, doc.Number),
			ForeignAmount:   fAmt(rawIVA, currency),
			ForeignCurrency: cur,
		})
	}

	return &journals.PostRequest{
		CompanyID:          companyID,
		Date:               doc.IssueDate,
		Description:        fmt.Sprintf("NA %s%d confirmada", doc.Prefix, doc.Number),
		Source:             "apidian",
		EntryType:          journals.EntryAutomatic,
		VoucherType:        "NA",
		SourceDocumentID:   doc.ID,
		SourceDocumentType: journals.SourceNA,
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
		return "240802", true // IVA retenido por pagar (2408 IVA × pagar → 240802)
	case "06":
		return "236540", true // Retefuente por pagar — Compras (genérico; Fase 2: cuenta por concepto)
	case "07":
		return "2368", true // ICA retenido por pagar (PUC 2368 = "Impuesto de industria y comercio retenido")
	default:
		return "", false
	}
}
