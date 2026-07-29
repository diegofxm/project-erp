import { useEffect, useState } from "react";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { listTaxTypes, listUnitMeasures } from "../../lib/catalogs";
import { amountToCents, centsToAmount, formatCOP } from "../../lib/currency";
import { previewLine } from "../../lib/invoiceMath";
import { listProducts } from "../../lib/products";
import { useCatalog } from "../../lib/useCatalog";
import type { DocumentLineInput, Product } from "../../lib/types";
import { Button } from "../ui/Button";
import { Combobox } from "../ui/Combobox";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";

interface LineItemsEditorProps {
  lines: DocumentLineInput[];
  onChange: (lines: DocumentLineInput[]) => void;
}

interface DraftLine {
  productId: string;
  description: string;
  unitCode: string;
  quantity: string;
  unitPrice: string;
  itemCode: string;
  itemTypeCode: string;
  taxTypeCode: string;
  taxPercent: string;
}

const EMPTY_DRAFT: DraftLine = {
  productId: "",
  description: "",
  unitCode: "",
  quantity: "1",
  unitPrice: "",
  itemCode: "",
  itemTypeCode: "",
  taxTypeCode: "",
  taxPercent: "",
};

function draftFromProduct(product: Product): DraftLine {
  return {
    productId: product.id,
    description: product.name,
    unitCode: product.unit_measure_code,
    quantity: "1",
    unitPrice: product.base_price.toString(),
    itemCode: product.standard_code ?? "",
    itemTypeCode: product.standard_code_id ?? "",
    taxTypeCode: product.tax_scheme_code !== "ZZ" ? product.tax_scheme_code : "",
    taxPercent: product.tax_rate?.toString() ?? "",
  };
}

function draftFromLine(line: DocumentLineInput): DraftLine {
  return {
    productId: "",
    description: line.description,
    unitCode: line.unit_code,
    quantity: line.quantity.toString(),
    unitPrice: centsToAmount(line.unit_price_cents),
    itemCode: line.item_code ?? "",
    itemTypeCode: line.item_type_code ?? "",
    taxTypeCode: line.tax_type_code ?? "",
    taxPercent: line.tax_percent?.toString() ?? "",
  };
}

