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

// updateQuote -- solo permitido mientras la cotización está en borrador (el backend lo valida).
export function updateQuote(id: string, payload: CreateQuotePayload): Promise<Quote> {
  return apiClient.put<Quote>(`/quotes/${id}`, payload);
}

// sendQuote — cambia el estado a "sent" sin generar ni enviar nada. Se mantiene por si se
// necesita marcar una cotización como enviada sin correo (ej. se entregó en persona). El botón
// "Enviar" del editor usa sendQuoteEmail, que sí genera el PDF y lo manda al cliente.
export function sendQuote(id: string): Promise<Quote> {
  return apiClient.post<Quote>(`/quotes/${id}/send`, undefined);
}

export async function getQuotePdfBlobUrl(id: string): Promise<string> {
  const blob = await apiClient.getBlob(`/quotes/${id}/pdf`);
  return URL.createObjectURL(blob);
}

// sendQuoteEmail — genera el PDF de la cotización y lo envía por correo al cliente (o a `to` si
// se especifica), y de paso la deja en estado "sent". Es el envío real detrás de "Enviar".
export function sendQuoteEmail(id: string, to?: string): Promise<Quote> {
  return apiClient.post<Quote>(`/quotes/${id}/send-email`, to ? { to } : undefined);
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
