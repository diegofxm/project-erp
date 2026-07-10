import { apiClient } from "./apiClient";
import type { Document, DocumentLine, DocumentLineInput, IssueCreditNotePayload, IssueDebitNotePayload, IssueInvoicePayload, ListDocumentsFilter, ListDocumentsResult } from "./types";

export async function listDocuments(filter?: ListDocumentsFilter): Promise<Document[]> {
  const params = new URLSearchParams();
  if (filter?.dian_document_type_code) params.set("dian_document_type_code", filter.dian_document_type_code);
  if (filter?.status) params.set("status", filter.status);
  if (filter?.from) params.set("from", filter.from);
  if (filter?.to) params.set("to", filter.to);
  if (filter?.limit) params.set("limit", String(filter.limit));
  if (filter?.offset) params.set("offset", String(filter.offset));
  if (filter?.source_document_id) params.set("source_document_id", filter.source_document_id);
  const query = params.toString();
  const res = await apiClient.get<ListDocumentsResult>(`/documents${query ? `?${query}` : ""}`);
  return res.documents;
}

export function getDocument(id: string): Promise<Document> {
  return apiClient.get<Document>(`/documents/${id}`);
}

// getDocumentPdfBlobUrl trae la representación gráfica (generada en memoria en el servidor,
// nunca guardada a disco, ver docs/apidian-architecture.md sección 9.39) y devuelve un
// Object URL para abrirla en una pestaña nueva — necesario porque el endpoint exige
// Authorization: Bearer, que un <a href> plano no puede mandar. Funciona para Factura, NC y ND.
export async function getDocumentPdfBlobUrl(id: string): Promise<string> {
  const blob = await apiClient.getBlob(`/documents/${id}/pdf`);
  return URL.createObjectURL(blob);
}

export function createInvoiceDraft(payload: IssueInvoicePayload): Promise<Document> {
  return apiClient.post<Document>("/invoices", payload);
}

export function updateInvoiceDraft(id: string, payload: IssueInvoicePayload): Promise<Document> {
  return apiClient.put<Document>(`/invoices/${id}`, payload);
}

export function createCreditNoteDraft(payload: IssueCreditNotePayload): Promise<Document> {
  return apiClient.post<Document>("/credit-notes", payload);
}

export function updateCreditNoteDraft(id: string, payload: IssueCreditNotePayload): Promise<Document> {
  return apiClient.put<Document>(`/credit-notes/${id}`, payload);
}

export function createDebitNoteDraft(payload: IssueDebitNotePayload): Promise<Document> {
  return apiClient.post<Document>("/debit-notes", payload);
}

export function updateDebitNoteDraft(id: string, payload: IssueDebitNotePayload): Promise<Document> {
  return apiClient.put<Document>(`/debit-notes/${id}`, payload);
}

export function deleteDraft(id: string): Promise<void> {
  return apiClient.del<void>(`/documents/${id}`);
}

// confirmDocument reclama el consecutivo real, firma y —si el ambiente lo permite— envía a la
// DIAN (ver docs/apidian-architecture.md sección 9.25). Sin body: todo lo que necesita ya está
// en el borrador persistido.
export function confirmDocument(id: string): Promise<Document> {
  return apiClient.post<Document>(`/documents/${id}/confirm`);
}

// sendDocumentEmail envía el documento (debe estar accepted) al correo del cliente con el PDF y
// el XML firmado adjuntos (ver docs/apidian-architecture.md sección 9.42). Funciona para
// Factura, NC y ND — sin body de petición ni de respuesta.
export function sendDocumentEmail(id: string): Promise<void> {
  return apiClient.post<void>(`/documents/${id}/send-email`);
}

// lineToInput es el inverso de lo que calcula el servidor — para editar un borrador ya
// guardado hay que volver a la forma de ENTRADA (cantidad/precio/% de impuesto), no mostrar
// line_extension_cents/taxes[] como si fueran editables. Solo toma el primer impuesto de la
// línea (0 o 1 por línea es lo que esta UI soporta, ver docs/apidian-architecture.md sección
// 9.37) — si una línea ya guardada tuviera más de uno (no posible desde esta UI, sí desde
// otro consumidor de la API), los siguientes se pierden al reeditar.
export function lineToInput(line: DocumentLine): DocumentLineInput {
  const tax = line.taxes?.[0];
  return {
    description: line.description,
    quantity: line.quantity,
    unit_code: line.unit_code,
    unit_price_cents: line.unit_price_cents,
    item_code: line.item_code,
    item_type_code: line.item_type_code,
    tax_type_code: tax?.type_code,
    tax_percent: tax?.percent,
  };
}
