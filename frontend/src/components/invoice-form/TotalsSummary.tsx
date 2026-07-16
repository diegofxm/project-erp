import { listTaxTypes } from "../../lib/catalogs";
import { formatCOP } from "../../lib/currency";
import { previewTotals } from "../../lib/invoiceMath";
import { useCatalog } from "../../lib/useCatalog";
import type { DocumentLineInput, Tax } from "../../lib/types";

interface TotalsSummaryProps {
  lines: DocumentLineInput[];
  withholdingTaxes?: Tax[];
}

// Subtotal/impuestos por tipo/total, recalculado en vivo desde las líneas actuales — vista
// previa con la misma fórmula que el servidor (ver lib/invoiceMath.ts), nunca la fuente de
// verdad final. withholdingTaxes: retenciones del Documento Soporte — se restan del total.
export function TotalsSummary({ lines, withholdingTaxes }: TotalsSummaryProps) {
  const { data: taxTypes } = useCatalog(listTaxTypes);
  const totals = previewTotals(lines);
  const withholdingTotal = withholdingTaxes?.reduce((s, t) => s + t.tax_amount_cents, 0) ?? 0;
  const netPayable = totals.payableCents - withholdingTotal;

  return (
    <div className="flex flex-col gap-1 rounded border border-(--border-color) bg-(--bg-primary) p-3 text-xs">
      <div className="flex justify-between">
        <span className="text-(--text-secondary)">Subtotal</span>
        <span className="font-mono text-(--text-primary)">{formatCOP.format(totals.lineExtensionCents / 100)}</span>
      </div>
      {totals.taxesByType.map((t) => (
        <div key={t.typeCode} className="flex justify-between">
          <span className="text-(--text-secondary)">{taxTypes.find((ty) => ty.code === t.typeCode)?.name ?? t.typeCode}</span>
          <span className="font-mono text-(--text-primary)">{formatCOP.format(t.taxAmountCents / 100)}</span>
        </div>
      ))}
      <div className="flex justify-between border-t border-(--border-color) pt-1 font-semibold">
        <span className="text-(--text-primary)">{withholdingTaxes && withholdingTaxes.length > 0 ? "Total bruto" : "Total"}</span>
        <span className="font-mono text-(--text-primary)">{formatCOP.format(totals.payableCents / 100)}</span>
      </div>
      {withholdingTaxes && withholdingTaxes.length > 0 && (
        <>
          {withholdingTaxes.map((t, i) => (
            <div key={i} className="flex justify-between">
              <span className="text-(--text-secondary)">{t.type_name} ({t.type_code})</span>
              <span className="font-mono text-(--danger)">− {formatCOP.format(t.tax_amount_cents / 100)}</span>
            </div>
          ))}
          <div className="flex justify-between border-t border-(--border-color) pt-1 font-semibold">
            <span className="text-(--text-primary)">A pagar</span>
            <span className="font-mono text-(--text-primary)">{formatCOP.format(netPayable / 100)}</span>
          </div>
        </>
      )}
    </div>
  );
}
