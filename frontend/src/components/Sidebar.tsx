import { useState, type ReactNode } from "react";
import { NavLink } from "react-router";
import { ChevronRight, FileMinus, FilePlus, FileText, Files, Home, Menu, Package, Settings, Users } from "lucide-react";

const COLLAPSED_KEY = "apidian.sidebarCollapsed";
const EXPANDED_GROUPS_KEY = "apidian.sidebarExpandedGroups";

interface NavLeaf {
  to: string;
  label: string;
  icon: ReactNode;
  enabled: boolean;
}

interface NavGroup {
  label: string;
  icon: ReactNode;
  children: NavLeaf[];
}

function isGroup(entry: NavLeaf | NavGroup): entry is NavGroup {
  return "children" in entry;
}

// Documentos es un grupo expandible (mismo patrón de árbol que ya describe el design system,
// secc. 6: chevron que rota 90° al expandir) — el backend ya maneja Factura/Nota Crédito/Nota
// Débito bajo un mismo documents.Service, así que la navegación los agrupa igual. Factura
// Electrónica ya tiene página propia; Nota Crédito/Nota Débito siguen deshabilitadas
// ("próximamente") hasta que tengan la suya, ver docs/frontend-architecture.md.
const NAV_ITEMS: (NavLeaf | NavGroup)[] = [
  { to: "/", label: "Inicio", icon: <Home className="h-3.5 w-3.5" />, enabled: true },
  {
    label: "Documentos",
    icon: <Files className="h-3.5 w-3.5" />,
    children: [
      { to: "/documents/invoices", label: "Factura Electrónica", icon: <FileText className="h-3.5 w-3.5" />, enabled: true },
      { to: "/documents/credit-notes", label: "Nota Crédito", icon: <FileMinus className="h-3.5 w-3.5" />, enabled: false },
      { to: "/documents/debit-notes", label: "Nota Débito", icon: <FilePlus className="h-3.5 w-3.5" />, enabled: false },
    ],
  },
  { to: "/customers", label: "Clientes", icon: <Users className="h-3.5 w-3.5" />, enabled: true },
  { to: "/products", label: "Productos", icon: <Package className="h-3.5 w-3.5" />, enabled: true },
  { to: "/settings", label: "Configuración", icon: <Settings className="h-3.5 w-3.5" />, enabled: true },
];

function readExpandedGroups(): Set<string> {
  try {
    const raw = localStorage.getItem(EXPANDED_GROUPS_KEY);
    return new Set(raw ? (JSON.parse(raw) as string[]) : ["Documentos"]);
  } catch {
    return new Set(["Documentos"]);
  }
}

export function Sidebar() {
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem(COLLAPSED_KEY) === "true");
  const [expandedGroups, setExpandedGroups] = useState(readExpandedGroups);

  function toggleCollapsed() {
    setCollapsed((current) => {
      const next = !current;
      localStorage.setItem(COLLAPSED_KEY, String(next));
      return next;
    });
  }

  function toggleGroup(label: string) {
    setExpandedGroups((current) => {
      const next = new Set(current);
      if (next.has(label)) next.delete(label);
      else next.add(label);
      localStorage.setItem(EXPANDED_GROUPS_KEY, JSON.stringify([...next]));
      return next;
    });
  }

  return (
    <aside
      className={`flex h-full flex-col border-r border-(--border-color) bg-(--bg-secondary) transition-all duration-300 ${collapsed ? "w-10" : "w-56"}`}
    >
      <div className="flex h-10 items-center justify-end border-b border-(--border-light) px-2">
        <button type="button" onClick={toggleCollapsed} className="rounded p-1.5 text-(--text-secondary) hover:bg-(--bg-hover)">
          <Menu className="h-4 w-4" />
        </button>
      </div>
      <nav className="flex flex-col gap-0.5 p-1.5">
        {NAV_ITEMS.map((item) => {
          if (isGroup(item)) {
            const expanded = expandedGroups.has(item.label);
            return (
              <div key={item.label}>
                <button
                  type="button"
                  onClick={() => toggleGroup(item.label)}
                  className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-xs font-medium text-(--text-secondary) hover:bg-(--bg-hover)"
                >
                  {item.icon}
                  {!collapsed && (
                    <>
                      <span className="flex-1 text-left">{item.label}</span>
                      <ChevronRight className={`h-3 w-3 transition-transform ${expanded ? "rotate-90" : ""}`} />
                    </>
                  )}
                </button>
                {!collapsed && expanded && (
                  <div className="flex flex-col gap-0.5 pl-5">
                    {item.children.map((child) => (
                      <NavLeafItem key={child.to} item={child} collapsed={collapsed} />
                    ))}
                  </div>
                )}
              </div>
            );
          }
          return <NavLeafItem key={item.to} item={item} collapsed={collapsed} />;
        })}
      </nav>
    </aside>
  );
}

function NavLeafItem({ item, collapsed }: { item: NavLeaf; collapsed: boolean }) {
  if (!item.enabled) {
    return (
      <span
        title={`${item.label} (próximamente)`}
        className="flex cursor-not-allowed items-center gap-2 rounded px-2 py-1.5 text-xs text-(--text-muted) opacity-60"
      >
        {item.icon}
        {!collapsed && <span>{item.label}</span>}
      </span>
    );
  }
  return (
    <NavLink
      to={item.to}
      className={({ isActive }) =>
        `flex items-center gap-2 rounded px-2 py-1.5 text-xs font-medium ${
          isActive ? "bg-(--bg-selected) text-(--accent-primary)" : "text-(--text-secondary) hover:bg-(--bg-hover)"
        }`
      }
    >
      {item.icon}
      {!collapsed && <span>{item.label}</span>}
    </NavLink>
  );
}
