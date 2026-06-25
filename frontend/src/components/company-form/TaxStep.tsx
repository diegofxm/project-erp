import { useEffect, useState } from "react";
import { listLiabilityCodes, listTaxRegimes, listTaxTypes } from "../../lib/catalogs";
import type { CatalogEntry } from "../../lib/types";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";
import { TagInput } from "../ui/TagInput";
import type { StepProps } from "./IdentificationStep";

export function TaxStep({ form, setField }: StepProps) {
  const [taxTypes, setTaxTypes] = useState<CatalogEntry[]>([]);
  const [taxRegimes, setTaxRegimes] = useState<CatalogEntry[]>([]);
  const [liabilityCodes, setLiabilityCodes] = useState<CatalogEntry[]>([]);

  useEffect(() => {
    listTaxTypes().then(setTaxTypes);
    listTaxRegimes().then(setTaxRegimes);
    listLiabilityCodes().then(setLiabilityCodes);
  }, []);

  const selectedLiabilityCodes = form.liability_codes ?? [];

  function toggleLiabilityCode(code: string) {
    const next = selectedLiabilityCodes.includes(code)
      ? selectedLiabilityCodes.filter((c) => c !== code)
      : [...selectedLiabilityCodes, code];
    setField("liability_codes", next);
  }

  return (
    <div className="flex flex-col gap-3 p-4">
      <Select
        label="Tipo de impuesto del régimen"
        value={form.tax_scheme_code ?? "ZZ"}
        onChange={(e) => setField("tax_scheme_code", e.target.value)}
      >
        {taxTypes.map((t) => (
          <option key={t.code} value={t.code}>
            {t.code} — {t.name}
          </option>
        ))}
      </Select>
      <Select
        label="Tipo de régimen"
        value={form.tax_regime_code ?? ""}
        onChange={(e) => setField("tax_regime_code", e.target.value)}
      >
        <option value="">No aplica</option>
        {taxRegimes.map((t) => (
          <option key={t.code} value={t.code}>
            {t.code} — {t.name}
          </option>
        ))}
      </Select>
      <div className="flex flex-col gap-1">
        <span className="text-xs font-medium text-(--text-secondary)">Responsabilidades fiscales</span>
        <div className="flex flex-col gap-1 rounded border border-(--border-color) bg-(--bg-primary) p-2">
          {liabilityCodes.map((l) => (
            <label key={l.code} className="flex items-center gap-2 text-xs text-(--text-primary)">
              <input type="checkbox" checked={selectedLiabilityCodes.includes(l.code)} onChange={() => toggleLiabilityCode(l.code)} />
              <span className="font-mono">{l.code}</span>
              <span className="text-(--text-secondary)">{l.name}</span>
            </label>
          ))}
        </div>
      </div>
      <TagInput
        label="Códigos CIIU (actividad económica)"
        values={form.industry_classification_codes ?? []}
        onChange={(values) => setField("industry_classification_codes", values)}
        max={4}
        placeholder="Ej. 4711"
      />
      <Input
        label="Matrícula mercantil (opcional)"
        value={form.merchant_registration_number ?? ""}
        onChange={(e) => setField("merchant_registration_number", e.target.value)}
      />
    </div>
  );
}
