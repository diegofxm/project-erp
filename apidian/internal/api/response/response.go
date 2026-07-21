package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/diegofxm/apidian/internal/auth"
	"github.com/diegofxm/apidian/internal/customers"
	"github.com/diegofxm/apidian/internal/documents"
	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/diegofxm/apidian/internal/numbering"
	"github.com/diegofxm/apidian/internal/products"
	"github.com/diegofxm/apidian/internal/vendors"
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
		errors.Is(err, documents.ErrDocumentNotFound),
		errors.Is(err, customers.ErrCustomerNotFound),
		errors.Is(err, products.ErrProductNotFound),
		errors.Is(err, vendors.ErrVendorNotFound),
		errors.Is(err, auth.ErrUserNotFound),
		errors.Is(err, auth.ErrIssuerAccessDenied):
		return http.StatusNotFound, err.Error()

	// ── 401 ──────────────────────────────────────────────────────────────────
	case errors.Is(err, auth.ErrInvalidCredentials),
		errors.Is(err, auth.ErrUserInactive),
		errors.Is(err, auth.ErrInvalidInviteToken):
		return http.StatusUnauthorized, err.Error()

	// ── 409 ──────────────────────────────────────────────────────────────────
	case errors.Is(err, issuers.ErrNITAlreadyExists),
		errors.Is(err, auth.ErrEmailAlreadyExists),
		errors.Is(err, documents.ErrDocumentNotDraft),
		errors.Is(err, documents.ErrDocumentNotSigned),
		errors.Is(err, products.ErrDuplicateItemCode):
		return http.StatusConflict, err.Error()

	// ── 400 (validación / datos faltantes) ────────────────────────────────────
	case errors.Is(err, issuers.ErrEmptyNIT),
		errors.Is(err, issuers.ErrEmptyBusinessName),
		errors.Is(err, issuers.ErrEmptySoftwareID),
		errors.Is(err, issuers.ErrEmptySoftwarePIN),
		errors.Is(err, issuers.ErrEmptyCertificate),
		errors.Is(err, issuers.ErrInvalidCertificate),
		errors.Is(err, issuers.ErrInvalidEnvironment),
		errors.Is(err, issuers.ErrTooManyIndustryClassificationCodes),
		errors.Is(err, issuers.ErrInvalidLiabilityCode),
		errors.Is(err, issuers.ErrInvalidTaxSchemeCode),
		errors.Is(err, issuers.ErrEmptyLogo),
		errors.Is(err, issuers.ErrInvalidLogoContentType),
		errors.Is(err, issuers.ErrInvalidIdentificationNumber),
		errors.Is(err, numbering.ErrMissingIssuer),
		errors.Is(err, numbering.ErrMissingDocumentType),
		errors.Is(err, numbering.ErrEmptyPrefix),
		errors.Is(err, numbering.ErrInvalidRange),
		errors.Is(err, numbering.ErrInvalidEnvironment),
		errors.Is(err, numbering.ErrNextNumberOutOfRange),
		errors.Is(err, numbering.ErrMissingTestSetID),
		errors.Is(err, documents.ErrMissingIssuer),
		errors.Is(err, documents.ErrMissingNumberingRange),
		errors.Is(err, documents.ErrEmptyLines),
		errors.Is(err, documents.ErrMissingCustomer),
		errors.Is(err, documents.ErrMissingPaymentMeans),
		errors.Is(err, documents.ErrInvalidPaymentTerm),
		errors.Is(err, documents.ErrInvalidPaymentMethod),
		errors.Is(err, documents.ErrInvalidLiabilityCode),
		errors.Is(err, documents.ErrInvalidTaxSchemeCode),
		errors.Is(err, documents.ErrInvalidTaxTypeCode),
		errors.Is(err, documents.ErrInvalidItemStandardCode),
		errors.Is(err, documents.ErrMissingBillingReference),
		errors.Is(err, auth.ErrEmptyEmail),
		errors.Is(err, auth.ErrEmptyPassword),
		errors.Is(err, auth.ErrPasswordTooShort),
		errors.Is(err, auth.ErrEmptyName),
		errors.Is(err, auth.ErrNoEmailSender),
		errors.Is(err, auth.ErrInviteTokenExpired),
		errors.Is(err, auth.ErrInviteAlreadyUsed),
		errors.Is(err, customers.ErrEmptyName),
		errors.Is(err, customers.ErrEmptyIdentification),
		errors.Is(err, customers.ErrInvalidLiabilityCode),
		errors.Is(err, customers.ErrInvalidTaxSchemeCode),
		errors.Is(err, customers.ErrInvalidIdentificationNumber),
		errors.Is(err, vendors.ErrEmptyName),
		errors.Is(err, vendors.ErrEmptyIdentification),
		errors.Is(err, vendors.ErrInvalidLiabilityCode),
		errors.Is(err, vendors.ErrInvalidTaxSchemeCode),
		errors.Is(err, vendors.ErrInvalidIdentificationNumber),
		errors.Is(err, products.ErrEmptyDescription),
		errors.Is(err, products.ErrEmptyUnitCode),
		errors.Is(err, products.ErrInvalidUnitPrice),
		errors.Is(err, products.ErrInvalidTaxTypeCode),
		errors.Is(err, products.ErrInvalidItemStandardCode):
		return http.StatusBadRequest, err.Error()

	// ── 422 (regla de negocio: la petición está bien formada pero no se puede
	//         cumplir tal como está) ───────────────────────────────────────────
	case errors.Is(err, numbering.ErrRangeExhausted),
		errors.Is(err, documents.ErrWrongDocumentType),
		errors.Is(err, documents.ErrNumberingRangeIssuerMismatch),
		errors.Is(err, documents.ErrCustomerIssuerMismatch),
		errors.Is(err, documents.ErrIssuerNotReadyToIssue),
		errors.Is(err, documents.ErrDocumentNotAccepted),
		errors.Is(err, documents.ErrCustomerEmailMissing):
		return http.StatusUnprocessableEntity, err.Error()
	}

	// ── 500 ──────────────────────────────────────────────────────────────────
	return http.StatusInternalServerError, "error interno del servidor"
}
