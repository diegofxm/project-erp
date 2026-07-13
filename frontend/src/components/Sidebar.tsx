import { useState, type ReactNode } from "react";
import { NavLink, useLocation } from "react-router";
import { Files, Home, Menu, Package, Settings, Users } from "lucide-react";

const COLLAPSED_KEY = "apidian.sidebarCollapsed";

interface NavItem {
  to: string;
  label: string;
  icon: ReactNode;
  activePrefix?: string;
  end?: boolean;
}

const NAV_ITEMS: NavItem[] = [
  { to: "/", label: "Inicio", icon: <Home className="h-3.5 w-3.5" />, end: true },
  { to: "/documents/invoices", label: "Documentos", icon: <Files className="h-3.5 w-3.5" />, activePrefix: "/documents" },
  { to: "/customers", label: "Clientes", icon: <Users className="h-3.5 w-3.5" />, activePrefix: "/customers" },
  { to: "/products", label: "Productos", icon: <Package className="h-3.5 w-3.5" />, activePrefix: "/products" },
  { to: "/settings", label: "Configuración", icon: <Settings className="h-3.5 w-3.5" />, activePrefix: "/settings" },
];

export function Sidebar() {
  const location = useLocation();
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem(COLLAPSED_KEY) === "true");

  function toggleCollapsed() {
    setCollapsed((current) => {
      const next = !current;
      localStorage.setItem(COLLAPSED_KEY, String(next));
      return next;
    });
  }

  return (
    <aside
      className={`flex h-full flex-col overflow-hidden border-r border-(--border-color) bg-(--bg-secondary) transition-all duration-200 ${
        collapsed ? "w-10" : "w-48"
      }`}
    >
      <div className="flex h-10 shrink-0 items-center border-b border-(--border-light) px-2">
        <button
          type="button"
          onClick={toggleCollapsed}
          className="rounded p-1.5 text-(--text-secondary) transition-colors hover:bg-(--bg-hover)"
          title={collapsed ? "Expandir menú" : "Colapsar menú"}
        >
          <Menu className="h-4 w-4" />
        </button>
      </div>
      <nav className="flex flex-col gap-0.5 p-1.5">
        {NAV_ITEMS.map((item) => {
          const active = item.activePrefix
            ? location.pathname.startsWith(item.activePrefix)
            : item.end
            ? location.pathname === "/"
            : location.pathname.startsWith(item.to);
          return (
            <NavLink
              key={item.to}
              to={item.to}
              title={collapsed ? item.label : undefined}
              className={`flex items-center rounded py-1.5 text-xs font-medium transition-colors ${
                collapsed ? "justify-center" : "gap-2 px-2"
              } ${
                active
                  ? "bg-(--bg-selected) text-(--accent-primary)"
                  : "text-(--text-secondary) hover:bg-(--bg-hover)"
              }`}
            >
              {item.icon}
              {!collapsed && <span className="whitespace-nowrap">{item.label}</span>}
            </NavLink>
          );
        })}
      </nav>
    </aside>
  );
}
