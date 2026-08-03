import { SlidersHorizontal } from "lucide-react";
import { ThemeToggle } from "../components/ThemeToggle";
import { Card } from "../components/ui/Card";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { useTheme } from "../context/ThemeContext";
import { usePdfFormat } from "../lib/usePdfFormat";

export function SettingsGeneralPage() {
  const { theme } = useTheme();
  const [pdfFormat, setPdfFormat] = usePdfFormat();

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Configuración", to: "/settings/general" }, { label: "General" }]} />
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <SlidersHorizontal className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        General
      </h1>
      <div className="flex flex-wrap gap-3">
        <Card className="flex items-center justify-between gap-6 p-3">
          <div>
            <p className="text-xs font-medium text-(--text-primary)">Tema de la interfaz</p>
            <p className="text-xs text-(--text-secondary)">{theme === "light" ? "Claro" : "Oscuro"}</p>
          </div>
          <ThemeToggle />
        </Card>

        <Card className="flex items-center justify-between gap-6 p-3">
          <div>
            <p className="text-xs font-medium text-(--text-primary)">Formato de PDF predeterminado</p>
            <p className="text-xs text-(--text-secondary)">Se aplica al generar y enviar documentos</p>
          </div>
          <select
            value={pdfFormat}
            onChange={(e) => setPdfFormat(e.target.value as "full_a4" | "half_a4")}
            className="rounded border border-(--border-color) bg-(--bg-primary) px-2 py-1 text-xs text-(--text-primary) transition-colors"
          >
            <option value="full_a4">A4 completo</option>
            <option value="half_a4">Media página (2 copias)</option>
          </select>
        </Card>
      </div>
    </div>
  );
}
