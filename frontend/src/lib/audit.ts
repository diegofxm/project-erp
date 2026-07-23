import { apiClient } from "./apiClient";
import type { AuditEvent, ListAuditEventsFilter } from "./types";

export async function listAuditEvents(filter?: ListAuditEventsFilter): Promise<AuditEvent[]> {
  const params = new URLSearchParams();
  if (filter?.resource_id) params.set("resource_id", filter.resource_id);
  if (filter?.limit) params.set("limit", String(filter.limit));
  if (filter?.offset) params.set("offset", String(filter.offset));
  const query = params.toString();
  const res = await apiClient.get<{ events: AuditEvent[]; count: number }>(
    `/audit-events${query ? `?${query}` : ""}`
  );
  return res.events;
}
