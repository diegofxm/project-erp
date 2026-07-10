import { useEffect, useState } from "react";
import { Plus, Trash2 } from "lucide-react";
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
  // itemTypeCode es el selector de @schemeID que copia el producto (ver ProductPayload) — sin
  // UI propia aquí, item_type_name/item_type_agency_id se derivan en el servidor.
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
    description: product.description,
    unitCode: product.unit_code,
    quantity: "1",
    unitPrice: centsToAmount(product.unit_price_cents),
    itemCode: product.item_code ?? "",
    itemTypeCode: product.item_type_code ?? "",
    taxTypeCode: product.tax_type_code ?? "",
    taxPercent: product.tax_percent?.toString() ?? "",
  };
}

// Selector de "producto guardado" (copia descripción/unidad/precio/código/impuesto por
// defecto) o entrada manual; cantidad, precio unitario e impuesto editables siempre después de
// elegir. La vista previa de cada línea (agregada o en captura) usa lib/invoiceMath.ts — el
// mismo cálculo que hará el servidor al guardar, pero solo para feedback inmediato (ver
// docs/apidian-architecture.md sección 9.37: el servidor es la fuente de verdad).
export function LineItemsEditor({ lines, onChange }: LineItemsEditorProps) {
  const [products, setProducts] = useState<Product[]>([]);
  const { data: unitMeasures, loading: loadingUnitMeasures } = useCatalog(listUnitMeasures);
  const { data: taxTypes, loading: loadingTaxTypes } = useCatalog(listTaxTypes);
  const [draft, setDraft] = useState<DraftLine>(EMPTY_DRAFT);

  useEffect(() => {
    listProducts()
      .then(setProducts)
      .catch(() => setProducts([]));
  }, []);

  function handleProductSelect(id: string) {
    if (!id) {
      setDraft(EMPTY_DRAFT);
      return;
    }
    const product = products.find((p) => p.id === id);
    if (product) setDraft(draftFromProduct(product));
  }

  function handleAddLine() {
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
    onChange([...lines, line]);
    setDraft(EMPTY_DRAFT);
  }

  function handleRemoveLine(index: number) {
    onChange(lines.filter((_, i) => i !== index));
  }

  const canAdd = draft.description.trim() !== "" && draft.unitCode !== "" && Number(draft.quantity) > 0 && Number(draft.unitPrice) > 0;
  const draftPreview = previewLine({
    quantity: Number(draft.quantity || 0),
    unit_price_cents: amountToCents(draft.unitPrice),
    tax_percent: draft.taxTypeCode ? Number(draft.taxPercent || 0) : undefined,
  });

  return (
    <div className="flex flex-col gap-3">
      {lines.length > 0 && (
        <div className="overflow-hidden rounded border border-(--border-color)">
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
                return (
                  <tr key={i} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                    <td className="px-3 py-2 text-(--text-primary)">{line.description}</td>
                    <td className="px-3 py-2 font-mono text-(--text-secondary)">{line.quantity}</td>
                    <td className="px-3 py-2 font-mono text-(--text-secondary)">{formatCOP.format(line.unit_price_cents / 100)}</td>
                    <td className="px-3 py-2 text-(--text-secondary)">
                      {line.tax_type_code ? `${line.tax_type_code} (${line.tax_percent ?? 0}%)` : "—"}
                    </td>
                    <td className="px-3 py-2 font-mono text-(--text-primary)">{formatCOP.format(preview.totalCents / 100)}</td>
                    <td className="px-3 py-2">
                      <button
                        type="button"
                        title="Quitar línea"
                        onClick={() => handleRemoveLine(i)}
                        className="rounded p-1 text-(--color-danger) hover:bg-(--bg-hover)"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      <div className="rounded border border-(--border-color) bg-(--bg-secondary) p-3">
        <div className="grid grid-cols-12 gap-3">
          <div className="col-span-5">
            <Select label="Producto guardado (opcional)" value={draft.productId} onChange={(e) => handleProductSelect(e.target.value)}>
              <option value="">Entrada manual</option>
              {products.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.description}
                </option>
              ))}
            </Select>
          </div>
          <div className="col-span-7">
            <Input
              label="Descripción"
              required
              value={draft.description}
              onChange={(e) => setDraft({ ...draft, description: e.target.value })}
            />
          </div>

          <div className="col-span-3">
            <Combobox
              label="Unidad de medida"
              disabled={loadingUnitMeasures}
              value={draft.unitCode}
              onChange={(unitCode) => setDraft({ ...draft, unitCode })}
              options={unitMeasures.map((u) => ({ value: u.code, label: `${u.code} — ${u.name}` }))}
              placeholder={loadingUnitMeasures ? "Cargando…" : "Buscar unidad…"}
            />
          </div>
          <div className="col-span-2">
            <Input
              label="Cantidad"
              type="number"
              min="0"
              step="0.01"
              value={draft.quantity}
              onChange={(e) => setDraft({ ...draft, quantity: e.target.value })}
            />
          </div>
          <div className="col-span-3">
            <Input
              label="Precio unitario (COP)"
              type="number"
              min="0"
              step="0.01"
              value={draft.unitPrice}
              onChange={(e) => setDraft({ ...draft, unitPrice: e.target.value })}
            />
          </div>
          <div className="col-span-4">
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
            <div className="col-span-3">
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
          <div className={draft.taxTypeCode ? "col-span-5 flex items-end justify-between" : "col-span-8 flex items-end justify-between"}>
            <span className="text-xs text-(--text-secondary)">
              Total de la línea: <span className="font-mono text-(--text-primary)">{formatCOP.format(draftPreview.totalCents / 100)}</span>
            </span>
          </div>
          <div className="col-span-12">
            <Button type="button" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} disabled={!canAdd} onClick={handleAddLine}>
              Agregar línea
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
