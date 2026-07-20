import { apiClient } from "./apiClient";
import type { BillingEntry, Issuer, IssuerSettings, Plan, RenewalEntry, Subscription } from "./types";

export async function adminListPlans(): Promise<Plan[]> {
  const res = await apiClient.get<{ plans: Plan[]; count: number }>("/admin/plans");
  return res.plans;
}

export async function adminCreatePlan(data: Omit<Plan, "id" | "created_at" | "updated_at">): Promise<Plan> {
  return apiClient.post<Plan>("/admin/plans", data);
}

export async function adminUpdatePlan(id: string, data: Partial<Plan>): Promise<Plan> {
  return apiClient.patch<Plan>(`/admin/plans/${id}`, data);
}

export async function adminApplyPlanIncrement(planId: string): Promise<Plan> {
  return apiClient.post<Plan>(`/admin/plans/${planId}/apply-increment`, {});
}

export async function adminGetIssuer(id: string): Promise<Issuer> {
  return apiClient.get<Issuer>(`/admin/issuers/${id}`);
}

export async function adminGetSubscription(issuerId: string): Promise<Subscription> {
  return apiClient.get<Subscription>(`/admin/issuers/${issuerId}/subscription`);
}

export async function adminAssignPlan(issuerId: string, planId: string): Promise<Subscription> {
  return apiClient.post<Subscription>(`/admin/issuers/${issuerId}/subscription`, { plan_id: planId });
}

export async function adminGetIssuerSettings(issuerId: string): Promise<IssuerSettings> {
  return apiClient.get<IssuerSettings>(`/admin/issuers/${issuerId}/settings`);
}

export async function adminUpdateIssuerSettings(issuerId: string, data: Partial<IssuerSettings>): Promise<IssuerSettings> {
  return apiClient.patch<IssuerSettings>(`/admin/issuers/${issuerId}/settings`, data);
}

export async function adminAffiliateIssuer(issuerId: string, feePaidCOP: number): Promise<IssuerSettings> {
  return apiClient.post<IssuerSettings>(`/admin/issuers/${issuerId}/affiliate`, { fee_paid_cop: feePaidCOP });
}

export async function adminRenewIssuer(issuerId: string, feePaidCOP: number): Promise<IssuerSettings> {
  return apiClient.post<IssuerSettings>(`/admin/issuers/${issuerId}/renew`, { fee_paid_cop: feePaidCOP });
}

export async function adminGetBillingSummary(): Promise<BillingEntry[]> {
  const res = await apiClient.get<{ entries: BillingEntry[]; count: number }>("/admin/billing/summary");
  return res.entries;
}

export async function adminGetRenewalsSummary(): Promise<RenewalEntry[]> {
  const res = await apiClient.get<{ entries: RenewalEntry[]; count: number }>("/admin/billing/renewals");
  return res.entries;
}
