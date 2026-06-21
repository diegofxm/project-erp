package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/diegofxm/api-dian/internal/documents"
	"github.com/diegofxm/api-dian/internal/issuers"
	"github.com/diegofxm/api-dian/internal/numbering"
)

// Error es el JSON body devuelto en respuestas no-2xx.
type Error struct {
	Error string `json:"error"`
}

// WriteJSON codifica v como JSON con el status dado.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError mapea un error de dominio a un status HTTP + body JSON.
func WriteError(w http.ResponseWriter, err error) {
	status, msg := classify(err)
	WriteJSON(w, status, Error{Error: msg})
}

// classify mapea errores de dominio a (httpStatus, mensaje para el usuario).
func classify(err error) (int, string) {
	// ── 404 ──────────────────────────────────────────────────────────────────
	switch {
	case errors.Is(err, issuers.ErrIssuerNotFound),
		errors.Is(err, numbering.ErrRangeNotFound),
		errors.Is(err, documents.ErrDocumentNotFound):
		return http.StatusNotFound, err.Error()

	// ── 409 ──────────────────────────────────────────────────────────────────
	case errors.Is(err, issuers.ErrNITAlreadyExists):
		return http.StatusConflict, err.Error()

	// ── 400 (validación / datos faltantes) ────────────────────────────────────
	case errors.Is(err, issuers.ErrEmptyNIT),
		errors.Is(err, issuers.ErrEmptyBusinessName),
		errors.Is(err, issuers.ErrEmptySoftwareID),
		errors.Is(err, issuers.ErrEmptySoftwarePIN),
		errors.Is(err, issuers.ErrEmptyCertificate),
		errors.Is(err, issuers.ErrInvalidEnvironment),
		errors.Is(err, numbering.ErrMissingIssuer),
		errors.Is(err, numbering.ErrMissingDocumentType),
		errors.Is(err, numbering.ErrEmptyPrefix),
		errors.Is(err, numbering.ErrInvalidRange),
		errors.Is(err, numbering.ErrInvalidEnvironment),
		errors.Is(err, documents.ErrMissingIssuer),
		errors.Is(err, documents.ErrMissingNumberingRange),
		errors.Is(err, documents.ErrEmptyLines),
		errors.Is(err, documents.ErrMissingCustomer),
		errors.Is(err, documents.ErrMissingBillingReference):
		return http.StatusBadRequest, err.Error()

	// ── 422 (regla de negocio: la petición está bien formada pero no se puede
	//         cumplir tal como está) ───────────────────────────────────────────
	case errors.Is(err, numbering.ErrRangeExhausted),
		errors.Is(err, documents.ErrWrongDocumentType):
		return http.StatusUnprocessableEntity, err.Error()
	}

	// ── 500 ──────────────────────────────────────────────────────────────────
	return http.StatusInternalServerError, "error interno del servidor"
}
