import { apiClient } from "./apiClient";
import type { PayableBalance, PurchasePayment, RecordPurchasePaymentPayload } from "./types";

export function recordPurchasePayment(payload: RecordPurchasePaymentPayload): Promise<PurchasePayment> {
  return apiClient.post<PurchasePayment>("/purchase-payments", payload);
}

export function listPaymentsByPurchase(purchaseId: string): Promise<PurchasePayment[]> {
  return apiClient.get<PurchasePayment[]>(`/purchase-payments/purchases/${purchaseId}`);
}

// Cuentas por pagar — órdenes recibidas con saldo pendiente (ver purchase/application/
// payment.go GetPayables). No incluye órdenes ya pagadas por completo.
export function getPayables(): Promise<PayableBalance[]> {
  return apiClient.get<PayableBalance[]>("/payables");
}
