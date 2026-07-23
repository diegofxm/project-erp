package api

import (
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/diegofxm/apidian/internal/api/middleware"
	"github.com/diegofxm/apidian/internal/api/response"
	"github.com/diegofxm/apidian/internal/audit"
	"github.com/diegofxm/apidian/internal/auth"
	"github.com/diegofxm/apidian/internal/catalogs"
	"github.com/diegofxm/apidian/internal/customers"
	"github.com/diegofxm/apidian/internal/database"
	"github.com/diegofxm/apidian/internal/documents"
	"github.com/diegofxm/apidian/internal/email"
	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/diegofxm/apidian/internal/numbering"
	"github.com/diegofxm/apidian/internal/payments"
	"github.com/diegofxm/apidian/internal/plans"
	"github.com/diegofxm/apidian/internal/prospects"
	"github.com/diegofxm/apidian/internal/products"
	"github.com/diegofxm/apidian/internal/settings"
	"github.com/diegofxm/apidian/internal/subscriptions"
	"github.com/diegofxm/apidian/internal/vendors"
)

// API agrupa los servicios de dominio y expone el http.Handler.
type API struct {
	log            *zap.Logger
	issuers        *issuers.Service
	numbering      *numbering.Service
	documents      *documents.Service
	auth           *auth.Service
	tokens         *auth.TokenIssuer
	customers      *customers.Service
	products       *products.Service
	vendors        *vendors.Service
	plans          *plans.Service
	subscriptions  *subscriptions.Service
	settings       *settings.Service
	payments       *payments.Service
	prospects      *prospects.Service
	auditSvc       *audit.Service  // nil en tests que no usan Postgres
	catalogs       catalogs.Repository
	allowedOrigins []string
	appBaseURL     string
}

// New conecta los siete dominios sobre una sola base de datos y devuelve la API.
// authJWTSecret firma los tokens de sesión (HS256) — deliberadamente distinto de
// issuerSecretsKey (ver internal/config.Config.AuthJWTSecret). allowedOrigins son los
// orígenes permitidos por CORS (ver internal/api/middleware/cors.go). smtpCfg configura el
// envío de correo al cliente (ver docs/apidian-architecture.md sección 9.42) — vacío es válido,
// SendInvoiceEmail simplemente falla con un mensaje claro si nunca se configuró.
func New(log *zap.Logger, db *database.DB, issuerSecretsKey, authJWTSecret []byte, allowedOrigins []string, smtpCfg email.Config, appBaseURL string) *API {
	catalogsRepo := catalogs.NewPostgresRepository(db.Pool)
	issuerSvc := issuers.New(issuers.NewPostgresRepository(db.Pool, issuerSecretsKey), documents.ValidateCertificate, documents.ParseCertificate, catalogsRepo)
	numberingSvc := numbering.New(numbering.NewPostgresRepository(db.Pool, issuerSecretsKey))
	customersSvc := customers.New(customers.NewPostgresRepository(db.Pool), catalogsRepo)
	productsSvc := products.New(products.NewPostgresRepository(db.Pool), catalogsRepo)
	vendorsSvc := vendors.New(vendors.NewPostgresRepository(db.Pool), catalogsRepo)
	plansSvc := plans.NewService(plans.NewPostgresRepository(db.Pool))
	subsSvc := subscriptions.NewService(subscriptions.NewPostgresRepository(db.Pool))
	settingsSvc := settings.NewService(settings.NewPostgresRepository(db.Pool))
	paymentsSvc := payments.NewService(payments.NewPostgresRepository(db.Pool))
	prospectsSvc := prospects.NewService(prospects.NewPostgresRepository(db))
	emailSender := email.NewSMTPSender(smtpCfg)
	documentsSvc := documents.New(documents.NewPostgresRepository(db.Pool), issuerSvc, numberingSvc, customersSvc, catalogsRepo, emailSender).
		WithVendors(vendorsSvc).
		WithSubscriptions(subsSvc).
		WithSettings(settingsSvc)
	tokens := auth.NewTokenIssuer(authJWTSecret)
	authSvc := auth.New(auth.NewPostgresRepository(db.Pool), issuerSvc, tokens).
		WithEmail(emailSender, appBaseURL)
	auditSvc := audit.New(audit.NewPostgresRepository(db.Pool))

	a := NewFromServices(log, issuerSvc, numberingSvc, documentsSvc, authSvc, tokens, customersSvc, productsSvc, vendorsSvc, plansSvc, subsSvc, settingsSvc, paymentsSvc, prospectsSvc, catalogsRepo, allowedOrigins, appBaseURL)
	a.auditSvc = auditSvc
	return a
}

