import { useState } from "react";
import { Tabs, type TabItem } from "../ui/Tabs";
import { Button } from "../ui/Button";
import type { CreateCompanyPayload } from "../../lib/types";
import { IdentificationStep } from "./IdentificationStep";
import { LocationStep } from "./LocationStep";
import { TaxStep } from "./TaxStep";
import { ReviewStep } from "./ReviewStep";

export type TabId = "identification" | "location" | "tax" | "review";

export interface MissingField {
  tab: TabId;
  label: string;
}

const TABS: TabItem[] = [
  { id: "identification", label: "Identificación" },
  { id: "location", label: "Ubicación y contacto" },
  { id: "tax", label: "Información tributaria" },
  { id: "review", label: "Revisión" },
];

const EMPTY_FORM: CreateCompanyPayload = {
  nit: "",
  check_digit: "",
  business_name: "",
  trade_name: "",
  identification_type_code: "31",
  department_code: "",
  municipality_code: "",
  address_line: "",
  email: "",
  phone: "",
  environment: "2",
  entity_type_code: "",
  tax_scheme_code: "ZZ",
  liability_codes: ["R-99-PN"],
  tax_regime_code: "",
  industry_classification_codes: [],
  merchant_registration_number: "",
};

// Solo se exigen aquí los campos que ya eran obligatorios en el formulario plano anterior —
// issuers.Service.validateIssuer en el backend solo exige nit/business_name/environment, el
// resto (check_digit/identification_type_code/department_code/municipality_code/address_line/
// email) son obligatorios a nivel de UX para que la empresa quede realmente lista para emitir,
// no por una regla del backend.
function computeMissingFields(form: CreateCompanyPayload): MissingField[] {
  const missing: MissingField[] = [];
  if (!form.nit.trim()) missing.push({ tab: "identification", label: "NIT" });
  if (!form.check_digit.trim()) missing.push({ tab: "identification", label: "Dígito de verificación" });
  if (!form.business_name.trim()) missing.push({ tab: "identification", label: "Razón social" });
  if (!form.identification_type_code) missing.push({ tab: "identification", label: "Tipo de identificación" });
  if (!form.environment) missing.push({ tab: "identification", label: "Ambiente" });
  if (!form.department_code) missing.push({ tab: "location", label: "Departamento" });
  if (!form.municipality_code) missing.push({ tab: "location", label: "Municipio" });
  if (!form.address_line.trim()) missing.push({ tab: "location", label: "Dirección" });
  if (!form.email.trim()) missing.push({ tab: "location", label: "Correo de la empresa" });
  return missing;
}

interface CompanyFormProps {
  onSubmit: (payload: CreateCompanyPayload) => void;
  loading: boolean;
  onCancel?: () => void;
}

export function CompanyForm({ onSubmit, loading, onCancel }: CompanyFormProps) {
  const [form, setForm] = useState<CreateCompanyPayload>(EMPTY_FORM);
  const [activeTab, setActiveTab] = useState<TabId>("identification");

  function setField<K extends keyof CreateCompanyPayload>(key: K, value: CreateCompanyPayload[K]) {
    setForm((current) => ({ ...current, [key]: value }));
  }

  const missingFields = computeMissingFields(form);
  const tabIndex = TABS.findIndex((t) => t.id === activeTab);

  function goTo(direction: 1 | -1) {
    const next = TABS[tabIndex + direction];
    if (next) setActiveTab(next.id as TabId);
  }

  function handleSubmit() {
    if (missingFields.length > 0) return;
    onSubmit({
      ...form,
      tax_regime_code: form.tax_regime_code || undefined,
      merchant_registration_number: form.merchant_registration_number || undefined,
      trade_name: form.trade_name || undefined,
      phone: form.phone || undefined,
    });
  }

  return (
    <div className="flex flex-col gap-3">
      <Tabs tabs={TABS} activeId={activeTab} onChange={(id) => setActiveTab(id as TabId)} />

      {activeTab === "identification" && <IdentificationStep form={form} setField={setField} />}
      {activeTab === "location" && <LocationStep form={form} setField={setField} />}
      {activeTab === "tax" && <TaxStep form={form} setField={setField} />}
      {activeTab === "review" && (
        <ReviewStep form={form} missingFields={missingFields} onEditTab={setActiveTab} onSubmit={handleSubmit} loading={loading} />
      )}

      <div className="flex items-center gap-2 px-4 pb-4">
        {onCancel && (
          <Button type="button" variant="ghost" onClick={onCancel}>
            Cancelar
          </Button>
        )}
        <div className="flex-1" />
        {tabIndex > 0 && (
          <Button type="button" variant="secondary" onClick={() => goTo(-1)}>
            Atrás
          </Button>
        )}
        {activeTab !== "review" && <Button type="button" onClick={() => goTo(1)}>Siguiente</Button>}
      </div>
    </div>
  );
}
