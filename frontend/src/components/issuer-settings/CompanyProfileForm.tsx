import { useEffect, useState, type FormEvent } from "react";
import { Pencil, X } from "lucide-react";
import { useAuth } from "../../context/AuthContext";
import { useToast } from "../../context/ToastContext";
import { ApiError } from "../../lib/apiClient";
import {
  listDepartments,
  listLiabilityCodes,
  listMunicipalities,
  listTaxRegimes,
  listTaxTypes,
} from "../../lib/catalogs";
import { listCiiuOptions } from "../../lib/ciiu";
import { useCatalog } from "../../lib/useCatalog";
import type { CatalogEntry, UpdateIssuerProfilePayload } from "../../lib/types";
import { Card } from "../ui/Card";
import { Input } from "../ui/Input";
import { Button } from "../ui/Button";
import { Combobox } from "../ui/Combobox";

const ENTITY_TYPE_OPTIONS = [
  { code: "1", name: "Persona jurídica y asimilada" },
  { code: "2", name: "Persona natural y asimilada" },
];

const ID_TYPE_LABEL: Record<string, string> = {
  "31": "NIT",
  "13": "Cédula de ciudadanía",
  "22": "Cédula de extranjería",
  "47": "NIT (otro país)",
  "50": "NIT (DIAN)",
  "91": "NUIP",
  "11": "Registro civil",
  "12": "Tarjeta de identidad",
  "41": "Pasaporte",
  "42": "Documento extranjero",
};

