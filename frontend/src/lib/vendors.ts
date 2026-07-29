import { apiClient } from "./apiClient";
import type { ListVendorsResult, Vendor, VendorPayload } from "./types";

export async function listVendors(): Promise<Vendor[]> {
  const res = await apiClient.get<ListVendorsResult>("/suppliers");
  return res.vendors;
}

export function fetchVendor(id: string): Promise<Vendor> {
  return apiClient.get<Vendor>(`/suppliers/${id}`);
}

// Convierte el payload anidado del formulario (VendorPayload) al formato plano del ERP.
function toERPPayload(p: VendorPayload) {
  return {
    entity_type_code: p.entity_type_code ?? "1",
    identification_type_code: p.identification.type_code,
    identification_number: p.identification.number,
    check_digit: p.identification.verification_code ?? "",
    merchant_registration_number: "",
    name: p.name,
    tax_scheme_code: p.tax_scheme_code ?? "ZZ",
    tax_scheme_name: p.tax_scheme_name ?? "No aplica",
    tax_regime_code: p.tax_regime_code ?? null,
    liability_codes: p.liability_codes ?? [],
    department_code: p.address?.state_code ?? "",
    municipality_code: p.address?.city_code ?? "",
    address_line: p.address?.line ?? "",
    address_city_name: p.address?.city_name ?? "",
    address_state_name: p.address?.state_name ?? "",
    address_country_code: p.address?.country_code ?? "CO",
    address_country_name: p.address?.country_name ?? "Colombia",
    email: p.email ?? "",
    phone: p.phone ?? "",
    payment_terms_days: 0,
  };
}

export function createVendor(payload: VendorPayload): Promise<Vendor> {
  return apiClient.post<Vendor>("/suppliers", toERPPayload(payload));
}

export function updateVendor(id: string, payload: VendorPayload): Promise<Vendor> {
  return apiClient.put<Vendor>(`/suppliers/${id}`, toERPPayload(payload));
}

export function deleteVendor(id: string): Promise<void> {
  return apiClient.del<void>(`/suppliers/${id}`);
}

// vendorToPayload — convierte un Vendor del catálogo ERP (campos planos) al VendorPayload
// anidado que se embebe en documentos (IssueSupportDocumentPayload.vendor).
export function vendorToPayload(v: Vendor): VendorPayload {
  return {
    entity_type_code: v.entity_type_code || undefined,
    identification: {
      number: v.identification_number,
      type_code: v.identification_type_code,
      verification_code: v.check_digit || undefined,
    },
    name: v.name,
    address: {
      line: v.address_line || undefined,
      city_code: v.municipality_code || undefined,
      city_name: v.address_city_name || undefined,
      state_code: v.department_code || undefined,
      state_name: v.address_state_name || undefined,
      country_code: v.address_country_code || undefined,
      country_name: v.address_country_name || undefined,
    },
    tax_scheme_code: v.tax_scheme_code,
    tax_scheme_name: v.tax_scheme_name,
    liability_codes: v.liability_codes,
    tax_regime_code: v.tax_regime_code ?? undefined,
    phone: v.phone || undefined,
    email: v.email || undefined,
  };
}
