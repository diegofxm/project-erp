import { useAuth } from "../../context/AuthContext";

const ENTITY_TYPE: Record<string, string> = {
  "1": "Persona jurídica y asimilada",
  "2": "Persona natural y asimilada",
};

const ID_TYPE: Record<string, string> = {
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

const TAX_REGIME: Record<string, string> = {
  "00": "No aplica",
  "02": "Gravamen a los movimientos financieros",
  "03": "Impuesto al patrimonio",
  "04": "Renta y complementario – régimen especial",
  "05": "Renta y complementario – régimen ordinario",
};

const LIABILITY_CODE: Record<string, string> = {
  "O-13": "Gran contribuyente",
  "O-15": "Autorretenedor",
  "O-23": "Agente de retención IVA",
  "O-47": "Régimen simple de tributación",
  "R-99-PJ": "No aplica (persona jurídica)",
  "R-99-PN": "No aplica (persona natural)",
};

function Field({ label, value }: { label: string; value?: string | null }) {
  if (!value) return null;
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-[10px] font-semibold uppercase tracking-wide text-(--text-muted)">{label}</span>
      <span className="text-xs text-(--text-primary)">{value}</span>
    </div>
  );
}

export function CompanyDataPanel() {
  const { activeIssuer } = useAuth();
  if (!activeIssuer) return null;

  const idTypeLabel = ID_TYPE[activeIssuer.identification_type_code] ?? activeIssuer.identification_type_code;

  const nit =
    activeIssuer.identification_type_code === "31"
      ? `${activeIssuer.nit}-${activeIssuer.check_digit}`
      : activeIssuer.nit;

  const location = [
    activeIssuer.address_line,
    activeIssuer.municipality_name ?? activeIssuer.municipality_code,
    activeIssuer.department_name ?? activeIssuer.department_code,
  ]
    .filter(Boolean)
    .join(", ");

  const entityType = activeIssuer.entity_type_code ? (ENTITY_TYPE[activeIssuer.entity_type_code] ?? activeIssuer.entity_type_code) : undefined;

  const taxRegime = activeIssuer.tax_regime_code
    ? (TAX_REGIME[activeIssuer.tax_regime_code] ?? activeIssuer.tax_regime_code)
    : undefined;

  const liabilities = activeIssuer.liability_codes
    ?.map((c) => LIABILITY_CODE[c] ?? c)
    .join(", ");

  const taxScheme = activeIssuer.tax_scheme_code
    ? `${activeIssuer.tax_scheme_name ?? activeIssuer.tax_scheme_code}`
    : undefined;

  const ciiu = activeIssuer.industry_classification_codes?.join(", ");

  return (
    <div className="rounded-lg border border-(--border-color) bg-(--bg-secondary) px-5 py-4">
      {/* Identidad principal */}
      <div className="mb-4">
        <h2 className="text-sm font-bold text-(--text-primary) leading-tight">{activeIssuer.business_name}</h2>
        {activeIssuer.trade_name && (
          <p className="mt-0.5 text-xs text-(--text-secondary)">{activeIssuer.trade_name}</p>
        )}
        <p className="mt-1 text-xs text-(--text-muted)">
          {idTypeLabel} {nit}
          {location ? ` · ${location}` : ""}
        </p>
      </div>

      {/* Detalle */}
      <div className="grid grid-cols-2 gap-x-8 gap-y-3 border-t border-(--border-color) pt-4">
        <Field label="Correo electrónico" value={activeIssuer.email} />
        <Field label="Teléfono" value={activeIssuer.phone} />
        <Field label="Tipo de persona" value={entityType} />
        <Field label="Régimen tributario" value={taxRegime} />
        <div className="col-span-2">
          <Field label="Responsabilidades fiscales" value={liabilities} />
        </div>
        <Field label="Esquema de impuesto" value={taxScheme} />
        <Field label="Matrícula mercantil" value={activeIssuer.merchant_registration_number ?? undefined} />
        {ciiu && (
          <div className="col-span-2">
            <Field label="Códigos CIIU" value={ciiu} />
          </div>
        )}
      </div>
    </div>
  );
}
