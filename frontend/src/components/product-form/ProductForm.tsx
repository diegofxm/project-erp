import { useState, type FormEvent } from "react";
import { listItemStandards, listTaxTypes, listUnitMeasures } from "../../lib/catalogs";
import { useCatalog } from "../../lib/useCatalog";
import type { Product, ProductPayload } from "../../lib/types";
import { Combobox } from "../ui/Combobox";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";
import { Button } from "../ui/Button";

// Placeholder del código real dentro del estándar elegido — "999" (estándar propio,
// predeterminado) usa el propio código del contribuyente; el resto son estándares externos
// reales con códigos que apidian no valida contra un catálogo completo (ver ItemStandard,
// docs/apidian-architecture.md sección 9.45).
const ITEM_CODE_PLACEHOLDERS: Record<string, string> = {
  "001": "Código UNSPSC real, ej. 43211500",
  "010": "Código GTIN/EAN real",
  "020": "Partida arancelaria real",
};

interface ProductFormProps {
  initial: Product | null;
  onSubmit: (payload: ProductPayload) => void;
  onCancel: () => void;
  loading: boolean;
}

// unit_price_cents viaja en centavos (mismo criterio que domain.Line en cofacture) — el
// formulario lo muestra/edita en la unidad de moneda normal y convierte al guardar.
function centsToAmount(cents: number | undefined): string {
  return cents ? (cents / 100).toString() : "";
}

// Formulario único (sin pestañas): un producto tiene menos campos que una empresa o un
// cliente — sin identificación, sin dirección. Misma grilla fija de 12 columnas y mismos
// hooks (useCatalog) que el resto del frontend (ver docs/frontend-architecture.md).
export function ProductForm({ initial, onSubmit, onCancel, loading }: ProductFormProps) {
  const { data: unitMeasures, loading: loadingUnitMeasures } = useCatalog(listUnitMeasures);
  const { data: taxTypes, loading: loadingTaxTypes } = useCatalog(listTaxTypes);
  const { data: itemStandards, loading: loadingItemStandards } = useCatalog(listItemStandards);

  const [description, setDescription] = useState(initial?.description ?? "");
  const [unitCode, setUnitCode] = useState(initial?.unit_code ?? "");
  const [unitPrice, setUnitPrice] = useState(centsToAmount(initial?.unit_price_cents));
  const [itemCode, setItemCode] = useState(initial?.item_code ?? "");
  const [itemTypeCode, setItemTypeCode] = useState(initial?.item_type_code ?? "");
  const [taxTypeCode, setTaxTypeCode] = useState(initial?.tax_type_code ?? "");
  const [taxPercent, setTaxPercent] = useState(initial?.tax_percent?.toString() ?? "");

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    onSubmit({
      description,
      unit_code: unitCode,
      unit_price_cents: Math.round(Number(unitPrice || 0) * 100),
      item_code: itemCode || undefined,
      item_type_code: itemTypeCode || undefined,
      tax_type_code: taxTypeCode || undefined,
      tax_percent: taxPercent ? Number(taxPercent) : undefined,
    });
  }

  return (
    <form className="flex flex-col gap-3 p-4" onSubmit={handleSubmit}>
      <div className="grid grid-cols-12 gap-3">
        <div className="col-span-8">
          <Input label="Descripción" required value={description} onChange={(e) => setDescription(e.target.value)} />
        </div>
        <div className="col-span-4">
          <Input
            label="Código del ítem (opcional)"
            value={itemCode}
            onChange={(e) => setItemCode(e.target.value)}
            placeholder={ITEM_CODE_PLACEHOLDERS[itemTypeCode] ?? "Tu código interno"}
          />
        </div>

        <div className="col-span-4">
          <Combobox
            label="Unidad de medida"
            disabled={loadingUnitMeasures}
            value={unitCode}
            onChange={setUnitCode}
            options={unitMeasures.map((u) => ({ value: u.code, label: `${u.code} — ${u.name}` }))}
            placeholder={loadingUnitMeasures ? "Cargando…" : "Buscar unidad…"}
          />
        </div>
        <div className="col-span-4">
          <Input
            label="Precio unitario (COP)"
            type="number"
            step="0.01"
            min="0"
            required
            value={unitPrice}
            onChange={(e) => setUnitPrice(e.target.value)}
          />
        </div>
        <div className="col-span-4">
          <Select label="Tipo de impuesto por defecto" disabled={loadingTaxTypes} value={taxTypeCode} onChange={(e) => setTaxTypeCode(e.target.value)}>
            {loadingTaxTypes ? (
              <option>Cargando…</option>
            ) : (
              <>
                <option value="">Sin impuesto por defecto</option>
                {taxTypes.map((t) => (
                  <option key={t.code} value={t.code}>
                    {t.code} — {t.name}
                  </option>
                ))}
              </>
            )}
          </Select>
        </div>

        <div className="col-span-4">
          <Input label="Porcentaje de impuesto (%)" type="number" step="0.01" min="0" value={taxPercent} onChange={(e) => setTaxPercent(e.target.value)} />
        </div>
        <div className="col-span-8">
          <Select
            label="Estándar de clasificación del ítem"
            disabled={loadingItemStandards}
            value={itemTypeCode}
            onChange={(e) => setItemTypeCode(e.target.value)}
          >
            {loadingItemStandards ? (
              <option>Cargando…</option>
            ) : (
              <>
                <option value="">Sin clasificar (mi propio código)</option>
                {itemStandards.filter((s) => s.code !== "999").map((s) => (
                  <option key={s.code} value={s.code}>
                    {s.name}
                  </option>
                ))}
              </>
            )}
          </Select>
        </div>
      </div>

      <div className="flex gap-2">
        <Button type="button" variant="secondary" onClick={onCancel} className="flex-1">
          Cancelar
        </Button>
        <Button type="submit" loading={loading} className="flex-1">
          {initial ? "Guardar cambios" : "Crear producto"}
        </Button>
      </div>
    </form>
  );
}
