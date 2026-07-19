import { useState } from "react";
import { Outlet } from "react-router";
import { useAuth } from "../context/AuthContext";
import { Navbar } from "./Navbar";
import { Sidebar } from "./Sidebar";
import { SubNav } from "./SubNav";
import { OnboardingPage } from "../pages/OnboardingPage";

const COLLAPSED_KEY = "apidian.sidebarCollapsed";

export function DashboardLayout() {
  const { activeIssuer } = useAuth();
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem(COLLAPSED_KEY) === "true");

  function toggleSidebar() {
    setCollapsed((c) => {
      const next = !c;
      localStorage.setItem(COLLAPSED_KEY, String(next));
      return next;
    });
  }

  if (!activeIssuer) return <OnboardingPage />;

  return (
    <div className="flex h-screen flex-col">
      <Navbar onToggleSidebar={toggleSidebar} />
      <div className="flex flex-1 overflow-hidden">
        <Sidebar collapsed={collapsed} />
        <div className="flex flex-1 flex-col overflow-hidden">
          <SubNav />
          <main className="flex-1 overflow-auto bg-(--bg-primary)">
            <Outlet />
          </main>
        </div>
      </div>
    </div>
  );
}
