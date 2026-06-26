// Cliente de rangos de numeración de la empresa activa — a diferencia de lib/catalogs.ts, NO
// se memoiza: son datos propios del tenant, no catálogos de referencia estáticos.
import { apiClient } from "./apiClient";
import type { CreateNumberingRangePayload, ListNumberingRangesResult, NumberingRange } from "./types";

export async function listNumberingRanges(): Promise<NumberingRange[]> {
  const res = await apiClient.get<ListNumberingRangesResult>("/numbering-ranges");
  return res.numbering_ranges;
}

export function createNumberingRange(payload: CreateNumberingRangePayload): Promise<NumberingRange> {
  return apiClient.post<NumberingRange>("/numbering-ranges", payload);
}
