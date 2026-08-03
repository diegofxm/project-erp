package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	accountingapp "github.com/diegofxm/erp/internal/accounting/application"
	accountingpostgres "github.com/diegofxm/erp/internal/accounting/infrastructure/persistence/postgres"
	accountingseed "github.com/diegofxm/erp/internal/accounting/infrastructure/persistence/postgres/seed"
	accountinghttp "github.com/diegofxm/erp/internal/accounting/interfaces/http"
	catalogpostgres "github.com/diegofxm/erp/internal/catalog/infrastructure/persistence/postgres"
	"github.com/diegofxm/erp/internal/catalog/infrastructure/persistence/postgres/seed"
	cataloghttp "github.com/diegofxm/erp/internal/catalog/interfaces/http"
	companyapp "github.com/diegofxm/erp/internal/company/application"
	companypostgres "github.com/diegofxm/erp/internal/company/infrastructure/persistence/postgres"
	companyhttp "github.com/diegofxm/erp/internal/company/interfaces/http"
	customerapp "github.com/diegofxm/erp/internal/customer/application"
	customerpostgres "github.com/diegofxm/erp/internal/customer/infrastructure/persistence/postgres"
	customerhttp "github.com/diegofxm/erp/internal/customer/interfaces/http"
	electronicapp "github.com/diegofxm/erp/internal/electronic/application"
	electroniccofacture "github.com/diegofxm/erp/internal/electronic/infrastructure/cofacture"
	electroniccompany "github.com/diegofxm/erp/internal/electronic/infrastructure/company"
	electronicpostgres "github.com/diegofxm/erp/internal/electronic/infrastructure/persistence/postgres"
	electronichttp "github.com/diegofxm/erp/internal/electronic/interfaces/http"
	hrapp "github.com/diegofxm/erp/internal/hr/application"
	hrpostgres "github.com/diegofxm/erp/internal/hr/infrastructure/persistence/postgres"
	hrhttp "github.com/diegofxm/erp/internal/hr/interfaces/http"
	inventoryapp "github.com/diegofxm/erp/internal/inventory/application"
	inventorypostgres "github.com/diegofxm/erp/internal/inventory/infrastructure/persistence/postgres"
	inventoryhttp "github.com/diegofxm/erp/internal/inventory/interfaces/http"
	payrollapp "github.com/diegofxm/erp/internal/payroll/application"
	payrollpostgres "github.com/diegofxm/erp/internal/payroll/infrastructure/persistence/postgres"
	payrollseed "github.com/diegofxm/erp/internal/payroll/infrastructure/persistence/postgres/seed"
	payrollhttp "github.com/diegofxm/erp/internal/payroll/interfaces/http"
	productapp "github.com/diegofxm/erp/internal/product/application"
	productpostgres "github.com/diegofxm/erp/internal/product/infrastructure/persistence/postgres"
	producthttp "github.com/diegofxm/erp/internal/product/interfaces/http"
	purchaseapp "github.com/diegofxm/erp/internal/purchase/application"
	purchasecompany "github.com/diegofxm/erp/internal/purchase/infrastructure/company"
	purchasepostgres "github.com/diegofxm/erp/internal/purchase/infrastructure/persistence/postgres"
	purchasehttp "github.com/diegofxm/erp/internal/purchase/interfaces/http"
	salesapp "github.com/diegofxm/erp/internal/sales/application"
	salescompany "github.com/diegofxm/erp/internal/sales/infrastructure/company"
	salespostgres "github.com/diegofxm/erp/internal/sales/infrastructure/persistence/postgres"
	saleshttp "github.com/diegofxm/erp/internal/sales/interfaces/http"
	securityapp "github.com/diegofxm/erp/internal/security/application"
	securityjwt "github.com/diegofxm/erp/internal/security/infrastructure/jwt"
	securitypostgres "github.com/diegofxm/erp/internal/security/infrastructure/persistence/postgres"
	securityhttp "github.com/diegofxm/erp/internal/security/interfaces/http"
	"github.com/diegofxm/erp/internal/shared/cors"
	"github.com/diegofxm/erp/internal/shared/events"
	"github.com/diegofxm/erp/internal/shared/logger"
	notificationapp "github.com/diegofxm/erp/internal/shared/notification/application"
	notificationdomain "github.com/diegofxm/erp/internal/shared/notification/domain"
	notificationnoop "github.com/diegofxm/erp/internal/shared/notification/infrastructure/email/noop"
	notificationsmtp "github.com/diegofxm/erp/internal/shared/notification/infrastructure/email/smtp"
	notificationcompany "github.com/diegofxm/erp/internal/shared/notification/infrastructure/company"
	notificationnumbering "github.com/diegofxm/erp/internal/shared/notification/infrastructure/numbering"
	notificationhttp "github.com/diegofxm/erp/internal/shared/notification/interfaces/http"
	reportsdomain "github.com/diegofxm/erp/internal/shared/reports/domain"
	htmlreports "github.com/diegofxm/erp/internal/shared/reports/infrastructure/html"
	multireports "github.com/diegofxm/erp/internal/shared/reports/infrastructure/multi"
	pdfreports "github.com/diegofxm/erp/internal/shared/reports/infrastructure/pdf"
	"github.com/diegofxm/erp/internal/shared/tenant"
	auditapp "github.com/diegofxm/erp/internal/audit/application"
	auditpostgres "github.com/diegofxm/erp/internal/audit/infrastructure/persistence/postgres"
	audithttp "github.com/diegofxm/erp/internal/audit/interfaces/http"
	publichttp "github.com/diegofxm/erp/internal/public/interfaces/http"
	statsapp "github.com/diegofxm/erp/internal/stats/application"
	statspostgres "github.com/diegofxm/erp/internal/stats/infrastructure/persistence/postgres"
	statshttp "github.com/diegofxm/erp/internal/stats/interfaces/http"
	supplierapp "github.com/diegofxm/erp/internal/supplier/application"
	supplierpostgres "github.com/diegofxm/erp/internal/supplier/infrastructure/persistence/postgres"
	supplierhttp "github.com/diegofxm/erp/internal/supplier/interfaces/http"
)

