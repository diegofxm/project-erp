import { Outlet } from "react-router";
import { useAuth } from "../context/AuthContext";
import { Navbar } from "./Navbar";
import { Sidebar } from "./Sidebar";
import { OnboardingPage } from "../pages/OnboardingPage";

// Gate de empresa activa: sin ella, casi toda la API responde 409 (middleware.RequireTenant en
// apidian) — así que ni siquiera se monta el shell del dashboard, se manda directo al
// onboarding (crear/seleccionar empresa).
export function DashboardLayout() {
  const { activeIssuer } = useAuth();

  if (!activeIssuer) return <OnboardingPage />;

  return (
    <div className="flex h-screen flex-col">
      <Navbar />
      <div className="flex flex-1 overflow-hidden">
        <Sidebar />
        <main className="flex-1 overflow-auto bg-(--bg-primary)">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
