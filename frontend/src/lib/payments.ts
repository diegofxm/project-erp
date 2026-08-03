import { apiClient } from "./apiClient";
import type { ReceivableBalance, RecordPaymentPayload, SalePayment } from "./types";

export function recordPayment(payload: RecordPaymentPayload): Promise<SalePayment> {
  return apiClient.post<SalePayment>("/payments", payload);
}

export function listPaymentsBySale(saleId: string): Promise<SalePayment[]> {
  return apiClient.get<SalePayment[]>(`/payments/sales/${saleId}`);
}

// Cartera — ventas confirmadas con saldo pendiente (ver sales/application/payment.go
// GetReceivables). No incluye ventas ya pagadas por completo.
export function getReceivables(): Promise<ReceivableBalance[]> {
  return apiClient.get<ReceivableBalance[]>("/receivables");
}
