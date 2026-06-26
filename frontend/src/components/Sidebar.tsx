import { useState, type ReactNode } from "react";
import { NavLink } from "react-router";
import { FileText, Hash, Home, Menu, Package, Settings, Users } from "lucide-react";

const STORAGE_KEY = "apidian.sidebarCollapsed";

interface NavItem {
  to: string;
  label: string;
  icon: ReactNode;
  enabled: boolean;
}

// Resto de items deshabilitados a propósito en esta fase — todavía no tienen página propia
// (ver docs/frontend-architecture.md, "Pendiente"). Se muestran igual para que la información
// arquitectónica (qué va a vivir en el sidebar) sea visible desde ya.
const NAV_ITEMS: NavItem[] = [
  { to: "/", label: "Inicio", icon: <Home className="h-3.5 w-3.5" />, enabled: true },
  { to: "/invoices", label: "Facturas", icon: <FileText className="h-3.5 w-3.5" />, enabled: false },
  { to: "/customers", label: "Clientes", icon: <Users className="h-3.5 w-3.5" />, enabled: false },
  { to: "/products", label: "Productos", icon: <Package className="h-3.5 w-3.5" />, enabled: false },
  { to: "/numbering", label: "Numeración", icon: <Hash className="h-3.5 w-3.5" />, enabled: false },
  { to: "/settings", label: "Configuración", icon: <Settings className="h-3.5 w-3.5" />, enabled: true },
];

export function Sidebar() {
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem(STORAGE_KEY) === "true");

  function toggle() {
    setCollapsed((current) => {
      const next = !current;
      localStorage.setItem(STORAGE_KEY, String(next));
      return next;
    });
  }

  return (
    <aside
      className={`flex h-full flex-col border-r border-(--border-color) bg-(--bg-secondary) transition-all duration-300 ${collapsed ? "w-10" : "w-56"}`}
    >
      <div className="flex h-10 items-center justify-end border-b border-(--border-light) px-2">
        <button type="button" onClick={toggle} className="rounded p-1.5 text-(--text-secondary) hover:bg-(--bg-hover)">
          <Menu className="h-4 w-4" />
        </button>
      </div>
      <nav className="flex flex-col gap-0.5 p-1.5">
        {NAV_ITEMS.map((item) =>
          item.enabled ? (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                `flex items-center gap-2 rounded px-2 py-1.5 text-xs font-medium ${
                  isActive
                    ? "bg-(--bg-selected) text-(--accent-primary)"
                    : "text-(--text-secondary) hover:bg-(--bg-hover)"
                }`
              }
            >
              {item.icon}
              {!collapsed && <span>{item.label}</span>}
            </NavLink>
          ) : (
            <span
              key={item.to}
              title={`${item.label} (próximamente)`}
              className="flex cursor-not-allowed items-center gap-2 rounded px-2 py-1.5 text-xs text-(--text-muted) opacity-60"
            >
              {item.icon}
              {!collapsed && <span>{item.label}</span>}
            </span>
          ),
        )}
      </nav>
    </aside>
  );
}
