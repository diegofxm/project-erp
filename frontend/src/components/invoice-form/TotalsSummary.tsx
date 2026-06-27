import { listTaxTypes } from "../../lib/catalogs";
import { formatCOP } from "../../lib/currency";
import { previewTotals } from "../../lib/invoiceMath";
import { useCatalog } from "../../lib/useCatalog";
import type { DocumentLineInput } from "../../lib/types";

interface TotalsSummaryProps {
  lines: DocumentLineInput[];
}

// Subtotal/impuestos por tipo/total, recalculado en vivo desde las líneas actuales — vista
// previa con la misma fórmula que el servidor (ver lib/invoiceMath.ts), nunca la fuente de
// verdad final.
export function TotalsSummary({ lines }: TotalsSummaryProps) {
  const { data: taxTypes } = useCatalog(listTaxTypes);
  const totals = previewTotals(lines);

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
        <span className="text-(--text-primary)">Total</span>
        <span className="font-mono text-(--text-primary)">{formatCOP.format(totals.payableCents / 100)}</span>
      </div>
    </div>
  );
}
