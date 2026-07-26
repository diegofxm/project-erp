// Cliente del catálogo de proveedores/terceros no obligados a facturar — mismo patrón que
// lib/customers.ts.
import { apiClient } from "./apiClient";
import type { ListVendorsResult, Vendor, VendorPayload } from "./types";

export async function listVendors(): Promise<Vendor[]> {
  const res = await apiClient.get<ListVendorsResult>("/vendors");
  return res.vendors;
}

export function fetchVendor(id: string): Promise<Vendor> {
  return apiClient.get<Vendor>(`/vendors/${id}`);
}

export function createVendor(payload: VendorPayload): Promise<Vendor> {
  return apiClient.post<Vendor>("/vendors", payload);
}

export function updateVendor(id: string, payload: VendorPayload): Promise<Vendor> {
  return apiClient.put<Vendor>(`/vendors/${id}`, payload);
}

export function deleteVendor(id: string): Promise<void> {
  return apiClient.del<void>(`/vendors/${id}`);
}

// vendorToPayload despoja id/created_at/updated_at — usado para precargar VendorSection en
// edición y para copiar los campos de un vendor guardado al armar un Documento Soporte.
export function vendorToPayload(vendor: Vendor): VendorPayload {
  return {
    identification: vendor.identification,
    name: vendor.name,
    entity_type_code: vendor.entity_type_code,
    address: vendor.address,
    tax_scheme_code: vendor.tax_scheme_code,
    liability_codes: vendor.liability_codes,
    tax_regime_code: vendor.tax_regime_code,
    phone: vendor.phone,
    email: vendor.email,
  };
}
