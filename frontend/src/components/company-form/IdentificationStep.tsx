import { useEffect, useState } from "react";
import { listIdentificationTypes } from "../../lib/catalogs";
import type { CatalogEntry, CreateIssuerPayload } from "../../lib/types";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";

export interface StepProps {
  form: CreateIssuerPayload;
  setField: <K extends keyof CreateIssuerPayload>(key: K, value: CreateIssuerPayload[K]) => void;
}

export function IdentificationStep({ form, setField }: StepProps) {
  const [identificationTypes, setIdentificationTypes] = useState<CatalogEntry[]>([]);

  useEffect(() => {
    listIdentificationTypes().then(setIdentificationTypes);
  }, []);

  return (
    <div className="flex flex-col gap-3 p-4">
      <div className="grid grid-cols-2 gap-2">
        <Input label="NIT" required value={form.nit} onChange={(e) => setField("nit", e.target.value)} />
        <Input
          label="Dígito verificación"
          required
          value={form.check_digit}
          onChange={(e) => setField("check_digit", e.target.value)}
        />
      </div>
      <Select
        label="Tipo de identificación"
        required
        value={form.identification_type_code}
        onChange={(e) => setField("identification_type_code", e.target.value)}
      >
        {identificationTypes.map((t) => (
          <option key={t.code} value={t.code}>
            {t.name}
          </option>
        ))}
      </Select>
      <Input label="Razón social" required value={form.business_name} onChange={(e) => setField("business_name", e.target.value)} />
      <Input label="Nombre comercial" value={form.trade_name ?? ""} onChange={(e) => setField("trade_name", e.target.value)} />
      <Select
        label="Tipo de entidad"
        value={form.entity_type_code ?? ""}
        onChange={(e) => setField("entity_type_code", e.target.value)}
      >
        <option value="">Automático según el tipo de identificación</option>
        <option value="1">Persona jurídica</option>
        <option value="2">Persona natural</option>
      </Select>
      <Select
        label="Ambiente"
        required
        value={form.environment}
        onChange={(e) => setField("environment", e.target.value as CreateIssuerPayload["environment"])}
      >
        <option value="2">Habilitación</option>
        <option value="1">Producción</option>
      </Select>
    </div>
  );
}
