import { apiClient } from "./apiClient";
import type { AdminUser, BillingEntry, Company, IssuerSettings, Payment, Plan, Prospect, RenewalEntry, Subscription } from "./types";

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

export async function adminGetCompany(id: string): Promise<Company> {
  return apiClient.get<Company>(`/admin/issuers/${id}`);
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

export async function adminAffiliateCompany(issuerId: string, feePaidCOP: number): Promise<IssuerSettings> {
  return apiClient.post<IssuerSettings>(`/admin/issuers/${issuerId}/affiliate`, { fee_paid_cop: feePaidCOP });
}

export async function adminRenewCompany(issuerId: string, feePaidCOP: number): Promise<IssuerSettings> {
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

export async function adminListUsers(): Promise<AdminUser[]> {
  const res = await apiClient.get<{ users: AdminUser[]; count: number }>("/admin/users");
  return res.users;
}

export async function adminCreateUser(data: { email: string; name: string }): Promise<AdminUser> {
  return apiClient.post<AdminUser>("/admin/users", data);
}

export async function adminListCompanyPayments(issuerId: string): Promise<Payment[]> {
  const res = await apiClient.get<{ payments: Payment[]; count: number }>(`/admin/issuers/${issuerId}/payments`);
  return res.payments;
}

export async function adminListProspects(): Promise<Prospect[]> {
  const res = await apiClient.get<{ prospects: Prospect[]; count: number }>("/admin/prospects");
  return res.prospects;
}

export async function adminApproveProspect(id: string): Promise<Prospect> {
  return apiClient.post<Prospect>(`/admin/prospects/${id}/approve`, {});
}

export async function adminRejectProspect(id: string, notes?: string): Promise<Prospect> {
  return apiClient.post<Prospect>(`/admin/prospects/${id}/reject`, { notes: notes ?? "" });
}

export function adminProspectCedulaUrl(id: string): string {
  return `/admin/prospects/${id}/cedula`;
}

export function adminProspectRutUrl(id: string): string {
  return `/admin/prospects/${id}/rut`;
}
