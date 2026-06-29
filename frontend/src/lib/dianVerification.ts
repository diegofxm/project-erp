// Cliente de la ayuda OPCIONAL al capturar un NIT (GetAcquirer, ver
// docs/apidian-architecture.md sección 9.41) — nunca bloqueante: found=false es el resultado
// normal y esperado para la mayoría de identificaciones, no un error.
import { apiClient } from "./apiClient";

export interface AcquirerVerification {
  found: boolean;
  receiver_name?: string;
  receiver_email?: string;
}

export function verifyAcquirer(typeCode: string, number: string): Promise<AcquirerVerification> {
  const query = `?identification_type_code=${encodeURIComponent(typeCode)}&identification_number=${encodeURIComponent(number)}`;
  return apiClient.get<AcquirerVerification>(`/dian/verify-acquirer${query}`);
}
