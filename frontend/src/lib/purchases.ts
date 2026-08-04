import { apiClient } from "./apiClient";
import type { CreatePurchasePayload, Purchase, PurchaseWithholding } from "./types";

export function listPurchases(): Promise<Purchase[]> {
  return apiClient.get<Purchase[]>("/purchases");
}

export function fetchPurchase(id: string): Promise<Purchase> {
  return apiClient.get<Purchase>(`/purchases/${id}`);
}

export function createPurchase(payload: CreatePurchasePayload): Promise<Purchase> {
  return apiClient.post<Purchase>("/purchases", payload);
}

export function confirmPurchase(id: string): Promise<Purchase> {
  return apiClient.post<Purchase>(`/purchases/${id}/confirm`, undefined);
}

// Al recibir, el backend dispara purchase.received: accounting/ contabiliza y inventory/
// aumenta stock automáticamente (excepto productos de servicio).
export function receivePurchase(id: string): Promise<Purchase> {
  return apiClient.post<Purchase>(`/purchases/${id}/receive`, undefined);
}

export function cancelPurchase(id: string): Promise<void> {
  return apiClient.post<void>(`/purchases/${id}/cancel`, undefined);
}

export function deletePurchase(id: string): Promise<void> {
  return apiClient.del<void>(`/purchases/${id}`);
}

export async function getPurchasePdfBlobUrl(id: string): Promise<string> {
  const blob = await apiClient.getBlob(`/purchases/${id}/pdf`);
  return URL.createObjectURL(blob);
}

// sendPurchaseEmail — genera el PDF de la orden y lo envía por correo al proveedor (o a `to`
// si se especifica). La orden debe estar confirmada o recibida.
export function sendPurchaseEmail(id: string, to?: string): Promise<void> {
  return apiClient.post<void>(`/purchases/${id}/send-email`, to ? { to } : undefined);
}

// addWithholding — aplica una retención a una orden confirmada (antes de recibirla). base va
// en pesos (igual que unit_price en las líneas), el backend calcula el monto con la tarifa
// del concepto elegido.
export function addWithholding(purchaseId: string, conceptId: string, base: number): Promise<PurchaseWithholding> {
  return apiClient.post<PurchaseWithholding>(`/purchases/${purchaseId}/withholdings`, { concept_id: conceptId, base });
}

export function listWithholdings(purchaseId: string): Promise<PurchaseWithholding[]> {
  return apiClient.get<PurchaseWithholding[]>(`/purchases/${purchaseId}/withholdings`);
}
