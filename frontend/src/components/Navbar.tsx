import { useEffect, useRef, useState } from "react";
import { Link } from "react-router";
import { Building2, ChevronDown, LogOut, Menu, Server, UserCircle } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { NotificationBell } from "./NotificationBell";

function getInitials(name: string | undefined): string {
  if (!name) return "?";
  const [first, second] = name.trim().split(/\s+/);
  return ((first?.[0] ?? "") + (second?.[0] ?? "")).toUpperCase();
}

interface NavbarProps {
  onToggleSidebar?: () => void;
}

export function Navbar({ onToggleSidebar }: NavbarProps) {
  const { user, activeCompany, logout } = useAuth();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!menuOpen) return;
    function onMouseDown(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setMenuOpen(false);
    }
    document.addEventListener("mousedown", onMouseDown);
    return () => document.removeEventListener("mousedown", onMouseDown);
  }, [menuOpen]);

  return (
    <header className="flex h-10 items-stretch bg-(--navbar-bg) text-(--navbar-text)">
      {/* Desktop: celda de marca w-48 fija — hamburguesa con su border-r, logo llena el resto */}
      <div className="hidden md:flex w-48 shrink-0 items-center">
        {onToggleSidebar && (
          <div className="flex w-10 shrink-0 items-center justify-center border-r border-white/15 self-stretch">
            <button
              type="button"
              onClick={onToggleSidebar}
              className="rounded p-1.5 opacity-70 transition-colors hover:bg-white/10 hover:opacity-100"
              title="Menú"
            >
              <Menu className="h-4 w-4" />
            </button>
          </div>
        )}
        <Link to="/" className="flex items-center gap-2 px-3">
          <img src="/logo.svg" alt="cofacture" className="h-5 w-5 shrink-0" />
          <span className="text-base font-semibold">cofacture</span>
        </Link>
      </div>

      {/* Móvil: hamburguesa sola */}
      {onToggleSidebar && (
        <div className="flex md:hidden w-10 shrink-0 items-center justify-center border-r border-white/15">
          <button
            type="button"
            onClick={onToggleSidebar}
            className="rounded p-1.5 opacity-70 transition-colors hover:bg-white/10 hover:opacity-100"
            title="Menú"
          >
            <Menu className="h-4 w-4" />
          </button>
        </div>
      )}

      {/* Separador corto alineado exactamente con el borde de la celda de marca (w-48) */}
      {activeCompany && (
        <span className="hidden md:block h-4 w-px bg-(--navbar-text) opacity-30 self-center" />
      )}

      {/* Resto del navbar */}
      <div className="flex flex-1 items-center justify-between px-3">
        <div className="flex min-w-0 items-center gap-3">
          {/* Logo solo en móvil */}
          <Link to="/" className="flex md:hidden shrink-0 items-center gap-2">
            <img src="/logo.svg" alt="cofacture" className="h-5 w-5" />
            <span className="text-base font-semibold">cofacture</span>
          </Link>
          {activeCompany && (
            <div className="hidden min-w-0 items-center gap-1.5 text-xs opacity-70 sm:flex">
              <Server className="h-3.5 w-3.5 shrink-0" />
              <span className="shrink-0">Empresa:</span>
              <span className="truncate font-mono">{activeCompany.business_name}</span>
            </div>
          )}
        </div>

        <div className="flex items-center gap-1">
          <Link
            to="/issuers"
            title="Mis empresas"
            className="rounded p-1.5 opacity-80 transition-colors hover:bg-white/10 hover:opacity-100"
          >
            <Building2 className="h-4 w-4" />
          </Link>
          <NotificationBell />
          <div ref={menuRef} className="relative">
            <button
              type="button"
              onClick={() => setMenuOpen((open) => !open)}
              className="flex items-center gap-1 rounded px-1.5 py-1 hover:bg-white/10"
              title={user?.name}
            >
              <span className="flex h-5 w-5 items-center justify-center rounded-full bg-(--accent-primary) text-[10px] font-semibold text-white">
                {getInitials(user?.name)}
              </span>
              <ChevronDown className="h-3 w-3" />
            </button>
            <div className={`absolute right-0 top-full z-10 mt-1 w-48 rounded border border-(--border-light) bg-(--bg-secondary) text-(--text-primary) shadow-lg origin-top-right transition-all duration-150 ease-out ${menuOpen ? "opacity-100 scale-100" : "pointer-events-none opacity-0 scale-95"}`}>
              <div className="border-b border-(--border-light) px-3 py-2">
                <p className="text-xs font-medium">{user?.name}</p>
                <p className="text-xs text-(--text-muted)">{user?.email}</p>
              </div>
              <Link
                to="/settings/account"
                onClick={() => setMenuOpen(false)}
                className="flex items-center gap-1.5 px-3 py-2 text-xs text-(--text-primary) hover:bg-(--bg-hover)"
              >
                <UserCircle className="h-3.5 w-3.5" />
                Mi cuenta
              </Link>
              <button
                type="button"
                onClick={logout}
                className="flex w-full items-center gap-1.5 border-t border-(--border-light) px-3 py-2 text-left text-xs text-(--color-danger) hover:bg-(--bg-hover)"
              >
                <LogOut className="h-3.5 w-3.5" />
                Cerrar sesión
              </button>
            </div>
          </div>
        </div>
      </div>
    </header>
  );
}
