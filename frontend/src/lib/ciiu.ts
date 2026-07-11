import { apiClient } from "./apiClient";
import type { ComboboxOption } from "../components/ui/Combobox";

interface CiiuEntry {
  code: string;
  description: string;
}

let cache: ComboboxOption[] | null = null;

export async function listCiiuOptions(): Promise<ComboboxOption[]> {
  if (cache) return cache;
  const res = await apiClient.get<{ ciiu_codes: CiiuEntry[] }>("/catalogs/ciiu-codes");
  cache = res.ciiu_codes.map((e) => ({ value: e.code, label: `${e.code} - ${e.description}` }));
  return cache;
}
