import { apiClient } from "./apiClient";
import type { Customer, CustomerPayload, ListCustomersResult } from "./types";

export async function listCustomers(): Promise<Customer[]> {
  const res = await apiClient.get<ListCustomersResult>("/customers");
  return res.customers;
}

export function fetchCustomer(id: string): Promise<Customer> {
  return apiClient.get<Customer>(`/customers/${id}`);
}

// Convierte el payload anidado del formulario (CustomerPayload) al formato plano del ERP.
function toERPPayload(p: CustomerPayload) {
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
  };
}

export function createCustomer(payload: CustomerPayload): Promise<Customer> {
  return apiClient.post<Customer>("/customers", toERPPayload(payload));
}

export function updateCustomer(id: string, payload: CustomerPayload): Promise<Customer> {
  return apiClient.put<Customer>(`/customers/${id}`, toERPPayload(payload));
}

export function deleteCustomer(id: string): Promise<void> {
  return apiClient.del<void>(`/customers/${id}`);
}

// customerToPayload — convierte un Customer del catálogo ERP (campos planos) al CustomerPayload
// anidado que se embebe en documentos electrónicos (IssueInvoicePayload.customer).
export function customerToPayload(c: Customer): CustomerPayload {
  return {
    entity_type_code: c.entity_type_code || undefined,
    identification: {
      number: c.identification_number,
      type_code: c.identification_type_code,
      verification_code: c.check_digit || undefined,
    },
    name: c.name,
    address: {
      line: c.address_line || undefined,
      city_code: c.municipality_code || undefined,
      city_name: c.address_city_name || undefined,
      state_code: c.department_code || undefined,
      state_name: c.address_state_name || undefined,
      country_code: c.address_country_code || undefined,
      country_name: c.address_country_name || undefined,
    },
    tax_scheme_code: c.tax_scheme_code,
    tax_scheme_name: c.tax_scheme_name,
    liability_codes: c.liability_codes,
    tax_regime_code: c.tax_regime_code ?? undefined,
    phone: c.phone || undefined,
    email: c.email || undefined,
  };
}

// customerFromPayload — reconstruye un Customer del catálogo a partir de un CustomerPayload
// (usado cuando se edita el cliente directamente desde la sección de la factura).
export function customerFromPayload(p: CustomerPayload, existing: Customer): Customer {
  return {
    ...existing,
    entity_type_code: p.entity_type_code ?? existing.entity_type_code,
    identification_type_code: p.identification.type_code,
    identification_number: p.identification.number,
    check_digit: p.identification.verification_code ?? "",
    name: p.name,
    tax_scheme_code: p.tax_scheme_code ?? existing.tax_scheme_code,
    tax_scheme_name: p.tax_scheme_name ?? existing.tax_scheme_name,
    tax_regime_code: p.tax_regime_code ?? null,
    liability_codes: p.liability_codes ?? [],
    department_code: p.address?.state_code ?? existing.department_code,
    municipality_code: p.address?.city_code ?? existing.municipality_code,
    address_line: p.address?.line ?? existing.address_line,
    address_city_name: p.address?.city_name ?? existing.address_city_name,
    address_state_name: p.address?.state_name ?? existing.address_state_name,
    address_country_code: p.address?.country_code ?? existing.address_country_code,
    address_country_name: p.address?.country_name ?? existing.address_country_name,
    email: p.email ?? existing.email,
    phone: p.phone ?? existing.phone,
  };
}