// NewFromServices crea una API a partir de servicios ya construidos — útil para tests que
// usan repositorios en memoria en vez de Postgres real.
func NewFromServices(
	log *zap.Logger,
	issuerSvc *issuers.Service,
	numberingSvc *numbering.Service,
	documentsSvc *documents.Service,
	authSvc *auth.Service,
	tokens *auth.TokenIssuer,
	customersSvc *customers.Service,
	productsSvc *products.Service,
	vendorsSvc *vendors.Service,
	plansSvc *plans.Service,
	subsSvc *subscriptions.Service,
	settingsSvc *settings.Service,
	paymentsSvc *payments.Service,
	prospectsSvc *prospects.Service,
	catalogsRepo catalogs.Repository,
	allowedOrigins []string,
	appBaseURL string,
) *API {
	return &API{
		log: log, issuers: issuerSvc, numbering: numberingSvc, documents: documentsSvc,
		auth: authSvc, tokens: tokens, customers: customersSvc, products: productsSvc,
		vendors: vendorsSvc, plans: plansSvc, subscriptions: subsSvc, settings: settingsSvc,
		payments: paymentsSvc, prospects: prospectsSvc, catalogs: catalogsRepo, allowedOrigins: allowedOrigins, appBaseURL: appBaseURL,
	}
}

// Handler devuelve el http.Handler del subárbol /api/v1 con el middleware aplicado.
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	a.registerRoutes(mux)

	// Middleware aplicado de afuera hacia adentro: RequestID → CORS → Logging → Recovery.
	// CORS va antes de Logging a propósito: un preflight OPTIONS se responde y corta ahí
	// mismo (ver middleware.CORS), nunca llega a generar una línea de log ni a tocar el mux
	// (que no tiene rutas OPTIONS registradas).
	h := middleware.Recovery(a.log)(mux)
	h = middleware.Logging(a.log)(h)
	h = middleware.CORS(a.allowedOrigins)(h)
	h = middleware.RequestID(h)
	return h
}

