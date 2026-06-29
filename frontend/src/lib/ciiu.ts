import type { ComboboxOption } from "../components/ui/Combobox";

// Catálogo CIIU (clasificación de actividad económica, DANE) — fuente externa, no del backend
// de apidian (CIIU nunca tuvo catálogo propio aquí, ver TaxStep.tsx). ~430 filas, se carga
// completa una sola vez y se memoiza a nivel de módulo, mismo criterio que lib/catalogs.ts.
const CIIU_URL = "https://diegofxm.github.io/ciiu-classification/ciiu.json";

interface CiiuEntry {
  codigo: string;
  descripcion: string;
}

let cache: ComboboxOption[] | null = null;

export async function listCiiuOptions(): Promise<ComboboxOption[]> {
  if (cache) return cache;
  const res = await fetch(CIIU_URL);
  const entries: CiiuEntry[] = await res.json();
  cache = entries.map((e) => ({ value: e.codigo, label: `${e.codigo} — ${e.descripcion}` }));
  return cache;
}
