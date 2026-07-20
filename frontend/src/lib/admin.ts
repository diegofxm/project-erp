import { apiClient } from "./apiClient";
import type { Issuer, Plan, Subscription } from "./types";

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

export async function adminGetIssuer(id: string): Promise<Issuer> {
  return apiClient.get<Issuer>(`/admin/issuers/${id}`);
}

export async function adminGetSubscription(issuerId: string): Promise<Subscription> {
  return apiClient.get<Subscription>(`/admin/issuers/${issuerId}/subscription`);
}

export async function adminAssignPlan(issuerId: string, planId: string): Promise<Subscription> {
  return apiClient.post<Subscription>(`/admin/issuers/${issuerId}/subscription`, { plan_id: planId });
}
