import { useState } from "react";
import { X, SlidersHorizontal } from "lucide-react";
import type { MovementType, Product, Warehouse } from "../../lib/types";
import { Button } from "../ui/Button";
import { Combobox } from "../ui/Combobox";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";
import { Banner } from "../ui/Banner";

interface Props {
  products: Product[];
  warehouses: Warehouse[];
  onSubmit: (payload: { product_id: string; warehouse_id: string; type: MovementType; quantity: number; description: string }) => Promise<void>;
  onClose: () => void;
}

const TYPE_LABEL: Record<"entry" | "exit", string> = {
  entry: "Entrada (suma stock)",
  exit: "Salida (resta stock)",
};

export function AdjustStockModal({ products, warehouses, onSubmit, onClose }: Props) {
  const [productId, setProductId] = useState("");
  const [warehouseId, setWarehouseId] = useState(warehouses.find((w) => w.is_default)?.id ?? warehouses[0]?.id ?? "");
  const [type, setType] = useState<"entry" | "exit" | "adjust">("adjust");
  const [quantity, setQuantity] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const qtyNum = Number(quantity || 0);
  const valid = !!productId && !!warehouseId && qtyNum > 0 && description.trim() !== "";

  async function handleSubmit() {
    if (!valid) return;
    setError(null);
    setSaving(true);
    try {
      await onSubmit({ product_id: productId, warehouse_id: warehouseId, type, quantity: qtyNum, description });
    } catch {
      setError("No se pudo registrar el movimiento.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div className="w-full max-w-md rounded-lg border border-(--border-color) bg-(--bg-primary) p-5 shadow-xl" onClick={(e) => e.stopPropagation()}>
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <SlidersHorizontal className="h-4 w-4 text-(--accent-primary)" />
            <h2 className="text-sm font-semibold text-(--text-primary)">Ajuste manual de inventario</h2>
          </div>
          <button type="button" onClick={onClose} className="text-(--text-muted) hover:text-(--text-primary) transition-colors">
            <X className="h-4 w-4" />
          </button>
        </div>

        {error && <Banner tone="danger">{error}</Banner>}

        <div className="space-y-3">
          <Combobox
            label="Producto"
            value={productId}
            onChange={setProductId}
            options={products.filter((p) => !p.is_service).map((p) => ({ value: p.id, label: `${p.code} — ${p.name}` }))}
            placeholder="Buscar producto…"
          />
          <Select label="Bodega" value={warehouseId} onChange={(e) => setWarehouseId(e.target.value)}>
            {warehouses.map((w) => (
              <option key={w.id} value={w.id}>{w.name}{w.is_default ? " (por defecto)" : ""}</option>
            ))}
          </Select>
          <Select label="Tipo de movimiento" value={type} onChange={(e) => setType(e.target.value as "entry" | "exit" | "adjust")}>
            <option value="entry">{TYPE_LABEL.entry}</option>
            <option value="exit">{TYPE_LABEL.exit}</option>
            <option value="adjust">Ajuste (conteo físico — suma la diferencia)</option>
          </Select>
          <Input label="Cantidad" type="number" min="0" step="0.01" value={quantity} onChange={(e) => setQuantity(e.target.value)} />
          <Input label="Motivo" required value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Ej. Conteo físico de fin de mes, producto dañado…" />
        </div>

        <div className="mt-5 flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={onClose}>Cancelar</Button>
          <Button type="button" disabled={!valid} loading={saving} onClick={handleSubmit}>Registrar</Button>
        </div>
      </div>
    </div>
  );
}
