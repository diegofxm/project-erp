import { apiClient } from "./apiClient";
import type { CreateQuotePayload, Quote } from "./types";

export function listQuotes(): Promise<Quote[]> {
  return apiClient.get<Quote[]>("/quotes");
}

export function fetchQuote(id: string): Promise<Quote> {
  return apiClient.get<Quote>(`/quotes/${id}`);
}

export function createQuote(payload: CreateQuotePayload): Promise<Quote> {
  return apiClient.post<Quote>("/quotes", payload);
}

export function sendQuote(id: string): Promise<Quote> {
  return apiClient.post<Quote>(`/quotes/${id}/send`, undefined);
}

export function acceptQuote(id: string): Promise<Quote> {
  return apiClient.post<Quote>(`/quotes/${id}/accept`, undefined);
}

export function rejectQuote(id: string): Promise<void> {
  return apiClient.post<void>(`/quotes/${id}/reject`, undefined);
}

// El backend devuelve la venta completa (saleDTO) — acá solo se toma lo necesario para
// confirmar al usuario y, cuando exista SalesPage (siguiente paso), navegar a /sales/{id}.
export function convertQuoteToSale(id: string): Promise<{ id: string; number: string }> {
  return apiClient.post<{ id: string; number: string }>(`/quotes/${id}/convert-to-sale`, undefined);
}

export function deleteQuote(id: string): Promise<void> {
  return apiClient.del<void>(`/quotes/${id}`);
}
