import { useEffect, useMemo, useState } from "react";
import { Boxes, AlertTriangle } from "lucide-react";
import { listStock } from "../lib/inventory";
import { listProducts } from "../lib/products";
import { listWarehouses } from "../lib/warehouses";
import { ApiError } from "../lib/apiClient";
import type { Product, StockEntry, Warehouse } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";

export function InventoryStockPage() {
  const [stock, setStock] = useState<StockEntry[] | null>(null);
  const [products, setProducts] = useState<Product[]>([]);
  const [warehouses, setWarehouses] = useState<Warehouse[]>([]);
  const [warehouseFilter, setWarehouseFilter] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listStock()
      .then(setStock)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar el stock"));
    listProducts().then(setProducts).catch(() => setProducts([]));
    listWarehouses().then(setWarehouses).catch(() => setWarehouses([]));
  }, []);

  const productMap = useMemo(() => new Map(products.map((p) => [p.id, p])), [products]);
  const warehouseMap = useMemo(() => new Map(warehouses.map((w) => [w.id, w])), [warehouses]);

  const rows = useMemo(() => {
    if (!stock) return [];
    return stock
      .filter((e) => !warehouseFilter || e.warehouse_id === warehouseFilter)
      .map((e) => {
        const product = productMap.get(e.product_id);
        const minStock = product?.min_stock ?? 0;
        return { entry: e, product, warehouse: warehouseMap.get(e.warehouse_id), low: minStock > 0 && e.quantity < minStock };
      })
      .sort((a, b) => (a.product?.name ?? "").localeCompare(b.product?.name ?? ""));
  }, [stock, warehouseFilter, productMap, warehouseMap]);

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Inventario", to: "/inventory" }, { label: "Existencias" }]} />
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <Boxes className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Existencias
        </h1>
        <select
          value={warehouseFilter}
          onChange={(e) => setWarehouseFilter(e.target.value)}
          className="rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1 text-xs text-(--text-primary) transition-colors"
        >
          <option value="">Todas las bodegas</option>
          {warehouses.map((w) => (
            <option key={w.id} value={w.id}>{w.name}</option>
          ))}
        </select>
      </div>

      {/* El aviso agregado de stock bajo vive en la campana de notificaciones (ver
          shared/notification StockPort) — acá solo queda la marca por fila, útil en contexto
          al mirar la tabla, no una notificación duplicada. */}

      {error && <Banner tone="danger">{error}</Banner>}

      {stock === null ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : rows.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">Todavía no hay movimientos de inventario — el stock aparece acá en cuanto confirmes una venta o recibas una compra.</p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Producto</th>
                <th className="px-3 py-2 font-medium">Código</th>
                <th className="px-3 py-2 font-medium">Bodega</th>
                <th className="px-3 py-2 font-medium">Cantidad</th>
                <th className="px-3 py-2 font-medium">Mínimo</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {rows.map((r, i) => (
                <tr key={r.entry.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 text-(--text-primary)">{r.product?.name ?? "—"}</td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">{r.product?.code ?? "—"}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">{r.warehouse?.name ?? "—"}</td>
                  <td className={`px-3 py-2 font-mono font-semibold ${r.low ? "text-(--color-danger-text)" : "text-(--text-primary)"}`}>{r.entry.quantity}</td>
                  <td className="px-3 py-2 font-mono text-(--text-secondary)">{r.product?.min_stock || "—"}</td>
                  <td className="px-3 py-2">
                    {r.low && (
                      <span className="inline-flex items-center gap-1 rounded bg-(--color-danger-bg) px-1.5 py-0.5 text-[10px] font-medium text-(--color-danger-text)">
                        <AlertTriangle className="h-2.5 w-2.5" /> Stock bajo
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
