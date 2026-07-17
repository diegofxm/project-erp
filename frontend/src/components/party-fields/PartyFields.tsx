import { useEffect, useState } from "react";
import { Search } from "lucide-react";
import { listDepartments, listIdentificationTypes, listLiabilityCodes, listMunicipalities, listTaxRegimes, listTaxTypes } from "../../lib/catalogs";
import { ApiError } from "../../lib/apiClient";
import { verifyAcquirer } from "../../lib/dianVerification";
import { useCatalog } from "../../lib/useCatalog";
import type { CustomerPayload, VendorPayload, Municipality } from "../../lib/types";
import { Button } from "../ui/Button";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";

type PartyPayload = CustomerPayload | VendorPayload;

interface PartyFieldsProps {
  value: PartyPayload;
  onChange: (next: PartyPayload) => void;
}

// Campos de un CustomerPayload (identificación, nombre, tipo de entidad, contacto, dirección
// departamento→municipio, información tributaria, responsabilidades fiscales) como componente
// controlado, sin chrome de guardar/cancelar — extraído de CustomerForm para reusarlo también
// en la sección de cliente de Factura Electrónica (ver docs/frontend-architecture.md). Misma
// grilla fija de 12 columnas que el resto del frontend.
export function PartyFields({ value, onChange }: PartyFieldsProps) {
  const { data: identificationTypes, loading: loadingIdentificationTypes } = useCatalog(listIdentificationTypes);
  const { data: departments, loading: loadingDepartments } = useCatalog(listDepartments);
  const { data: taxTypes, loading: loadingTaxTypes } = useCatalog(listTaxTypes);
  const { data: taxRegimes, loading: loadingTaxRegimes } = useCatalog(listTaxRegimes);
  const { data: liabilityCodes, loading: loadingLiabilityCodes } = useCatalog(listLiabilityCodes);

  const departmentCode = value.address?.state_code ?? "";
  const municipalityCode = value.address?.city_code ?? "";
  const selectedLiabilityCodes = value.liability_codes ?? [];

  const [municipalities, setMunicipalities] = useState<Municipality[]>([]);
  const [loadingMunicipalities, setLoadingMunicipalities] = useState(false);

  const [verifying, setVerifying] = useState(false);
  const [verifyResult, setVerifyResult] = useState<string | null>(null);

  // Verificación opcional contra la DIAN (GetAcquirer, sección 9.41) — solo aplica a NIT, solo
  // a pedido del usuario, nunca bloquea ni autocompleta nada.
  async function handleVerify() {
    setVerifying(true);
    setVerifyResult(null);
    try {
      const result = await verifyAcquirer(value.identification.type_code, value.identification.number);
      setVerifyResult(
        result.found
          ? `Encontrado en la DIAN: ${result.receiver_name ?? "—"} (${result.receiver_email ?? "sin correo"})`
          : "Sin registro en la DIAN — normal para la mayoría de NIT, no impide guardar.",
      );
    } catch (err) {
      setVerifyResult(err instanceof ApiError ? err.message : "No se pudo verificar contra la DIAN");
    } finally {
      setVerifying(false);
    }
  }

  useEffect(() => {
    if (!departmentCode) {
      setMunicipalities([]);
      return;
    }
    setLoadingMunicipalities(true);
    listMunicipalities(departmentCode)
      .then(setMunicipalities)
      .finally(() => setLoadingMunicipalities(false));
  }, [departmentCode]);

  function handleDepartmentChange(code: string) {
    const department = departments.find((d) => d.code === code);
    onChange({
      ...value,
      address: { ...value.address, state_code: code || undefined, state_name: department?.name, city_code: undefined, city_name: undefined },
    });
  }

  function handleMunicipalityChange(code: string) {
    const municipality = municipalities.find((m) => m.code === code);
    onChange({ ...value, address: { ...value.address, city_code: code || undefined, city_name: municipality?.name } });
  }

  function toggleLiabilityCode(code: string) {
    const next = selectedLiabilityCodes.includes(code)
      ? selectedLiabilityCodes.filter((c) => c !== code)
      : [...selectedLiabilityCodes, code];
    onChange({ ...value, liability_codes: next });
  }

  return (
    <div className="grid grid-cols-12 gap-3">
      <div className="col-span-3">
        <Select
          label="Tipo de identificación"
          required
          disabled={loadingIdentificationTypes}
          value={value.identification.type_code}
          onChange={(e) => onChange({ ...value, identification: { ...value.identification, type_code: e.target.value } })}
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
      <div className="col-span-1">
        <Input
          label="DV"
          value={value.identification.verification_code ?? ""}
          onChange={(e) => onChange({ ...value, identification: { ...value.identification, verification_code: e.target.value || undefined } })}
        />
      </div>
      <div className="col-span-4">
        <Input
          label="Número de identificación"
          required
          value={value.identification.number}
          onChange={(e) => onChange({ ...value, identification: { ...value.identification, number: e.target.value } })}
        />
      </div>
      <div className="col-span-4">
        <Input label="Nombre / Razón social" required value={value.name} onChange={(e) => onChange({ ...value, name: e.target.value })} />
      </div>

      {value.identification.type_code === "31" && (
        <div className="col-span-12 flex flex-col gap-1">
          <Button
            type="button"
            variant="secondary"
            icon={<Search className="h-3.5 w-3.5" />}
            loading={verifying}
            disabled={!value.identification.number}
            onClick={handleVerify}
            className="self-start"
          >
            Verificar en la DIAN
          </Button>
          {verifyResult && <p className="text-xs text-(--text-secondary)">{verifyResult}</p>}
        </div>
      )}

      <div className="col-span-4">
        <Select
          label="Tipo de entidad"
          value={value.entity_type_code ?? ""}
          onChange={(e) => onChange({ ...value, entity_type_code: e.target.value || undefined })}
        >
          <option value="">Sin especificar</option>
          <option value="1">Persona jurídica</option>
          <option value="2">Persona natural</option>
        </Select>
      </div>
      <div className="col-span-4">
        <Input
          label="Correo"
          type="email"
          value={value.email ?? ""}
          onChange={(e) => onChange({ ...value, email: e.target.value || undefined })}
        />
      </div>
      <div className="col-span-4">
        <Input label="Teléfono" value={value.phone ?? ""} onChange={(e) => onChange({ ...value, phone: e.target.value || undefined })} />
      </div>

      <div className="col-span-3">
        <Select label="Departamento" disabled={loadingDepartments} value={departmentCode} onChange={(e) => handleDepartmentChange(e.target.value)}>
          {loadingDepartments ? (
            <option>Cargando…</option>
          ) : (
            <>
              <option value="">Selecciona…</option>
              {departments.map((d) => (
                <option key={d.code} value={d.code}>
                  {d.name}
                </option>
              ))}
            </>
          )}
        </Select>
      </div>
      <div className="col-span-5">
        <Select
          label="Municipio"
          disabled={!departmentCode || loadingMunicipalities}
          value={municipalityCode}
          onChange={(e) => handleMunicipalityChange(e.target.value)}
        >
          {loadingMunicipalities ? (
            <option>Cargando…</option>
          ) : (
            <>
              <option value="">{departmentCode ? "Selecciona…" : "Elige un departamento primero"}</option>
              {municipalities.map((m) => (
                <option key={m.code} value={m.code}>
                  {m.name}
                </option>
              ))}
            </>
          )}
        </Select>
      </div>
      <div className="col-span-4">
        <Input
          label="Dirección"
          value={value.address?.line ?? ""}
          onChange={(e) => onChange({ ...value, address: { ...value.address, line: e.target.value || undefined } })}
        />
      </div>

      <div className="col-span-6">
        <Select
          label="Tipo de impuesto del régimen"
          disabled={loadingTaxTypes}
          value={value.tax_scheme_code ?? "ZZ"}
          onChange={(e) => onChange({ ...value, tax_scheme_code: e.target.value })}
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
      <div className="col-span-6">
        <Select
          label="Tipo de régimen"
          disabled={loadingTaxRegimes}
          value={value.tax_regime_code ?? ""}
          onChange={(e) => onChange({ ...value, tax_regime_code: e.target.value || undefined })}
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

      <div className="col-span-12 flex flex-col gap-1">
        <span className="text-xs font-medium text-(--text-secondary)">Responsabilidades fiscales</span>
        <div className="grid grid-cols-2 gap-1 rounded border border-(--border-color) bg-(--bg-primary) p-2">
          {loadingLiabilityCodes ? (
            <div className="col-span-2 flex min-h-16 items-center justify-center">
              <span className="text-xs text-(--text-muted)">Cargando…</span>
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
    </div>
  );
}
