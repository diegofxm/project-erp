// Cliente de los catálogos DIAN de solo lectura expuestos por apidian (internal/catalogs) —
// estáticos dentro de una sesión, por eso cada función memoiza su resultado a nivel de módulo
// en vez de volver a pedirlo cada vez que el usuario cambia de pestaña en un formulario.
import { apiClient } from "./apiClient";
import type { CatalogEntry, Municipality } from "./types";

function memoized<T>(fetcher: () => Promise<T>): () => Promise<T> {
  let cache: T | null = null;
  return async () => {
    if (cache === null) cache = await fetcher();
    return cache;
  };
}

export const listDepartments = memoized(async () => {
  const res = await apiClient.get<{ departments: CatalogEntry[] }>("/catalogs/departments");
  return res.departments;
});

export const listIdentificationTypes = memoized(async () => {
  const res = await apiClient.get<{ identification_types: CatalogEntry[] }>("/catalogs/identification-types");
  return res.identification_types;
});

export const listTaxTypes = memoized(async () => {
  const res = await apiClient.get<{ tax_types: CatalogEntry[] }>("/catalogs/tax-types");
  return res.tax_types;
});

export const listTaxRegimes = memoized(async () => {
  const res = await apiClient.get<{ tax_regimes: CatalogEntry[] }>("/catalogs/tax-regimes");
  return res.tax_regimes;
});

export const listLiabilityCodes = memoized(async () => {
  const res = await apiClient.get<{ liability_codes: CatalogEntry[] }>("/catalogs/liability-codes");
  return res.liability_codes;
});

const municipalitiesCache = new Map<string, Municipality[]>();

export async function listMunicipalities(departmentCode: string): Promise<Municipality[]> {
  const cached = municipalitiesCache.get(departmentCode);
  if (cached) return cached;
  const res = await apiClient.get<{ municipalities: Municipality[] }>(
    `/catalogs/municipalities?department_code=${encodeURIComponent(departmentCode)}`,
  );
  municipalitiesCache.set(departmentCode, res.municipalities);
  return res.municipalities;
}
