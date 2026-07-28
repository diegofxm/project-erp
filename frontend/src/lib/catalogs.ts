// Cliente de los catálogos DIAN de solo lectura expuestos por apidian (internal/catalogs) —
// estáticos dentro de una sesión, por eso cada función memoiza su resultado a nivel de módulo
// en vez de volver a pedirlo cada vez que el usuario cambia de pestaña en un formulario.
import { apiClient } from "./apiClient";
import type { CatalogEntry, Currency, ItemStandard, Municipality } from "./types";

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
  const res = await apiClient.get<{ document_types: CatalogEntry[] }>("/catalogs/document-types");
  return res.document_types;
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

// 1.081/1.081 códigos reales UN/ECE Rec. 20 (tabla 13.3.6, ver
// docs/apidian-architecture.md sección 9.46) — por el volumen, los selects que lo usan
// (LineItemsEditor, ProductForm) lo consumen vía components/ui/Combobox, no un <select> plano.
export const listUnitMeasures = memoized(async () => {
  const res = await apiClient.get<{ unit_measures: CatalogEntry[] }>("/catalogs/unit-measures");
  return res.unit_measures;
});

export const listCurrencies = memoized(async () => {
  const res = await apiClient.get<{ currencies: Currency[] }>("/catalogs/currencies");
  return res.currencies;
});

// Selector de clasificación de ítems (UNSPSC/GTIN/Partida Arancelaria/propio) — ver
// ItemStandard en lib/types.ts.
export const listItemStandards = memoized(async () => {
  const res = await apiClient.get<{ item_standards: ItemStandard[] }>("/catalogs/item-standards");
  return res.item_standards;
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