export function CompanyProfileForm() {
  const { activeIssuer, updateIssuerProfile } = useAuth();
  const toast = useToast();
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);

  // Catálogos
  const { data: departments } = useCatalog(listDepartments);
  const { data: taxTypes } = useCatalog(listTaxTypes);
  const { data: taxRegimes } = useCatalog(listTaxRegimes);
  const { data: liabilityCatalog } = useCatalog(listLiabilityCodes);
  const { data: ciiuOptions, loading: loadingCiiu } = useCatalog(listCiiuOptions);
  const [municipalities, setMunicipalities] = useState<CatalogEntry[]>([]);
  const [ciiuPicker, setCiiuPicker] = useState("");

  // Estado del formulario — inicializado desde activeIssuer cuando se abre
  const [form, setForm] = useState<UpdateIssuerProfilePayload>({
    business_name: "",
    trade_name: "",
    department_code: "",
    municipality_code: "",
    address_line: "",
    email: "",
    phone: "",
    entity_type_code: "1",
    tax_scheme_code: "ZZ",
    liability_codes: [],
    tax_regime_code: null,
    industry_classification_codes: [],
    merchant_registration_number: null,
  });

  // Al abrir el formulario, copiar los valores actuales del emisor
  function openForm() {
    if (!activeIssuer) return;
    setForm({
      business_name: activeIssuer.business_name,
      trade_name: activeIssuer.trade_name ?? "",
      department_code: activeIssuer.department_code,
      municipality_code: activeIssuer.municipality_code,
      address_line: activeIssuer.address_line,
      email: activeIssuer.email,
      phone: activeIssuer.phone ?? "",
      entity_type_code: activeIssuer.entity_type_code ?? "1",
      tax_scheme_code: activeIssuer.tax_scheme_code ?? "ZZ",
      liability_codes: activeIssuer.liability_codes ?? [],
      tax_regime_code: activeIssuer.tax_regime_code ?? null,
      industry_classification_codes: activeIssuer.industry_classification_codes ?? [],
      merchant_registration_number: activeIssuer.merchant_registration_number ?? null,
    });
    setCiiuPicker("");
    setEditing(true);
  }

  // Cargar municipios cuando cambia el departamento
  useEffect(() => {
    if (!form.department_code) { setMunicipalities([]); return; }
    listMunicipalities(form.department_code)
      .then(setMunicipalities)
      .catch(() => setMunicipalities([]));
  }, [form.department_code]);

  function setField<K extends keyof UpdateIssuerProfilePayload>(key: K, value: UpdateIssuerProfilePayload[K]) {
    setForm((f) => ({ ...f, [key]: value }));
  }

  function toggleLiability(code: string) {
    setForm((f) => {
      const codes = f.liability_codes.includes(code)
        ? f.liability_codes.filter((c) => c !== code)
        : [...f.liability_codes, code];
      return { ...f, liability_codes: codes };
    });
  }

  const ciiuCodes = form.industry_classification_codes ?? [];
  const ciiuAtLimit = ciiuCodes.length >= 4;

  function handlePickCiiu(code: string) {
    if (code && !ciiuCodes.includes(code) && !ciiuAtLimit) {
      setField("industry_classification_codes", [...ciiuCodes, code]);
    }
    setCiiuPicker("");
  }

  function removeCiiu(code: string) {
    setField("industry_classification_codes", ciiuCodes.filter((c) => c !== code));
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!form.business_name.trim()) {
      toast.error("La razón social es obligatoria.");
      return;
    }
    setSaving(true);
    try {
      await updateIssuerProfile({
        ...form,
        trade_name: form.trade_name || "",
        phone: form.phone || "",
        tax_regime_code: form.tax_regime_code || null,
        merchant_registration_number: form.merchant_registration_number || null,
      });
      toast.success("Datos de la empresa actualizados.");
      setEditing(false);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudieron guardar los datos");
    } finally {
      setSaving(false);
    }
  }

  if (!activeIssuer) return null;

  const idTypeLabel = ID_TYPE_LABEL[activeIssuer.identification_type_code] ?? activeIssuer.identification_type_code;
  const formattedNit =
    activeIssuer.identification_type_code === "31"
      ? `${activeIssuer.nit}-${activeIssuer.check_digit}`
      : activeIssuer.nit;
  const envLabel = activeIssuer.environment === "1" ? "Producción" : "Habilitación";

  return (
    <Card className="flex flex-col gap-3 p-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xs font-semibold text-(--text-primary)">Datos de la empresa</h2>
        {!editing && (
          <Button type="button" variant="secondary" icon={<Pencil className="h-3.5 w-3.5" />} onClick={openForm}>
            Editar
          </Button>
        )}
      </div>

      {!editing ? (
        <div className="flex flex-col gap-3">
          {/* Identidad (inmutable) */}
          <div>
            <p className="text-sm font-bold leading-tight text-(--text-primary)">{activeIssuer.business_name}</p>
            {activeIssuer.trade_name && (
              <p className="mt-0.5 text-xs text-(--text-secondary)">{activeIssuer.trade_name}</p>
            )}
            <p className="mt-1 text-xs text-(--text-muted)">
              {idTypeLabel} {formattedNit} · {envLabel}
            </p>
          </div>

          {/* Campos editables */}
          <div className="grid grid-cols-2 gap-x-6 gap-y-2 border-t border-(--border-color) pt-3 text-xs">
            <Row label="Correo" value={activeIssuer.email} />
            {activeIssuer.phone && <Row label="Teléfono" value={activeIssuer.phone} />}
            <Row
              label="Dirección"
              value={[
                activeIssuer.address_line,
                activeIssuer.municipality_name ?? activeIssuer.municipality_code,
                activeIssuer.department_name ?? activeIssuer.department_code,
              ]
                .filter(Boolean)
                .join(", ")}
            />
            {activeIssuer.tax_scheme_code && (
              <Row
                label="Esquema tributario"
                value={`${activeIssuer.tax_scheme_code}${activeIssuer.tax_scheme_name ? ` – ${activeIssuer.tax_scheme_name}` : ""}`}
              />
            )}
            {activeIssuer.liability_codes && activeIssuer.liability_codes.length > 0 && (
              <Row label="Responsabilidades" value={activeIssuer.liability_codes.join(", ")} />
            )}
            {activeIssuer.merchant_registration_number && (
              <Row label="Matrícula mercantil" value={activeIssuer.merchant_registration_number} />
            )}
            {activeIssuer.industry_classification_codes && activeIssuer.industry_classification_codes.length > 0 && (
              <div className="col-span-2 flex flex-col gap-0.5">
                <span className="text-(--text-muted)">Códigos CIIU</span>
                <div className="flex flex-col gap-0.5">
                  {activeIssuer.industry_classification_codes.map((code) => {
                    const opt = ciiuOptions.find((o) => o.value === code);
                    const desc = opt?.label.includes(" - ") ? opt.label.slice(opt.label.indexOf(" - ") + 3) : undefined;
                    return (
                      <span key={code} className="text-(--text-primary)">
                        <span className="font-mono font-medium">{code}</span>
                        {desc && <span className="text-(--text-secondary)"> – {desc}</span>}
                      </span>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="flex flex-col gap-3">
          <div className="grid grid-cols-12 gap-3">
            <div className="col-span-6">
              <Input
                label="Razón social"
                value={form.business_name}
                onChange={(e) => setField("business_name", e.target.value)}
                required
              />
            </div>
            <div className="col-span-6">
              <Input
                label="Nombre comercial (opcional)"
                value={form.trade_name}
                onChange={(e) => setField("trade_name", e.target.value)}
              />
            </div>
            <div className="col-span-4">
              <label className="flex flex-col gap-1">
                <span className="text-xs font-medium text-(--text-secondary)">Tipo de persona</span>
                <select
                  value={form.entity_type_code}
                  onChange={(e) => setField("entity_type_code", e.target.value)}
                  className="rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1.5 text-xs text-(--text-primary)"
                >
                  {ENTITY_TYPE_OPTIONS.map((o) => <option key={o.code} value={o.code}>{o.code} – {o.name}</option>)}
                </select>
              </label>
            </div>
            <div className="col-span-8">
              <Input label="Correo" type="email" value={form.email} onChange={(e) => setField("email", e.target.value)} required />
            </div>
            <div className="col-span-4">
              <Input label="Teléfono (opcional)" value={form.phone} onChange={(e) => setField("phone", e.target.value)} />
            </div>
            <div className="col-span-4">
              <label className="flex flex-col gap-1">
                <span className="text-xs font-medium text-(--text-secondary)">Departamento</span>
                <select
                  value={form.department_code}
                  onChange={(e) => { setField("department_code", e.target.value); setField("municipality_code", ""); }}
                  className="rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1.5 text-xs text-(--text-primary)"
                >
                  <option value="">Seleccionar…</option>
                  {departments.map((d) => <option key={d.code} value={d.code}>{d.name}</option>)}
                </select>
              </label>
            </div>
            <div className="col-span-4">
              <label className="flex flex-col gap-1">
                <span className="text-xs font-medium text-(--text-secondary)">Municipio</span>
                <select
                  value={form.municipality_code}
                  onChange={(e) => setField("municipality_code", e.target.value)}
                  disabled={!form.department_code}
                  className="rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1.5 text-xs text-(--text-primary) disabled:opacity-50"
                >
                  <option value="">{form.department_code ? "Seleccionar…" : "Elige un departamento primero"}</option>
                  {municipalities.map((m) => <option key={m.code} value={m.code}>{m.name}</option>)}
                </select>
              </label>
            </div>
            <div className="col-span-12">
              <Input label="Dirección" value={form.address_line} onChange={(e) => setField("address_line", e.target.value)} required />
            </div>
            <div className="col-span-4">
              <label className="flex flex-col gap-1">
                <span className="text-xs font-medium text-(--text-secondary)">Tipo de impuesto del régimen</span>
                <select
                  value={form.tax_scheme_code}
                  onChange={(e) => setField("tax_scheme_code", e.target.value)}
                  className="rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1.5 text-xs text-(--text-primary)"
                >
                  {taxTypes.map((t) => <option key={t.code} value={t.code}>{t.code} – {t.name}</option>)}
                </select>
              </label>
            </div>
            <div className="col-span-4">
              <label className="flex flex-col gap-1">
                <span className="text-xs font-medium text-(--text-secondary)">Tipo de régimen (opcional)</span>
                <select
                  value={form.tax_regime_code ?? ""}
                  onChange={(e) => setField("tax_regime_code", e.target.value || null)}
                  className="rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1.5 text-xs text-(--text-primary)"
                >
                  <option value="">No aplica</option>
                  {taxRegimes.map((r) => <option key={r.code} value={r.code}>{r.code} – {r.name}</option>)}
                </select>
              </label>
            </div>
            <div className="col-span-4">
              <Input
                label="Matrícula mercantil (opcional)"
                value={form.merchant_registration_number ?? ""}
                onChange={(e) => setField("merchant_registration_number", e.target.value || null)}
              />
            </div>
            <div className="col-span-12 flex flex-col gap-1">
              <span className="text-xs font-medium text-(--text-secondary)">Responsabilidades fiscales</span>
              <div className="grid grid-cols-2 gap-1 rounded border border-(--border-color) bg-(--bg-primary) p-2">
                {liabilityCatalog.map((l) => (
                  <label key={l.code} className="flex cursor-pointer items-center gap-2 text-xs text-(--text-primary)">
                    <input
                      type="checkbox"
                      checked={form.liability_codes.includes(l.code)}
                      onChange={() => toggleLiability(l.code)}
                    />
                    <span className="font-mono">{l.code}</span>
                    <span className="text-(--text-secondary)">{l.name}</span>
                  </label>
                ))}
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
              {ciiuCodes.length > 0 && (
                <div className="flex flex-col gap-1">
                  {ciiuCodes.map((code) => {
                    const opt = ciiuOptions.find((o) => o.value === code);
                    const desc = opt?.label.includes(" - ") ? opt.label.slice(opt.label.indexOf(" - ") + 3) : "";
                    return (
                      <div
                        key={code}
                        className="flex items-center justify-between rounded border border-(--border-color) bg-(--bg-secondary) px-2 py-1.5 text-xs"
                      >
                        <span>
                          <span className="font-mono font-medium text-(--text-primary)">{code}</span>
                          {desc && <span className="text-(--text-secondary)"> – {desc}</span>}
                        </span>
                        <button
                          type="button"
                          onClick={() => removeCiiu(code)}
                          className="ml-2 shrink-0 text-(--text-muted) hover:text-(--color-danger)"
                        >
                          <X className="h-3 w-3" />
                        </button>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </div>

          <div className="flex gap-2 pt-1">
            <Button type="submit" loading={saving}>
              Guardar cambios
            </Button>
            <Button type="button" variant="secondary" onClick={() => setEditing(false)} disabled={saving}>
              Cancelar
            </Button>
          </div>
        </form>
      )}
    </Card>
  );
}

function Row({ label, value }: { label: string; value?: string }) {
  if (!value) return null;
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-(--text-muted)">{label}</span>
      <span className="text-(--text-primary)">{value}</span>
    </div>
  );
}
