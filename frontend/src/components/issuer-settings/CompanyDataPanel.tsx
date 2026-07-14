import { Building2 } from "lucide-react";
import { useAuth } from "../../context/AuthContext";
import { Card } from "../ui/Card";

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
  "42": "Tipo de documento extranjero",
};

function Row({ label, value }: { label: string; value?: string | null }) {
  if (!value) return null;
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-[10px] font-medium uppercase tracking-wide text-(--text-muted)">{label}</span>
      <span className="text-xs text-(--text-primary)">{value}</span>
    </div>
  );
}

export function CompanyDataPanel() {
  const { activeIssuer } = useAuth();
  if (!activeIssuer) return null;

  const nit =
    activeIssuer.identification_type_code === "31"
      ? `${activeIssuer.nit}-${activeIssuer.check_digit}`
      : activeIssuer.nit;

  const idTypeName = ID_TYPE[activeIssuer.identification_type_code] ?? activeIssuer.identification_type_code;
  const entityTypeName = activeIssuer.entity_type_code ? (ENTITY_TYPE[activeIssuer.entity_type_code] ?? activeIssuer.entity_type_code) : undefined;

  const location = [
    activeIssuer.address_line,
    activeIssuer.municipality_name ?? activeIssuer.municipality_code,
    activeIssuer.department_name ?? activeIssuer.department_code,
  ]
    .filter(Boolean)
    .join(", ");

  const liabilities = activeIssuer.liability_codes?.join(", ");
  const ciiu = activeIssuer.industry_classification_codes?.join(", ");

  return (
    <Card className="flex flex-col gap-3 p-4">
      <h2 className="flex items-center gap-1.5 text-xs font-semibold text-(--text-primary)">
        <Building2 className="h-3.5 w-3.5 shrink-0 text-(--accent-primary)" />
        Datos de la empresa
      </h2>
      <p className="text-xs text-(--text-secondary)">
        Información registrada ante la DIAN. Para modificarla crea una empresa nueva o contacta al soporte.
      </p>
      <div className="grid grid-cols-2 gap-x-6 gap-y-3">
        <Row label="Razón social" value={activeIssuer.business_name} />
        <Row label="Nombre comercial" value={activeIssuer.trade_name} />
        <Row label={idTypeName} value={nit} />
        <Row label="Tipo de persona" value={entityTypeName} />
        <Row label="Correo electrónico" value={activeIssuer.email} />
        <Row label="Teléfono" value={activeIssuer.phone} />
        <div className="col-span-2">
          <Row label="Dirección" value={location} />
        </div>
        <Row label="Régimen tributario" value={activeIssuer.tax_regime_code ?? undefined} />
        <Row label="Esquema de impuesto" value={activeIssuer.tax_scheme_code ? `${activeIssuer.tax_scheme_code} – ${activeIssuer.tax_scheme_name}` : undefined} />
        <div className="col-span-2">
          <Row label="Responsabilidades fiscales" value={liabilities} />
        </div>
        <Row label="Códigos CIIU" value={ciiu} />
        <Row label="Matrícula mercantil" value={activeIssuer.merchant_registration_number ?? undefined} />
      </div>
    </Card>
  );
}
