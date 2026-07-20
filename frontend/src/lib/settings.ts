import { apiClient } from "./apiClient";
import type { IssuerSettings } from "./types";

export async function getMySettings(): Promise<IssuerSettings> {
  return apiClient.get<IssuerSettings>("/issuers/me/settings");
}

export async function updateBrandColor(color: string): Promise<IssuerSettings> {
  return apiClient.patch<IssuerSettings>("/issuers/me/settings", { brand_color: color });
}
