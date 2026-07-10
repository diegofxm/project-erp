import { Building2 } from "lucide-react";
import { SoftwareCertificateForm } from "../components/issuer-settings/SoftwareCertificateForm";
import { LogoForm } from "../components/issuer-settings/LogoForm";
import { PublicRegistrationPanel } from "../components/issuer-settings/PublicRegistrationPanel";
import { NumberingRangesPanel } from "../components/issuer-settings/NumberingRangesPanel";

export function SettingsCompanyPage() {
  return (
    <div className="p-4">
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <Building2 className="h-4 w-4 shrink-0 text-(--text-secondary)" />
        Empresa
      </h1>
      <div className="flex flex-col gap-4">
        <SoftwareCertificateForm />
        <LogoForm />
        <PublicRegistrationPanel />
        <NumberingRangesPanel />
      </div>
    </div>
  );
}
