import { useEffect, useState } from "react";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { listUnitMeasures } from "../../lib/catalogs";
import { formatCOP } from "../../lib/currency";
import { listProducts } from "../../lib/products";
import { useCatalog } from "../../lib/useCatalog";
import type { Product, SalesLineInput } from "../../lib/types";
import { Button } from "../ui/Button";
import { Combobox } from "../ui/Combobox";
import { Input } from "../ui/Input";

interface SalesLineItemsEditorProps {
  lines: SalesLineInput[];
  onChange: (lines: SalesLineInput[]) => void;
  disabled?: boolean;
}

interface DraftLine {
  productId: string;
  description: string;
  unitCode: string;
  quantity: string;
  unitPrice: string;
  discount: string;
  taxRate: string;
}

const EMPTY_DRAFT: DraftLine = { productId: "", description: "", unitCode: "", quantity: "1", unitPrice: "", discount: "", taxRate: "" };

function draftFromProduct(product: Product): DraftLine {
  return {
    productId: product.id,
    description: product.name,
    unitCode: product.unit_measure_code,
    quantity: "1",
    unitPrice: product.base_price.toString(),
    discount: "",
    taxRate: product.tax_rate?.toString() ?? "",
  };
}

function lineTotal(l: { quantity: number; unit_price: number; discount: number; tax_rate: number }) {
  const gross = l.quantity * l.unit_price;
  const subtotal = gross - (gross * l.discount) / 100;
  return subtotal + (subtotal * l.tax_rate) / 100;
}

