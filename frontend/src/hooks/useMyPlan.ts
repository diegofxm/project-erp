import { useEffect, useState } from "react";
import { useAuth } from "../context/AuthContext";
import { getMyPlan } from "../lib/saas";

// useMyPlan trae los módulos habilitados por el plan de la empresa activa — usado por el Sidebar
// para ocultar secciones no incluidas en el plan contratado (ver docs/Diseno_ERP_Go_Arquitectura_Hexagonal.md,
// jerarquía "Documentos Electrónicos / Nómina / ERP completo").
//
// modules === null significa "todavía no se sabe" (cargando, o la empresa no tiene suscripción
// asignada) — en ese caso el Sidebar NO oculta nada, para no bloquear a un usuario por un dato de
// facturación que aún no existe (mismo criterio "sin suscripción = sin límite" que ya usa el
// backend en billing).
export function useMyPlan(): string[] | null {
  const { activeCompany } = useAuth();
  const [modules, setModules] = useState<string[] | null>(null);

  useEffect(() => {
    if (!activeCompany) {
      setModules(null);
      return;
    }
    let cancelled = false;
    getMyPlan()
      .then((p) => { if (!cancelled) setModules(p.modules); })
      .catch(() => { if (!cancelled) setModules(null); });
    return () => { cancelled = true; };
  }, [activeCompany?.id]);

  return modules;
}
