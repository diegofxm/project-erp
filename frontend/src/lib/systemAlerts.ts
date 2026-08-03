// Cliente de alertas de sistema (certificado y rangos de numeración por vencer/vencidos) —
// el cálculo vive en el backend (erp/internal/shared/notification), este archivo solo consume
// el endpoint. Ver NotificationContext.tsx para cómo se combina con el estado de leído/descartado
// (ese sí sigue siendo local, en localStorage).
import { apiClient } from "./apiClient";

export interface SystemAlert {
  id: string;
  severity: "warning" | "danger";
  title: string;
  message: string;
  link_to: string;
}

interface ListSystemAlertsResult {
  alerts: SystemAlert[];
  count: number;
}

export async function listSystemAlerts(): Promise<SystemAlert[]> {
  const res = await apiClient.get<ListSystemAlertsResult>("/notifications/system-alerts");
  return res.alerts;
}
