import { useState } from "react";
import { useSearchParams } from "react-router";
import { Tabs, type TabItem } from "../components/ui/Tabs";
import { ThemeToggle } from "../components/ThemeToggle";
import { SoftwareCertificateForm } from "../components/issuer-settings/SoftwareCertificateForm";
import { LogoForm } from "../components/issuer-settings/LogoForm";
import { NumberingRangesPanel } from "../components/issuer-settings/NumberingRangesPanel";
import { PublicRegistrationPanel } from "../components/issuer-settings/PublicRegistrationPanel";
import { useAuth } from "../context/AuthContext";
import { useTheme } from "../context/ThemeContext";

type SettingsTab = "general" | "account" | "company";

const TABS: TabItem[] = [
  { id: "general", label: "General" },
  { id: "account", label: "Mi cuenta" },
  { id: "company", label: "Empresa" },
];

function resolveInitialTab(value: string | null): SettingsTab {
  if (value === "account" || value === "company") return value;
  return "general";
}

// Configuración general del aplicativo, de la cuenta del usuario, y de la empresa activa
// (software/certificado/numeración) — esto último deliberadamente NO incluye datos de
// identificación/ubicación/tributarios (ya completos desde CompanyForm en la creación).
export function SettingsPage() {
  const [searchParams] = useSearchParams();
  const [activeTab, setActiveTab] = useState<SettingsTab>(resolveInitialTab(searchParams.get("tab")));
  const { user } = useAuth();
  const { theme } = useTheme();

  return (
    <div className="p-4">
      <h1 className="mb-3 text-sm font-semibold text-(--text-primary)">Configuración</h1>
      <div className="rounded-lg border border-(--border-light) bg-(--bg-secondary)">
        <Tabs tabs={TABS} activeId={activeTab} onChange={(id) => setActiveTab(id as SettingsTab)} />

        {activeTab === "general" && (
          // flex-wrap + tarjetas sin ancho forzado: a diferencia de un formulario con varios
          // campos, aquí hay un solo control — meterlo en una grilla a todo el ancho solo deja
          // un hueco enorme entre la etiqueta y el switch. La tarjeta se ajusta a su contenido.
          <div className="flex flex-wrap gap-3 p-4">
            <div className="flex items-center gap-6 rounded border border-(--border-color) p-3">
              <div>
                <p className="text-xs font-medium text-(--text-primary)">Tema de la interfaz</p>
                <p className="text-xs text-(--text-secondary)">{theme === "light" ? "Claro" : "Oscuro"}</p>
              </div>
              <ThemeToggle />
            </div>
          </div>
        )}

        {activeTab === "account" && (
          <div className="flex flex-wrap gap-3 p-4">
            <div className="rounded border border-(--border-color) p-3 text-xs">
              <p className="text-(--text-secondary)">Nombre</p>
              <p className="text-(--text-primary)">{user?.name}</p>
            </div>
            <div className="rounded border border-(--border-color) p-3 text-xs">
              <p className="text-(--text-secondary)">Correo</p>
              <p className="text-(--text-primary)">{user?.email}</p>
            </div>
          </div>
        )}

        {activeTab === "company" && (
          <div className="flex flex-col gap-4 p-4">
            <SoftwareCertificateForm />
            <LogoForm />
            <PublicRegistrationPanel />
            <NumberingRangesPanel />
          </div>
        )}
      </div>
    </div>
  );
}
