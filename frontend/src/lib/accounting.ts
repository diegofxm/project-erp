import { apiClient } from "./apiClient";
import type {
  Account,
  AccountBalance,
  AccountingPeriod,
  Budget,
  BudgetActualRow,
  BudgetLine,
  BankAccount,
  DepreciationRun,
  FixedAsset,
  ICADeclaration,
  ICATariff,
  IncomeTaxDeclaration,
  IVADeclaration,
  JournalEntry,
  LedgerLine,
  ExchangeRate,
  OpenLine,
  PostJournalPayload,
  ReconciliationCandidate,
  StatementLine,
  TrialBalanceRow,
  VoucherType,
  WithholdingCertificate,
  WithholdingConcept,
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

export function reopenPeriod(id: string): Promise<{ status: string }> {
  return apiClient.post<{ status: string }>(`/accounting/periods/${id}/reopen`);
}

// ── Tipos de comprobante ──────────────────────────────────────────────────────

export function listVoucherTypes(): Promise<VoucherType[]> {
  return apiClient.get<VoucherType[]>("/accounting/voucher-types");
}

export function createVoucherType(payload: { code: string; name: string; resets_annually: boolean }): Promise<VoucherType> {
  return apiClient.post<VoucherType>("/accounting/voucher-types", payload);
}

// ── Retenciones ───────────────────────────────────────────────────────────────

export function listWithholdingConcepts(): Promise<WithholdingConcept[]> {
  return apiClient.get<WithholdingConcept[]>("/accounting/withholding-concepts");
}

export function issueCertificates(payload: { supplier_id: string; third_party_nit: string; fiscal_year: number }): Promise<WithholdingCertificate[]> {
  return apiClient.post<WithholdingCertificate[]>("/accounting/withholding-certificates", payload);
}

export function listCertificates(year: number): Promise<WithholdingCertificate[]> {
  return apiClient.get<WithholdingCertificate[]>(`/accounting/withholding-certificates?year=${year}`);
}

// ── Bancos ────────────────────────────────────────────────────────────────────

export function listBankAccounts(): Promise<BankAccount[]> {
  return apiClient.get<BankAccount[]>("/accounting/bank-accounts");
}

export function createBankAccount(payload: { name: string; bank_name: string; account_no: string; account_code: string }): Promise<BankAccount> {
  return apiClient.post<BankAccount>("/accounting/bank-accounts", payload);
}

export function listStatement(bankAccountId: string): Promise<StatementLine[]> {
  return apiClient.get<StatementLine[]>(`/accounting/bank-accounts/${bankAccountId}/statement-lines`);
}

export function addStatementLine(bankAccountId: string, payload: { date: string; description: string; debit_cents: number; credit_cents: number; reference?: string }): Promise<StatementLine> {
  return apiClient.post<StatementLine>(`/accounting/bank-accounts/${bankAccountId}/statement-lines`, payload);
}

export function getCandidates(lineId: string): Promise<ReconciliationCandidate[]> {
  return apiClient.get<ReconciliationCandidate[]>(`/accounting/statement-lines/${lineId}/candidates`);
}

export function reconcile(lineId: string, journalLineId: string): Promise<{ status: string }> {
  return apiClient.post<{ status: string }>(`/accounting/statement-lines/${lineId}/reconcile`, { journal_line_id: journalLineId });
}

// ── Activos fijos ─────────────────────────────────────────────────────────────

export function listFixedAssets(): Promise<FixedAsset[]> {
  return apiClient.get<FixedAsset[]>("/accounting/fixed-assets");
}

export interface CreateFixedAssetPayload {
  code: string;
  name: string;
  description?: string;
  asset_account: string;
  depreciation_account: string;
  accumulated_account: string;
  acquisition_date: string;
  acquisition_cost_cents: number;
  salvage_value_cents: number;
  useful_life_months: number;
  third_party_nit?: string;
}

export function createFixedAsset(payload: CreateFixedAssetPayload): Promise<FixedAsset> {
  return apiClient.post<FixedAsset>("/accounting/fixed-assets", payload);
}

export function runDepreciation(date: string): Promise<DepreciationRun> {
  return apiClient.post<DepreciationRun>("/accounting/depreciation-runs", { date });
}

export function listDepreciationRuns(): Promise<DepreciationRun[]> {
  return apiClient.get<DepreciationRun[]>("/accounting/depreciation-runs");
}

// ── Presupuestos ──────────────────────────────────────────────────────────────

export function listBudgets(year?: number): Promise<Budget[]> {
  return apiClient.get<Budget[]>(`/accounting/budgets${year ? `?year=${year}` : ""}`);
}

export function createBudget(year: number, name: string): Promise<Budget> {
  return apiClient.post<Budget>("/accounting/budgets", { year, name });
}

export function setBudgetLine(budgetId: string, accountCode: string, months: number[]): Promise<BudgetLine> {
  return apiClient.post<BudgetLine>(`/accounting/budgets/${budgetId}/lines`, { account_code: accountCode, months });
}

export function listBudgetLines(budgetId: string): Promise<BudgetLine[]> {
  return apiClient.get<BudgetLine[]>(`/accounting/budgets/${budgetId}/lines`);
}

export function compareBudget(budgetId: string): Promise<BudgetActualRow[]> {
  return apiClient.get<BudgetActualRow[]>(`/accounting/budgets/${budgetId}/compare`);
}

export function approveBudget(budgetId: string): Promise<{ status: string }> {
  return apiClient.post<{ status: string }>(`/accounting/budgets/${budgetId}/approve`);
}

// ── Declaración de IVA ────────────────────────────────────────────────────────

export function generateIVA(periodStart: string, periodEnd: string, periodType: string): Promise<IVADeclaration> {
  return apiClient.post<IVADeclaration>("/accounting/declarations/iva", { period_start: periodStart, period_end: periodEnd, period_type: periodType });
}

export function listIVA(): Promise<IVADeclaration[]> {
  return apiClient.get<IVADeclaration[]>("/accounting/declarations/iva");
}

export function fileIVA(id: string): Promise<{ status: string }> {
  return apiClient.post<{ status: string }>(`/accounting/declarations/iva/${id}/file`);
}

// ── Declaración de Renta ──────────────────────────────────────────────────────

export function generateIncomeTax(fiscalYear: number): Promise<IncomeTaxDeclaration> {
  return apiClient.post<IncomeTaxDeclaration>("/accounting/declarations/income-tax", { fiscal_year: fiscalYear });
}

export function listIncomeTax(): Promise<IncomeTaxDeclaration[]> {
  return apiClient.get<IncomeTaxDeclaration[]>("/accounting/declarations/income-tax");
}

export function fileIncomeTax(id: string): Promise<{ status: string }> {
  return apiClient.post<{ status: string }>(`/accounting/declarations/income-tax/${id}/file`);
}

// ── Declaración de ICA ────────────────────────────────────────────────────────

export function setICATariff(payload: { municipality_code: string; ciiu_code: string; fiscal_year: number; rate_bp: number; surcharge_bp: number }): Promise<ICATariff> {
  return apiClient.post<ICATariff>("/accounting/ica-tariffs", payload);
}

export function listICATariffs(): Promise<ICATariff[]> {
  return apiClient.get<ICATariff[]>("/accounting/ica-tariffs");
}

export interface GenerateICAPayload {
  municipality_code: string;
  ciiu_code: string;
  period_start: string;
  period_end: string;
  period_type: string;
  deductions_cents?: number;
}

export function generateICA(payload: GenerateICAPayload): Promise<ICADeclaration> {
  return apiClient.post<ICADeclaration>("/accounting/declarations/ica", payload);
}

export function listICA(): Promise<ICADeclaration[]> {
  return apiClient.get<ICADeclaration[]>("/accounting/declarations/ica");
}

export function fileICA(id: string): Promise<{ status: string }> {
  return apiClient.post<{ status: string }>(`/accounting/declarations/ica/${id}/file`);
}

// ── Tasas de cambio (TRM) ──────────────────────────────────────────────────────

export interface SetExchangeRatePayload {
  rate_date: string;
  from_currency: string;
  to_currency?: string;
  rate: number;
  source?: string;
  description?: string;
}

export function setExchangeRate(payload: SetExchangeRatePayload): Promise<ExchangeRate> {
  return apiClient.post<ExchangeRate>("/accounting/exchange-rates", payload);
}

export function listExchangeRates(from: string, to: string): Promise<ExchangeRate[]> {
  return apiClient.get<ExchangeRate[]>(`/accounting/exchange-rates?from=${from}&to=${to}`);
}

// Consulta la TRM oficial vigente (Superfinanciera, vía el servicio propio de TRM) y la guarda
// como tasa del día — si falla (sin red, servicio caído), la captura manual de arriba sigue
// siendo el respaldo. También corre sola todos los días a la 1 a.m. hora Colombia (ver
// accounting.RunTRMDailySync en el backend) — este botón es solo el respaldo manual.
export function syncExchangeRate(): Promise<ExchangeRate> {
  return apiClient.post<ExchangeRate>("/accounting/exchange-rates/sync");
}

// Busca la TRM de una fecha puntual — primero en la base local; si no está, el backend la trae
// una sola vez del servicio TRM propio y la guarda (la TRM histórica no cambia). Pensada para que
// el contador consulte cualquier fecha pasada sin esperar a que el disparador diario haya pasado
// por ahí.
export function lookupExchangeRate(date: string, from = "USD"): Promise<ExchangeRate> {
  return apiClient.get<ExchangeRate>(`/accounting/exchange-rates/lookup?date=${date}&from=${from}`);
}

// ── Conciliación de cuentas (cruce de partidas) ─────────────────────────────────

export function listOpenLines(accountCode: string): Promise<OpenLine[]> {
  return apiClient.get<OpenLine[]>(`/accounting/reconciliation/open-lines?account_code=${accountCode}`);
}

export function markReconciled(journalLineId: string, reconciledWith: string | null, note: string): Promise<void> {
  return apiClient.post<void>("/accounting/reconciliation", {
    journal_line_id: journalLineId, reconciled_with: reconciledWith, note,
  });
}

export function unmarkReconciled(journalLineId: string): Promise<void> {
  return apiClient.del<void>(`/accounting/reconciliation/${journalLineId}`);
}
