package main

import (
	"context"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	catalogpostgres "github.com/diegofxm/erp/internal/catalog/infrastructure/persistence/postgres"
	"github.com/diegofxm/erp/internal/catalog/infrastructure/persistence/postgres/seed"
	cataloghttp "github.com/diegofxm/erp/internal/catalog/interfaces/http"
	companyapp "github.com/diegofxm/erp/internal/company/application"
	companypostgres "github.com/diegofxm/erp/internal/company/infrastructure/persistence/postgres"
	companyhttp "github.com/diegofxm/erp/internal/company/interfaces/http"
	customerapp "github.com/diegofxm/erp/internal/customer/application"
	customerpostgres "github.com/diegofxm/erp/internal/customer/infrastructure/persistence/postgres"
	customerhttp "github.com/diegofxm/erp/internal/customer/interfaces/http"
	inventoryapp "github.com/diegofxm/erp/internal/inventory/application"
	inventorypostgres "github.com/diegofxm/erp/internal/inventory/infrastructure/persistence/postgres"
	inventoryhttp "github.com/diegofxm/erp/internal/inventory/interfaces/http"
	productapp "github.com/diegofxm/erp/internal/product/application"
	purchaseapp "github.com/diegofxm/erp/internal/purchase/application"
	purchasepostgres "github.com/diegofxm/erp/internal/purchase/infrastructure/persistence/postgres"
	purchasehttp "github.com/diegofxm/erp/internal/purchase/interfaces/http"
	accountingapp "github.com/diegofxm/erp/internal/accounting/application"
	accountingpostgres "github.com/diegofxm/erp/internal/accounting/infrastructure/persistence/postgres"
	accountingseed "github.com/diegofxm/erp/internal/accounting/infrastructure/persistence/postgres/seed"
	accountinghttp "github.com/diegofxm/erp/internal/accounting/interfaces/http"
	electronicapp "github.com/diegofxm/erp/internal/electronic/application"
	electroniccofacture "github.com/diegofxm/erp/internal/electronic/infrastructure/cofacture"
	electroniccompany "github.com/diegofxm/erp/internal/electronic/infrastructure/company"
	electronicpostgres "github.com/diegofxm/erp/internal/electronic/infrastructure/persistence/postgres"
	electronichttp "github.com/diegofxm/erp/internal/electronic/interfaces/http"
	reportsdomain "github.com/diegofxm/erp/internal/shared/reports/domain"
	htmlreports "github.com/diegofxm/erp/internal/shared/reports/infrastructure/html"
	multireports "github.com/diegofxm/erp/internal/shared/reports/infrastructure/multi"
	pdfreports "github.com/diegofxm/erp/internal/shared/reports/infrastructure/pdf"
	hrapp "github.com/diegofxm/erp/internal/hr/application"
	hrpostgres "github.com/diegofxm/erp/internal/hr/infrastructure/persistence/postgres"
	hrhttp "github.com/diegofxm/erp/internal/hr/interfaces/http"
	payrollapp "github.com/diegofxm/erp/internal/payroll/application"
	payrollpostgres "github.com/diegofxm/erp/internal/payroll/infrastructure/persistence/postgres"
	payrollseed "github.com/diegofxm/erp/internal/payroll/infrastructure/persistence/postgres/seed"
	payrollhttp "github.com/diegofxm/erp/internal/payroll/interfaces/http"
	salesapp "github.com/diegofxm/erp/internal/sales/application"
	salespostgres "github.com/diegofxm/erp/internal/sales/infrastructure/persistence/postgres"
	saleshttp "github.com/diegofxm/erp/internal/sales/interfaces/http"
	productpostgres "github.com/diegofxm/erp/internal/product/infrastructure/persistence/postgres"
	producthttp "github.com/diegofxm/erp/internal/product/interfaces/http"
	securityapp "github.com/diegofxm/erp/internal/security/application"
	supplierapp "github.com/diegofxm/erp/internal/supplier/application"
	supplierpostgres "github.com/diegofxm/erp/internal/supplier/infrastructure/persistence/postgres"
	supplierhttp "github.com/diegofxm/erp/internal/supplier/interfaces/http"
	securityjwt "github.com/diegofxm/erp/internal/security/infrastructure/jwt"
	securitypostgres "github.com/diegofxm/erp/internal/security/infrastructure/persistence/postgres"
	securityhttp "github.com/diegofxm/erp/internal/security/interfaces/http"
	"github.com/diegofxm/erp/internal/shared/events"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

func main() {
	// Carga erp/.env si existe; en producción las variables vienen del entorno directamente.
	_ = godotenv.Load()

	databaseURL := mustEnv("DATABASE_URL")
	addr := envOr("ADDR", ":8080")

	// ENCRYPTION_KEY: hex de 32 bytes (64 chars hex) para AES-256-GCM
	encryptionKey := mustHexKey("ENCRYPTION_KEY")

	// ── Base de datos ──────────────────────────────────────────────────────────
	pool := mustOpenDB(databaseURL)
	defer pool.Close()

	// ── Migraciones ─────────────────────────────────────────────────────────────
	mustMigrate("catalog", catalogpostgres.Migrate(databaseURL))
	mustMigrate("security", securitypostgres.Migrate(databaseURL))
	mustMigrate("company", companypostgres.Migrate(databaseURL))
	mustMigrate("customer", customerpostgres.Migrate(databaseURL))
	mustMigrate("supplier", supplierpostgres.Migrate(databaseURL))
	mustMigrate("product", productpostgres.Migrate(databaseURL))
	mustMigrate("inventory", inventorypostgres.Migrate(databaseURL))
	mustMigrate("purchase", purchasepostgres.Migrate(databaseURL))
	mustMigrate("sales", salespostgres.Migrate(databaseURL))
	mustMigrate("accounting", accountingpostgres.Migrate(databaseURL))
	mustMigrate("electronic", electronicpostgres.Migrate(databaseURL))
	mustMigrate("payroll", payrollpostgres.Migrate(databaseURL))
	mustMigrate("hr", hrpostgres.Migrate(databaseURL))

	// ── Bus de eventos ──────────────────────────────────────────────────────────
	bus := events.NewBus()

	// ── Seed ────────────────────────────────────────────────────────────────────
	if err := seed.All(context.Background(), pool); err != nil {
		log.Fatalf("seed catálogos: %v", err)
	}
	if err := accountingseed.All(context.Background(), pool); err != nil {
		log.Fatalf("seed accounting: %v", err)
	}
	if err := payrollseed.All(context.Background(), pool); err != nil {
		log.Fatalf("seed payroll: %v", err)
	}

	// ── JWT ──────────────────────────────────────────────────────────────────────
	jwtSvc := securityjwt.NewTokenService([]byte(mustEnv("JWT_SECRET")))

	// ── Repositorios ────────────────────────────────────────────────────────────
	catalogRepo  := catalogpostgres.NewRepository(pool)
	securityRepo := securitypostgres.NewRepository(pool)
	companyRepo  := companypostgres.NewRepository(pool, encryptionKey)
	customerRepo  := customerpostgres.NewRepository(pool)
	supplierRepo   := supplierpostgres.NewRepository(pool)
	productRepo    := productpostgres.NewRepository(pool)
	inventoryRepo  := inventorypostgres.NewRepository(pool)
	purchaseRepo   := purchasepostgres.NewRepository(pool)
	salesRepo      := salespostgres.NewRepository(pool)

	// ── Repositorios — accounting ───────────────────────────────────────────────
	accountingAccountRepo  := accountingpostgres.NewAccountRepository(pool)
	accountingPeriodRepo   := accountingpostgres.NewPeriodRepository(pool)
	accountingJournalRepo  := accountingpostgres.NewJournalRepository(pool)

	// ── Repositorios — electronic ───────────────────────────────────────────────
	electronicDocRepo      := electronicpostgres.NewDocumentRepository(pool)
	electronicNumRepo      := electronicpostgres.NewNumberingRepository(pool, encryptionKey)
	electronicCompanyPort  := electroniccompany.New(companyRepo)
	electronicAdapter      := electroniccofacture.New()

	// ── Casos de uso — security ─────────────────────────────────────────────────
	registerUC      := securityapp.NewRegisterUseCase(securityRepo, jwtSvc)
	loginUC         := securityapp.NewLoginUseCase(securityRepo, jwtSvc)
	selectCompanyUC := securityapp.NewSelectCompanyUseCase(securityRepo, jwtSvc)
	inviteUserUC    := securityapp.NewInviteUserUseCase(securityRepo)
	acceptInviteUC  := securityapp.NewAcceptInviteUseCase(securityRepo, jwtSvc)
	updateProfileUC := securityapp.NewUpdateProfileUseCase(securityRepo)
	getProfileUC    := securityapp.NewGetProfileUseCase(securityRepo)

	// ── Casos de uso — purchase ─────────────────────────────────────────────────
	createPurchaseUC  := purchaseapp.NewCreateUseCase(purchaseRepo)
	getPurchaseUC     := purchaseapp.NewGetUseCase(purchaseRepo)
	confirmPurchaseUC := purchaseapp.NewConfirmUseCase(purchaseRepo)
	cancelPurchaseUC  := purchaseapp.NewCancelUseCase(purchaseRepo)
	receivePurchaseUC := purchaseapp.NewReceiveUseCase(purchaseRepo, bus)

	// ── Casos de uso — sales ────────────────────────────────────────────────────
	createSaleUC  := salesapp.NewCreateUseCase(salesRepo)
	getSaleUC     := salesapp.NewGetUseCase(salesRepo)
	confirmSaleUC := salesapp.NewConfirmUseCase(salesRepo, bus)
	cancelSaleUC  := salesapp.NewCancelUseCase(salesRepo)

	// ── Casos de uso — accounting ───────────────────────────────────────────────
	postJournalUC   := accountingapp.NewPostJournalUseCase(accountingAccountRepo, accountingPeriodRepo, accountingJournalRepo)
	getJournalUC    := accountingapp.NewGetJournalUseCase(accountingJournalRepo)
	voidJournalUC   := accountingapp.NewVoidJournalUseCase(accountingJournalRepo)
	managePeriodUC  := accountingapp.NewManagePeriodUseCase(accountingPeriodRepo)
	onSaleConfirmed := accountingapp.NewOnSaleConfirmed(accountingAccountRepo, accountingPeriodRepo, accountingJournalRepo)
	onSaleConfirmed.Register(bus)
	inventoryOnSaleConfirmed := inventoryapp.NewOnSaleConfirmed(inventoryRepo)
	inventoryOnSaleConfirmed.Register(bus)

	onPurchaseReceived := accountingapp.NewOnPurchaseReceived(accountingAccountRepo, accountingPeriodRepo, accountingJournalRepo)
	onPurchaseReceived.Register(bus)
	inventoryOnPurchaseReceived := inventoryapp.NewOnPurchaseReceived(inventoryRepo)
	inventoryOnPurchaseReceived.Register(bus)

	onPayrollGenerated := accountingapp.NewOnPayrollGenerated(accountingAccountRepo, accountingPeriodRepo, accountingJournalRepo)
	onPayrollGenerated.Register(bus)

	// ── Casos de uso — electronic ───────────────────────────────────────────────
	electronicCreateDraftUC := electronicapp.NewCreateDraftUseCase(electronicDocRepo, electronicNumRepo, electronicCompanyPort, catalogRepo)
	electronicConfirmUC     := electronicapp.NewConfirmUseCase(electronicDocRepo, electronicNumRepo, electronicCompanyPort, electronicAdapter, electronicAdapter, electronicAdapter)
	electronicGetUC         := electronicapp.NewGetDocumentUseCase(electronicDocRepo)
	electronicListUC        := electronicapp.NewListDocumentsUseCase(electronicDocRepo)
	electronicNumberingUC   := electronicapp.NewManageNumberingUseCase(electronicNumRepo)

	// ── Renderer de reportes ─────────────────────────────────────────────────────
	htmlRenderer, err := htmlreports.NewRenderer()
	if err != nil {
		log.Fatalf("renderer HTML: %v", err)
	}
	pdfGenerator := pdfreports.NewGenerator(htmlRenderer)
	multiRenderer := multireports.New(map[reportsdomain.Format]reportsdomain.Renderer{
		reportsdomain.FormatHTML: htmlRenderer,
		reportsdomain.FormatPDF:  pdfGenerator,
	})
	electronicPDFUC := electronicapp.NewGetDocumentPDFUseCase(electronicDocRepo, electronicCompanyPort, electronicNumRepo, multiRenderer)

	// ── Repositorios — hr ──────────────────────────────────────────────────────
	hrAbsenceRepo := hrpostgres.NewAbsenceRepository(pool)

	// ── Casos de uso — hr ───────────────────────────────────────────────────────
	hrAbsenceUC := hrapp.NewAbsenceUseCase(hrAbsenceRepo)

	// ── Repositorios — payroll ──────────────────────────────────────────────────
	payrollEmpRepo      := payrollpostgres.NewEmployeeRepository(pool)
	payrollContractRepo := payrollpostgres.NewContractRepository(pool)
	payrollPayslipRepo  := payrollpostgres.NewPayslipRepository(pool)

	// ── Casos de uso — payroll ──────────────────────────────────────────────────
	payrollEmpUC      := payrollapp.NewEmployeeUseCase(payrollEmpRepo)
	payrollContractUC := payrollapp.NewContractUseCase(payrollContractRepo, payrollEmpRepo)
	payrollPayslipUC  := payrollapp.NewPayslipUseCase(payrollPayslipRepo, payrollContractRepo, payrollEmpRepo, bus)

	// ── Casos de uso — inventory ────────────────────────────────────────────────
	moveInventoryUC := inventoryapp.NewMoveUseCase(inventoryRepo)
	getInventoryUC  := inventoryapp.NewGetUseCase(inventoryRepo)

	// ── Casos de uso — supplier ─────────────────────────────────────────────────
	createSupplierUC := supplierapp.NewCreateUseCase(supplierRepo)
	getSupplierUC    := supplierapp.NewGetUseCase(supplierRepo)
	updateSupplierUC := supplierapp.NewUpdateUseCase(supplierRepo)
	deleteSupplierUC := supplierapp.NewDeleteUseCase(supplierRepo)

	// ── Casos de uso — product ──────────────────────────────────────────────────
	createProductUC := productapp.NewCreateUseCase(productRepo)
	getProductUC    := productapp.NewGetUseCase(productRepo)
	updateProductUC := productapp.NewUpdateUseCase(productRepo)
	deleteProductUC := productapp.NewDeleteUseCase(productRepo)

	// ── Casos de uso — customer ─────────────────────────────────────────────────
	createCustomerUC := customerapp.NewCreateUseCase(customerRepo)
	getCustomerUC    := customerapp.NewGetUseCase(customerRepo)
	updateCustomerUC := customerapp.NewUpdateUseCase(customerRepo)
	deleteCustomerUC := customerapp.NewDeleteUseCase(customerRepo)

	// ── Repositorios — company (extra) ─────────────────────────────────────────
	warehouseRepo := companypostgres.NewWarehouseRepository(pool)

	// ── Repositorios — sales (extra) ────────────────────────────────────────────
	quoteRepo   := salespostgres.NewQuoteRepository(pool)
	paymentRepo := salespostgres.NewPaymentRepository(pool)

	// ── Casos de uso — company ──────────────────────────────────────────────────
	// securityRepo satisface MembershipLinker estructuralmente (AddCompany toma string)
	createCompanyUC      := companyapp.NewCreateUseCase(companyRepo, securityRepo)
	getCompanyUC         := companyapp.NewGetUseCase(companyRepo)
	updateCompanyProfile := companyapp.NewUpdateProfileUseCase(companyRepo)
	updateCompanyCreds   := companyapp.NewUpdateCredentialsUseCase(companyRepo)
	updateCompanyLogo    := companyapp.NewUpdateLogoUseCase(companyRepo)
	warehouseUC := companyapp.NewWarehouseUseCase(warehouseRepo)

	// ── Casos de uso — sales (extra) ────────────────────────────────────────────
	quoteUC   := salesapp.NewQuoteUseCase(quoteRepo, salesRepo)
	paymentUC := salesapp.NewPaymentUseCase(paymentRepo, salesRepo)

	// ── Casos de uso — electronic (from-sale) ───────────────────────────────────
	fromSaleUC := electronicapp.NewCreateFromSaleUseCase(electronicCreateDraftUC, salesRepo, customerRepo, productRepo)

	// ── Handlers HTTP ────────────────────────────────────────────────────────────
	mux := http.NewServeMux()
	cataloghttp.NewHandler(catalogRepo).RegisterRoutes(mux)
	securityhttp.NewHandler(
		registerUC, loginUC, selectCompanyUC,
		inviteUserUC, acceptInviteUC,
		updateProfileUC, getProfileUC,
	).RegisterRoutes(mux)
	companyhttp.NewHandler(
		createCompanyUC, getCompanyUC,
		updateCompanyProfile, updateCompanyCreds, updateCompanyLogo,
		warehouseUC,
	).RegisterRoutes(mux)
	customerhttp.NewHandler(createCustomerUC, getCustomerUC, updateCustomerUC, deleteCustomerUC).RegisterRoutes(mux)
	supplierhttp.NewHandler(createSupplierUC, getSupplierUC, updateSupplierUC, deleteSupplierUC).RegisterRoutes(mux)
	producthttp.NewHandler(createProductUC, getProductUC, updateProductUC, deleteProductUC).RegisterRoutes(mux)
	inventoryhttp.NewHandler(moveInventoryUC, getInventoryUC).RegisterRoutes(mux)
	purchasehttp.NewHandler(createPurchaseUC, getPurchaseUC, confirmPurchaseUC, cancelPurchaseUC, receivePurchaseUC).RegisterRoutes(mux)
	saleshttp.NewHandler(createSaleUC, getSaleUC, confirmSaleUC, cancelSaleUC, quoteUC, paymentUC).RegisterRoutes(mux)
	accountinghttp.NewHandler(postJournalUC, getJournalUC, voidJournalUC, managePeriodUC, accountingAccountRepo).RegisterRoutes(mux)
	electronichttp.NewHandler(electronicCreateDraftUC, electronicConfirmUC, electronicGetUC, electronicListUC, electronicNumberingUC, electronicPDFUC, fromSaleUC).RegisterRoutes(mux)
	payrollhttp.NewHandler(payrollEmpUC, payrollContractUC, payrollPayslipUC).RegisterRoutes(mux)
	hrhttp.NewHandler(hrAbsenceUC).RegisterRoutes(mux)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// ── Middleware ───────────────────────────────────────────────────────────────
	handler := tenant.Middleware(jwtSvc)(mux)

	log.Printf("servidor ERP en %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("variable de entorno requerida: %s", key)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// mustHexKey decodifica una clave hex de 64 chars (32 bytes) para AES-256.
func mustHexKey(envVar string) []byte {
	raw := mustEnv(envVar)
	key, err := hex.DecodeString(raw)
	if err != nil || len(key) != 32 {
		log.Fatalf("%s debe ser exactamente 64 caracteres hexadecimales (32 bytes)", envVar)
	}
	return key
}

func mustOpenDB(url string) *pgxpool.Pool {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		log.Fatalf("parsear DATABASE_URL: %v", err)
	}
	// Todas las conexiones operan en hora de Colombia (UTC-5).
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET TIME ZONE 'America/Bogota'")
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		log.Fatalf("abrir base de datos: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping base de datos: %v", err)
	}
	return pool
}

func mustMigrate(module string, err error) {
	if err != nil {
		log.Fatalf("migración %s: %v", module, err)
	}
}
