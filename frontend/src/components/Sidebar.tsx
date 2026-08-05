import { type ReactNode } from "react";
import { NavLink, useLocation } from "react-router";
import {
  Banknote, Calculator, ClipboardCheck, Crown, Files, Home, Package, Settings,
  ShoppingBag, ShoppingCart, Truck, User, UserCheck, Users,
} from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { useMyPlan } from "../hooks/useMyPlan";

interface NavItem {
  to: string;
  label: string;
  icon: ReactNode;
  activePrefix?: string;
  end?: boolean;
  superAdminOnly?: boolean;
  // Sin página propia todavía (ej. Nómina/RRHH/Empleados — módulo backend existe, sin
  // frontend) — se muestra en el sidebar para reflejar la jerarquía completa, pero no navega.
  disabled?: boolean;
  // Módulo del plan SaaS que debe incluir la empresa activa para ver este ítem (ver
  // useMyPlan) — sin esto, el ítem siempre se muestra (ej. Catálogos, Configuración).
  moduleCode?: string;
}

interface NavGroup {
  // undefined = sin encabezado (Inicio/Comando, sueltos arriba).
  title?: string;
  items: NavItem[];
}

const NAV_GROUPS: NavGroup[] = [
  {
    items: [
      { to: "/", label: "Inicio", icon: <Home className="h-3.5 w-3.5" />, end: true },
      { to: "/admin", label: "Comando", icon: <Crown className="h-3.5 w-3.5" />, activePrefix: "/admin", superAdminOnly: true },
    ],
  },
  {
    title: "Finanzas",
    items: [
      { to: "/accounting", label: "Contabilidad", icon: <Calculator className="h-3.5 w-3.5" />, activePrefix: "/accounting", moduleCode: "erp_core" },
    ],
  },
  {
    title: "Operación",
    items: [
      { to: "/documents/invoices", label: "Documentos", icon: <Files className="h-3.5 w-3.5" />, activePrefix: "/documents", moduleCode: "electronic_invoicing" },
      { to: "/sales", label: "Ventas", icon: <ShoppingCart className="h-3.5 w-3.5" />, activePrefix: "/sales", moduleCode: "erp_core" },
      { to: "/purchases", label: "Compras", icon: <ShoppingBag className="h-3.5 w-3.5" />, activePrefix: "/purchases", moduleCode: "erp_core" },
      { to: "/inventory", label: "Inventario", icon: <ClipboardCheck className="h-3.5 w-3.5" />, activePrefix: "/inventory", moduleCode: "erp_core" },
    ],
  },
  {
    title: "Nómina y RRHH",
    items: [
      { to: "/payroll", label: "Nómina", icon: <Banknote className="h-3.5 w-3.5" />, disabled: true, moduleCode: "payroll_hr" },
      { to: "/hr", label: "RRHH", icon: <UserCheck className="h-3.5 w-3.5" />, disabled: true, moduleCode: "payroll_hr" },
      { to: "/employees", label: "Empleados", icon: <User className="h-3.5 w-3.5" />, disabled: true, moduleCode: "payroll_hr" },
    ],
  },
  {
    title: "Catálogos",
    items: [
      { to: "/customers", label: "Clientes", icon: <Users className="h-3.5 w-3.5" />, activePrefix: "/customers" },
      { to: "/suppliers", label: "Proveedores", icon: <Truck className="h-3.5 w-3.5" />, activePrefix: "/suppliers" },
      { to: "/products", label: "Productos", icon: <Package className="h-3.5 w-3.5" />, activePrefix: "/products" },
    ],
  },
];

// Anclado al fondo del sidebar (el nav de arriba usa flex-1 + overflow-y-auto para empujarlo).
const BOTTOM_NAV_ITEMS: NavItem[] = [
  { to: "/settings", label: "Configuración", icon: <Settings className="h-3.5 w-3.5" />, activePrefix: "/settings" },
];

interface SidebarProps {
  collapsed: boolean;
}

function ItemLabel({ label, collapsed }: { label: string; collapsed: boolean }) {
  return (
    <span
      className={`whitespace-nowrap overflow-hidden transition-[max-width,opacity,margin-left] duration-200 ${
        collapsed ? "max-w-0 opacity-0 ml-0" : "max-w-30 opacity-100 ml-2"
      }`}
    >
      {label}
    </span>
  );
}

function renderItem(item: NavItem, collapsed: boolean, active: boolean) {
  if (item.disabled) {
    return (
      <div
        key={item.to}
        title={collapsed ? `${item.label} (próximamente)` : "Próximamente"}
        className="flex cursor-default items-center rounded py-1.5 pl-1.75 pr-2 text-xs font-medium text-(--text-muted) opacity-60"
      >
        <span className="shrink-0">{item.icon}</span>
        <ItemLabel label={item.label} collapsed={collapsed} />
      </div>
    );
  }
  return (
    <NavLink
      key={item.to}
      to={item.to}
      title={collapsed ? item.label : undefined}
      className={`flex items-center rounded py-1.5 pl-1.75 pr-2 text-xs font-medium transition-colors ${
        active
          ? "bg-(--bg-selected) text-(--accent-primary)"
          : "text-(--text-secondary) hover:bg-(--bg-hover)"
      }`}
    >
      <span className="shrink-0">{item.icon}</span>
      <ItemLabel label={item.label} collapsed={collapsed} />
    </NavLink>
  );
}

export function Sidebar({ collapsed }: SidebarProps) {
  const location = useLocation();
  const { user } = useAuth();
  const myModules = useMyPlan();

  function isActive(item: NavItem) {
    return item.activePrefix
      ? location.pathname.startsWith(item.activePrefix)
      : item.end
      ? location.pathname === "/"
      : location.pathname.startsWith(item.to);
  }

  return (
    <aside
      className={`flex h-full flex-col overflow-hidden border-r border-(--border-color) bg-(--bg-secondary) transition-[width] duration-200 ${
        collapsed ? "w-10" : "w-48"
      }`}
    >
      <nav className="flex flex-1 flex-col gap-px overflow-y-auto p-1.5 pt-2">
        {NAV_GROUPS.map((group, gi) => {
          const items = group.items.filter((item) => {
            if (item.superAdminOnly && !user?.is_superadmin) return false;
            if (item.moduleCode && myModules && !myModules.includes(item.moduleCode)) return false;
            return true;
          });
          if (items.length === 0) return null;
          return (
            <div key={group.title ?? `g${gi}`} className="flex flex-col gap-px">
              {group.title && !collapsed && (
                <div className="mt-0.5 px-1.75 pb-0 text-[8.5px] font-bold tracking-wide text-(--text-muted) uppercase">
                  {group.title}
                </div>
              )}
              {items.map((item) => renderItem(item, collapsed, isActive(item)))}
            </div>
          );
        })}
      </nav>
      <nav className="flex flex-col gap-px border-t border-(--border-color) p-1.5">
        {BOTTOM_NAV_ITEMS.map((item) => renderItem(item, collapsed, isActive(item)))}
      </nav>
    </aside>
  );
}