export function LineItemsEditor({ lines, onChange }: LineItemsEditorProps) {
  const [products, setProducts] = useState<Product[]>([]);
  const [loadingProducts, setLoadingProducts] = useState(true);
  const { data: unitMeasures, loading: loadingUnitMeasures } = useCatalog(listUnitMeasures);
  const { data: taxTypes, loading: loadingTaxTypes } = useCatalog(listTaxTypes);
  const [draft, setDraft] = useState<DraftLine>(EMPTY_DRAFT);
  const [showForm, setShowForm] = useState(false);
  const [editingIdx, setEditingIdx] = useState<number | null>(null);

  useEffect(() => {
    listProducts()
      .then(setProducts)
      .catch(() => setProducts([]))
      .finally(() => setLoadingProducts(false));
  }, []);

  const productOptions = products.map((p) => ({ value: p.id, label: `${p.code} — ${p.name}` }));

  function handleProductSelect(id: string) {
    if (!id) {
      setDraft((d) => ({ ...EMPTY_DRAFT, description: d.description }));
      return;
    }
    const product = products.find((p) => p.id === id);
    if (product) setDraft(draftFromProduct(product));
  }

  function handleStartEdit(index: number) {
    setEditingIdx(index);
    setDraft(draftFromLine(lines[index]));
    setShowForm(true);
  }

  function handleSaveLine() {
    const line: DocumentLineInput = {
      description: draft.description,
      quantity: Number(draft.quantity || 0),
      unit_code: draft.unitCode,
      unit_price_cents: amountToCents(draft.unitPrice),
      item_code: draft.itemCode || undefined,
      item_type_code: draft.itemTypeCode || undefined,
      tax_type_code: draft.taxTypeCode || undefined,
      tax_percent: draft.taxTypeCode ? Number(draft.taxPercent || 0) : undefined,
    };
    if (editingIdx !== null) {
      const updated = lines.map((l, i) => (i === editingIdx ? line : l));
      onChange(updated);
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
    if (editingIdx === index) {
      setDraft(EMPTY_DRAFT);
      setShowForm(false);
      setEditingIdx(null);
    }
  }

  const canAdd =
    draft.description.trim() !== "" &&
    draft.unitCode !== "" &&
    Number(draft.quantity) > 0 &&
    Number(draft.unitPrice) > 0;

  const draftPreview = previewLine({
    quantity: Number(draft.quantity || 0),
    unit_price_cents: amountToCents(draft.unitPrice),
    tax_percent: draft.taxTypeCode ? Number(draft.taxPercent || 0) : undefined,
  });

  return (
    <div className="flex flex-col gap-3">
      {/* Tabla de líneas agregadas */}
      {lines.length > 0 && (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">Descripción</th>
                <th className="px-3 py-2 font-medium">Cant.</th>
                <th className="px-3 py-2 font-medium">Precio unitario</th>
                <th className="px-3 py-2 font-medium">Impuesto</th>
                <th className="px-3 py-2 font-medium">Total línea</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {lines.map((line, i) => {
                const preview = previewLine(line);
                const isEditing = editingIdx === i;
                return (
                  <tr key={i} className={isEditing ? "bg-(--color-info-bg) outline-1 outline-(--color-info-border)" : i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                    <td className="px-3 py-2 text-(--text-primary)">{line.description}</td>
                    <td className="px-3 py-2 font-mono text-(--text-secondary)">{line.quantity}</td>
                    <td className="px-3 py-2 font-mono text-(--text-secondary)">{formatCOP.format(line.unit_price_cents / 100)}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">
                      {line.tax_type_code ? `${line.tax_type_code} (${line.tax_percent ?? 0}%)` : "—"}
                    </td>
                    <td className="px-3 py-2 font-mono text-(--text-primary)">{formatCOP.format(preview.totalCents / 100)}</td>
                    <td className="px-3 py-2">
                      <div className="flex items-center gap-1">
                        <button
                          type="button"
                          title="Editar línea"
                          onClick={() => handleStartEdit(i)}
                          className="rounded p-1 text-(--text-muted) hover:bg-(--bg-hover) hover:text-(--accent-primary)"
                        >
                          <Pencil className="h-3.5 w-3.5" />
                        </button>
                        <button
                          type="button"
                          title="Quitar línea"
                          onClick={() => handleRemoveLine(i)}
                          className="rounded p-1 text-(--color-danger) hover:bg-(--bg-hover)"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Estado vacío */}
      {lines.length === 0 && !showForm && (
        <p className="text-xs text-(--text-muted)">Aún no hay líneas — agrega al menos una para continuar.</p>
      )}

      {/* Botón o formulario */}
      {!showForm ? (
        <Button
          type="button"
          variant="secondary"
          icon={<Plus className="h-3.5 w-3.5" />}
          onClick={() => { setEditingIdx(null); setShowForm(true); }}
        >
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
              <Input
                label="Descripción"
                required
                value={draft.description}
                onChange={(e) => setDraft({ ...draft, description: e.target.value })}
              />
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
            <div className="col-span-6 sm:col-span-2">
              <Input
                label="Cantidad"
                type="number"
                min="0"
                step="0.01"
                value={draft.quantity}
                onChange={(e) => setDraft({ ...draft, quantity: e.target.value })}
              />
            </div>
            <div className="col-span-6 sm:col-span-3">
              <Input
                label="Precio unitario (COP)"
                type="number"
                min="0"
                step="0.01"
                value={draft.unitPrice}
                onChange={(e) => setDraft({ ...draft, unitPrice: e.target.value })}
              />
            </div>
            <div className="col-span-12 sm:col-span-4">
              <Select
                label="Impuesto"
                disabled={loadingTaxTypes}
                value={draft.taxTypeCode}
                onChange={(e) => setDraft({ ...draft, taxTypeCode: e.target.value })}
              >
                {loadingTaxTypes ? (
                  <option>Cargando…</option>
                ) : (
                  <>
                    <option value="">Sin impuesto</option>
                    {taxTypes.map((t) => (
                      <option key={t.code} value={t.code}>
                        {t.code} — {t.name}
                      </option>
                    ))}
                  </>
                )}
              </Select>
            </div>
            {draft.taxTypeCode && (
              <div className="col-span-12 sm:col-span-3">
                <Input
                  label="Porcentaje (%)"
                  type="number"
                  min="0"
                  step="0.01"
                  value={draft.taxPercent}
                  onChange={(e) => setDraft({ ...draft, taxPercent: e.target.value })}
                />
              </div>
            )}
            <div className="col-span-12 flex items-center justify-between pt-1">
              <span className="text-xs text-(--text-secondary)">
                Total de la línea:{" "}
                <span className="font-mono text-(--text-primary)">{formatCOP.format(draftPreview.totalCents / 100)}</span>
              </span>
              <div className="flex gap-2">
                <Button type="button" variant="ghost" onClick={handleCancel}>
                  Cancelar
                </Button>
                <Button
                  type="button"
                  icon={editingIdx !== null ? <Pencil className="h-3.5 w-3.5" /> : <Plus className="h-3.5 w-3.5" />}
                  disabled={!canAdd}
                  onClick={handleSaveLine}
                >
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
