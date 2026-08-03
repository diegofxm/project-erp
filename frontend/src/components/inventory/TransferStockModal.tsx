import { useState } from "react";
import { X, ArrowRightLeft } from "lucide-react";
import type { Product, Warehouse } from "../../lib/types";
import { Button } from "../ui/Button";
import { Combobox } from "../ui/Combobox";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";
import { Banner } from "../ui/Banner";

interface Props {
  products: Product[];
  warehouses: Warehouse[];
  onSubmit: (payload: { product_id: string; warehouse_id: string; to_warehouse_id: string; quantity: number; description: string }) => Promise<void>;
  onClose: () => void;
}

export function TransferStockModal({ products, warehouses, onSubmit, onClose }: Props) {
  const [productId, setProductId] = useState("");
  const [fromId, setFromId] = useState(warehouses.find((w) => w.is_default)?.id ?? warehouses[0]?.id ?? "");
  const [toId, setToId] = useState("");
  const [quantity, setQuantity] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const qtyNum = Number(quantity || 0);
  const valid = !!productId && !!fromId && !!toId && fromId !== toId && qtyNum > 0;

  async function handleSubmit() {
    if (!valid) return;
    setError(null);
    setSaving(true);
    try {
      await onSubmit({ product_id: productId, warehouse_id: fromId, to_warehouse_id: toId, quantity: qtyNum, description });
    } catch {
      setError("No se pudo trasladar el stock — revisa que haya suficiente en la bodega de origen.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div className="w-full max-w-md rounded-lg border border-(--border-color) bg-(--bg-primary) p-5 shadow-xl" onClick={(e) => e.stopPropagation()}>
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <ArrowRightLeft className="h-4 w-4 text-(--accent-primary)" />
            <h2 className="text-sm font-semibold text-(--text-primary)">Trasladar entre bodegas</h2>
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
          <div className="grid grid-cols-2 gap-3">
            <Select label="Desde" value={fromId} onChange={(e) => setFromId(e.target.value)}>
              {warehouses.map((w) => (
                <option key={w.id} value={w.id}>{w.name}</option>
              ))}
            </Select>
            <Select label="Hacia" value={toId} onChange={(e) => setToId(e.target.value)}>
              <option value="">Elegir…</option>
              {warehouses.filter((w) => w.id !== fromId).map((w) => (
                <option key={w.id} value={w.id}>{w.name}</option>
              ))}
            </Select>
          </div>
          {fromId && toId && fromId === toId && (
            <p className="text-xs text-(--color-danger-text)">La bodega de origen y destino no pueden ser la misma.</p>
          )}
          <Input label="Cantidad" type="number" min="0" step="0.01" value={quantity} onChange={(e) => setQuantity(e.target.value)} />
          <Input label="Referencia (opcional)" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Ej. Reabastecimiento sucursal norte" />
        </div>

        <div className="mt-5 flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={onClose}>Cancelar</Button>
          <Button type="button" disabled={!valid} loading={saving} onClick={handleSubmit}>Trasladar</Button>
        </div>
      </div>
    </div>
  );
}
