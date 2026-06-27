// Réplica de la aritmética que internal/documents/lines.go (linesFromInput) hace en el
// servidor — SOLO para la vista previa en el formulario, mientras el usuario escribe. La
// fuente de verdad siempre es la respuesta del servidor después de guardar (ver
// docs/apidian-architecture.md sección 9.37): si esta función y el servidor alguna vez
// calculan algo distinto, gana el servidor, no esta vista previa.
import type { DocumentLineInput } from "./types";

export interface LinePreview {
  lineExtensionCents: number;
  taxAmountCents: number;
  totalCents: number;
}

export function previewLine(line: Pick<DocumentLineInput, "quantity" | "unit_price_cents" | "tax_percent">): LinePreview {
  const lineExtensionCents = Math.round(line.quantity * line.unit_price_cents);
  const taxAmountCents = line.tax_percent ? Math.round((lineExtensionCents * line.tax_percent) / 100) : 0;
  return { lineExtensionCents, taxAmountCents, totalCents: lineExtensionCents + taxAmountCents };
}

export interface TotalsPreview {
  lineExtensionCents: number;
  taxAmountCents: number;
  payableCents: number;
  taxesByType: { typeCode: string; taxableAmountCents: number; taxAmountCents: number }[];
}

export function previewTotals(lines: DocumentLineInput[]): TotalsPreview {
  let lineExtensionCents = 0;
  let taxAmountCents = 0;
  const byType = new Map<string, { taxableAmountCents: number; taxAmountCents: number }>();

  for (const line of lines) {
    const preview = previewLine(line);
    lineExtensionCents += preview.lineExtensionCents;
    taxAmountCents += preview.taxAmountCents;
    if (line.tax_type_code) {
      const current = byType.get(line.tax_type_code) ?? { taxableAmountCents: 0, taxAmountCents: 0 };
      current.taxableAmountCents += preview.lineExtensionCents;
      current.taxAmountCents += preview.taxAmountCents;
      byType.set(line.tax_type_code, current);
    }
  }

  return {
    lineExtensionCents,
    taxAmountCents,
    payableCents: lineExtensionCents + taxAmountCents,
    taxesByType: [...byType.entries()].map(([typeCode, v]) => ({ typeCode, ...v })),
  };
}
