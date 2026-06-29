import { useState } from "react";
import { listLiabilityCodes, listTaxRegimes, listTaxTypes } from "../../lib/catalogs";
import { listCiiuOptions } from "../../lib/ciiu";
import { useCatalog } from "../../lib/useCatalog";
import { Combobox } from "../ui/Combobox";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";
import { Spinner } from "../ui/Spinner";
import { TagInput } from "../ui/TagInput";
import type { StepProps } from "./IdentificationStep";

export function TaxStep({ form, setField }: StepProps) {
  const { data: taxTypes, loading: loadingTaxTypes } = useCatalog(listTaxTypes);
  const { data: taxRegimes, loading: loadingTaxRegimes } = useCatalog(listTaxRegimes);
  const { data: liabilityCodes, loading: loadingLiabilityCodes } = useCatalog(listLiabilityCodes);
  const { data: ciiuOptions, loading: loadingCiiu } = useCatalog(listCiiuOptions);
  const [ciiuPicker, setCiiuPicker] = useState("");

  const selectedLiabilityCodes = form.liability_codes ?? [];
  const ciiuCodes = form.industry_classification_codes ?? [];
  const ciiuAtLimit = ciiuCodes.length >= 4;

  function handlePickCiiu(code: string) {
    if (code && !ciiuCodes.includes(code) && !ciiuAtLimit) {
      setField("industry_classification_codes", [...ciiuCodes, code]);
    }
    setCiiuPicker("");
  }

  function toggleLiabilityCode(code: string) {
    const next = selectedLiabilityCodes.includes(code)
      ? selectedLiabilityCodes.filter((c) => c !== code)
      : [...selectedLiabilityCodes, code];
    setField("liability_codes", next);
  }

  // Grilla de 12 columnas fijas, ver IdentificationStep.tsx para la explicación de por qué.
  return (
    <div className="grid grid-cols-12 gap-3 p-4">
      <div className="col-span-4">
        <Select
          label="Tipo de impuesto del régimen"
          disabled={loadingTaxTypes}
          value={form.tax_scheme_code ?? "ZZ"}
          onChange={(e) => setField("tax_scheme_code", e.target.value)}
        >
          {loadingTaxTypes ? (
            <option>Cargando…</option>
          ) : (
            taxTypes.map((t) => (
              <option key={t.code} value={t.code}>
                {t.code} — {t.name}
              </option>
            ))
          )}
        </Select>
      </div>
      <div className="col-span-4">
        <Select
          label="Tipo de régimen"
          disabled={loadingTaxRegimes}
          value={form.tax_regime_code ?? ""}
          onChange={(e) => setField("tax_regime_code", e.target.value)}
        >
          {loadingTaxRegimes ? (
            <option>Cargando…</option>
          ) : (
            <>
              <option value="">No aplica</option>
              {taxRegimes.map((t) => (
                <option key={t.code} value={t.code}>
                  {t.code} — {t.name}
                </option>
              ))}
            </>
          )}
        </Select>
      </div>
      <div className="col-span-4">
        <Input
          label="Matrícula mercantil (opcional)"
          value={form.merchant_registration_number ?? ""}
          onChange={(e) => setField("merchant_registration_number", e.target.value)}
        />
      </div>
      <div className="col-span-12 flex flex-col gap-1">
        <span className="text-xs font-medium text-(--text-secondary)">Responsabilidades fiscales</span>
        <div className="grid grid-cols-2 gap-1 rounded border border-(--border-color) bg-(--bg-primary) p-2">
          {loadingLiabilityCodes ? (
            <div className="col-span-2 flex min-h-16 items-center justify-center">
              <Spinner className="h-4 w-4 text-(--text-muted)" />
            </div>
          ) : (
            liabilityCodes.map((l) => (
              <label key={l.code} className="flex items-center gap-2 text-xs text-(--text-primary)">
                <input type="checkbox" checked={selectedLiabilityCodes.includes(l.code)} onChange={() => toggleLiabilityCode(l.code)} />
                <span className="font-mono">{l.code}</span>
                <span className="text-(--text-secondary)">{l.name}</span>
              </label>
            ))
          )}
        </div>
      </div>
      <div className="col-span-12 flex flex-col gap-2">
        <Combobox
          label="Códigos CIIU (actividad económica)"
          value={ciiuPicker}
          onChange={handlePickCiiu}
          options={ciiuOptions}
          disabled={loadingCiiu || ciiuAtLimit}
          placeholder={ciiuAtLimit ? "Máximo 4 códigos" : "Buscar por código o palabra…"}
        />
        <TagInput values={ciiuCodes} onChange={(values) => setField("industry_classification_codes", values)} max={4} disableInput />
      </div>
    </div>
  );
}
