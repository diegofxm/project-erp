import { apiClient } from "./apiClient";
import type { IssuerSettings } from "./types";

export async function getMySettings(): Promise<IssuerSettings> {
  return apiClient.get<IssuerSettings>("/issuers/me/settings");
}

export async function updateMySettings(data: Partial<Pick<IssuerSettings, "brand_color" | "price_per_document_cop">>): Promise<IssuerSettings> {
  return apiClient.patch<IssuerSettings>("/issuers/me/settings", data);
}
