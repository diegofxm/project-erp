package api

import (
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/diegofxm/api-dian/internal/api/middleware"
	"github.com/diegofxm/api-dian/internal/api/response"
	"github.com/diegofxm/api-dian/internal/database"
	"github.com/diegofxm/api-dian/internal/documents"
	"github.com/diegofxm/api-dian/internal/issuers"
	"github.com/diegofxm/api-dian/internal/numbering"
)

// API agrupa los tres servicios de dominio y expone el http.Handler. Es la primera vez que
// estos servicios se exponen por HTTP — antes de esto solo se llamaban directamente desde
// Go (tests, herramientas de verificación temporales).
type API struct {
	log       *zap.Logger
	issuers   *issuers.Service
	numbering *numbering.Service
	documents *documents.Service
}

// New conecta los tres dominios sobre una sola base de datos y devuelve la API.
func New(log *zap.Logger, db *database.DB, issuerSecretsKey []byte) *API {
	issuerSvc := issuers.New(issuers.NewPostgresRepository(db.Pool, issuerSecretsKey))
	numberingSvc := numbering.New(numbering.NewPostgresRepository(db.Pool, issuerSecretsKey))
	documentsSvc := documents.New(documents.NewPostgresRepository(db.Pool), issuerSvc, numberingSvc)

	return NewFromServices(log, issuerSvc, numberingSvc, documentsSvc)
}

// NewFromServices crea una API a partir de servicios ya construidos — útil para tests que
// usan repositorios en memoria en vez de Postgres real.
func NewFromServices(log *zap.Logger, issuerSvc *issuers.Service, numberingSvc *numbering.Service, documentsSvc *documents.Service) *API {
	return &API{log: log, issuers: issuerSvc, numbering: numberingSvc, documents: documentsSvc}
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
//	POST   /api/v1/issuers                              → alta de emisor/tenant
//	GET    /api/v1/issuers/{id}                          → consultar emisor
//	POST   /api/v1/issuers/{id}/numbering-ranges         → registrar rango de numeración
//	GET    /api/v1/numbering-ranges/{id}                 → consultar rango de numeración
//	POST   /api/v1/invoices                              → emitir Factura Electrónica de Venta
//	POST   /api/v1/credit-notes                          → emitir Nota Crédito
//	POST   /api/v1/debit-notes                           → emitir Nota Débito
//	GET    /api/v1/documents/{id}                        → consultar documento emitido
func (a *API) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/issuers", a.handleCreateIssuer)
	mux.HandleFunc("GET /api/v1/issuers/{id}", a.handleGetIssuer)
	mux.HandleFunc("POST /api/v1/issuers/{id}/numbering-ranges", a.handleCreateNumberingRange)
	mux.HandleFunc("GET /api/v1/numbering-ranges/{id}", a.handleGetNumberingRange)

	mux.HandleFunc("POST /api/v1/invoices", a.handleIssueInvoice)
	mux.HandleFunc("POST /api/v1/credit-notes", a.handleIssueCreditNote)
	mux.HandleFunc("POST /api/v1/debit-notes", a.handleIssueDebitNote)
	mux.HandleFunc("GET /api/v1/documents/{id}", a.handleGetDocument)
}

// ── helpers compartidos ─────────────────────────────────────────────────────────────────────

// parseUUID extrae un UUID de un path value y escribe un 400 si falla.
func parseUUID(w http.ResponseWriter, raw string) (uuid.UUID, bool) {
	return parseUUIDField(w, raw, "id")
}

// parseUUIDField valida un UUID que viene del BODY de la petición (no del path), nombrando el
// campo en el error — un uuid.UUID vacío o mal formado en un campo JSON hace que
// json.Decode falle como cualquier otro error de tipo, y antes eso se reportaba como el
// genérico "JSON inválido" sin decir cuál campo era ni por qué (issuer_id/numbering_range_id
// vacíos en Postman por correr los pasos fuera de orden, por ejemplo) — confuso de depurar.
func parseUUIDField(w http.ResponseWriter, raw, field string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: field + " inválido o vacío, se esperaba un UUID"})
		return uuid.Nil, false
	}
	return id, true
}
