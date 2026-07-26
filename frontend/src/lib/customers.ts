// Cliente del catálogo de adquirientes — datos propios del tenant, sin memoización (a
// diferencia de lib/catalogs.ts).
import { apiClient } from "./apiClient";
import type { Customer, CustomerPayload, ListCustomersResult } from "./types";

export async function listCustomers(): Promise<Customer[]> {
  const res = await apiClient.get<ListCustomersResult>("/customers");
  return res.customers;
}

export function fetchCustomer(id: string): Promise<Customer> {
  return apiClient.get<Customer>(`/customers/${id}`);
}

export function createCustomer(payload: CustomerPayload): Promise<Customer> {
  return apiClient.post<Customer>("/customers", payload);
}

export function updateCustomer(id: string, payload: CustomerPayload): Promise<Customer> {
  return apiClient.put<Customer>(`/customers/${id}`, payload);
}

export function deleteCustomer(id: string): Promise<void> {
  return apiClient.del<void>(`/customers/${id}`);
}

// customerToPayload despoja id/created_at/updated_at — usado tanto para precargar
// CustomerForm en edición como para copiar TODOS los campos de un cliente guardado al armar
// una factura (CustomerSection), sin dejar tax_scheme_name (ya no se manda, ver
// docs/apidian-architecture.md sección 9.37).
export function customerToPayload(customer: Customer): CustomerPayload {
  return {
    identification: customer.identification,
    name: customer.name,
    entity_type_code: customer.entity_type_code,
    address: customer.address,
    tax_scheme_code: customer.tax_scheme_code,
    liability_codes: customer.liability_codes,
    tax_regime_code: customer.tax_regime_code,
    phone: customer.phone,
    email: customer.email,
  };
}
