import { apiClient } from "./apiClient";
import type {
  AdminUser, BillingEntry, CompanyInfo, Payment, Plan, Prospect, RenewalEntry, SaasModule,
  SaasSettings, Subscription,
} from "./types";

export async function adminGetCompanyInfo(id: string): Promise<CompanyInfo> {
  return apiClient.get<CompanyInfo>(`/admin/companies/${id}`);
}

export async function adminListModules(): Promise<SaasModule[]> {
  const res = await apiClient.get<{ modules: SaasModule[] }>("/admin/modules");
  return res.modules;
}

export async function adminListPlans(): Promise<Plan[]> {
  const res = await apiClient.get<{ plans: Plan[]; count: number }>("/admin/plans");
  return res.plans;
}

export async function adminCreatePlan(data: Omit<Plan, "id" | "created_at" | "updated_at" | "is_active">): Promise<Plan> {
  return apiClient.post<Plan>("/admin/plans", data);
}

export async function adminUpdatePlan(id: string, data: Partial<Plan>): Promise<Plan> {
  return apiClient.patch<Plan>(`/admin/plans/${id}`, data);
}

export async function adminApplyPlanIncrement(planId: string): Promise<Plan> {
  return apiClient.post<Plan>(`/admin/plans/${planId}/apply-increment`, {});
}

export async function adminGetSettings(): Promise<SaasSettings> {
  return apiClient.get<SaasSettings>("/admin/settings");
}

export async function adminUpdateSettings(ivaRateBP: number): Promise<SaasSettings> {
  return apiClient.patch<SaasSettings>("/admin/settings", { iva_rate_bp: ivaRateBP });
}

export async function adminGetSubscription(companyId: string): Promise<Subscription> {
  return apiClient.get<Subscription>(`/admin/companies/${companyId}/subscription`);
}

export async function adminAssignPlan(companyId: string, planId: string, hasOwnCertificate: boolean): Promise<Subscription> {
  return apiClient.post<Subscription>(`/admin/companies/${companyId}/subscription`, {
    plan_id: planId, has_own_certificate: hasOwnCertificate,
  });
}

export async function adminRenewSubscription(companyId: string): Promise<Subscription> {
  return apiClient.post<Subscription>(`/admin/companies/${companyId}/subscription/renew`, {});
}

export async function adminGetBillingSummary(): Promise<BillingEntry[]> {
  const res = await apiClient.get<{ entries: BillingEntry[]; count: number }>("/admin/billing/summary");
  return res.entries;
}

export async function adminGetRenewalsSummary(withinDays = 90): Promise<RenewalEntry[]> {
  const res = await apiClient.get<{ entries: RenewalEntry[]; count: number }>(`/admin/billing/renewals?within_days=${withinDays}`);
  return res.entries;
}

export async function adminListUsers(): Promise<AdminUser[]> {
  const res = await apiClient.get<{ users: AdminUser[]; count: number }>("/admin/users");
  return res.users;
}

export async function adminSetUserSuperAdmin(id: string, isSuperAdmin: boolean): Promise<void> {
  await apiClient.patch(`/admin/users/${id}/superadmin`, { is_superadmin: isSuperAdmin });
}

export async function adminListCompanyPayments(companyId: string): Promise<Payment[]> {
  const res = await apiClient.get<{ payments: Payment[]; count: number }>(`/admin/companies/${companyId}/payments`);
  return res.payments;
}

export async function adminRecordPayment(companyId: string, data: { subscription_id?: string; type: Payment["type"]; amount_cents: number; note?: string; paid_at?: string }): Promise<Payment> {
  return apiClient.post<Payment>(`/admin/companies/${companyId}/payments`, data);
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
