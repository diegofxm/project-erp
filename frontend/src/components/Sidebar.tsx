import { type ReactNode } from "react";
import { NavLink, useLocation } from "react-router";
import { Boxes, Calculator, Crown, Files, Home, Package, Settings, ShoppingBag, ShoppingCart, Truck, Users } from "lucide-react";
import { useAuth } from "../context/AuthContext";

interface NavItem {
  to: string;
  label: string;
  icon: ReactNode;
  activePrefix?: string;
  end?: boolean;
  superAdminOnly?: boolean;
}

const NAV_ITEMS: NavItem[] = [
  { to: "/", label: "Inicio", icon: <Home className="h-3.5 w-3.5" />, end: true },
  { to: "/admin", label: "Comando", icon: <Crown className="h-3.5 w-3.5" />, activePrefix: "/admin", superAdminOnly: true },
  { to: "/documents/invoices", label: "Documentos", icon: <Files className="h-3.5 w-3.5" />, activePrefix: "/documents" },
  { to: "/sales", label: "Ventas", icon: <ShoppingCart className="h-3.5 w-3.5" />, activePrefix: "/sales" },
  { to: "/purchases", label: "Compras", icon: <ShoppingBag className="h-3.5 w-3.5" />, activePrefix: "/purchases" },
  { to: "/inventory", label: "Inventario", icon: <Boxes className="h-3.5 w-3.5" />, activePrefix: "/inventory" },
  { to: "/accounting", label: "Contabilidad", icon: <Calculator className="h-3.5 w-3.5" />, activePrefix: "/accounting" },
  { to: "/customers", label: "Clientes", icon: <Users className="h-3.5 w-3.5" />, activePrefix: "/customers" },
  { to: "/suppliers", label: "Proveedores", icon: <Truck className="h-3.5 w-3.5" />, activePrefix: "/suppliers" },
  { to: "/products", label: "Productos", icon: <Package className="h-3.5 w-3.5" />, activePrefix: "/products" },
];

// Anclado al fondo del sidebar (ver render: flex-1 en el nav de arriba empuja este bloque)
// — organización jerárquica completa (grupos por plan SaaS futuro) pendiente de confirmar,
// por ahora solo Configuración queda fija abajo, separada del resto.
const BOTTOM_NAV_ITEMS: NavItem[] = [
  { to: "/settings", label: "Configuración", icon: <Settings className="h-3.5 w-3.5" />, activePrefix: "/settings" },
];

interface SidebarProps {
  collapsed: boolean;
}

function renderItem(item: NavItem, collapsed: boolean, active: boolean) {
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
      <span
        className={`whitespace-nowrap overflow-hidden transition-[max-width,opacity,margin-left] duration-200 ${
          collapsed ? "max-w-0 opacity-0 ml-0" : "max-w-30 opacity-100 ml-2"
        }`}
      >
        {item.label}
      </span>
    </NavLink>
  );
}

export function Sidebar({ collapsed }: SidebarProps) {
  const location = useLocation();
  const { user } = useAuth();

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
      <nav className="flex flex-1 flex-col gap-0.5 overflow-y-auto p-1.5 pt-2">
        {NAV_ITEMS.map((item) => {
          if (item.superAdminOnly && !user?.is_superadmin) return null;
          return renderItem(item, collapsed, isActive(item));
        })}
      </nav>
      <nav className="flex flex-col gap-0.5 border-t border-(--border-color) p-1.5">
        {BOTTOM_NAV_ITEMS.map((item) => renderItem(item, collapsed, isActive(item)))}
      </nav>
    </aside>
  );
}
