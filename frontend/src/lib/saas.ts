import { apiClient } from "./apiClient";
import type { MyPlan, Prospect } from "./types";

// getMyPlan trae el plan contratado por la empresa activa — alimenta la página "Mi plan" y el
// gating de módulos del Sidebar (ver hooks/useMyPlan.ts).
export async function getMyPlan(): Promise<MyPlan> {
  return apiClient.get<MyPlan>("/saas/my-plan");
}

export interface SubmitProspectPayload {
  name: string;
  email: string;
  nit?: string;
  cedula_base64?: string;
  cedula_content_type?: string;
  rut_base64?: string;
  rut_content_type?: string;
}

// submitProspect es público (sin autenticación) — solicitud de acceso a la plataforma.
export async function submitProspect(data: SubmitProspectPayload): Promise<Prospect> {
  return apiClient.post<Prospect>("/public/prospects", data);
}
