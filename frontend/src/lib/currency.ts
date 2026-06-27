// *_cents siempre se muestra/edita en pesos en la UI, convertido en el borde (ver
// ProductForm.centsToAmount, mismo criterio aquí) — compartido por los componentes nuevos de
// Factura Electrónica (LineItemsEditor, TotalsSummary, InvoicesPage).
export const formatCOP = new Intl.NumberFormat("es-CO", { style: "currency", currency: "COP", maximumFractionDigits: 2 });

export function centsToAmount(cents: number | undefined): string {
  return cents ? (cents / 100).toString() : "";
}

export function amountToCents(amount: string): number {
  return Math.round(Number(amount || 0) * 100);
}
