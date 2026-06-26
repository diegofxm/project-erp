import { listIdentificationTypes } from "../../lib/catalogs";
import { useCatalog } from "../../lib/useCatalog";
import type { CreateIssuerPayload } from "../../lib/types";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";

export interface StepProps {
  form: CreateIssuerPayload;
  setField: <K extends keyof CreateIssuerPayload>(key: K, value: CreateIssuerPayload[K]) => void;
}

export function IdentificationStep({ form, setField }: StepProps) {
  const { data: identificationTypes, loading: loadingIdentificationTypes } = useCatalog(listIdentificationTypes);

  // Grilla de 12 columnas fijas (ver docs/frontend-architecture.md, regla de ancho): cada campo
  // tiene un ancho proporcional fijo (col-span-N) — al colapsar/expandir el Sidebar los campos
  // se agrandan o achican con el espacio disponible, pero NUNCA cambian de fila ni de posición
  // (a diferencia de una grilla auto-fit, que recalcula cuántas columnas entran y reordena).
  return (
    <div className="grid grid-cols-12 gap-3 p-4">
      <div className="col-span-3">
        <Input label="NIT" required value={form.nit} onChange={(e) => setField("nit", e.target.value)} />
      </div>
      <div className="col-span-1">
        <Input label="DV" required value={form.check_digit} onChange={(e) => setField("check_digit", e.target.value)} />
      </div>
      <div className="col-span-4">
        <Select
          label="Tipo de identificación"
          required
          disabled={loadingIdentificationTypes}
          value={form.identification_type_code}
          onChange={(e) => setField("identification_type_code", e.target.value)}
        >
          {loadingIdentificationTypes ? (
            <option>Cargando…</option>
          ) : (
            identificationTypes.map((t) => (
              <option key={t.code} value={t.code}>
                {t.name}
              </option>
            ))
          )}
        </Select>
      </div>
      <div className="col-span-4">
        <Input label="Razón social" required value={form.business_name} onChange={(e) => setField("business_name", e.target.value)} />
      </div>
      <div className="col-span-3">
        <Input label="Nombre comercial" value={form.trade_name ?? ""} onChange={(e) => setField("trade_name", e.target.value)} />
      </div>
      <div className="col-span-6">
        <Select
          label="Tipo de entidad"
          value={form.entity_type_code ?? ""}
          onChange={(e) => setField("entity_type_code", e.target.value)}
        >
          <option value="">Automático según el tipo de identificación</option>
          <option value="1">Persona jurídica</option>
          <option value="2">Persona natural</option>
        </Select>
      </div>
      <div className="col-span-3">
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
    </div>
  );
}