// registerRoutes conecta todos los endpoints REST al mux.
//
// Resumen de rutas:
//
//	POST   /api/v1/auth/register                         → crea el usuario, SIN empresa todavía (público, ver 9.32)
//	POST   /api/v1/auth/login                             → inicia sesión, devuelve el token (público, rate-limited: 10/min por IP)
//	GET    /api/v1/public/issuers/{id}                    → nombre del emisor para el formulario de autorregistro (público, ver 9.41)
//	GET    /api/v1/public/issuers/{id}/logo               → logo del emisor en crudo, sin sesión (404 si no tiene)
//	POST   /api/v1/public/issuers/{id}/customers          → autorregistro de cliente por QR, sin sesión (público, ver 9.41)
//
//	A partir de aquí, todo exige "Authorization: Bearer <token>". Un usuario puede tener cero,
//	una, o varias empresas vinculadas (sección 9.32) — estas tres rutas NO exigen una empresa
//	ACTIVA (son justamente las que se necesitan para conseguir una):
//
//	POST   /api/v1/issuers                                → crear una empresa nueva y vincularla al usuario (rol owner)
//	GET    /api/v1/issuers                                → listar las empresas del usuario
//	POST   /api/v1/issuers/{id}/select                    → reemitir el token con esa empresa como activa
//
//	El resto SÍ exige una empresa activa (middleware.RequireTenant — 409 si no hay una) — el
//	emisor/issuer_id nunca se recibe del cliente, siempre se toma del token
//	(middleware.GetTenantID):
//
//	GET    /api/v1/issuers/me                            → consultar la empresa activa
//	PUT    /api/v1/issuers/me                            → completar software/PIN/certificado/logo de la empresa activa (ver 9.25/9.39)
//	PATCH  /api/v1/issuers/me/profile                   → editar perfil (razón social, dirección, datos fiscales) — no toca secretos
//	GET    /api/v1/issuers/me/logo                       → logo en crudo de la empresa activa (404 si no tiene, ver 9.39)
//	DELETE /api/v1/issuers/me/logo                       → quitar el logo de la empresa activa
//	GET    /api/v1/issuers/me/settings                   → configuración de personalización (brand_color)
//	PATCH  /api/v1/issuers/me/settings                   → actualizar brand_color
//	POST   /api/v1/numbering-ranges                      → registrar rango de numeración de la empresa activa
//	GET    /api/v1/numbering-ranges                      → listar rangos de la empresa activa (?dian_document_type_code=)
//	GET    /api/v1/numbering-ranges/{id}                  → consultar rango (debe ser de la empresa activa)
//	DELETE /api/v1/numbering-ranges/{id}                  → desactivar rango
//	PUT    /api/v1/numbering-ranges/{id}/activate          → reactivar un rango previamente desactivado
//	POST   /api/v1/invoices                               → crear borrador de Factura (sin reclamar número)
//	PUT    /api/v1/invoices/{id}                          → reemplazar un borrador de Factura
//	POST   /api/v1/credit-notes                           → crear borrador de Nota Crédito
//	PUT    /api/v1/credit-notes/{id}                      → reemplazar un borrador de Nota Crédito
//	POST   /api/v1/debit-notes                            → crear borrador de Nota Débito
//	PUT    /api/v1/debit-notes/{id}                        → reemplazar un borrador de Nota Débito
//	POST   /api/v1/documents/{id}/confirm                  → reclamar número + firmar + enviar (ver 9.25)
//	DELETE /api/v1/documents/{id}                         → eliminar un borrador (debe seguir en borrador)
//	GET    /api/v1/documents                              → listar documentos de la empresa activa (filtros + ?limit=&offset=)
//	GET    /api/v1/documents/{id}                         → consultar documento (debe ser de la empresa activa)
//	GET    /api/v1/documents/{id}/pdf                      → representación gráfica en PDF, generada en memoria (ver 9.39)
//	GET    /api/v1/documents/{id}/xml                      → XML UBL firmado, solo si ya fue confirmado (ver handleGetDocumentXML)
//	POST   /api/v1/documents/{id}/send-email               → envía la Factura accepted al correo del cliente, PDF+XML adjuntos (ver 9.42)
//	GET    /api/v1/dian/verify-acquirer                    → ayuda opcional al capturar un NIT, no bloqueante (?identification_type_code=&identification_number=, ver 9.41)
//
//	POST/GET/PUT/DELETE /api/v1/customers[/{id}]          → catálogo de adquirientes (conveniencia, ver internal/customers)
//	POST/GET/PUT/DELETE /api/v1/products[/{id}]           → catálogo de ítems/servicios (conveniencia, ver internal/products)
//	POST/GET/PUT/DELETE /api/v1/vendors[/{id}]            → catálogo de proveedores/terceros no obligados (ver internal/vendors)
//
//	GET /api/v1/catalogs/{departments,municipalities,identification-types,tax-types,
//	    payment-methods,payment-terms,unit-measures,tax-regimes,liability-codes,
//	    dian-document-types,currencies,item-standards,ciiu-codes} → catálogos de referencia
//	    DIAN/DANE, de solo lectura, iguales para cualquier usuario autenticado (ver
//	    internal/catalogs). municipalities acepta ?department_code= para filtrar.
func (a *API) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", a.handleRegister)
	mux.Handle("POST /api/v1/auth/login", middleware.LoginRateLimit(http.HandlerFunc(a.handleLogin)))
	mux.HandleFunc("POST /api/v1/auth/accept-invite", a.handleAcceptInvite)
	mux.HandleFunc("GET /api/v1/public/issuers/{id}", a.handleGetPublicIssuer)
	mux.HandleFunc("GET /api/v1/public/issuers/{id}/logo", a.handleGetPublicIssuerLogo)
	mux.HandleFunc("POST /api/v1/public/issuers/{id}/customers", a.handleCreatePublicCustomer)
	mux.HandleFunc("POST /api/v1/public/prospects", a.handleSubmitProspect)

	protect := middleware.Auth(a.tokens)
	// handle exige una empresa ACTIVA (la mayoría de rutas); handleNoTenant solo exige estar
	// autenticado — para las rutas de gestión de empresas, que un usuario sin empresa activa
	// todavía necesita poder llamar (ver sección 9.32).
	handle := func(pattern string, h http.HandlerFunc) { mux.Handle(pattern, protect(middleware.RequireTenant(h))) }
	handleNoTenant := func(pattern string, h http.HandlerFunc) { mux.Handle(pattern, protect(h)) }

	handleNoTenant("PUT /api/v1/auth/me", a.handleUpdateMe)
	handleNoTenant("POST /api/v1/issuers", a.handleCreateIssuer)
	handleNoTenant("GET /api/v1/issuers", a.handleListMyIssuers)
	handleNoTenant("POST /api/v1/issuers/{id}/select", a.handleSelectIssuer)

	handle("GET /api/v1/issuers/me", a.handleGetMyIssuer)
	handle("PUT /api/v1/issuers/me", a.handleUpdateMyIssuer)
	handle("PATCH /api/v1/issuers/me/profile", a.handleUpdateMyIssuerProfile)
	handle("GET /api/v1/issuers/me/logo", a.handleGetMyIssuerLogo)
	handle("DELETE /api/v1/issuers/me/logo", a.handleDeleteMyIssuerLogo)
	handle("DELETE /api/v1/issuers/me/software", a.handleDeleteMyIssuerSoftware)
	handle("DELETE /api/v1/issuers/me/ne-software", a.handleDeleteMyIssuerNeSoftware)
	handle("DELETE /api/v1/issuers/me/certificate", a.handleDeleteMyIssuerCertificate)
	handle("POST /api/v1/numbering-ranges", a.handleCreateNumberingRange)
	handle("GET /api/v1/numbering-ranges", a.handleListNumberingRanges)
	handle("GET /api/v1/numbering-ranges/{id}", a.handleGetNumberingRange)
	handle("DELETE /api/v1/numbering-ranges/{id}", a.handleDeactivateNumberingRange)
	handle("PUT /api/v1/numbering-ranges/{id}/activate", a.handleActivateNumberingRange)

	handle("POST /api/v1/invoices", a.handleCreateInvoice)
	handle("PUT /api/v1/invoices/{id}", a.handleUpdateInvoice)
	handle("POST /api/v1/credit-notes", a.handleCreateCreditNote)
	handle("PUT /api/v1/credit-notes/{id}", a.handleUpdateCreditNote)
	handle("POST /api/v1/debit-notes", a.handleCreateDebitNote)
	handle("PUT /api/v1/debit-notes/{id}", a.handleUpdateDebitNote)
	handle("POST /api/v1/support-documents", a.handleCreateSupportDocument)
	handle("PUT /api/v1/support-documents/{id}", a.handleUpdateSupportDocument)
	handle("POST /api/v1/adjustment-notes", a.handleCreateAdjustmentNote)
	handle("PUT /api/v1/adjustment-notes/{id}", a.handleUpdateAdjustmentNote)
	handle("POST /api/v1/documents/{id}/confirm", a.handleConfirmDocument)
	handle("POST /api/v1/documents/{id}/clone", a.handleCloneDocument)
	handle("DELETE /api/v1/documents/{id}", a.handleDeleteDocument)
	handle("GET /api/v1/documents", a.handleListDocuments)
	handle("GET /api/v1/documents/{id}", a.handleGetDocument)
	handle("GET /api/v1/documents/{id}/pdf", a.handleGetDocumentPDF)
	handle("GET /api/v1/documents/{id}/xml", a.handleGetDocumentXML)
	handle("POST /api/v1/documents/{id}/send-email", a.handleSendDocumentEmail)
	handle("GET /api/v1/dian/verify-acquirer", a.handleVerifyAcquirer)
	handle("GET /api/v1/stats/billing", a.handleGetBillingStats)

	handle("GET /api/v1/issuers/me/settings", a.handleGetMySettings)
	handle("PATCH /api/v1/issuers/me/settings", a.handleUpdateMySettings)
	handle("GET /api/v1/audit-events", a.handleListAuditEvents)

	// Rutas de superadministrador — exigen is_superadmin=true además de estar autenticado.
	handleSA := func(pattern string, h http.HandlerFunc) {
		mux.Handle(pattern, protect(a.requireSuperAdmin(h)))
	}
	handleSA("GET /api/v1/admin/users", a.handleAdminListUsers)
	handleSA("POST /api/v1/admin/users", a.handleAdminCreateUser)
	handleSA("GET /api/v1/admin/plans", a.handleAdminListPlans)
	handleSA("POST /api/v1/admin/plans", a.handleAdminCreatePlan)
	handleSA("PATCH /api/v1/admin/plans/{id}", a.handleAdminUpdatePlan)
	handleSA("POST /api/v1/admin/plans/{id}/apply-increment", a.handleAdminApplyPlanIncrement)
	handleSA("GET /api/v1/admin/issuers/{id}", a.handleAdminGetIssuer)
	handleSA("GET /api/v1/admin/issuers/{id}/settings", a.handleAdminGetIssuerSettings)
	handleSA("PATCH /api/v1/admin/issuers/{id}/settings", a.handleAdminUpdateIssuerSettings)
	handleSA("GET /api/v1/admin/issuers/{id}/subscription", a.handleAdminGetIssuerSubscription)
	handleSA("POST /api/v1/admin/issuers/{id}/subscription", a.handleAdminAssignPlan)
	handleSA("GET /api/v1/admin/billing/summary", a.handleAdminBillingSummary)
	handleSA("GET /api/v1/admin/billing/renewals", a.handleAdminRenewalsSummary)
	handleSA("POST /api/v1/admin/issuers/{id}/affiliate", a.handleAdminAffiliateIssuer)
	handleSA("POST /api/v1/admin/issuers/{id}/renew", a.handleAdminRenewIssuer)
	handleSA("GET /api/v1/admin/issuers/{id}/payments", a.handleAdminListIssuerPayments)
	handleSA("GET /api/v1/admin/prospects", a.handleAdminListProspects)
	handleSA("POST /api/v1/admin/prospects/{id}/approve", a.handleAdminApproveProspect)
	handleSA("POST /api/v1/admin/prospects/{id}/reject", a.handleAdminRejectProspect)
	handleSA("GET /api/v1/admin/prospects/{id}/cedula", a.handleAdminGetProspectCedula)
	handleSA("GET /api/v1/admin/prospects/{id}/rut", a.handleAdminGetProspectRut)

	handle("POST /api/v1/customers", a.handleCreateCustomer)
	handle("GET /api/v1/customers", a.handleListCustomers)
	handle("GET /api/v1/customers/{id}", a.handleGetCustomer)
	handle("PUT /api/v1/customers/{id}", a.handleUpdateCustomer)
	handle("DELETE /api/v1/customers/{id}", a.handleDeleteCustomer)

	handle("POST /api/v1/products", a.handleCreateProduct)
	handle("GET /api/v1/products", a.handleListProducts)
	handle("GET /api/v1/products/{id}", a.handleGetProduct)
	handle("PUT /api/v1/products/{id}", a.handleUpdateProduct)
	handle("DELETE /api/v1/products/{id}", a.handleDeleteProduct)

	handle("POST /api/v1/vendors", a.handleCreateVendor)
	handle("GET /api/v1/vendors", a.handleListVendors)
	handle("GET /api/v1/vendors/{id}", a.handleGetVendor)
	handle("PUT /api/v1/vendors/{id}", a.handleUpdateVendor)
	handle("DELETE /api/v1/vendors/{id}", a.handleDeleteVendor)

	// Catálogos: iguales para cualquier usuario autenticado, no dependen de la empresa activa
	// — por eso usan handleNoTenant (solo Auth), no handle (que exigiría RequireTenant).
	handleNoTenant("GET /api/v1/catalogs/departments", a.handleListDepartments)
	handleNoTenant("GET /api/v1/catalogs/municipalities", a.handleListMunicipalities)
	handleNoTenant("GET /api/v1/catalogs/identification-types", a.handleListIdentificationTypes)
	handleNoTenant("GET /api/v1/catalogs/tax-types", a.handleListTaxTypes)
	handleNoTenant("GET /api/v1/catalogs/payment-methods", a.handleListPaymentMethods)
	handleNoTenant("GET /api/v1/catalogs/payment-terms", a.handleListPaymentTerms)
	handleNoTenant("GET /api/v1/catalogs/unit-measures", a.handleListUnitMeasures)
	handleNoTenant("GET /api/v1/catalogs/tax-regimes", a.handleListTaxRegimes)
	handleNoTenant("GET /api/v1/catalogs/liability-codes", a.handleListLiabilityCodes)
	handleNoTenant("GET /api/v1/catalogs/dian-document-types", a.handleListDianDocumentTypes)
	handleNoTenant("GET /api/v1/catalogs/currencies", a.handleListCurrencies)
	handleNoTenant("GET /api/v1/catalogs/item-standards", a.handleListItemStandards)
	handleNoTenant("GET /api/v1/catalogs/ciiu-codes", a.handleListCiiuCodes)
}

