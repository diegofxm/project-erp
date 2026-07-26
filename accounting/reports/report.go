package reports

// TrialBalanceRow es una fila del balance de comprobación.
type TrialBalanceRow struct {
	AccountCode string
	AccountName string
	Category    string
	TotalDebit  int64
	TotalCredit int64
	Balance     int64
}

// LedgerRow es una fila del libro mayor de una cuenta.
type LedgerRow struct {
	JournalID   string
	Date        string
	Description string
	Debit       int64
	Credit      int64
	RunningBal  int64
}

// IncomeStatementRow es una fila del estado de resultados.
type IncomeStatementRow struct {
	AccountCode string
	AccountName string
	Category    string
	Amount      int64
}

// IncomeStatement agrupa ingresos, gastos y la utilidad neta.
type IncomeStatement struct {
	Revenue     []IncomeStatementRow
	Expenses    []IncomeStatementRow
	NetIncome   int64
}

// BalanceSheetRow es una fila del balance general.
type BalanceSheetRow struct {
	AccountCode string
	AccountName string
	Balance     int64
}

// BalanceSheet agrupa activos, pasivos y patrimonio.
type BalanceSheet struct {
	Assets                    []BalanceSheetRow
	Liabilities               []BalanceSheetRow
	Equity                    []BalanceSheetRow
	TotalAssets               int64
	TotalLiabilitiesAndEquity int64
}

// CostCenterRow es una fila del reporte por centro de costo.
type CostCenterRow struct {
	CostCenter  string
	TotalDebit  int64
	TotalCredit int64
	Balance     int64 // debit - credit
}

// CostCenterReport agrupa las filas y el balance neto.
type CostCenterReport struct {
	Rows    []*CostCenterRow
	NetDebt int64 // suma de balance positivo (gastos netos por centro)
}
