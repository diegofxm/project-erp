import { apiClient } from "./apiClient";
import type {
  Account,
  AccountBalance,
  AccountingPeriod,
  JournalEntry,
  LedgerLine,
  PostJournalPayload,
  TrialBalanceRow,
} from "./types";

export function listAccounts(): Promise<Account[]> {
  return apiClient.get<Account[]>("/accounting/accounts");
}

export function getAccount(code: string): Promise<Account> {
  return apiClient.get<Account>(`/accounting/accounts/${code}`);
}

export function listPeriods(): Promise<AccountingPeriod[]> {
  return apiClient.get<AccountingPeriod[]>("/accounting/periods");
}

export function closePeriod(id: string): Promise<{ status: string }> {
  return apiClient.post<{ status: string }>(`/accounting/periods/${id}/close`);
}

export function listJournals(limit: number, offset: number): Promise<JournalEntry[]> {
  return apiClient.get<JournalEntry[]>(`/accounting/journals?limit=${limit}&offset=${offset}`);
}

export function getJournal(id: string): Promise<JournalEntry> {
  return apiClient.get<JournalEntry>(`/accounting/journals/${id}`);
}

export function postJournal(payload: PostJournalPayload): Promise<JournalEntry> {
  return apiClient.post<JournalEntry>("/accounting/journals", payload);
}

export function voidJournal(id: string): Promise<{ status: string }> {
  return apiClient.post<{ status: string }>(`/accounting/journals/${id}/void`);
}

export function getPLReport(year: number): Promise<AccountBalance[]> {
  return apiClient.get<AccountBalance[]>(`/accounting/reports/pl?year=${year}`);
}

export function getBSReport(asOf: string): Promise<AccountBalance[]> {
  return apiClient.get<AccountBalance[]>(`/accounting/reports/bs?as_of=${asOf}`);
}

export function getTrialBalance(from: string, to: string): Promise<TrialBalanceRow[]> {
  return apiClient.get<TrialBalanceRow[]>(`/accounting/reports/trial-balance?from=${from}&to=${to}`);
}

export function getAccountLedger(code: string, from: string, to: string): Promise<LedgerLine[]> {
  return apiClient.get<LedgerLine[]>(`/accounting/reports/ledger/${code}?from=${from}&to=${to}`);
}
