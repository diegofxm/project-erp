// Cliente del autorregistro público de clientes (patrón QR/D1, ver
// docs/apidian-architecture.md sección 9.41) — sin sesión: apiClient.request ya omite el header
// Authorization cuando no hay token en memoria (el caso normal para un visitante anónimo desde
// el QR), así que no hace falta ningún cliente especial "sin auth".
import { apiClient } from "./apiClient";

export interface PublicCompany {
  business_name: string;
  has_logo: boolean;
}

export interface PublicCustomerPayload {
  identification: { number: string; type_code: string };
  name: string;
  email?: string;
  phone?: string;
}

export function getPublicCompany(companyId: string): Promise<PublicCompany> {
  return apiClient.get<PublicCompany>(`/public/companies/${companyId}`);
}

// getPublicCompanyLogoBlobUrl trae el logo en crudo y devuelve un Object URL —
// mismo patrón que getInvoicePdfBlobUrl en lib/documents.ts.
export async function getPublicCompanyLogoBlobUrl(companyId: string): Promise<string> {
  const blob = await apiClient.getBlob(`/public/companies/${companyId}/logo`);
  return URL.createObjectURL(blob);
}

export function registerPublicCustomer(companyId: string, payload: PublicCustomerPayload): Promise<{ name: string }> {
  return apiClient.post<{ name: string }>(`/public/companies/${companyId}/customers`, payload);
}
