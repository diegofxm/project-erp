// Cliente de rangos de numeración de la empresa activa — a diferencia de lib/catalogs.ts, NO
// se memoiza: son datos propios del tenant, no catálogos de referencia estáticos.
import { apiClient } from "./apiClient";
import type { CreateNumberingRangePayload, ListNumberingRangesResult, NumberingRange } from "./types";

export async function listNumberingRanges(dianDocumentTypeCode?: string): Promise<NumberingRange[]> {
  const query = dianDocumentTypeCode ? `?dian_document_type_code=${encodeURIComponent(dianDocumentTypeCode)}` : "";
  const res = await apiClient.get<ListNumberingRangesResult>(`/numbering-ranges${query}`);
  return res.numbering_ranges;
}

export function createNumberingRange(payload: CreateNumberingRangePayload): Promise<NumberingRange> {
  return apiClient.post<NumberingRange>("/numbering-ranges", payload);
}

// "Borrar" un rango es desactivarlo (is_active=false) — nunca un DELETE real, ver
// numbering.Repository.Deactivate en apidian.
export function deactivateNumberingRange(id: string): Promise<void> {
  return apiClient.del<void>(`/numbering-ranges/${id}`);
}
