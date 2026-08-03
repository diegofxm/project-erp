import { apiClient } from "./apiClient";
import type { Warehouse, WarehousePayload } from "./types";

export function listWarehouses(): Promise<Warehouse[]> {
  return apiClient.get<Warehouse[]>("/companies/active/warehouses");
}

export function fetchWarehouse(id: string): Promise<Warehouse> {
  return apiClient.get<Warehouse>(`/companies/active/warehouses/${id}`);
}

export function createWarehouse(payload: WarehousePayload): Promise<Warehouse> {
  return apiClient.post<Warehouse>("/companies/active/warehouses", payload);
}

export function updateWarehouse(id: string, payload: WarehousePayload): Promise<Warehouse> {
  return apiClient.put<Warehouse>(`/companies/active/warehouses/${id}`, payload);
}

export function deactivateWarehouse(id: string): Promise<void> {
  return apiClient.del<void>(`/companies/active/warehouses/${id}`);
}

// setDefaultWarehouse — la marca como la que usan ventas/compras cuando el documento no elige
// una explícitamente. Desmarca automáticamente cualquier otra de la empresa (ver
// company/infrastructure WarehouseRepository.SetDefault).
export function setDefaultWarehouse(id: string): Promise<void> {
  return apiClient.put<void>(`/companies/active/warehouses/${id}/default`, undefined);
}
