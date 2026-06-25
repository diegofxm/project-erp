import { useState } from "react";
import { useSearchParams } from "react-router";
import { Tabs, type TabItem } from "../components/ui/Tabs";
import { ThemeToggle } from "../components/ThemeToggle";
import { useAuth } from "../context/AuthContext";
import { useTheme } from "../context/ThemeContext";

type SettingsTab = "general" | "account" | "company";

const TABS: TabItem[] = [
  { id: "general", label: "General" },
  { id: "account", label: "Mi cuenta" },
  { id: "company", label: "Empresa", disabled: true },
];

function resolveInitialTab(value: string | null): SettingsTab {
  return value === "account" ? "account" : "general";
}

// Configuración general del aplicativo y de la cuenta del usuario — la configuración de la
// empresa (software/certificado/numeración) queda deliberadamente fuera de aquí por ahora,
// mostrada solo como pestaña "próximamente" (mismo patrón que Sidebar.tsx ya usa).
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
          <div className="flex items-center justify-between p-4">
            <div>
              <p className="text-xs font-medium text-(--text-primary)">Tema de la interfaz</p>
              <p className="text-xs text-(--text-secondary)">{theme === "light" ? "Claro" : "Oscuro"}</p>
            </div>
            <ThemeToggle />
          </div>
        )}

        {activeTab === "account" && (
          <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-2 p-4 text-xs">
            <dt className="text-(--text-secondary)">Nombre</dt>
            <dd className="text-(--text-primary)">{user?.name}</dd>
            <dt className="text-(--text-secondary)">Correo</dt>
            <dd className="text-(--text-primary)">{user?.email}</dd>
          </dl>
        )}
      </div>
    </div>
  );
}
