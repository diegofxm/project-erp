package reports

// TrialBalanceRow es una fila del balance de comprobación.
type TrialBalanceRow struct {
	AccountCode string
	AccountName string
	Category    string
	TotalDebit  float64
	TotalCredit float64
	Balance     float64
}

// LedgerRow es una fila del libro mayor de una cuenta.
type LedgerRow struct {
	JournalID   string
	Date        string
	Description string
	Debit       float64
	Credit      float64
	RunningBal  float64
}

// IncomeStatementRow es una fila del estado de resultados.
type IncomeStatementRow struct {
	AccountCode string
	AccountName string
	Category    string
	Amount      float64
}

// IncomeStatement agrupa ingresos, gastos y la utilidad neta.
type IncomeStatement struct {
	Revenue     []IncomeStatementRow
	Expenses    []IncomeStatementRow
	NetIncome   float64
}

// BalanceSheetRow es una fila del balance general.
type BalanceSheetRow struct {
	AccountCode string
	AccountName string
	Balance     float64
}

// BalanceSheet agrupa activos, pasivos y patrimonio.
type BalanceSheet struct {
	Assets      []BalanceSheetRow
	Liabilities []BalanceSheetRow
	Equity      []BalanceSheetRow
	TotalAssets float64
	TotalLiabilitiesAndEquity float64
}
