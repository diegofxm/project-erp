package accounting_test

// Test de integración contra PostgreSQL real.
//
// Uso:
//
//	$env:TEST_DATABASE_URL = "postgres://usuario:clave@localhost:5432/nombre_db"
//	go test -v -run TestAccounting ./...
//
// Si TEST_DATABASE_URL no está definida el test se salta solo — no bloquea nada.
// Usa la misma base de datos que apidian (DATABASE_URL del .env de apidian).
// Las migraciones y el seed son idempotentes: correrlo varias veces no duplica datos.

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/diegofxm/accounting"
	"github.com/diegofxm/accounting/journals"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// companyID fijo para identificar fácilmente los datos de prueba en la BD.
var testCompanyID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

func openPool(t *testing.T) (*pgxpool.Pool, string) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL no definida — omitiendo test de integración")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("abrir pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool, url
}

func TestAccounting(t *testing.T) {
	pool, dbURL := openPool(t)
	ctx := context.Background()

	// ── 1. Migraciones ──────────────────────────────────────────────────────
	t.Log("── 1. Corriendo migraciones...")
	if err := accounting.Migrate(dbURL); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Log("   OK — schema accounting.* listo")

	// ── 2. Seed del PUC ─────────────────────────────────────────────────────
	t.Log("── 2. Cargando PUC (2500 cuentas)...")
	core := accounting.New(pool)
	if err := core.Seed(ctx); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Log("   OK — PUC cargado")

	// ── 3. Listar algunas cuentas ────────────────────────────────────────────
	t.Log("── 3. Cuentas de movimiento (is_posting=true), primeras 5:")
	all, err := core.Accounts.List(ctx)
	if err != nil {
		t.Fatalf("list accounts: %v", err)
	}
	postable := 0
	shown := 0
	for _, a := range all {
		if a.IsPosting {
			postable++
			if shown < 5 {
				t.Logf("   [%s] %-50s cat=%-20s nat=%s", a.Code, a.Name, a.Category, a.Nature())
				shown++
			}
		}
	}
	t.Logf("   Total cuentas: %d | De movimiento: %d", len(all), postable)

	// ── 4. Registrar asiento válido (simula FE confirmada) ───────────────────
	t.Log("── 4. Registrando asiento de FE (partida doble)...")
	entry, err := core.Journals.Post(ctx, journals.PostRequest{
		CompanyID:   testCompanyID,
		Date:        time.Now(),
		Description: "TEST — FE SETP-001 confirmada",
		Source:      "test",
		EntryType:   journals.EntryAutomatic,
		Lines: []journals.LineRequest{
			{AccountCode: "130505", Debit: 119000000, Description: "Cliente — total con IVA"},
			{AccountCode: "413505", Credit: 100000000, Description: "Venta — subtotal"},
			{AccountCode: "240805", Credit: 19000000, Description: "IVA generado 19%"},
		},
	})
	if err != nil {
		t.Fatalf("post journal: %v", err)
	}
	t.Logf("   OK — asiento ID: %s | status: %s | periodo: %s",
		entry.ID, entry.Status, entry.PeriodID)
	for _, l := range entry.Lines {
		t.Logf("      %s  déb=%-12d  cré=%-12d  %s",
			l.AccountCode, l.Debit, l.Credit, l.Description)
	}

	// ── 5. Intentar asiento desbalanceado (debe fallar) ──────────────────────
	t.Log("── 5. Intentando asiento desbalanceado (debe rechazarse)...")
	_, err = core.Journals.Post(ctx, journals.PostRequest{
		CompanyID:   testCompanyID,
		Date:        time.Now(),
		Description: "TEST — asiento inválido",
		Lines: []journals.LineRequest{
			{AccountCode: "130505", Debit: 50000000},
			{AccountCode: "413505", Credit: 30000000}, // ← no cuadra
		},
	})
	if err == nil {
		t.Fatal("   ERROR: debió rechazar el asiento desbalanceado")
	}
	t.Logf("   OK — rechazado: %v", err)

	// ── 6. Balance de comprobación ───────────────────────────────────────────
	t.Log("── 6. Balance de comprobación (desde inicio del mes):")
	from := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	to := time.Now()
	rows, err := core.Reports.TrialBalance(ctx, testCompanyID, from, to)
	if err != nil {
		t.Fatalf("trial balance: %v", err)
	}
	if len(rows) == 0 {
		t.Log("   (sin movimientos en el periodo)")
	}
	for _, r := range rows {
		t.Logf("   [%s] %-45s  déb=%12d  cré=%12d  saldo=%12d",
			r.AccountCode, r.AccountName, r.TotalDebit, r.TotalCredit, r.Balance)
	}

	// ── 7. Anular el asiento ─────────────────────────────────────────────────
	t.Log("── 7. Anulando el asiento...")
	if err := core.Journals.Void(ctx, entry.ID); err != nil {
		t.Fatalf("void: %v", err)
	}
	voided, err := core.Journals.Get(ctx, entry.ID)
	if err != nil {
		t.Fatalf("get voided: %v", err)
	}
	t.Logf("   OK — status ahora: %s", voided.Status)

	// ── 8. Libro mayor de 130505 ─────────────────────────────────────────────
	t.Log("── 8. Libro mayor cuenta 130505 (Clientes nacionales):")
	ledger, err := core.Reports.GeneralLedger(ctx, testCompanyID, "130505", from, to)
	if err != nil {
		t.Fatalf("general ledger: %v", err)
	}
	if len(ledger) == 0 {
		t.Log("   (sin movimientos — el asiento fue anulado, no aparece en el ledger)")
	}
	for _, l := range ledger {
		t.Logf("   %s  %s  déb=%10d  cré=%10d  saldo=%10d",
			l.Date, fmt.Sprintf("%-40s", l.Description), l.Debit, l.Credit, l.RunningBal)
	}

	t.Log("── Prueba completa ✓")
}
