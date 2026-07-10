import { SlidersHorizontal } from "lucide-react";
import { ThemeToggle } from "../components/ThemeToggle";
import { Card } from "../components/ui/Card";
import { useTheme } from "../context/ThemeContext";

export function SettingsGeneralPage() {
  const { theme } = useTheme();

  return (
    <div className="p-4">
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <SlidersHorizontal className="h-4 w-4 shrink-0 text-(--text-secondary)" />
        General
      </h1>
      <div className="flex flex-wrap gap-3">
        <Card className="flex items-center gap-6 p-3">
          <div>
            <p className="text-xs font-medium text-(--text-primary)">Tema de la interfaz</p>
            <p className="text-xs text-(--text-secondary)">{theme === "light" ? "Claro" : "Oscuro"}</p>
          </div>
          <ThemeToggle />
        </Card>
      </div>
    </div>
  );
}
