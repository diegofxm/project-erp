import { useEffect, useState, type ReactNode } from "react";
import { Building2 } from "lucide-react";
import {
  listDepartments,
  listIdentificationTypes,
  listLiabilityCodes,
  listMunicipalities,
  listTaxRegimes,
  listTaxTypes,
} from "../../lib/catalogs";
import type { CatalogEntry, CreateCompanyPayload, Municipality } from "../../lib/types";
import { Banner } from "../ui/Banner";
import { Button } from "../ui/Button";
import type { MissingField, TabId } from "./CompanyForm";

function findLabel(entries: CatalogEntry[], code: string | undefined): string {
  if (!code) return "—";
  const entry = entries.find((e) => e.code === code);
  return entry ? `${entry.code} — ${entry.name}` : code;
}

interface ReviewStepProps {
  form: CreateCompanyPayload;
  missingFields: MissingField[];
  onEditTab: (tab: TabId) => void;
  onSubmit: () => void;
  loading: boolean;
}

export function ReviewStep({ form, missingFields, onEditTab, onSubmit, loading }: ReviewStepProps) {
  const [departments, setDepartments] = useState<CatalogEntry[]>([]);
  const [municipalities, setMunicipalities] = useState<Municipality[]>([]);
  const [identificationTypes, setIdentificationTypes] = useState<CatalogEntry[]>([]);
  const [taxTypes, setTaxTypes] = useState<CatalogEntry[]>([]);
  const [taxRegimes, setTaxRegimes] = useState<CatalogEntry[]>([]);
  const [liabilityCodes, setLiabilityCodes] = useState<CatalogEntry[]>([]);

  useEffect(() => {
    listDepartments().then(setDepartments);
    listIdentificationTypes().then(setIdentificationTypes);
    listTaxTypes().then(setTaxTypes);
    listTaxRegimes().then(setTaxRegimes);
    listLiabilityCodes().then(setLiabilityCodes);
  }, []);

  useEffect(() => {
    if (!form.department_code) {
      setMunicipalities([]);
      return;
    }
    listMunicipalities(form.department_code).then(setMunicipalities);
  }, [form.department_code]);

  const liabilityLabels = (form.liability_codes ?? []).map((code) => findLabel(liabilityCodes, code)).join(", ") || "—";
  const ciiuLabels = (form.industry_classification_codes ?? []).join(", ") || "—";

  return (
    <div className="flex flex-col gap-4 p-4">
      {missingFields.length > 0 && (
        <Banner tone="danger">
          <p className="mb-1 font-medium">Faltan datos obligatorios:</p>
          <ul className="flex flex-col gap-0.5">
            {missingFields.map((f) => (
              <li key={`${f.tab}-${f.label}`}>
                <button type="button" onClick={() => onEditTab(f.tab)} className="underline hover:text-(--color-danger)">
                  {f.label}
                </button>
              </li>
            ))}
          </ul>
        </Banner>
      )}

      <ReviewSection title="Identificación" onEdit={() => onEditTab("identification")}>
        <ReviewRow label="NIT" value={form.nit ? `${form.nit}-${form.check_digit || "?"}` : "—"} />
        <ReviewRow label="Tipo de identificación" value={findLabel(identificationTypes, form.identification_type_code)} />
        <ReviewRow label="Razón social" value={form.business_name || "—"} />
        <ReviewRow label="Nombre comercial" value={form.trade_name || "—"} />
        <ReviewRow label="Ambiente" value={form.environment === "1" ? "Producción" : "Habilitación"} />
      </ReviewSection>

      <ReviewSection title="Ubicación y contacto" onEdit={() => onEditTab("location")}>
        <ReviewRow label="Departamento" value={findLabel(departments, form.department_code)} />
        <ReviewRow label="Municipio" value={findLabel(municipalities, form.municipality_code)} />
        <ReviewRow label="Dirección" value={form.address_line || "—"} />
        <ReviewRow label="Correo" value={form.email || "—"} />
        <ReviewRow label="Teléfono" value={form.phone || "—"} />
      </ReviewSection>

      <ReviewSection title="Información tributaria" onEdit={() => onEditTab("tax")}>
        <ReviewRow label="Tipo de impuesto" value={findLabel(taxTypes, form.tax_scheme_code)} />
        <ReviewRow label="Tipo de régimen" value={findLabel(taxRegimes, form.tax_regime_code)} />
        <ReviewRow label="Responsabilidades fiscales" value={liabilityLabels} />
        <ReviewRow label="Códigos CIIU" value={ciiuLabels} />
        <ReviewRow label="Matrícula mercantil" value={form.merchant_registration_number || "—"} />
      </ReviewSection>

      <Button onClick={onSubmit} loading={loading} disabled={missingFields.length > 0} icon={<Building2 className="h-3.5 w-3.5" />}>
        Crear empresa
      </Button>
    </div>
  );
}

function ReviewSection({ title, onEdit, children }: { title: string; onEdit: () => void; children: ReactNode }) {
  return (
    <div className="rounded border border-(--border-color)">
      <div className="flex items-center justify-between border-b border-(--border-light) bg-(--bg-tertiary) px-3 py-1.5">
        <span className="text-xs font-semibold text-(--text-primary)">{title}</span>
        <button type="button" onClick={onEdit} className="text-xs text-(--accent-primary) hover:text-(--accent-hover)">
          Editar
        </button>
      </div>
      <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 p-3 text-xs">{children}</dl>
    </div>
  );
}

function ReviewRow({ label, value }: { label: string; value: string }) {
  return (
    <>
      <dt className="text-(--text-secondary)">{label}</dt>
      <dd className="text-(--text-primary)">{value}</dd>
    </>
  );
}
