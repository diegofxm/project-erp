import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError } from "../lib/apiClient";

interface UseApiResourceOptions {
  // false = no dispara el fetch todavía (ej. falta un dato del que depende, como la empresa
  // activa) -- deja loading=true en vez de intentar una llamada que se sabe que va a fallar.
  enabled?: boolean;
  fallbackErrorMessage?: string;
}

interface UseApiResourceResult<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
  // refresh reintenta sin cambiar deps -- útil para el botón "Reintentar" o tras una acción que
  // invalida el dato (crear/editar/eliminar), sin duplicar la lógica de fetch en la página.
  refresh: () => void;
}

// useApiResource centraliza el patrón loading/error/cancelado-al-desmontar que antes cada página
// reimplementaba con variaciones -- ver docs/auditorias/2026-08-09/05-frontend.md punto 23
// (comparaba DashboardPage.tsx, que sí protegía contra setState post-desmontaje, con
// CustomersPage.tsx, que no lo hacía). No cancela la petición de red en sí (apiClient no expone
// un signal externo, y ya tiene su propio timeout de 20s desde el punto 22) -- solo evita hacer
// setState sobre un componente ya desmontado, igual que el patrón `cancelled` que ya usaban
// DashboardPage.tsx/useMyPlan.ts.
export function useApiResource<T>(
  fetcher: () => Promise<T>,
  deps: unknown[],
  options?: UseApiResourceOptions,
): UseApiResourceResult<T> {
  const enabled = options?.enabled ?? true;
  const fallbackErrorMessage = options?.fallbackErrorMessage ?? "No se pudo cargar la información";

  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(enabled);
  const [error, setError] = useState<string | null>(null);
  const [reloadToken, setReloadToken] = useState(0);

  // El fetcher puede ser una función nueva en cada render (closures sobre props/state) -- se
  // guarda en un ref para que el useEffect no lo necesite en sus deps y así deps sigue siendo la
  // única fuente de verdad de "cuándo volver a pedir el dato".
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;

  useEffect(() => {
    if (!enabled) {
      setLoading(true);
      return;
    }
    let cancelled = false;
    setLoading(true);
    setError(null);
    fetcherRef
      .current()
      .then((result) => {
        if (!cancelled) setData(result);
      })
      .catch((err: unknown) => {
        if (!cancelled) setError(err instanceof ApiError ? err.message : fallbackErrorMessage);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- deps la define quien llama al hook
  }, [enabled, reloadToken, ...deps]);

  const refresh = useCallback(() => setReloadToken((n) => n + 1), []);

  return { data, loading, error, refresh };
}
