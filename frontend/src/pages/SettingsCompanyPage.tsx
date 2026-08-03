import { Building2 } from "lucide-react";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { BrandColorForm } from "../components/issuer-settings/BrandColorForm";
import { CompanyProfileForm } from "../components/issuer-settings/CompanyProfileForm";
import { LogoForm } from "../components/issuer-settings/LogoForm";
import { NumberingRangesPanel } from "../components/issuer-settings/NumberingRangesPanel";
import { PublicRegistrationPanel } from "../components/issuer-settings/PublicRegistrationPanel";
import { SoftwareCertificateForm } from "../components/issuer-settings/SoftwareCertificateForm";

export function SettingsCompanyPage() {
  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Configuración", to: "/settings/general" }, { label: "Empresa" }]} />
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <Building2 className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        Empresa
      </h1>
      <div className="flex flex-col gap-4">
        <CompanyProfileForm />
        <SoftwareCertificateForm />
        <LogoForm />
        <BrandColorForm />
        <PublicRegistrationPanel />
        <NumberingRangesPanel />
      </div>
    </div>
  );
}
