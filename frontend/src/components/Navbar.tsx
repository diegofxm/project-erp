import { useEffect, useRef, useState } from "react";
import { Link } from "react-router";
import { Building2, ChevronDown, LogOut, Server, UserCircle } from "lucide-react";
import { useAuth } from "../context/AuthContext";
import { NotificationBell } from "./NotificationBell";

function getInitials(name: string | undefined): string {
  if (!name) return "?";
  const [first, second] = name.trim().split(/\s+/);
  return ((first?.[0] ?? "") + (second?.[0] ?? "")).toUpperCase();
}

// Barra única h-10, fondo oscuro — única zona oscura de la UI (sección 5 del design system).
// Solo lleva el avatar (iniciales) a la derecha — nombre/correo completos viven en el
// desplegable, no en la barra fija (mismo patrón que GitHub/Linear/Vercel).
export function Navbar() {
  const { user, activeIssuer, logout } = useAuth();
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
    <header className="flex h-10 items-center justify-between bg-(--navbar-bg) px-3 text-(--navbar-text)">
      <div className="flex items-center gap-3">
        <Link to="/" className="flex items-center gap-2">
          <img src="/logo.svg" alt="cofacture" className="h-5 w-5" />
          <span className="text-base font-semibold">cofacture</span>
        </Link>
        {activeIssuer && (
          <>
            <span className="h-4 w-px bg-(--navbar-text) opacity-30" />
            <div className="flex items-center gap-1.5 text-xs opacity-70">
              <Server className="h-3.5 w-3.5" />
              <span>Empresa:</span>
              <span className="font-mono">{activeIssuer.business_name}</span>
            </div>
          </>
        )}
      </div>

      <div className="flex items-center gap-1">
        <Link
          to="/issuers"
          title="Mis empresas"
          className="rounded p-1.5 text-(--navbar-text) opacity-80 transition-colors hover:bg-white/10 hover:opacity-100"
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
    </header>
  );
}
