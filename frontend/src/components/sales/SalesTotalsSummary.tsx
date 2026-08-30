import { formatCOP } from "../../lib/currency";
import { salesLinesBreakdown } from "./SalesLineItemsEditor";

interface SalesTotalsSummaryProps {
  lines: { quantity: number; unit_price: number; discount: number; tax_rate: number }[];
}

// Mismo look que TotalsSummary.tsx (factura electrónica) -- caja con borde, Subtotal/IVA/Total --
// pero para venta/cotización/compra, que usan pesos directo (SalesLineInput/PurchaseLineInput,
// sin desglose por tipo de impuesto DIAN) en vez de centavos.
export function SalesTotalsSummary({ lines }: SalesTotalsSummaryProps) {
  const { subtotal, tax, total } = salesLinesBreakdown(lines);

  return (
    <div className="flex flex-col gap-1 rounded border border-(--border-color) bg-(--bg-primary) p-3 text-xs">
      <div className="flex justify-between">
        <span className="text-(--text-secondary)">Subtotal</span>
        <span className="font-mono text-(--text-primary)">{formatCOP.format(subtotal)}</span>
      </div>
      {tax > 0 && (
        <div className="flex justify-between">
          <span className="text-(--text-secondary)">IVA</span>
          <span className="font-mono text-(--text-primary)">{formatCOP.format(tax)}</span>
        </div>
      )}
      <div className="flex justify-between border-t border-(--border-color) pt-1 font-semibold">
        <span className="text-(--text-primary)">Total</span>
        <span className="font-mono text-(--text-primary)">{formatCOP.format(total)}</span>
      </div>
    </div>
  );
}
