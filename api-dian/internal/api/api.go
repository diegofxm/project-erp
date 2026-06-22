package api

import (
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/diegofxm/api-dian/internal/api/middleware"
	"github.com/diegofxm/api-dian/internal/api/response"
	"github.com/diegofxm/api-dian/internal/auth"
	"github.com/diegofxm/api-dian/internal/customers"
	"github.com/diegofxm/api-dian/internal/database"
	"github.com/diegofxm/api-dian/internal/documents"
	"github.com/diegofxm/api-dian/internal/issuers"
	"github.com/diegofxm/api-dian/internal/numbering"
	"github.com/diegofxm/api-dian/internal/products"
)

// API agrupa los seis servicios de dominio y expone el http.Handler.
type API struct {
	log       *zap.Logger
	issuers   *issuers.Service
	numbering *numbering.Service
	documents *documents.Service
	auth      *auth.Service
	tokens    *auth.TokenIssuer
	customers *customers.Service
	products  *products.Service
}

// New conecta los seis dominios sobre una sola base de datos y devuelve la API.
// authJWTSecret firma los tokens de sesión (HS256) — deliberadamente distinto de
// issuerSecretsKey (ver internal/config.Config.AuthJWTSecret).
func New(log *zap.Logger, db *database.DB, issuerSecretsKey, authJWTSecret []byte) *API {
	issuerSvc := issuers.New(issuers.NewPostgresRepository(db.Pool, issuerSecretsKey))
	numberingSvc := numbering.New(numbering.NewPostgresRepository(db.Pool, issuerSecretsKey))
	customersSvc := customers.New(customers.NewPostgresRepository(db.Pool))
	productsSvc := products.New(products.NewPostgresRepository(db.Pool))
	documentsSvc := documents.New(documents.NewPostgresRepository(db.Pool), issuerSvc, numberingSvc, customersSvc)
	tokens := auth.NewTokenIssuer(authJWTSecret)
	authSvc := auth.New(auth.NewPostgresRepository(db.Pool), issuerSvc, tokens)

	return NewFromServices(log, issuerSvc, numberingSvc, documentsSvc, authSvc, tokens, customersSvc, productsSvc)
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
) *API {
	return &API{
		log: log, issuers: issuerSvc, numbering: numberingSvc, documents: documentsSvc,
		auth: authSvc, tokens: tokens, customers: customersSvc, products: productsSvc,
	}
}

// Handler devuelve el http.Handler del subárbol /api/v1 con el middleware aplicado.
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	a.registerRoutes(mux)

	// Middleware aplicado de afuera hacia adentro: RequestID → Logging → Recovery.
	h := middleware.Recovery(a.log)(mux)
	h = middleware.Logging(a.log)(h)
	h = middleware.RequestID(h)
	return h
}

// registerRoutes conecta todos los endpoints REST al mux.
//
// Resumen de rutas:
//
//	POST   /api/v1/auth/register                         → crea el emisor Y su primer usuario admin (público)
//	POST   /api/v1/auth/login                             → inicia sesión, devuelve el token (público)
//
//	A partir de aquí, todo exige "Authorization: Bearer <token>" — "un usuario = un emisor",
//	así que ningún endpoint recibe issuer_id del cliente: siempre se toma del token
//	(middleware.GetTenantID), nunca de algo que el cliente pueda elegir.
//
//	GET    /api/v1/issuers/me                            → consultar el emisor propio
//	POST   /api/v1/numbering-ranges                      → registrar rango de numeración del emisor propio
//	GET    /api/v1/numbering-ranges                      → listar rangos del emisor propio (?dian_document_type_code=)
//	GET    /api/v1/numbering-ranges/{id}                  → consultar rango (debe ser del emisor propio)
//	POST   /api/v1/invoices                               → emitir Factura Electrónica de Venta
//	POST   /api/v1/credit-notes                           → emitir Nota Crédito
//	POST   /api/v1/debit-notes                            → emitir Nota Débito
//	GET    /api/v1/documents                              → listar documentos del emisor propio (filtros + ?limit=&offset=)
//	GET    /api/v1/documents/{id}                         → consultar documento (debe ser del emisor propio)
//
//	POST/GET/PUT/DELETE /api/v1/customers[/{id}]          → catálogo de adquirientes (conveniencia, ver internal/customers)
//	POST/GET/PUT/DELETE /api/v1/products[/{id}]           → catálogo de ítems/servicios (conveniencia, ver internal/products)
func (a *API) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/register", a.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", a.handleLogin)

	protect := middleware.Auth(a.tokens)
	handle := func(pattern string, h http.HandlerFunc) { mux.Handle(pattern, protect(h)) }

	handle("GET /api/v1/issuers/me", a.handleGetMyIssuer)
	handle("POST /api/v1/numbering-ranges", a.handleCreateNumberingRange)
	handle("GET /api/v1/numbering-ranges", a.handleListNumberingRanges)
	handle("GET /api/v1/numbering-ranges/{id}", a.handleGetNumberingRange)

	handle("POST /api/v1/invoices", a.handleIssueInvoice)
	handle("POST /api/v1/credit-notes", a.handleIssueCreditNote)
	handle("POST /api/v1/debit-notes", a.handleIssueDebitNote)
	handle("GET /api/v1/documents", a.handleListDocuments)
	handle("GET /api/v1/documents/{id}", a.handleGetDocument)

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
