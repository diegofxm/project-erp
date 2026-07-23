import { useEffect, useState } from "react";
import { Outlet, useLocation } from "react-router";
import { useAuth } from "../context/AuthContext";
import { Navbar } from "./Navbar";
import { Sidebar } from "./Sidebar";
import { SubNav } from "./SubNav";
import { OnboardingPage } from "../pages/OnboardingPage";

const COLLAPSED_KEY = "apidian.sidebarCollapsed";

export function DashboardLayout() {
  const { activeIssuer } = useAuth();
  const location = useLocation();
  const [collapsed, setCollapsed] = useState(() => {
    const saved = localStorage.getItem(COLLAPSED_KEY);
    if (saved !== null) return saved === "true";
    return window.innerWidth < 1024; // tablet y split-screen arrancan con sidebar colapsado
  });
  const [mobileOpen, setMobileOpen] = useState(false);

  // Cierra el menú móvil al navegar
  useEffect(() => {
    setMobileOpen(false);
  }, [location.pathname]);

  function toggleSidebar() {
    if (window.innerWidth < 768) {
      setMobileOpen((open) => !open);
    } else {
      setCollapsed((c) => {
        const next = !c;
        localStorage.setItem(COLLAPSED_KEY, String(next));
        return next;
      });
    }
  }

  if (!activeIssuer) return <OnboardingPage />;

  return (
    <div className="flex h-screen flex-col">
      <Navbar onToggleSidebar={toggleSidebar} />
      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar desktop — oculto en móvil */}
        <div className="hidden md:flex h-full shrink-0">
          <Sidebar collapsed={collapsed} />
        </div>

        {/* Drawer móvil con backdrop */}
        {mobileOpen && (
          <div className="fixed inset-0 z-40 flex md:hidden">
            <div className="w-52 shrink-0 shadow-xl">
              <Sidebar collapsed={false} />
            </div>
            <div className="flex-1 bg-black/50" onClick={() => setMobileOpen(false)} />
          </div>
        )}

        <div className="flex min-w-0 flex-1 flex-col overflow-hidden">
          <SubNav />
          <main className="flex-1 overflow-auto bg-(--bg-primary)">
            <Outlet />
          </main>
        </div>
      </div>
    </div>
  );
}
