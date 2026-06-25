import { Moon, Sun } from "lucide-react";
import { useTheme } from "../context/ThemeContext";

// Sugerencia #3 del design system: la paleta oscura ya está modelada, lo que faltaba en el
// proyecto original era justo este control. Vive en Configuración → General (no en el Navbar,
// que ya no lleva controles propios fuera del menú de usuario).
export function ThemeToggle() {
  const { theme, toggleTheme } = useTheme();
  return (
    <button
      type="button"
      onClick={toggleTheme}
      title={theme === "light" ? "Cambiar a tema oscuro" : "Cambiar a tema claro"}
      className="rounded border border-(--border-color) p-1.5 text-(--text-secondary) hover:bg-(--bg-hover)"
    >
      {theme === "light" ? <Moon className="h-4 w-4" /> : <Sun className="h-4 w-4" />}
    </button>
  );
}
