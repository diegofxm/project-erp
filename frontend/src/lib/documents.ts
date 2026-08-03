import { apiClient } from "./apiClient";
import type { Document, DocumentLine, DocumentLineInput, IssueAdjustmentNotePayload, IssueCreditNotePayload, IssueDebitNotePayload, IssueInvoicePayload, IssueSupportDocumentPayload, ListDocumentsFilter, ListDocumentsResult } from "./types";

export async function listDocuments(filter?: ListDocumentsFilter): Promise<Document[]> {
  const params = new URLSearchParams();
  if (filter?.dian_document_type_code) params.set("dian_document_type_code", filter.dian_document_type_code);
  if (filter?.status) params.set("status", filter.status);
  if (filter?.from) params.set("from", filter.from);
  if (filter?.to) params.set("to", filter.to);
  if (filter?.limit) params.set("limit", String(filter.limit));
  if (filter?.offset) params.set("offset", String(filter.offset));
  if (filter?.source_document_id) params.set("source_document_id", filter.source_document_id);
  if (filter?.search) params.set("q", filter.search);
  const query = params.toString();
  const res = await apiClient.get<ListDocumentsResult>(`/electronic/documents${query ? `?${query}` : ""}`);
  return res.documents;
}

export function getDocument(id: string): Promise<Document> {
  return apiClient.get<Document>(`/electronic/documents/${id}`);
}

export type PDFFormat = "full_a4" | "half_a4";

export async function getDocumentPdfBlobUrl(id: string, format: PDFFormat = "full_a4"): Promise<string> {
  const query = format !== "full_a4" ? `?format=${format}` : "";
  const blob = await apiClient.getBlob(`/electronic/documents/${id}/pdf${query}`);
  return URL.createObjectURL(blob);
}

// XML UBL firmado — 404 solo si el documento sigue en borrador (aún no se firmó).
export async function getDocumentXmlBlobUrl(id: string): Promise<string> {
  const blob = await apiClient.getBlob(`/electronic/documents/${id}/xml`);
  return URL.createObjectURL(blob);
}

export function createInvoiceDraft(payload: IssueInvoicePayload): Promise<Document> {
  return apiClient.post<Document>("/electronic/invoices", payload);
}

// createInvoiceFromSale — arma el borrador de factura a partir de las líneas de una venta
// confirmada (erp/internal/electronic CreateFromSaleUseCase), en vez de digitarlas de nuevo.
// El borrador resultante se edita/confirma igual que cualquier otro (InvoiceEditorPage).
export function createInvoiceFromSale(saleId: string, numberingRangeId: string): Promise<Document> {
  return apiClient.post<Document>(`/electronic/invoices/from-sale/${saleId}`, { numbering_range_id: numberingRangeId });
}

export function updateInvoiceDraft(id: string, payload: IssueInvoicePayload): Promise<Document> {
  return apiClient.put<Document>(`/electronic/invoices/${id}`, payload);
}

export function createCreditNoteDraft(payload: IssueCreditNotePayload): Promise<Document> {
  return apiClient.post<Document>("/electronic/credit-notes", payload);
}

export function updateCreditNoteDraft(id: string, payload: IssueCreditNotePayload): Promise<Document> {
  return apiClient.put<Document>(`/electronic/credit-notes/${id}`, payload);
}

export function createDebitNoteDraft(payload: IssueDebitNotePayload): Promise<Document> {
  return apiClient.post<Document>("/electronic/debit-notes", payload);
}

export function updateDebitNoteDraft(id: string, payload: IssueDebitNotePayload): Promise<Document> {
  return apiClient.put<Document>(`/electronic/debit-notes/${id}`, payload);
}

export function createSupportDocumentDraft(payload: IssueSupportDocumentPayload): Promise<Document> {
  return apiClient.post<Document>("/electronic/support-documents", payload);
}

export function updateSupportDocumentDraft(id: string, payload: IssueSupportDocumentPayload): Promise<Document> {
  return apiClient.put<Document>(`/electronic/support-documents/${id}`, payload);
}

export function createAdjustmentNoteDraft(payload: IssueAdjustmentNotePayload): Promise<Document> {
  return apiClient.post<Document>("/electronic/adjustment-notes", payload);
}

export function updateAdjustmentNoteDraft(id: string, payload: IssueAdjustmentNotePayload): Promise<Document> {
  return apiClient.put<Document>(`/electronic/adjustment-notes/${id}`, payload);
}

export function deleteDraft(id: string): Promise<void> {
  return apiClient.del<void>(`/electronic/documents/${id}`);
}

export function confirmDocument(id: string): Promise<Document> {
  return apiClient.post<Document>(`/electronic/documents/${id}/confirm`);
}

export function cloneDocument(id: string): Promise<Document> {
  return apiClient.post<Document>(`/electronic/documents/${id}/clone`);
}

export function sendDocumentEmail(id: string, format: PDFFormat = "full_a4", to?: string, cc?: string[]): Promise<void> {
  const body: Record<string, unknown> = {};
  if (format !== "full_a4") body.pdf_format = format;
  if (to) body.to = to;
  if (cc && cc.length > 0) body.cc = cc;
  return apiClient.post<void>(`/electronic/documents/${id}/send-email`, Object.keys(body).length > 0 ? body : undefined);
}

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
