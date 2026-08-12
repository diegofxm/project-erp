import { useEffect, useMemo, useState } from "react";
import { History, SlidersHorizontal, ArrowRightLeft, Trash2 } from "lucide-react";
import { createMovement, deleteMovement, listMovements } from "../lib/inventory";
import { listProducts } from "../lib/products";
import { listWarehouses } from "../lib/warehouses";
import { ApiError } from "../lib/apiClient";
import { useConfirm } from "../context/ConfirmContext";
import { useToast } from "../context/ToastContext";
import type { Movement, MovementType, Product, Warehouse } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { InfoTip } from "../components/ui/InfoTip";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { StatusPill, type StatusTone } from "../components/ui/StatusPill";
import { AdjustStockModal } from "../components/inventory/AdjustStockModal";
import { TransferStockModal } from "../components/inventory/TransferStockModal";

const TYPE_LABEL: Record<MovementType, string> = {
  entry: "Entrada", exit: "Salida", transfer: "Traslado", adjust: "Ajuste",
};
const TYPE_TONE: Record<MovementType, StatusTone> = {
  entry: "success", exit: "danger", transfer: "info", adjust: "neutral",
};

export function InventoryMovementsPage() {
  const [movements, setMovements] = useState<Movement[] | null>(null);
  const [products, setProducts] = useState<Product[]>([]);
  const [warehouses, setWarehouses] = useState<Warehouse[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [showAdjust, setShowAdjust] = useState(false);
  const [showTransfer, setShowTransfer] = useState(false);
  const toast = useToast();
  const confirmDialog = useConfirm();

  function refresh() {
    listMovements()
      .then(setMovements)
      .catch((err) => setError(err instanceof ApiError ? err.message : "No se pudo cargar el historial de movimientos"));
  }

  useEffect(() => {
    refresh();
    listProducts().then(setProducts).catch(() => setProducts([]));
    listWarehouses().then(setWarehouses).catch(() => setWarehouses([]));
  }, []);

  const productName = useMemo(() => {
    const map = new Map(products.map((p) => [p.id, p.name]));
    return (id: string) => map.get(id) ?? "—";
  }, [products]);

  const warehouseName = useMemo(() => {
    const map = new Map(warehouses.map((w) => [w.id, w.name]));
    return (id: string) => map.get(id) ?? "—";
  }, [warehouses]);

  async function handleAdjust(payload: { product_id: string; warehouse_id: string; type: MovementType; quantity: number; description: string }) {
    await createMovement(payload);
    toast.success("Movimiento registrado.");
    setShowAdjust(false);
    refresh();
  }

  async function handleTransfer(payload: { product_id: string; warehouse_id: string; to_warehouse_id: string; quantity: number; description: string }) {
    await createMovement({ ...payload, type: "transfer" });
    toast.success("Traslado registrado.");
    setShowTransfer(false);
    refresh();
  }

  async function handleDelete(m: Movement) {
    const extra = m.type === "transfer" ? " Se eliminarán las dos entradas del traslado (origen y destino)." : "";
    if (!(await confirmDialog(
      `¿Eliminar el movimiento ${m.number}? Esto revierte su efecto sobre el stock.${extra}`,
      { tone: "danger" }
    ))) return;
    try {
      await deleteMovement(m.id);
      toast.success("Movimiento eliminado.");
      refresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo eliminar el movimiento");
    }
  }

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Inventario", to: "/inventory" }, { label: "Movimientos" }]} />
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
          <History className="h-4 w-4 shrink-0 text-(--accent-primary)" />
          Movimientos
          <InfoTip>
            Historial completo de inventario: <strong>entradas</strong> y <strong>salidas</strong> automáticas
            (al recibir una compra o confirmar una venta), <strong>ajustes</strong> manuales (ej. conteo físico),
            y <strong>traslados</strong> entre bodegas. Trasladar requiere al menos dos bodegas activas.
          </InfoTip>
        </h1>
        <div className="flex items-center gap-1.5">
          <Button type="button" variant="secondary" icon={<ArrowRightLeft className="h-3.5 w-3.5" />} onClick={() => setShowTransfer(true)} disabled={warehouses.length < 2}>
            Trasladar
          </Button>
          <Button type="button" variant="secondary" icon={<SlidersHorizontal className="h-3.5 w-3.5" />} onClick={() => setShowAdjust(true)}>
            Ajuste manual
          </Button>
        </div>
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {movements === null ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spinner className="h-5 w-5 text-(--text-muted)" />
        </div>
      ) : movements.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">Todavía no hay movimientos registrados.</p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Folio</th>
                <th className="px-3 py-2 font-medium">Fecha</th>
                <th className="px-3 py-2 font-medium">Producto</th>
                <th className="px-3 py-2 font-medium">Bodega</th>
                <th className="px-3 py-2 font-medium">Tipo</th>
                <th className="px-3 py-2 font-medium">Cantidad</th>
                <th className="px-3 py-2 font-medium">Referencia</th>
                <th className="px-3 py-2 font-medium">Descripción</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {movements.map((m, i) => (
                <tr key={m.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 font-mono text-(--text-primary)">{m.number}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">{new Date(m.created_at).toLocaleString("es-CO")}</td>
                  <td className="px-3 py-2 text-(--text-primary)">{productName(m.product_id)}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">{warehouseName(m.warehouse_id)}</td>
                  <td className="px-3 py-2"><StatusPill tone={TYPE_TONE[m.type]} label={TYPE_LABEL[m.type]} /></td>
                  <td className="px-3 py-2 font-mono text-(--text-primary)">{m.quantity}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">{m.reference || "—"}</td>
                  <td className="px-3 py-2 text-(--text-secondary)">{m.description || "—"}</td>
                  <td className="px-3 py-2 text-right">
                    <button
                      type="button"
                      title="Eliminar movimiento"
                      aria-label={`Eliminar movimiento ${m.number}`}
                      onClick={() => handleDelete(m)}
                      className="rounded p-1 text-(--color-danger) transition-colors hover:bg-(--bg-hover)"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {showAdjust && (
        <AdjustStockModal products={products} warehouses={warehouses} onSubmit={handleAdjust} onClose={() => setShowAdjust(false)} />
      )}
      {showTransfer && (
        <TransferStockModal products={products} warehouses={warehouses} onSubmit={handleTransfer} onClose={() => setShowTransfer(false)} />
      )}
    </div>
  );
}
