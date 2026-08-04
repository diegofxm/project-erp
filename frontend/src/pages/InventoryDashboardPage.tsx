import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { AlertTriangle, ArrowRight, ArrowRightLeft, Boxes, Warehouse as WarehouseIcon } from "lucide-react";
import { listStock, listMovements } from "../lib/inventory";
import { listWarehouses } from "../lib/warehouses";
import { listProducts } from "../lib/products";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import { useAuth } from "../context/AuthContext";
import type { Movement, MovementType, Product, StockEntry, Warehouse } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Card } from "../components/ui/Card";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { StatusPill, type StatusTone } from "../components/ui/StatusPill";

const TYPE_LABEL: Record<MovementType, string> = { entry: "Entrada", exit: "Salida", transfer: "Traslado", adjust: "Ajuste" };
const TYPE_TONE: Record<MovementType, StatusTone> = { entry: "success", exit: "danger", transfer: "info", adjust: "neutral" };

function money(v: number): string {
  return formatCOP.format(v);
}

function monthKey(d: Date): string {
  return `${d.getFullYear()}-${d.getMonth()}`;
}

export function InventoryDashboardPage() {
  const { activeCompany } = useAuth();
  const navigate = useNavigate();

  const [stock, setStock] = useState<StockEntry[] | null>(null);
  const [movements, setMovements] = useState<Movement[] | null>(null);
  const [warehouses, setWarehouses] = useState<Warehouse[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([listStock(), listMovements(), listWarehouses(), listProducts()])
      .then(([s, m, w, p]) => { setStock(s); setMovements(m); setWarehouses(w); setProducts(p); })
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar el panel de inventario"));
  }, []);

  const productMap = useMemo(() => new Map(products.map((p) => [p.id, p])), [products]);
  const warehouseName = useMemo(() => {
    const map = new Map(warehouses.map((w) => [w.id, w.name]));
    return (id: string) => map.get(id) ?? "—";
  }, [warehouses]);

  const stockValue = useMemo(() => {
    if (!stock) return 0;
    return stock.reduce((sum, e) => sum + e.quantity * (productMap.get(e.product_id)?.base_price ?? 0), 0);
  }, [stock, productMap]);

  const lowStock = useMemo(() => {
    if (!stock) return [];
    return stock.filter((e) => {
      const min = productMap.get(e.product_id)?.min_stock ?? 0;
      return min > 0 && e.quantity < min;
    });
  }, [stock, productMap]);

  const now = new Date();
  const curKey = monthKey(now);
  const movementsThisMonth = useMemo(() => (movements ?? []).filter((m) => monthKey(new Date(m.created_at)) === curKey), [movements, curKey]);
  const transfersThisMonth = useMemo(() => movementsThisMonth.filter((m) => m.type === "transfer"), [movementsThisMonth]);
  const adjustsThisMonth = useMemo(() => movementsThisMonth.filter((m) => m.type === "adjust"), [movementsThisMonth]);

  const activeWarehouses = useMemo(() => warehouses.filter((w) => w.is_active), [warehouses]);

  const recent = useMemo(() => [...(movements ?? [])].slice(0, 5), [movements]);

  const loading = stock === null || movements === null;

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Inventario" }]} />
      <div className="mb-3">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <Boxes className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Panel de inventario
        </h1>
        <p className="mt-0.5 text-xs text-(--text-secondary)">{activeCompany?.trade_name || activeCompany?.business_name}</p>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {loading ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : (
        <>
          <div className="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Valor de inventario <Boxes className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">{money(stockValue)}</div>
              <div className="mt-1 text-xs text-(--text-secondary)">estimado a precio de venta</div>
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Stock bajo <AlertTriangle className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className={`mt-2 text-xl font-bold tabular-nums ${lowStock.length > 0 ? "text-(--color-danger-text)" : "text-(--text-primary)"}`}>{lowStock.length}</div>
              <div className="mt-1 text-xs text-(--text-secondary)">productos bajo el mínimo configurado</div>
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Movimientos del mes <ArrowRightLeft className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">{movementsThisMonth.length}</div>
              <div className="mt-1 text-xs text-(--text-secondary)">{transfersThisMonth.length} traslados · {adjustsThisMonth.length} ajustes</div>
            </Card>
            <Card className="p-4">
              <div className="flex items-center justify-between text-xs font-semibold uppercase tracking-wider text-(--text-muted)">
                Bodegas activas <WarehouseIcon className="h-3.5 w-3.5 text-(--accent-primary)" />
              </div>
              <div className="mt-2 text-xl font-bold tabular-nums text-(--text-primary)">{activeWarehouses.length}</div>
              <div className="mt-1 text-xs text-(--text-secondary)">de {warehouses.length} registradas</div>
            </Card>
          </div>

          <div className="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
            <Card className="flex flex-col p-4">
              <div className="text-xs font-semibold text-(--text-primary)">Existencias</div>
              <div className={`mt-2 text-lg font-bold tabular-nums ${lowStock.length > 0 ? "text-(--color-danger-text)" : "text-(--text-primary)"}`}>{lowStock.length}</div>
              <span className="mt-0.5 text-xs text-(--text-muted)">con stock bajo el mínimo</span>
              <div className="mt-3 flex flex-1 items-end justify-end border-t border-(--border-light) pt-2 text-xs">
                <button type="button" onClick={() => navigate("/inventory/stock")} className="flex items-center gap-1 text-(--text-secondary) hover:text-(--accent-primary)">Ver existencias <ArrowRight className="h-3 w-3" /></button>
              </div>
            </Card>
            <Card className="flex flex-col p-4">
              <div className="text-xs font-semibold text-(--text-primary)">Movimientos</div>
              <div className="mt-2 text-lg font-bold tabular-nums text-(--text-primary)">{movementsThisMonth.length}</div>
              <span className="mt-0.5 text-xs text-(--text-muted)">este mes</span>
              <div className="mt-3 flex flex-1 items-end justify-between gap-2 border-t border-(--border-light) pt-2 text-xs">
                <button type="button" onClick={() => navigate("/inventory/movements")} className="font-medium text-(--accent-primary) hover:underline">Nuevo traslado</button>
                <button type="button" onClick={() => navigate("/inventory/movements")} className="flex items-center gap-1 text-(--text-secondary) hover:text-(--accent-primary)">Ver <ArrowRight className="h-3 w-3" /></button>
              </div>
            </Card>
            <Card className="flex flex-col p-4">
              <div className="text-xs font-semibold text-(--text-primary)">Bodegas</div>
              <div className="mt-2 text-lg font-bold tabular-nums text-(--text-primary)">{activeWarehouses.length}</div>
              <span className="mt-0.5 text-xs text-(--text-muted)">activas</span>
              <div className="mt-3 flex flex-1 items-end justify-end border-t border-(--border-light) pt-2 text-xs">
                <button type="button" onClick={() => navigate("/inventory/warehouses")} className="flex items-center gap-1 text-(--text-secondary) hover:text-(--accent-primary)">Ver <ArrowRight className="h-3 w-3" /></button>
              </div>
            </Card>
          </div>

          <Card className="p-4">
            <div className="mb-2 flex items-center justify-between">
              <h2 className="text-xs font-semibold uppercase tracking-wider text-(--text-muted)">Actividad reciente</h2>
              <button type="button" onClick={() => navigate("/inventory/movements")} className="flex items-center gap-1 text-xs text-(--accent-primary) hover:underline">
                Ver todos <ArrowRight className="h-3 w-3" />
              </button>
            </div>
            {recent.length === 0 ? (
              <p className="py-4 text-center text-xs text-(--text-muted)">Aún no hay movimientos registrados</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="text-(--text-secondary)">
                    <tr>
                      <th className="py-1.5 pr-3 font-medium">Fecha</th>
                      <th className="py-1.5 pr-3 font-medium">Producto</th>
                      <th className="py-1.5 pr-3 font-medium">Bodega</th>
                      <th className="py-1.5 pr-3 font-medium">Tipo</th>
                      <th className="py-1.5 text-right font-medium">Cantidad</th>
                    </tr>
                  </thead>
                  <tbody>
                    {recent.map((m, i) => (
                      <tr key={m.id} className={`border-t border-(--border-light) ${i % 2 === 1 ? "bg-(--bg-secondary)" : ""}`}>
                        <td className="py-1.5 pr-3 text-(--text-secondary)">{new Date(m.created_at).toLocaleDateString("es-CO")}</td>
                        <td className="py-1.5 pr-3 text-(--text-primary)">{productMap.get(m.product_id)?.name ?? "—"}</td>
                        <td className="py-1.5 pr-3 text-(--text-secondary)">{warehouseName(m.warehouse_id)}</td>
                        <td className="py-1.5 pr-3"><StatusPill tone={TYPE_TONE[m.type]} label={TYPE_LABEL[m.type]} /></td>
                        <td className="py-1.5 text-right font-mono text-(--text-primary)">{m.quantity}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Card>
        </>
      )}
    </div>
  );
}
