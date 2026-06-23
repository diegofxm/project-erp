import { Moon, Sun } from "lucide-react";
import { useTheme } from "../context/ThemeContext";

// Sugerencia #3 del design system: la paleta oscura ya está modelada, lo que faltaba en el
// proyecto original era justo este control.
export function ThemeToggle() {
  const { theme, toggleTheme } = useTheme();
  return (
    <button
      type="button"
      onClick={toggleTheme}
      title={theme === "light" ? "Cambiar a tema oscuro" : "Cambiar a tema claro"}
      className="rounded p-1.5 text-(--navbar-text) opacity-80 hover:bg-white/10 hover:opacity-100"
    >
      {theme === "light" ? <Moon className="h-4 w-4" /> : <Sun className="h-4 w-4" />}
    </button>
  );
}
