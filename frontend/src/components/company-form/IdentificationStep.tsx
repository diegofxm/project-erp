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

  const isJuridica = form.entity_type_code === "1" || form.identification_type_code === "31";

  function handleEntityTypeChange(val: string) {
    setField("entity_type_code", val);
    if (val === "1" && form.identification_type_code !== "31") {
      setField("identification_type_code", "31");
    } else if (val === "2" && form.identification_type_code === "31") {
      setField("identification_type_code", "13");
    }
  }

  function handleIdentificationTypeChange(val: string) {
    setField("identification_type_code", val);
    if (val === "31") {
      setField("entity_type_code", "1");
    } else if (form.entity_type_code === "1") {
      setField("entity_type_code", "2");
    }
  }

  return (
    <div className="grid grid-cols-12 gap-3 p-4">
      {/* Tipo de persona + ambiente — arriba del todo */}
      <div className="col-span-6">
        <Select
          label="Tipo de persona"
          value={form.entity_type_code ?? ""}
          onChange={(e) => handleEntityTypeChange(e.target.value)}
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

      {/* Tipo de identificación | NIT/número | DV (solo jurídica) | Razón social */}
      <div className="col-span-12 sm:col-span-4">
        <Select
          label="Tipo de identificación"
          required
          disabled={loadingIdentificationTypes}
          value={form.identification_type_code}
          onChange={(e) => handleIdentificationTypeChange(e.target.value)}
        >
          {loadingIdentificationTypes ? (
            <option>Cargando…</option>
          ) : (
            identificationTypes.map((t) => (
              <option key={t.code} value={t.code}>{t.name}</option>
            ))
          )}
        </Select>
      </div>
      <div className={isJuridica ? "col-span-9 sm:col-span-3" : "col-span-12 sm:col-span-4"}>
        <Input
          label={isJuridica ? "NIT" : "Número de identificación"}
          required
          value={form.nit}
          onChange={(e) => setField("nit", e.target.value)}
        />
      </div>
      {isJuridica && (
        <div className="col-span-3 sm:col-span-1">
          <Input
            label="DV"
            required
            value={form.check_digit}
            onChange={(e) => setField("check_digit", e.target.value)}
          />
        </div>
      )}
      <div className="col-span-12 sm:col-span-4">
        <Input
          label={isJuridica ? "Razón social" : "Nombre"}
          required
          value={form.business_name}
          onChange={(e) => setField("business_name", e.target.value)}
        />
      </div>

      {/* Nombre comercial */}
      <div className="col-span-6">
        <Input
          label="Nombre comercial (opcional)"
          value={form.trade_name ?? ""}
          onChange={(e) => setField("trade_name", e.target.value)}
        />
      </div>
    </div>
  );
}
