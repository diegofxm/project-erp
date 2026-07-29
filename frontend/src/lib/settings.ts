import { apiClient } from "./apiClient";
import type { IssuerSettings } from "./types";

export async function getMySettings(): Promise<IssuerSettings> {
  return apiClient.get<IssuerSettings>("/companies/active/settings");
}

export async function updateMySettings(data: Partial<Pick<IssuerSettings, "brand_color" | "price_per_document_cop">>): Promise<IssuerSettings> {
  return apiClient.patch<IssuerSettings>("/companies/active/settings", data);
}
