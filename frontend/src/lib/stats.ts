import { apiClient } from "./apiClient";

export interface PeriodStats {
  revenue_cents: number;
  document_count: number;
  accepted_count: number;
  rejected_count: number;
  draft_count: number;
}

export interface TypeStats {
  type_code: string;
  type_name: string;
  count: number;
  revenue_cents: number;
}

export interface MonthSeries {
  month: string; // "YYYY-MM"
  revenue_cents: number;
  count: number;
  accepted_count: number;
}

export interface BillingStats {
  current_month: PeriodStats;
  previous_month: PeriodStats;
  ytd: PeriodStats;
  by_type: TypeStats[];
  series: MonthSeries[];
}

export function getBillingStats(): Promise<BillingStats> {
  return apiClient.get<BillingStats>("/stats/billing");
}
