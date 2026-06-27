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

export const listDianDocumentTypes = memoized(async () => {
  const res = await apiClient.get<{ dian_document_types: CatalogEntry[] }>("/catalogs/dian-document-types");
  return res.dian_document_types;
});

// payment_terms (forma de pago: "1" contado/"2" crédito) y payment_methods (medio de pago:
// efectivo, transferencia, etc.) — payment_means de un documento necesita un código de cada
// uno (ver documents.PaymentMean).
export const listPaymentTerms = memoized(async () => {
  const res = await apiClient.get<{ payment_terms: CatalogEntry[] }>("/catalogs/payment-terms");
  return res.payment_terms;
});

export const listPaymentMethods = memoized(async () => {
  const res = await apiClient.get<{ payment_methods: CatalogEntry[] }>("/catalogs/payment-methods");
  return res.payment_methods;
});

// Catálogo incompleto a propósito (11 de los cientos de códigos reales UN/ECE Rec. 20) — se
// usa igual aquí porque un select con 11 opciones reales es mejor que un input de texto libre,
// y no se valida del lado del servidor por la misma razón (ver documents.Service).
export const listUnitMeasures = memoized(async () => {
  const res = await apiClient.get<{ unit_measures: CatalogEntry[] }>("/catalogs/unit-measures");
  return res.unit_measures;
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