// ── helpers compartidos ─────────────────────────────────────────────────────────────────────

// parseUUID extrae un UUID de un path value y escribe un 400 si falla.
func parseUUID(w http.ResponseWriter, raw string) (uuid.UUID, bool) {
	return parseUUIDField(w, raw, "id")
}

// parseUUIDField valida un UUID que viene del BODY de la petición (no del path), nombrando el
// campo en el error — un uuid.UUID vacío o mal formado en un campo JSON hace que
// json.Decode falle como cualquier otro error de tipo, y antes eso se reportaba como el
// genérico "JSON inválido" sin decir cuál campo era ni por qué (numbering_range_id vacío en
// Postman por correr los pasos fuera de orden, por ejemplo) — confuso de depurar.
func parseUUIDField(w http.ResponseWriter, raw, field string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: field + " inválido o vacío, se esperaba un UUID"})
		return uuid.Nil, false
	}
	return id, true
}

// parseOptionalUUIDField es como parseUUIDField, pero para campos que SÍ pueden venir vacíos
// (ej. customer_id en una emisión, ver handler_documents.go) — "" es válido y significa "no
// se mandó", no un error.
func parseOptionalUUIDField(w http.ResponseWriter, raw, field string) (*uuid.UUID, bool) {
	if raw == "" {
		return nil, true
	}
	id, ok := parseUUIDField(w, raw, field)
	if !ok {
		return nil, false
	}
	return &id, true
}