// Mismo layout de línea que LineItemsEditor.tsx (factura electrónica): producto, descripción,
// unidad de medida, cantidad, precio. Se diferencia solo en lo que sales/ no necesita porque no
// es un documento fiscal DIAN: sin item_code/item_type_code (clasificación DIAN del ítem, se
// deriva del producto solo al generar la factura electrónica, ver electronic/application/
// from_sale.go) y con descuento por línea + impuesto como número libre (%) en vez del catálogo
// de tipos de impuesto DIAN.
export function SalesLineItemsEditor({ lines, onChange, disabled }: SalesLineItemsEditorProps) {
  const [products, setProducts] = useState<Product[]>([]);
  const [loadingProducts, setLoadingProducts] = useState(true);
  const { data: unitMeasures, loading: loadingUnitMeasures } = useCatalog(listUnitMeasures);
  const [draft, setDraft] = useState<DraftLine>(EMPTY_DRAFT);
  const [showForm, setShowForm] = useState(false);
  const [editingIdx, setEditingIdx] = useState<number | null>(null);

  useEffect(() => {
    listProducts().then(setProducts).catch(() => setProducts([])).finally(() => setLoadingProducts(false));
  }, []);

  const productOptions = products.map((p) => ({ value: p.id, label: `${p.code} — ${p.name}` }));

  function handleProductSelect(id: string) {
    if (!id) return;
    const product = products.find((p) => p.id === id);
    if (product) setDraft(draftFromProduct(product));
  }

  function handleStartEdit(index: number) {
    const l = lines[index];
    setEditingIdx(index);
    setDraft({
      productId: l.product_id,
      description: l.description,
      unitCode: l.unit_code,
      quantity: l.quantity.toString(),
      unitPrice: l.unit_price.toString(),
      discount: l.discount ? l.discount.toString() : "",
      taxRate: l.tax_rate.toString(),
    });
    setShowForm(true);
  }

  function handleSaveLine() {
    const line: SalesLineInput = {
      product_id: draft.productId,
      description: draft.description,
      unit_code: draft.unitCode,
      quantity: Number(draft.quantity || 0),
      unit_price: Number(draft.unitPrice || 0),
      discount: Number(draft.discount || 0),
      tax_rate: Number(draft.taxRate || 0),
    };
    if (editingIdx !== null) {
      onChange(lines.map((l, i) => (i === editingIdx ? line : l)));
    } else {
      onChange([...lines, line]);
    }
    setDraft(EMPTY_DRAFT);
    setShowForm(false);
    setEditingIdx(null);
  }

  function handleCancel() {
    setDraft(EMPTY_DRAFT);
    setShowForm(false);
    setEditingIdx(null);
  }

  function handleRemoveLine(index: number) {
    onChange(lines.filter((_, i) => i !== index));
    if (editingIdx === index) handleCancel();
  }

  const canAdd =
    draft.description.trim() !== "" &&
    draft.unitCode !== "" &&
    Number(draft.quantity) > 0 &&
    Number(draft.unitPrice) > 0;
  const draftPreview = lineTotal({
    quantity: Number(draft.quantity || 0),
    unit_price: Number(draft.unitPrice || 0),
    discount: Number(draft.discount || 0),
    tax_rate: Number(draft.taxRate || 0),
  });

  return (
    <div className="flex flex-col gap-3">
      {lines.length > 0 && (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Descripción</th>
                <th className="px-3 py-2 font-medium">Unidad</th>
                <th className="px-3 py-2 font-medium">Cant.</th>
                <th className="px-3 py-2 font-medium">Precio unitario</th>
                <th className="px-3 py-2 font-medium">Dto.</th>
                <th className="px-3 py-2 font-medium">Impuesto</th>
                <th className="px-3 py-2 font-medium">Total línea</th>
                {!disabled && <th className="px-3 py-2" />}
              </tr>
            </thead>
            <tbody>
              {lines.map((line, i) => {
                const isEditing = editingIdx === i;
                return (
                  <tr key={i} className={isEditing ? "bg-(--color-info-bg) outline-1 outline-(--color-info-border)" : i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                    <td className="px-3 py-2 text-(--text-primary)">{line.description}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{line.unit_code || "—"}</td>
                    <td className="px-3 py-2 font-mono text-(--text-secondary)">{line.quantity}</td>
                    <td className="px-3 py-2 font-mono text-(--text-secondary)">{formatCOP.format(line.unit_price)}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{line.discount ? `${line.discount}%` : "—"}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">{line.tax_rate ? `${line.tax_rate}%` : "—"}</td>
                    <td className="px-3 py-2 font-mono text-(--text-primary)">{formatCOP.format(lineTotal(line))}</td>
                    {!disabled && (
                      <td className="px-3 py-2">
                        <div className="flex items-center gap-1">
                          <button type="button" title="Editar línea" aria-label="Editar línea" onClick={() => handleStartEdit(i)} className="rounded p-1 text-(--text-muted) hover:bg-(--bg-hover) hover:text-(--accent-primary)">
                            <Pencil className="h-3.5 w-3.5" />
                          </button>
                          <button type="button" title="Quitar línea" aria-label="Quitar línea" onClick={() => handleRemoveLine(i)} className="rounded p-1 text-(--color-danger) hover:bg-(--bg-hover)">
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      </td>
                    )}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {lines.length === 0 && !showForm && (
        <p className="text-xs text-(--text-muted)">Aún no hay líneas — agrega al menos una para continuar.</p>
      )}

      {disabled ? null : !showForm ? (
        <Button type="button" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => { setEditingIdx(null); setShowForm(true); }}>
          Agregar línea
        </Button>
      ) : (
        <div className="rounded border border-(--border-color) bg-(--bg-secondary) p-3">
          <div className="grid grid-cols-12 gap-3">
            <div className="col-span-12 sm:col-span-5">
              <Combobox
                label="Buscar producto"
                value={draft.productId}
                onChange={handleProductSelect}
                options={productOptions}
                disabled={loadingProducts}
                placeholder={loadingProducts ? "Cargando…" : "Buscar por nombre…"}
              />
            </div>
            <div className="col-span-12 sm:col-span-7">
              <Input label="Descripción" required value={draft.description} onChange={(e) => setDraft({ ...draft, description: e.target.value })} />
            </div>
            <div className="col-span-12 sm:col-span-3">
              <Combobox
                label="Unidad de medida"
                disabled={loadingUnitMeasures}
                value={draft.unitCode}
                onChange={(unitCode) => setDraft({ ...draft, unitCode })}
                options={unitMeasures.map((u) => ({ value: u.code, label: `${u.code} — ${u.name}` }))}
                placeholder={loadingUnitMeasures ? "Cargando…" : "Buscar unidad…"}
              />
            </div>
            <div className="col-span-6 sm:col-span-3">
              <Input label="Cantidad" type="number" min="0" step="0.01" value={draft.quantity} onChange={(e) => setDraft({ ...draft, quantity: e.target.value })} />
            </div>
            <div className="col-span-6 sm:col-span-3">
              <Input label="Precio unitario (COP)" type="number" min="0" step="0.01" value={draft.unitPrice} onChange={(e) => setDraft({ ...draft, unitPrice: e.target.value })} />
            </div>
            <div className="col-span-6 sm:col-span-3">
              <Input label="Descuento (%)" type="number" min="0" max="100" step="0.01" value={draft.discount} onChange={(e) => setDraft({ ...draft, discount: e.target.value })} />
            </div>
            <div className="col-span-6 sm:col-span-3">
              <Input label="Impuesto (%)" type="number" min="0" step="0.01" value={draft.taxRate} onChange={(e) => setDraft({ ...draft, taxRate: e.target.value })} />
            </div>
            <div className="col-span-12 flex items-center justify-between pt-1">
              <span className="text-xs text-(--text-secondary)">
                Total de la línea: <span className="font-mono text-(--text-primary)">{formatCOP.format(draftPreview)}</span>
              </span>
              <div className="flex gap-2">
                <Button type="button" variant="ghost" onClick={handleCancel}>Cancelar</Button>
                <Button type="button" icon={editingIdx !== null ? <Pencil className="h-3.5 w-3.5" /> : <Plus className="h-3.5 w-3.5" />} disabled={!canAdd} onClick={handleSaveLine}>
                  {editingIdx !== null ? "Guardar cambios" : "Agregar"}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export function salesLinesTotal(lines: { quantity: number; unit_price: number; discount: number; tax_rate: number }[]) {
  return lines.reduce((sum, l) => sum + lineTotal(l), 0);
}

// salesLinesBreakdown -- mismos números que salesLinesTotal, pero separando subtotal/impuesto
// para el cuadro de totales (ver components/sales/SalesTotalsSummary.tsx), igual que
// TotalsSummary.tsx hace para factura electrónica.
export function salesLinesBreakdown(lines: { quantity: number; unit_price: number; discount: number; tax_rate: number }[]) {
  let subtotal = 0;
  let tax = 0;
  for (const l of lines) {
    const gross = l.quantity * l.unit_price;
    const lineSubtotal = gross - (gross * l.discount) / 100;
    subtotal += lineSubtotal;
    tax += (lineSubtotal * l.tax_rate) / 100;
  }
  return { subtotal, tax, total: subtotal + tax };
}
