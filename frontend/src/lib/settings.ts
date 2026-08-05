import { apiClient } from "./apiClient";
import type { CompanySettings } from "./types";

export async function getMySettings(): Promise<CompanySettings> {
  return apiClient.get<CompanySettings>("/companies/active/settings");
}

export async function updateMySettings(data: Partial<Pick<CompanySettings, "brand_color">>): Promise<CompanySettings> {
  return apiClient.patch<CompanySettings>("/companies/active/settings", data);
}
