import { useState, type FormEvent } from "react";
import { listItemStandards, listTaxTypes, listUnitMeasures } from "../../lib/catalogs";
import { useCatalog } from "../../lib/useCatalog";
import type { Product, ProductPayload } from "../../lib/types";
import { Combobox } from "../ui/Combobox";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";
import { Button } from "../ui/Button";

interface ProductFormProps {
  initial: Product | null;
  onSubmit: (payload: ProductPayload) => void;
  onCancel: () => void;
  loading: boolean;
}

const ITEM_CODE_PLACEHOLDERS: Record<string, string> = {
  "001": "Código UNSPSC real, ej. 43211500",
  "010": "Código GTIN/EAN real",
  "020": "Partida arancelaria real",
};

function defaultPayload(): ProductPayload {
  return {
    code: "",
    name: "",
    description: "",
    unit_measure_code: "94",
    standard_code: "",
    standard_code_type: "Estándar propio",
    standard_code_id: "999",
    standard_code_agency_id: "",
    is_service: false,
    tax_scheme_code: "ZZ",
    tax_scheme_name: "No aplica",
    tax_rate: 0,
    base_price: 0,
  };
}

function fromProduct(p: Product): ProductPayload {
  return {
    code: p.code,
    name: p.name,
    description: p.description,
    unit_measure_code: p.unit_measure_code,
    standard_code: p.standard_code,
    standard_code_type: p.standard_code_type,
    standard_code_id: p.standard_code_id,
    standard_code_agency_id: p.standard_code_agency_id,
    is_service: p.is_service,
    tax_scheme_code: p.tax_scheme_code,
    tax_scheme_name: p.tax_scheme_name,
    tax_rate: p.tax_rate,
    base_price: p.base_price,
  };
}

export function ProductForm({ initial, onSubmit, onCancel, loading }: ProductFormProps) {
  const { data: unitMeasures, loading: loadingUnits } = useCatalog(listUnitMeasures);
  const { data: taxTypes, loading: loadingTaxTypes } = useCatalog(listTaxTypes);
  const { data: itemStandards, loading: loadingStandards } = useCatalog(listItemStandards);

  const [form, setForm] = useState<ProductPayload>(() => initial ? fromProduct(initial) : defaultPayload());

  function set<K extends keyof ProductPayload>(key: K, value: ProductPayload[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  function handleStandardChange(standardId: string) {
    const std = itemStandards.find((s) => s.code === standardId);
    setForm((prev) => ({
      ...prev,
      standard_code_id: standardId,
      standard_code_type: std?.name ?? "Estándar propio",
      standard_code_agency_id: std?.agency_id ?? "",
      standard_code: standardId === "999" ? prev.standard_code : "",
    }));
  }

  function handleTaxTypeChange(code: string) {
    const t = taxTypes.find((t) => t.code === code);
    setForm((prev) => ({
      ...prev,
      tax_scheme_code: code,
      tax_scheme_name: t?.name ?? "No aplica",
    }));
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    onSubmit(form);
  }

  return (
    <form className="flex flex-col gap-3 p-4" onSubmit={handleSubmit}>
      <div className="grid grid-cols-12 gap-3">
        {/* Código interno + Nombre */}
        <div className="col-span-12 sm:col-span-3">
          <Input
            label="Código interno"
            required
            value={form.code}
            onChange={(e) => set("code", e.target.value)}
            placeholder="P001"
          />
        </div>
        <div className="col-span-12 sm:col-span-9">
          <Input
            label="Nombre del producto/servicio"
            required
            value={form.name}
            onChange={(e) => set("name", e.target.value)}
          />
        </div>

        {/* Descripción */}
        <div className="col-span-12">
          <Input
            label="Descripción (opcional)"
            value={form.description}
            onChange={(e) => set("description", e.target.value)}
          />
        </div>

        {/* Unidad de medida + Precio base + Tipo (bien/servicio) */}
        <div className="col-span-12 sm:col-span-4">
          <Combobox
            label="Unidad de medida"
            disabled={loadingUnits}
            value={form.unit_measure_code}
            onChange={(v) => set("unit_measure_code", v)}
            options={unitMeasures.map((u) => ({ value: u.code, label: `${u.code} — ${u.name}` }))}
            placeholder={loadingUnits ? "Cargando…" : "Buscar unidad…"}
          />
        </div>
        <div className="col-span-8 sm:col-span-4">
          <Input
            label="Precio base (COP)"
            type="number"
            step="0.01"
            min="0"
            required
            value={form.base_price || ""}
            onChange={(e) => set("base_price", Number(e.target.value) || 0)}
          />
        </div>
        <div className="col-span-4 sm:col-span-4 flex items-end pb-1">
          <label className="flex items-center gap-2 text-xs text-(--text-primary) cursor-pointer">
            <input
              type="checkbox"
              checked={form.is_service}
              onChange={(e) => set("is_service", e.target.checked)}
              className="h-3.5 w-3.5"
            />
            <span>Es servicio</span>
          </label>
        </div>

        {/* Estándar de clasificación DIAN */}
        <div className="col-span-12 sm:col-span-5">
          <Select
            label="Estándar de clasificación (DIAN)"
            disabled={loadingStandards}
            value={form.standard_code_id}
            onChange={(e) => handleStandardChange(e.target.value)}
          >
            {loadingStandards ? (
              <option>Cargando…</option>
            ) : (
              <>
                <option value="999">Sin clasificar (código propio)</option>
                {itemStandards.filter((s) => s.code !== "999").map((s) => (
                  <option key={s.code} value={s.code}>{s.name}</option>
                ))}
              </>
            )}
          </Select>
        </div>
        <div className="col-span-12 sm:col-span-7">
          <Input
            label="Código del ítem en el estándar"
            value={form.standard_code}
            onChange={(e) => set("standard_code", e.target.value)}
            placeholder={ITEM_CODE_PLACEHOLDERS[form.standard_code_id] ?? "Tu código interno"}
          />
        </div>

        {/* Impuesto */}
        <div className="col-span-8 sm:col-span-6">
          <Select
            label="Tipo de impuesto por defecto"
            disabled={loadingTaxTypes}
            value={form.tax_scheme_code}
            onChange={(e) => handleTaxTypeChange(e.target.value)}
          >
            {loadingTaxTypes ? (
              <option>Cargando…</option>
            ) : (
              taxTypes.map((t) => (
                <option key={t.code} value={t.code}>{t.code} — {t.name}</option>
              ))
            )}
          </Select>
        </div>
        <div className="col-span-4 sm:col-span-6">
          <Input
            label="Porcentaje (%)"
            type="number"
            step="0.01"
            min="0"
            value={form.tax_rate || ""}
            onChange={(e) => set("tax_rate", Number(e.target.value) || 0)}
          />
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
