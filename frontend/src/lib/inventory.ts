import { apiClient } from "./apiClient";
import type { Movement, MovementPayload, StockEntry } from "./types";

export function listStock(): Promise<StockEntry[]> {
  return apiClient.get<StockEntry[]>("/inventory/stock");
}

export function listMovements(productId?: string): Promise<Movement[]> {
  const query = productId ? `?product_id=${encodeURIComponent(productId)}` : "";
  return apiClient.get<Movement[]>(`/inventory/movements${query}`);
}

// createMovement — entrada/salida/ajuste manual, o traslado entre bodegas (type="transfer" +
// to_warehouse_id). Las entradas/salidas automáticas de ventas y compras confirmadas no pasan
// por acá — las genera el backend directamente al confirmar/recibir.
export function createMovement(payload: MovementPayload): Promise<Movement> {
  return apiClient.post<Movement>("/inventory/movements", payload);
}
