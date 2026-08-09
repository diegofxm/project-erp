package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	accountingapp "github.com/diegofxm/erp/internal/accounting/application"
	accountingdomain "github.com/diegofxm/erp/internal/accounting/domain"
	accountingpostgres "github.com/diegofxm/erp/internal/accounting/infrastructure/persistence/postgres"
	accountingseed "github.com/diegofxm/erp/internal/accounting/infrastructure/persistence/postgres/seed"
	accountingtrmapi "github.com/diegofxm/erp/internal/accounting/infrastructure/trmapi"
	accountinghttp "github.com/diegofxm/erp/internal/accounting/interfaces/http"
	auditapp "github.com/diegofxm/erp/internal/audit/application"
	auditpostgres "github.com/diegofxm/erp/internal/audit/infrastructure/persistence/postgres"
	auditsecurity "github.com/diegofxm/erp/internal/audit/infrastructure/security"
	audithttp "github.com/diegofxm/erp/internal/audit/interfaces/http"
	catalogpostgres "github.com/diegofxm/erp/internal/catalog/infrastructure/persistence/postgres"
	"github.com/diegofxm/erp/internal/catalog/infrastructure/persistence/postgres/seed"
	cataloghttp "github.com/diegofxm/erp/internal/catalog/interfaces/http"
	companyapp "github.com/diegofxm/erp/internal/company/application"
	companypostgres "github.com/diegofxm/erp/internal/company/infrastructure/persistence/postgres"
	companysaas "github.com/diegofxm/erp/internal/company/infrastructure/saas"
	companyhttp "github.com/diegofxm/erp/internal/company/interfaces/http"
	electronicapp "github.com/diegofxm/erp/internal/electronic/application"
	electroniccofacture "github.com/diegofxm/erp/internal/electronic/infrastructure/cofacture"
	electroniccompany "github.com/diegofxm/erp/internal/electronic/infrastructure/company"
	electronicpostgres "github.com/diegofxm/erp/internal/electronic/infrastructure/persistence/postgres"
	electronicthirdparty "github.com/diegofxm/erp/internal/electronic/infrastructure/thirdparty"
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
	publichttp "github.com/diegofxm/erp/internal/public/interfaces/http"
	purchaseapp "github.com/diegofxm/erp/internal/purchase/application"
	purchasecompany "github.com/diegofxm/erp/internal/purchase/infrastructure/company"
	purchasepostgres "github.com/diegofxm/erp/internal/purchase/infrastructure/persistence/postgres"
	purchasethirdparty "github.com/diegofxm/erp/internal/purchase/infrastructure/thirdparty"
	purchasehttp "github.com/diegofxm/erp/internal/purchase/interfaces/http"
	saasapp "github.com/diegofxm/erp/internal/saas/application"
	saascompany "github.com/diegofxm/erp/internal/saas/infrastructure/company"
	saaselectronic "github.com/diegofxm/erp/internal/saas/infrastructure/electronic"
	saaspostgres "github.com/diegofxm/erp/internal/saas/infrastructure/persistence/postgres"
	saasseed "github.com/diegofxm/erp/internal/saas/infrastructure/persistence/postgres/seed"
	saassecurity "github.com/diegofxm/erp/internal/saas/infrastructure/security"
	saashttp "github.com/diegofxm/erp/internal/saas/interfaces/http"
	salesapp "github.com/diegofxm/erp/internal/sales/application"
	salescompany "github.com/diegofxm/erp/internal/sales/infrastructure/company"
	salespostgres "github.com/diegofxm/erp/internal/sales/infrastructure/persistence/postgres"
	salesthirdparty "github.com/diegofxm/erp/internal/sales/infrastructure/thirdparty"
	saleshttp "github.com/diegofxm/erp/internal/sales/interfaces/http"
	securityapp "github.com/diegofxm/erp/internal/security/application"
	securityjwt "github.com/diegofxm/erp/internal/security/infrastructure/jwt"
	securitypostgres "github.com/diegofxm/erp/internal/security/infrastructure/persistence/postgres"
	securityseed "github.com/diegofxm/erp/internal/security/infrastructure/persistence/postgres/seed"
	securityhttp "github.com/diegofxm/erp/internal/security/interfaces/http"
	"github.com/diegofxm/erp/internal/shared/cors"
	"github.com/diegofxm/erp/internal/shared/events"
	"github.com/diegofxm/erp/internal/shared/logger"
	notificationapp "github.com/diegofxm/erp/internal/shared/notification/application"
	notificationdomain "github.com/diegofxm/erp/internal/shared/notification/domain"
	notificationcompany "github.com/diegofxm/erp/internal/shared/notification/infrastructure/company"
	notificationnoop "github.com/diegofxm/erp/internal/shared/notification/infrastructure/email/noop"
	notificationsmtp "github.com/diegofxm/erp/internal/shared/notification/infrastructure/email/smtp"
	notificationnumbering "github.com/diegofxm/erp/internal/shared/notification/infrastructure/numbering"
	notificationstock "github.com/diegofxm/erp/internal/shared/notification/infrastructure/stock"
	notificationhttp "github.com/diegofxm/erp/internal/shared/notification/interfaces/http"
	reportsdomain "github.com/diegofxm/erp/internal/shared/reports/domain"
	htmlreports "github.com/diegofxm/erp/internal/shared/reports/infrastructure/html"
	multireports "github.com/diegofxm/erp/internal/shared/reports/infrastructure/multi"
	pdfreports "github.com/diegofxm/erp/internal/shared/reports/infrastructure/pdf"
	"github.com/diegofxm/erp/internal/shared/secheaders"
	"github.com/diegofxm/erp/internal/shared/tenant"
	thirdpartyapp "github.com/diegofxm/erp/internal/thirdparty/application"
	thirdpartypostgres "github.com/diegofxm/erp/internal/thirdparty/infrastructure/persistence/postgres"
	thirdpartyhttp "github.com/diegofxm/erp/internal/thirdparty/interfaces/http"
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
	mustMigrate(mlog, "saas", saaspostgres.Migrate(databaseURL))
	mustMigrate(mlog, "thirdparty", thirdpartypostgres.Migrate(databaseURL))
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
	if err := saasseed.All(context.Background(), pool); err != nil {
		slog.Error("seed saas fallido", "error", err)
		os.Exit(1)
	}
	slog.Info("saas OK")
	if err := securityseed.All(context.Background(), pool); err != nil {
		slog.Error("seed security fallido", "error", err)
		os.Exit(1)
	}
	slog.Info("security OK")
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
	thirdpartyRepo := thirdpartypostgres.NewRepository(pool)
	productRepo := productpostgres.NewRepository(pool)
	inventoryRepo := inventorypostgres.NewRepository(pool)
	purchaseRepo := purchasepostgres.NewRepository(pool)
	purchasePaymentRepo := purchasepostgres.NewPaymentRepository(pool)
	salesRepo := salespostgres.NewRepository(pool)
	warehouseRepo := companypostgres.NewWarehouseRepository(pool)
	quoteRepo := salespostgres.NewQuoteRepository(pool)

	// ── Repositorios — saas (la parte que createCompanyUC necesita) ─────────────
	saasModuleRepo := saaspostgres.NewModuleRepository(pool)
	saasPlanRepo := saaspostgres.NewPlanRepository(pool)
	saasSubscriptionRepo := saaspostgres.NewSubscriptionRepository(pool)
	saasSubscriptionUC := saasapp.NewSubscriptionUseCase(saasSubscriptionRepo, saasPlanRepo)
	companyPlanAssigner := companysaas.New(saasSubscriptionUC, saasPlanRepo)

	// createCompanyUC se declara aquí (antes de lo que sería su lugar natural más abajo, junto al
	// resto de casos de uso de company) porque saasCompanyPort ya lo necesita para aprovisionar la
	// empresa de un dueño nuevo al aprobar un prospecto (ver saas/application/prospect.go), y
	// porque companyPlanAssigner le asigna el plan Interno a la empresa de un superadmin apenas
	// se crea (ver company/application/create.go).
	createCompanyUC := companyapp.NewCreateUseCase(companyRepo, securityRepo, warehouseRepo, companyPlanAssigner)
	inviteOwnerUC := securityapp.NewInviteOwnerUseCase(securityRepo)

	// ── Repositorios — saas (el resto) ───────────────────────────────────────────
	saasPaymentRepo := saaspostgres.NewPaymentRepository(pool)
	saasSettingsRepo := saaspostgres.NewSettingsRepository(pool)
	saasProspectRepo := saaspostgres.NewProspectRepository(pool)
	saasDocumentCounter := saaselectronic.New(pool)
	saasCompanyPort := saascompany.New(companyRepo, createCompanyUC)
	saasUserPort := saassecurity.New(securityRepo, inviteOwnerUC)
	paymentRepo := salespostgres.NewPaymentRepository(pool)

	// ── Repositorios — accounting ───────────────────────────────────────────────
	accountingAccountRepo := accountingpostgres.NewAccountRepository(pool)
	accountingWithholdingRepo := accountingpostgres.NewWithholdingConceptRepository(pool)
	accountingCertificateRepo := accountingpostgres.NewWithholdingCertificateRepository(pool)
	accountingBankAccountRepo := accountingpostgres.NewBankAccountRepository(pool)
	accountingBankStatementRepo := accountingpostgres.NewBankStatementRepository(pool)
	accountingFixedAssetRepo := accountingpostgres.NewFixedAssetRepository(pool)
	accountingDepreciationRepo := accountingpostgres.NewDepreciationRepository(pool)
	accountingBudgetRepo := accountingpostgres.NewBudgetRepository(pool)
	accountingIVARepo := accountingpostgres.NewIVADeclarationRepository(pool)
	accountingIncomeTaxRepo := accountingpostgres.NewIncomeTaxDeclarationRepository(pool)
	accountingICATariffRepo := accountingpostgres.NewICATariffRepository(pool)
	accountingICARepo := accountingpostgres.NewICADeclarationRepository(pool)
	accountingPeriodRepo := accountingpostgres.NewPeriodRepository(pool)
	accountingExchangeRateRepo := accountingpostgres.NewExchangeRateRepository(pool)
	accountingReconciliationRepo := accountingpostgres.NewReconciliationRepository(pool)
	accountingJournalRepo := accountingpostgres.NewJournalRepository(pool)

	// ── Repositorios — electronic ───────────────────────────────────────────────
	electronicDocRepo := electronicpostgres.NewDocumentRepository(pool)
	electronicNumRepo := electronicpostgres.NewNumberingRepository(pool, encryptionKey)
	electronicCompanyPort := electroniccompany.New(companyRepo)
	electronicAdapter := electroniccofacture.New()

	// ── Puertos locales — terceros (customer/supplier unificados) ──────────────
	salesCustomerPort := salesthirdparty.New(thirdpartyRepo)
	purchaseSupplierPort := purchasethirdparty.New(thirdpartyRepo)
	electronicThirdpartyPort := electronicthirdparty.New(thirdpartyRepo)

	// ── Casos de uso — security ─────────────────────────────────────────────────
	loginRateLimiter := securityapp.NewLoginRateLimiter()
	registerUC := securityapp.NewRegisterUseCase(securityRepo, jwtSvc)
	loginUC := securityapp.NewLoginUseCase(securityRepo, jwtSvc, loginRateLimiter)
	selectCompanyUC := securityapp.NewSelectCompanyUseCase(securityRepo, jwtSvc)
	inviteUserUC := securityapp.NewInviteUserUseCase(securityRepo)
	acceptInviteUC := securityapp.NewAcceptInviteUseCase(securityRepo, jwtSvc)
	updateProfileUC := securityapp.NewUpdateProfileUseCase(securityRepo)
	getProfileUC := securityapp.NewGetProfileUseCase(securityRepo)
	changePasswordUC := securityapp.NewChangePasswordUseCase(securityRepo, jwtSvc)
	logoutUC := securityapp.NewLogoutUseCase(securityRepo)
	// sessionVerifier reemplaza a jwtSvc como tenant.Verifier -- además de la firma/expiración
	// del JWT, confirma contra BD que la sesión no fue revocada (logout, cambio de contraseña).
	// Ver plan de acción 2026-08-09 Fase 2 punto 14.
	sessionVerifier := securityapp.NewSessionVerifier(jwtSvc, securityRepo)

	// ── Audit ───────────────────────────────────────────────────────────────────
	// Se construye acá (antes que purchase/sales/payroll) porque sus casos de uso de
	// confirmar/recibir/pagar ahora reciben auditUC para registrar si falla algún suscriptor
	// del bus de eventos (ver events.PublishAndAudit, plan de acción 2026-08-09 Fase 2 punto 10).
	// auditSecurityAdapter resuelve email/nombre de usuario para la vista de auditoría sin que
	// audit/ consulte security.users directamente (ver plan de acción 2026-08-09 Fase 2 punto 15).
	auditSecurityAdapter := auditsecurity.New(securityRepo)
	auditUC := auditapp.NewUseCase(auditpostgres.NewRepository(pool), logger.New("audit"), auditSecurityAdapter)

	// ── Casos de uso — purchase ─────────────────────────────────────────────────
	createPurchaseUC := purchaseapp.NewCreateUseCase(purchaseRepo)
	getPurchaseUC := purchaseapp.NewGetUseCase(purchaseRepo)
	confirmPurchaseUC := purchaseapp.NewConfirmUseCase(purchaseRepo)
	cancelPurchaseUC := purchaseapp.NewCancelUseCase(purchaseRepo)
	receivePurchaseUC := purchaseapp.NewReceiveUseCase(purchaseRepo, bus, auditUC)
	deletePurchaseUC := purchaseapp.NewDeleteUseCase(purchaseRepo)
	purchasePaymentUC := purchaseapp.NewPaymentUseCase(purchasePaymentRepo, purchaseRepo, bus, auditUC)
	purchaseWithholdingUC := purchaseapp.NewAddWithholdingUseCase(purchaseRepo, purchaseRepo, accountingWithholdingRepo)
	setPurchaseCounterUC := purchaseapp.NewSetNumberCounterUseCase(purchaseRepo)
	issueCertificatesUC := accountingapp.NewIssueWithholdingCertificatesUseCase(purchaseRepo, accountingCertificateRepo)
	manageBankUC := accountingapp.NewManageBankUseCase(accountingBankAccountRepo, accountingBankStatementRepo, accountingAccountRepo)
	fixedAssetUC := accountingapp.NewFixedAssetUseCase(accountingFixedAssetRepo, accountingAccountRepo)
	runDepreciationUC := accountingapp.NewRunDepreciationUseCase(accountingFixedAssetRepo, accountingDepreciationRepo, accountingAccountRepo, accountingPeriodRepo, accountingJournalRepo)
	budgetUC := accountingapp.NewBudgetUseCase(accountingBudgetRepo, accountingAccountRepo)
	ivaUC := accountingapp.NewIVAUseCase(accountingIVARepo, accountingJournalRepo, accountingAccountRepo)
	incomeTaxUC := accountingapp.NewIncomeTaxUseCase(accountingIncomeTaxRepo, accountingJournalRepo)
	icaUC := accountingapp.NewICAUseCase(accountingICARepo, accountingICATariffRepo, accountingJournalRepo)
	exchangeRateUC := accountingapp.NewExchangeRateUseCase(accountingExchangeRateRepo, accountingtrmapi.New(os.Getenv("TRM_API_URL")))
	reconciliationUC := accountingapp.NewReconciliationUseCase(accountingReconciliationRepo)

	// ── Casos de uso — sales ────────────────────────────────────────────────────
	createSaleUC := salesapp.NewCreateUseCase(salesRepo)
	getSaleUC := salesapp.NewGetUseCase(salesRepo)
	confirmSaleUC := salesapp.NewConfirmUseCase(salesRepo, bus, productRepo, inventoryRepo, salesCustomerPort, paymentRepo, warehouseRepo, auditUC)
	cancelSaleUC := salesapp.NewCancelUseCase(salesRepo)
	quoteUC := salesapp.NewQuoteUseCase(quoteRepo, salesRepo)
	paymentUC := salesapp.NewPaymentUseCase(paymentRepo, salesRepo, bus, auditUC)
	setSalesCounterUC := salesapp.NewSetNumberCounterUseCase(salesRepo, quoteRepo)

	// ── Casos de uso — accounting ───────────────────────────────────────────────
	postJournalUC := accountingapp.NewPostJournalUseCase(accountingAccountRepo, accountingPeriodRepo, accountingJournalRepo)
	getJournalUC := accountingapp.NewGetJournalUseCase(accountingJournalRepo)
	voidJournalUC := accountingapp.NewVoidJournalUseCase(accountingJournalRepo, accountingPeriodRepo)
	managePeriodUC := accountingapp.NewManagePeriodUseCase(accountingPeriodRepo)
	// Loggers de los suscriptores del bus de eventos (ver plan de acción 2026-08-09 Fase 2
	// punto 11) -- antes usaban log.Printf sin componente identificable.
	accountingEventsLog := logger.New("accounting")
	inventoryEventsLog := logger.New("inventory")
	onSaleConfirmed := accountingapp.NewOnSaleConfirmed(accountingAccountRepo, accountingPeriodRepo, accountingJournalRepo, accountingEventsLog)
	onSaleConfirmed.Register(bus)
	inventoryOnSaleConfirmed := inventoryapp.NewOnSaleConfirmed(inventoryRepo, productRepo, warehouseRepo, inventoryEventsLog)
	inventoryOnSaleConfirmed.Register(bus)
	onPurchaseReceived := accountingapp.NewOnPurchaseReceived(accountingAccountRepo, accountingPeriodRepo, accountingJournalRepo, accountingEventsLog)
	onPurchaseReceived.Register(bus)
	inventoryOnPurchaseReceived := inventoryapp.NewOnPurchaseReceived(inventoryRepo, productRepo, warehouseRepo, inventoryEventsLog)
	inventoryOnPurchaseReceived.Register(bus)
	onPayrollGenerated := accountingapp.NewOnPayrollGenerated(accountingAccountRepo, accountingPeriodRepo, accountingJournalRepo, accountingEventsLog)
	onPayrollGenerated.Register(bus)
	onSalePaymentRecorded := accountingapp.NewOnSalePaymentRecorded(accountingAccountRepo, accountingPeriodRepo, accountingJournalRepo, accountingEventsLog)
	onSalePaymentRecorded.Register(bus)
	onPurchasePaymentRecorded := accountingapp.NewOnPurchasePaymentRecorded(accountingAccountRepo, accountingPeriodRepo, accountingJournalRepo, accountingEventsLog)
	onPurchasePaymentRecorded.Register(bus)

	// ── Casos de uso — saas ──────────────────────────────────────────────────────
	// saasSubscriptionUC ya se construyó más arriba (companyPlanAssigner lo necesita).
	saasPlanUC := saasapp.NewPlanUseCase(saasPlanRepo, saasModuleRepo)
	saasSettingsUC := saasapp.NewSettingsUseCase(saasSettingsRepo)
	saasBillingUC := saasapp.NewBillingUseCase(saasSubscriptionRepo, saasPlanRepo, saasSettingsRepo, saasDocumentCounter, saasCompanyPort)
	saasPaymentUC := saasapp.NewPaymentUseCase(saasPaymentRepo)
	saasMyPlanUC := saasapp.NewMyPlanUseCase(saasSubscriptionRepo, saasPlanRepo, saasDocumentCounter)
	// saasProspectUC se construye más abajo (necesita `notifier`/`appURL`, que todavía no existen aquí).

	// ── Notificaciones de sistema (certificado/rangos por vencer) ───────────────
	notificationCompanyPort := notificationcompany.New(companyRepo)
	notificationNumberingPort := notificationnumbering.New(electronicNumRepo)
	notificationStockPort := notificationstock.New(inventoryRepo, productRepo, warehouseRepo)
	notificationAlertsUC := notificationapp.NewListSystemAlertsUseCase(notificationCompanyPort, notificationNumberingPort, notificationStockPort)

	// ── Casos de uso — electronic ───────────────────────────────────────────────
	electronicCreateDraftUC := electronicapp.NewCreateDraftUseCase(electronicDocRepo, electronicNumRepo, electronicCompanyPort, catalogRepo)
	electronicConfirmUC := electronicapp.NewConfirmUseCase(electronicDocRepo, electronicNumRepo, electronicCompanyPort, electronicAdapter, electronicAdapter, electronicAdapter)
	electronicGetUC := electronicapp.NewGetDocumentUseCase(electronicDocRepo)
	electronicListUC := electronicapp.NewListDocumentsUseCase(electronicDocRepo)
	electronicNumberingUC := electronicapp.NewManageNumberingUseCase(electronicNumRepo)
	electronicDianRangesUC := electronicapp.NewGetDianRangesUseCase(electronicCompanyPort, electronicAdapter)
	fromSaleUC := electronicapp.NewCreateFromSaleUseCase(electronicCreateDraftUC, salesRepo, electronicThirdpartyPort, productRepo)
	fromPurchaseUC := electronicapp.NewCreateFromPurchaseUseCase(electronicCreateDraftUC, purchaseRepo, electronicThirdpartyPort, productRepo)

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
	saasProspectUC := saasapp.NewProspectUseCase(saasProspectRepo, saasUserPort, saasCompanyPort, notifier, appURL)

	// ── Casos de uso — PDF/email de cotizaciones ────────────────────────────────
	salesCompanyPort := salescompany.New(companyRepo)
	quotePDFUC := salesapp.NewGetQuotePDFUseCase(quoteRepo, salesCustomerPort, salesCompanyPort, multiRenderer)
	sendQuoteEmailUC := salesapp.NewSendQuoteEmailUseCase(quoteRepo, salesCustomerPort, salesCompanyPort, multiRenderer, notifier, appURL)

	// ── Casos de uso — PDF/email de órdenes de compra ───────────────────────────
	purchaseCompanyPort := purchasecompany.New(companyRepo)
	purchasePDFUC := purchaseapp.NewGetPurchaseOrderPDFUseCase(purchaseRepo, purchaseSupplierPort, purchaseCompanyPort, multiRenderer)
	sendPurchaseEmailUC := purchaseapp.NewSendPurchaseOrderEmailUseCase(purchaseRepo, purchaseSupplierPort, purchaseCompanyPort, multiRenderer, notifier, appURL)

	// ── Casos de uso — hr / payroll / company / inventory / product ────────────
	hrAbsenceRepo := hrpostgres.NewAbsenceRepository(pool)
	hrAbsenceUC := hrapp.NewAbsenceUseCase(hrAbsenceRepo)

	payrollEmpRepo := payrollpostgres.NewEmployeeRepository(pool)
	payrollContractRepo := payrollpostgres.NewContractRepository(pool)
	payrollPayslipRepo := payrollpostgres.NewPayslipRepository(pool)
	payrollEmpUC := payrollapp.NewEmployeeUseCase(payrollEmpRepo)
	payrollContractUC := payrollapp.NewContractUseCase(payrollContractRepo, payrollEmpRepo)
	payrollPayslipUC := payrollapp.NewPayslipUseCase(payrollPayslipRepo, payrollContractRepo, payrollEmpRepo, bus, auditUC)

	moveInventoryUC := inventoryapp.NewMoveUseCase(inventoryRepo)
	getInventoryUC := inventoryapp.NewGetUseCase(inventoryRepo)

	createProductUC := productapp.NewCreateUseCase(productRepo)
	getProductUC := productapp.NewGetUseCase(productRepo)
	updateProductUC := productapp.NewUpdateUseCase(productRepo)
	deleteProductUC := productapp.NewDeleteUseCase(productRepo)

	// Terceros — un mismo juego de casos de uso sirve a los catálogos de Clientes y Proveedores;
	// el rol lo fija el handler HTTP (NewCustomerHandler/NewSupplierHandler) en cada llamada.
	createPartyUC := thirdpartyapp.NewCreateUseCase(thirdpartyRepo)
	getPartyUC := thirdpartyapp.NewGetUseCase(thirdpartyRepo)
	updatePartyUC := thirdpartyapp.NewUpdateUseCase(thirdpartyRepo)
	deletePartyUC := thirdpartyapp.NewDeleteUseCase(thirdpartyRepo)

	getCompanyUC := companyapp.NewGetUseCase(companyRepo, securityRepo)
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
		updateProfileUC, getProfileUC, changePasswordUC, logoutUC, auditUC,
	).RegisterRoutes(mux)
	companyhttp.NewHandler(
		createCompanyUC, getCompanyUC,
		updateCompanyProfile, updateCompanyCreds, clearCompanyCreds, updateCompanyLogo,
		warehouseUC, auditUC,
	).RegisterRoutes(mux)
	thirdpartyhttp.NewCustomerHandler(createPartyUC, getPartyUC, updatePartyUC, deletePartyUC, auditUC).RegisterRoutes(mux)
	thirdpartyhttp.NewSupplierHandler(createPartyUC, getPartyUC, updatePartyUC, deletePartyUC, auditUC).RegisterRoutes(mux)
	producthttp.NewHandler(createProductUC, getProductUC, updateProductUC, deleteProductUC, auditUC).RegisterRoutes(mux)
	inventoryhttp.NewHandler(moveInventoryUC, getInventoryUC, auditUC).RegisterRoutes(mux)
	purchasehttp.NewHandler(createPurchaseUC, getPurchaseUC, confirmPurchaseUC, cancelPurchaseUC, receivePurchaseUC, deletePurchaseUC, purchasePaymentUC, purchasePDFUC, sendPurchaseEmailUC, purchaseWithholdingUC, setPurchaseCounterUC, auditUC).RegisterRoutes(mux)
	saleshttp.NewHandler(createSaleUC, getSaleUC, confirmSaleUC, cancelSaleUC, quoteUC, paymentUC, quotePDFUC, sendQuoteEmailUC, setSalesCounterUC, auditUC).RegisterRoutes(mux)
	accountinghttp.NewHandler(postJournalUC, getJournalUC, voidJournalUC, managePeriodUC, accountingAccountRepo, accountingWithholdingRepo, issueCertificatesUC, manageBankUC, fixedAssetUC, runDepreciationUC, budgetUC, ivaUC, incomeTaxUC, icaUC, exchangeRateUC, reconciliationUC, auditUC).RegisterRoutes(mux)
	electronichttp.NewHandler(electronicCreateDraftUC, electronicConfirmUC, electronicGetUC, electronicListUC, electronicNumberingUC, electronicPDFUC, fromSaleUC, fromPurchaseUC, electronicDianRangesUC, electronicSendEmailUC, auditUC).RegisterRoutes(mux)
	audithttp.NewHandler(auditUC).RegisterRoutes(mux)
	notificationhttp.NewHandler(notificationAlertsUC).RegisterRoutes(mux)
	publichttp.NewHandler(getCompanyUC, createPartyUC).RegisterRoutes(mux)
	payrollhttp.NewHandler(payrollEmpUC, payrollContractUC, payrollPayslipUC, auditUC).RegisterRoutes(mux)
	hrhttp.NewHandler(hrAbsenceUC, auditUC).RegisterRoutes(mux)

	saasHandler := saashttp.NewHandler(saasPlanUC, saasSettingsUC, saasSubscriptionUC, saasBillingUC, saasPaymentUC, saasProspectUC, saasMyPlanUC, saasUserPort, saasCompanyPort, auditUC)
	saasHandler.RegisterAdminRoutes(mux)
	saasHandler.RegisterTenantRoutes(mux)
	saasHandler.RegisterPublicRoutes(mux)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","service":"cofacture erp"}`))
	})

	// ── CORS: CORS_ALLOWED_ORIGINS es obligatoria salvo que se opte explícitamente por
	// CORS_DEV_MODE=true (refleja cualquier origen) -- antes, si la variable faltaba, el
	// servidor caía en modo permisivo por defecto en vez de fallar al arrancar (ver plan de
	// acción 2026-08-09 Fase 2 punto 13).
	corsDevMode := envOr("CORS_DEV_MODE", "") == "true"
	var corsAllowedOrigins []string
	if corsDevMode {
		log.Warn("CORS_DEV_MODE=true -- se refleja cualquier origen, NO usar en producción")
	} else {
		for _, o := range strings.Split(mustEnv(log, "CORS_ALLOWED_ORIGINS"), ",") {
			if o = strings.TrimSpace(o); o != "" {
				corsAllowedOrigins = append(corsAllowedOrigins, o)
			}
		}
	}

	// ── Middleware (orden: recover → request-id → secheaders → cors → logger → tenant/JWT) ──
	// recover va lo más afuera para cubrir un panic en cualquiera de los middleware
	// siguientes, no solo en los handlers de negocio (ver plan de acción 2026-08-09
	// Fase 2 punto 12 -- antes un panic tumbaba la conexión sin dejar rastro ni responder
	// nada coherente al cliente).
	httpLog := logger.New("http")
	handler := logger.Recover(httpLog)(logger.WithRequestID(secheaders.Middleware(cors.Middleware(corsAllowedOrigins, corsDevMode)(logger.Middleware(httpLog)(tenant.Middleware(sessionVerifier)(mux))))))

	// ── Disparador diario de TRM (1:00 a.m. hora Colombia) ──────────────────────
	if os.Getenv("TRM_API_URL") != "" {
		trmLog := logger.New("trm")
		go accountingapp.RunTRMDailySync(context.Background(), exchangeRateUC, func(rate *accountingdomain.ExchangeRate, err error) {
			if err != nil {
				trmLog.Error("sincronización diaria de TRM falló", "error", err)
				return
			}
			trmLog.Info("TRM sincronizada automáticamente", "rate", rate.Rate(), "date", rate.RateDate.Format("2006-01-02"))
		})
	}

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
