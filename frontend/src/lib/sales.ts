import { apiClient } from "./apiClient";
import type { CreateSalePayload, Sale, UpdateSalePayload } from "./types";

export function listSales(): Promise<Sale[]> {
  return apiClient.get<Sale[]>("/sales");
}

export function fetchSale(id: string): Promise<Sale> {
  return apiClient.get<Sale>(`/sales/${id}`);
}

export function createSale(payload: CreateSalePayload): Promise<Sale> {
  return apiClient.post<Sale>("/sales", payload);
}

// updateSale -- solo permitido mientras la venta está en borrador (el backend lo valida).
export function updateSale(id: string, payload: UpdateSalePayload): Promise<Sale> {
  return apiClient.put<Sale>(`/sales/${id}`, payload);
}

// Al confirmar, el backend dispara sale.confirmed: accounting/ contabiliza y inventory/
// descuenta stock automáticamente — por eso una venta confirmada ya no se puede editar.
export function confirmSale(id: string): Promise<Sale> {
  return apiClient.post<Sale>(`/sales/${id}/confirm`, undefined);
}

export function cancelSale(id: string): Promise<void> {
  return apiClient.post<void>(`/sales/${id}/cancel`, undefined);
}

// deleteSale — solo permitido mientras la venta está en borrador (el backend lo valida y
// devuelve 422 si no). Para una venta ya confirmada, usa cancelSale.
export function deleteSale(id: string): Promise<void> {
  return apiClient.del(`/sales/${id}`);
}
