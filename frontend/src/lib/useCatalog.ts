import { useEffect, useState } from "react";

// Envuelve un fetcher de lib/catalogs.ts (ya memoizado a nivel de módulo) con un estado de
// `loading` explícito — sin esto, cada Select que depende de un catálogo arranca con un
// array vacío indistinguible de "ya cargó y no hay nada", mostrando un <select> vacío en vez
// de un estado de carga real mientras llega la primera respuesta.
export function useCatalog<T>(fetcher: () => Promise<T[]>): { data: T[]; loading: boolean } {
  const [data, setData] = useState<T[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    fetcher()
      .then(setData)
      .finally(() => setLoading(false));
  }, [fetcher]);

  return { data, loading };
}
