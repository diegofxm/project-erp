package http

import "net/http"

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Cuentas (PUC — globales, sin company_id)
	mux.HandleFunc("GET /api/v1/accounting/accounts", h.handleListAccounts)
	mux.HandleFunc("GET /api/v1/accounting/withholding-concepts", h.handleListWithholdingConcepts)
	mux.HandleFunc("POST /api/v1/accounting/withholding-certificates", h.handleIssueCertificates)
	mux.HandleFunc("GET /api/v1/accounting/withholding-certificates", h.handleListCertificates)

	mux.HandleFunc("POST /api/v1/accounting/bank-accounts", h.handleCreateBankAccount)
	mux.HandleFunc("GET /api/v1/accounting/bank-accounts", h.handleListBankAccounts)
	mux.HandleFunc("PUT /api/v1/accounting/bank-accounts/{id}", h.handleUpdateBankAccount)
	mux.HandleFunc("DELETE /api/v1/accounting/bank-accounts/{id}", h.handleDeactivateBankAccount)
	mux.HandleFunc("POST /api/v1/accounting/bank-accounts/{id}/statement-lines", h.handleAddStatementLine)
	mux.HandleFunc("GET /api/v1/accounting/bank-accounts/{id}/statement-lines", h.handleListStatement)
	mux.HandleFunc("GET /api/v1/accounting/statement-lines/{line_id}/candidates", h.handleCandidates)
	mux.HandleFunc("POST /api/v1/accounting/statement-lines/{line_id}/reconcile", h.handleReconcile)

	mux.HandleFunc("POST /api/v1/accounting/fixed-assets", h.handleCreateFixedAsset)
	mux.HandleFunc("GET /api/v1/accounting/fixed-assets", h.handleListFixedAssets)
	mux.HandleFunc("POST /api/v1/accounting/depreciation-runs", h.handleRunDepreciation)
	mux.HandleFunc("GET /api/v1/accounting/depreciation-runs", h.handleListDepreciationRuns)

	mux.HandleFunc("POST /api/v1/accounting/budgets", h.handleCreateBudget)
	mux.HandleFunc("GET /api/v1/accounting/budgets", h.handleListBudgets)
	mux.HandleFunc("POST /api/v1/accounting/budgets/{id}/lines", h.handleSetBudgetLine)
	mux.HandleFunc("GET /api/v1/accounting/budgets/{id}/lines", h.handleListBudgetLines)
	mux.HandleFunc("GET /api/v1/accounting/budgets/{id}/compare", h.handleBudgetCompare)
	mux.HandleFunc("POST /api/v1/accounting/budgets/{id}/approve", h.handleApproveBudget)

	mux.HandleFunc("POST /api/v1/accounting/declarations/iva", h.handleGenerateIVA)
	mux.HandleFunc("GET /api/v1/accounting/declarations/iva", h.handleListIVA)
	mux.HandleFunc("POST /api/v1/accounting/declarations/iva/{id}/file", h.handleFileIVA)

	mux.HandleFunc("POST /api/v1/accounting/declarations/income-tax", h.handleGenerateIncomeTax)
	mux.HandleFunc("GET /api/v1/accounting/declarations/income-tax", h.handleListIncomeTax)
	mux.HandleFunc("POST /api/v1/accounting/declarations/income-tax/{id}/file", h.handleFileIncomeTax)

	mux.HandleFunc("POST /api/v1/accounting/ica-tariffs", h.handleSetICATariff)
	mux.HandleFunc("GET /api/v1/accounting/ica-tariffs", h.handleListICATariffs)
	mux.HandleFunc("POST /api/v1/accounting/declarations/ica", h.handleGenerateICA)
	mux.HandleFunc("GET /api/v1/accounting/declarations/ica", h.handleListICA)
	mux.HandleFunc("POST /api/v1/accounting/declarations/ica/{id}/file", h.handleFileICA)
	mux.HandleFunc("GET /api/v1/accounting/accounts/{code}", h.handleGetAccount)

	// Períodos contables
	mux.HandleFunc("GET /api/v1/accounting/periods", h.handleListPeriods)
	mux.HandleFunc("POST /api/v1/accounting/periods/{id}/close", h.handleClosePeriod)
	mux.HandleFunc("POST /api/v1/accounting/periods/{id}/reopen", h.handleReopenPeriod)
	mux.HandleFunc("POST /api/v1/accounting/voucher-types", h.handleCreateVoucherType)
	mux.HandleFunc("GET /api/v1/accounting/voucher-types", h.handleListVoucherTypes)
	mux.HandleFunc("POST /api/v1/accounting/voucher-counters", h.handleSetVoucherCounter)

	// Asientos contables
	mux.HandleFunc("GET /api/v1/accounting/journals", h.handleListJournals)
	mux.HandleFunc("POST /api/v1/accounting/journals", h.handlePostJournal)
	mux.HandleFunc("GET /api/v1/accounting/journals/{id}", h.handleGetJournal)
	mux.HandleFunc("POST /api/v1/accounting/journals/{id}/void", h.handleVoidJournal)

	// Tasas de cambio (TRM)
	mux.HandleFunc("POST /api/v1/accounting/exchange-rates", h.handleSetExchangeRate)
	mux.HandleFunc("GET /api/v1/accounting/exchange-rates", h.handleListExchangeRates)
	mux.HandleFunc("POST /api/v1/accounting/exchange-rates/sync", h.handleSyncExchangeRate)
	mux.HandleFunc("GET /api/v1/accounting/exchange-rates/lookup", h.handleLookupExchangeRate)
	mux.HandleFunc("GET /api/v1/accounting/exchange-rates/today", h.handleGetTodayExchangeRate)

	// Conciliación de cuentas (cruce de partidas, distinto de la conciliación bancaria)
	mux.HandleFunc("GET /api/v1/accounting/reconciliation/open-lines", h.handleListOpenLines)
	mux.HandleFunc("POST /api/v1/accounting/reconciliation", h.handleMarkReconciled)
	mux.HandleFunc("DELETE /api/v1/accounting/reconciliation/{line_id}", h.handleUnmarkReconciled)

	// Reportes financieros
	mux.HandleFunc("GET /api/v1/accounting/reports/pl", h.handlePLReport)
	mux.HandleFunc("GET /api/v1/accounting/reports/bs", h.handleBSReport)
	mux.HandleFunc("GET /api/v1/accounting/reports/trial-balance", h.handleTrialBalance)
	mux.HandleFunc("GET /api/v1/accounting/reports/ledger/{code}", h.handleAccountLedger)
}