func main() {
	_ = godotenv.Load()

	log := logger.New("server")

	databaseURL := mustEnv(log, "DATABASE_URL")
	addr := envOr("ADDR", ":8080")
	encryptionKey := mustHexKey(log, "ENCRYPTION_KEY")

	// ── Base de datos ──────────────────────────────────────────────────────────
	pool := mustOpenDB(log, databaseURL)
	defer pool.Close()

	// ── Migraciones ─────────────────────────────────────────────────────────────
	mlog := logger.New("migrate")
	mustMigrate(mlog, "catalog", catalogpostgres.Migrate(databaseURL))
	mustMigrate(mlog, "security", securitypostgres.Migrate(databaseURL))
	mustMigrate(mlog, "company", companypostgres.Migrate(databaseURL))
	mustMigrate(mlog, "customer", customerpostgres.Migrate(databaseURL))
	mustMigrate(mlog, "supplier", supplierpostgres.Migrate(databaseURL))
	mustMigrate(mlog, "product", productpostgres.Migrate(databaseURL))
	mustMigrate(mlog, "inventory", inventorypostgres.Migrate(databaseURL))
	mustMigrate(mlog, "purchase", purchasepostgres.Migrate(databaseURL))
	mustMigrate(mlog, "sales", salespostgres.Migrate(databaseURL))
	mustMigrate(mlog, "accounting", accountingpostgres.Migrate(databaseURL))
	mustMigrate(mlog, "electronic", electronicpostgres.Migrate(databaseURL))
	mustMigrate(mlog, "payroll", payrollpostgres.Migrate(databaseURL))
	mustMigrate(mlog, "hr", hrpostgres.Migrate(databaseURL))
	mustMigrate(mlog, "audit", auditpostgres.Migrate(databaseURL))

	// ── Bus de eventos ──────────────────────────────────────────────────────────
	bus := events.NewBus()

	// ── Seed ────────────────────────────────────────────────────────────────────
	slog := logger.New("seed")
	if err := seed.All(context.Background(), pool); err != nil {
		slog.Error("seed catálogos fallido", "error", err)
		os.Exit(1)
	}
	slog.Info("catalog OK")
	if err := accountingseed.All(context.Background(), pool); err != nil {
		slog.Error("seed accounting fallido", "error", err)
		os.Exit(1)
	}
	slog.Info("accounting OK")
	if err := payrollseed.All(context.Background(), pool); err != nil {
		slog.Error("seed payroll fallido", "error", err)
		os.Exit(1)
	}
	slog.Info("payroll OK")

	// ── JWT ──────────────────────────────────────────────────────────────────────
	jwtSvc := securityjwt.NewTokenService([]byte(mustEnv(log, "JWT_SECRET")))

	// ── Repositorios ────────────────────────────────────────────────────────────
	catalogRepo := catalogpostgres.NewRepository(pool)
	securityRepo := securitypostgres.NewRepository(pool)
	companyRepo := companypostgres.NewRepository(pool, encryptionKey)
	customerRepo := customerpostgres.NewRepository(pool)
	supplierRepo := supplierpostgres.NewRepository(pool)
	productRepo := productpostgres.NewRepository(pool)
	inventoryRepo := inventorypostgres.NewRepository(pool)
	purchaseRepo := purchasepostgres.NewRepository(pool)
	purchasePaymentRepo := purchasepostgres.NewPaymentRepository(pool)
	salesRepo := salespostgres.NewRepository(pool)
	warehouseRepo := companypostgres.NewWarehouseRepository(pool)
	quoteRepo := salespostgres.NewQuoteRepository(pool)
	paymentRepo := salespostgres.NewPaymentRepository(pool)

	// ── Repositorios — accounting ───────────────────────────────────────────────
	accountingAccountRepo := accountingpostgres.NewAccountRepository(pool)
	accountingPeriodRepo := accountingpostgres.NewPeriodRepository(pool)
	accountingJournalRepo := accountingpostgres.NewJournalRepository(pool)

	// ── Repositorios — electronic ───────────────────────────────────────────────
	electronicDocRepo := electronicpostgres.NewDocumentRepository(pool)
	electronicNumRepo := electronicpostgres.NewNumberingRepository(pool, encryptionKey)
	electronicCompanyPort := electroniccompany.New(companyRepo)
	electronicAdapter := electroniccofacture.New()

	// ── Casos de uso — security ─────────────────────────────────────────────────
	registerUC := securityapp.NewRegisterUseCase(securityRepo, jwtSvc)
	loginUC := securityapp.NewLoginUseCase(securityRepo, jwtSvc)
	selectCompanyUC := securityapp.NewSelectCompanyUseCase(securityRepo, jwtSvc)
	inviteUserUC := securityapp.NewInviteUserUseCase(securityRepo)
	acceptInviteUC := securityapp.NewAcceptInviteUseCase(securityRepo, jwtSvc)
	updateProfileUC := securityapp.NewUpdateProfileUseCase(securityRepo)
	getProfileUC := securityapp.NewGetProfileUseCase(securityRepo)

	// ── Casos de uso — purchase ─────────────────────────────────────────────────
	createPurchaseUC := purchaseapp.NewCreateUseCase(purchaseRepo)
	getPurchaseUC := purchaseapp.NewGetUseCase(purchaseRepo)
	confirmPurchaseUC := purchaseapp.NewConfirmUseCase(purchaseRepo)
	cancelPurchaseUC := purchaseapp.NewCancelUseCase(purchaseRepo)
	receivePurchaseUC := purchaseapp.NewReceiveUseCase(purchaseRepo, bus)
	deletePurchaseUC := purchaseapp.NewDeleteUseCase(purchaseRepo)
	purchasePaymentUC := purchaseapp.NewPaymentUseCase(purchasePaymentRepo, purchaseRepo)

	// ── Casos de uso — sales ────────────────────────────────────────────────────
	createSaleUC := salesapp.NewCreateUseCase(salesRepo)
	getSaleUC := salesapp.NewGetUseCase(salesRepo)
	confirmSaleUC := salesapp.NewConfirmUseCase(salesRepo, bus, productRepo, inventoryRepo, customerRepo, paymentRepo)
	cancelSaleUC := salesapp.NewCancelUseCase(salesRepo)
	quoteUC := salesapp.NewQuoteUseCase(quoteRepo, salesRepo)
	paymentUC := salesapp.NewPaymentUseCase(paymentRepo, salesRepo)

	// ── Casos de uso — accounting ───────────────────────────────────────────────
	postJournalUC := accountingapp.NewPostJournalUseCase(accountingAccountRepo, accountingPeriodRepo, accountingJournalRepo)
	getJournalUC := accountingapp.NewGetJournalUseCase(accountingJournalRepo)
	voidJournalUC := accountingapp.NewVoidJournalUseCase(accountingJournalRepo)
	managePeriodUC := accountingapp.NewManagePeriodUseCase(accountingPeriodRepo)
	onSaleConfirmed := accountingapp.NewOnSaleConfirmed(accountingAccountRepo, accountingPeriodRepo, accountingJournalRepo)
	onSaleConfirmed.Register(bus)
	inventoryOnSaleConfirmed := inventoryapp.NewOnSaleConfirmed(inventoryRepo, productRepo)
	inventoryOnSaleConfirmed.Register(bus)
	onPurchaseReceived := accountingapp.NewOnPurchaseReceived(accountingAccountRepo, accountingPeriodRepo, accountingJournalRepo)
	onPurchaseReceived.Register(bus)
	inventoryOnPurchaseReceived := inventoryapp.NewOnPurchaseReceived(inventoryRepo, productRepo)
	inventoryOnPurchaseReceived.Register(bus)
	onPayrollGenerated := accountingapp.NewOnPayrollGenerated(accountingAccountRepo, accountingPeriodRepo, accountingJournalRepo)
	onPayrollGenerated.Register(bus)

	// ── Audit ───────────────────────────────────────────────────────────────────
	auditUC := auditapp.NewUseCase(auditpostgres.NewRepository(pool), logger.New("audit"))

	// ── Notificaciones de sistema (certificado/rangos por vencer) ───────────────
	notificationCompanyPort := notificationcompany.New(companyRepo)
	notificationNumberingPort := notificationnumbering.New(electronicNumRepo)
	notificationAlertsUC := notificationapp.NewListSystemAlertsUseCase(notificationCompanyPort, notificationNumberingPort)

	// ── Casos de uso — electronic ───────────────────────────────────────────────
	electronicCreateDraftUC := electronicapp.NewCreateDraftUseCase(electronicDocRepo, electronicNumRepo, electronicCompanyPort, catalogRepo)
	electronicConfirmUC := electronicapp.NewConfirmUseCase(electronicDocRepo, electronicNumRepo, electronicCompanyPort, electronicAdapter, electronicAdapter, electronicAdapter)
	electronicGetUC := electronicapp.NewGetDocumentUseCase(electronicDocRepo)
	electronicListUC := electronicapp.NewListDocumentsUseCase(electronicDocRepo)
	electronicNumberingUC := electronicapp.NewManageNumberingUseCase(electronicNumRepo)
	electronicDianRangesUC := electronicapp.NewGetDianRangesUseCase(electronicCompanyPort, electronicAdapter)
	fromSaleUC := electronicapp.NewCreateFromSaleUseCase(electronicCreateDraftUC, salesRepo, customerRepo, productRepo)
	fromPurchaseUC := electronicapp.NewCreateFromPurchaseUseCase(electronicCreateDraftUC, purchaseRepo, supplierRepo, productRepo)

	// ── Renderer de reportes ─────────────────────────────────────────────────────
	htmlRenderer, err := htmlreports.NewRenderer()
	if err != nil {
		log.Error("renderer HTML fallido", "error", err)
		os.Exit(1)
	}
	pdfGenerator := pdfreports.NewGenerator(htmlRenderer)
	multiRenderer := multireports.New(map[reportsdomain.Format]reportsdomain.Renderer{
		reportsdomain.FormatHTML: htmlRenderer,
		reportsdomain.FormatPDF:  pdfGenerator,
	})
	electronicPDFUC := electronicapp.NewGetDocumentPDFUseCase(electronicDocRepo, electronicCompanyPort, electronicNumRepo, multiRenderer)

	// ── Notificaciones ──────────────────────────────────────────────────────────
	smtpHost := os.Getenv("SMTP_HOST")
	var notifier notificationdomain.Notifier
	if smtpHost != "" {
		smtpPort := 587
		if p := os.Getenv("SMTP_PORT"); p != "" {
			if n, err := fmt.Sscanf(p, "%d", &smtpPort); n == 0 || err != nil {
				smtpPort = 587
			}
		}
		tlsMode := os.Getenv("SMTP_TLS") == "true"
		notifier = notificationsmtp.New(notificationsmtp.Config{
			Host:        smtpHost,
			Port:        smtpPort,
			Username:    os.Getenv("SMTP_USERNAME"),
			Password:    os.Getenv("SMTP_PASSWORD"),
			FromAddress: os.Getenv("SMTP_FROM_ADDRESS"),
			FromName:    os.Getenv("SMTP_FROM_NAME"),
			TLS:         tlsMode,
		})
		log.Info("notificaciones por SMTP configuradas", "host", smtpHost)
	} else {
		notifier = notificationnoop.New()
		log.Info("notificaciones: modo noop (SMTP_HOST no configurado)")
	}
	appURL := os.Getenv("APP_URL")
	electronicSendEmailUC := electronicapp.NewSendDocumentEmailUseCase(electronicDocRepo, electronicCompanyPort, electronicNumRepo, multiRenderer, notifier, electronicAdapter, appURL)

	// ── Casos de uso — PDF/email de cotizaciones ────────────────────────────────
	salesCompanyPort := salescompany.New(companyRepo)
	quotePDFUC := salesapp.NewGetQuotePDFUseCase(quoteRepo, customerRepo, salesCompanyPort, multiRenderer)
	sendQuoteEmailUC := salesapp.NewSendQuoteEmailUseCase(quoteRepo, customerRepo, salesCompanyPort, multiRenderer, notifier, appURL)

	// ── Casos de uso — PDF/email de órdenes de compra ───────────────────────────
	purchaseCompanyPort := purchasecompany.New(companyRepo)
	purchasePDFUC := purchaseapp.NewGetPurchaseOrderPDFUseCase(purchaseRepo, supplierRepo, purchaseCompanyPort, multiRenderer)
	sendPurchaseEmailUC := purchaseapp.NewSendPurchaseOrderEmailUseCase(purchaseRepo, supplierRepo, purchaseCompanyPort, multiRenderer, notifier, appURL)

	// ── Casos de uso — hr / payroll / company / inventory / supplier / product / customer ──
	hrAbsenceRepo := hrpostgres.NewAbsenceRepository(pool)
	hrAbsenceUC := hrapp.NewAbsenceUseCase(hrAbsenceRepo)

	payrollEmpRepo := payrollpostgres.NewEmployeeRepository(pool)
	payrollContractRepo := payrollpostgres.NewContractRepository(pool)
	payrollPayslipRepo := payrollpostgres.NewPayslipRepository(pool)
	payrollEmpUC := payrollapp.NewEmployeeUseCase(payrollEmpRepo)
	payrollContractUC := payrollapp.NewContractUseCase(payrollContractRepo, payrollEmpRepo)
	payrollPayslipUC := payrollapp.NewPayslipUseCase(payrollPayslipRepo, payrollContractRepo, payrollEmpRepo, bus)

	moveInventoryUC := inventoryapp.NewMoveUseCase(inventoryRepo)
	getInventoryUC := inventoryapp.NewGetUseCase(inventoryRepo)

	createSupplierUC := supplierapp.NewCreateUseCase(supplierRepo)
	getSupplierUC := supplierapp.NewGetUseCase(supplierRepo)
	updateSupplierUC := supplierapp.NewUpdateUseCase(supplierRepo)
	deleteSupplierUC := supplierapp.NewDeleteUseCase(supplierRepo)

	createProductUC := productapp.NewCreateUseCase(productRepo)
	getProductUC := productapp.NewGetUseCase(productRepo)
	updateProductUC := productapp.NewUpdateUseCase(productRepo)
	deleteProductUC := productapp.NewDeleteUseCase(productRepo)

	createCustomerUC := customerapp.NewCreateUseCase(customerRepo)
	getCustomerUC := customerapp.NewGetUseCase(customerRepo)
	updateCustomerUC := customerapp.NewUpdateUseCase(customerRepo)
	deleteCustomerUC := customerapp.NewDeleteUseCase(customerRepo)

	createCompanyUC := companyapp.NewCreateUseCase(companyRepo, securityRepo)
	getCompanyUC := companyapp.NewGetUseCase(companyRepo)
	updateCompanyProfile := companyapp.NewUpdateProfileUseCase(companyRepo)
	updateCompanyCreds := companyapp.NewUpdateCredentialsUseCase(companyRepo)
	clearCompanyCreds := companyapp.NewClearCredentialUseCase(companyRepo)
	updateCompanyLogo := companyapp.NewUpdateLogoUseCase(companyRepo)
	warehouseUC := companyapp.NewWarehouseUseCase(warehouseRepo)

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
		updateCompanyProfile, updateCompanyCreds, clearCompanyCreds, updateCompanyLogo,
		warehouseUC,
	).RegisterRoutes(mux)
	customerhttp.NewHandler(createCustomerUC, getCustomerUC, updateCustomerUC, deleteCustomerUC).RegisterRoutes(mux)
	supplierhttp.NewHandler(createSupplierUC, getSupplierUC, updateSupplierUC, deleteSupplierUC).RegisterRoutes(mux)
	producthttp.NewHandler(createProductUC, getProductUC, updateProductUC, deleteProductUC).RegisterRoutes(mux)
	inventoryhttp.NewHandler(moveInventoryUC, getInventoryUC).RegisterRoutes(mux)
	purchasehttp.NewHandler(createPurchaseUC, getPurchaseUC, confirmPurchaseUC, cancelPurchaseUC, receivePurchaseUC, deletePurchaseUC, purchasePaymentUC, purchasePDFUC, sendPurchaseEmailUC).RegisterRoutes(mux)
	saleshttp.NewHandler(createSaleUC, getSaleUC, confirmSaleUC, cancelSaleUC, quoteUC, paymentUC, quotePDFUC, sendQuoteEmailUC).RegisterRoutes(mux)
	accountinghttp.NewHandler(postJournalUC, getJournalUC, voidJournalUC, managePeriodUC, accountingAccountRepo).RegisterRoutes(mux)
	electronichttp.NewHandler(electronicCreateDraftUC, electronicConfirmUC, electronicGetUC, electronicListUC, electronicNumberingUC, electronicPDFUC, fromSaleUC, fromPurchaseUC, electronicDianRangesUC, electronicSendEmailUC, auditUC).RegisterRoutes(mux)
	statshttp.NewHandler(statsapp.NewGetBillingStatsUseCase(statspostgres.NewRepository(pool))).RegisterRoutes(mux)
	audithttp.NewHandler(auditUC).RegisterRoutes(mux)
	notificationhttp.NewHandler(notificationAlertsUC).RegisterRoutes(mux)
	publichttp.NewHandler(getCompanyUC, createCustomerUC).RegisterRoutes(mux)
	payrollhttp.NewHandler(payrollEmpUC, payrollContractUC, payrollPayslipUC).RegisterRoutes(mux)
	hrhttp.NewHandler(hrAbsenceUC).RegisterRoutes(mux)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"cofacture erp"}`))
	})

	// ── Middleware (orden: logger → tenant/JWT) ──────────────────────────────────
	handler := cors.Middleware(logger.Middleware(logger.New("http"))(tenant.Middleware(jwtSvc)(mux)))

	log.Info("ERP iniciado", "addr", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Error("servidor detenido", "error", err)
		os.Exit(1)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func mustEnv(log *slog.Logger, key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Error("variable de entorno requerida", "key", key)
		os.Exit(1)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustHexKey(log *slog.Logger, envVar string) []byte {
	raw := mustEnv(log, envVar)
	key, err := hex.DecodeString(raw)
	if err != nil || len(key) != 32 {
		log.Error("clave inválida: se esperan 64 caracteres hex (32 bytes)", "key", envVar)
		os.Exit(1)
	}
	return key
}

func mustOpenDB(log *slog.Logger, url string) *pgxpool.Pool {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		log.Error("parsear DATABASE_URL", "error", err)
		os.Exit(1)
	}
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET TIME ZONE 'America/Bogota'")
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		log.Error("abrir base de datos", "error", err)
		os.Exit(1)
	}
	if err := pool.Ping(ctx); err != nil {
		log.Error("ping base de datos", "error", err)
		os.Exit(1)
	}
	log.Info("base de datos conectada")
	return pool
}

func mustMigrate(log *slog.Logger, module string, err error) {
	if err != nil {
		log.Error("fallida", "module", module, "error", err)
		os.Exit(1)
	}
	log.Info("OK", "module", module)
}
