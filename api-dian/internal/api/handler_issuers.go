package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/diegofxm/api-dian/internal/api/middleware"
	"github.com/diegofxm/api-dian/internal/issuers"
	"github.com/diegofxm/api-dian/internal/numbering"
	"github.com/google/uuid"

	"github.com/diegofxm/api-dian/internal/api/response"
)

// createIssuerRequest es el payload de datos del emisor — ya no se expone como
// POST /api/v1/issuers independiente (abierto sin autenticación); ahora viaja embebido en
// registerRequest (handler_auth.go), que crea el emisor y su primer usuario admin juntos.
// CertificateBase64/CertificatePassword/SoftwarePIN son secretos: se cifran en
// internal/issuers antes de guardar (AES-256-GCM) y NUNCA se devuelven en la respuesta — ver
// issuerResponse.
type createIssuerRequest struct {
	NIT                        string   `json:"nit"`
	CheckDigit                 string   `json:"check_digit"`
	BusinessName               string   `json:"business_name"`
	TradeName                  string   `json:"trade_name,omitempty"`
	IdentificationTypeCode     string   `json:"identification_type_code"`
	DepartmentCode             string   `json:"department_code"`
	MunicipalityCode           string   `json:"municipality_code"`
	AddressLine                string   `json:"address_line"`
	Email                      string   `json:"email"`
	Phone                      string   `json:"phone,omitempty"`
	Environment                string   `json:"environment"`
	EntityTypeCode             string   `json:"entity_type_code,omitempty"`
	TaxSchemeCode              string   `json:"tax_scheme_code,omitempty"`
	TaxSchemeName              string   `json:"tax_scheme_name,omitempty"`
	LiabilityCodes             []string `json:"liability_codes,omitempty"`
	MerchantRegistrationNumber *string  `json:"merchant_registration_number,omitempty"`
	SoftwareID                 string   `json:"software_id"`
	SoftwarePIN                string   `json:"software_pin"`
	CertificateBase64          string   `json:"certificate_base64"`
	CertificatePassword        string   `json:"certificate_password"`
}

// issuerResponse es deliberadamente angosto — nunca incluye Certificate/SoftwarePIN/
// CertificatePassword, ni siquiera cifrados: una vez guardados, esta API no los expone más.
type issuerResponse struct {
	ID                     uuid.UUID `json:"id"`
	NIT                    string    `json:"nit"`
	BusinessName           string    `json:"business_name"`
	IdentificationTypeCode string    `json:"identification_type_code"`
	Environment            string    `json:"environment"`
	IsActive               bool      `json:"is_active"`
	CreatedAt              time.Time `json:"created_at"`
}

func issuerToResponse(iss *issuers.Issuer) issuerResponse {
	return issuerResponse{
		ID:                     iss.ID,
		NIT:                    iss.NIT,
		BusinessName:           iss.BusinessName,
		IdentificationTypeCode: iss.IdentificationTypeCode,
		Environment:            string(iss.Environment),
		IsActive:               iss.IsActive,
		CreatedAt:              iss.CreatedAt,
	}
}

// issuerFromRequest convierte el DTO público en el issuers.Issuer de dominio — único lugar
// donde se hace esta conversión, usado por handleRegister (handler_auth.go).
func issuerFromRequest(req createIssuerRequest, cert []byte) issuers.Issuer {
	return issuers.Issuer{
		NIT:                        req.NIT,
		CheckDigit:                 req.CheckDigit,
		BusinessName:               req.BusinessName,
		TradeName:                  req.TradeName,
		IdentificationTypeCode:     req.IdentificationTypeCode,
		DepartmentCode:             req.DepartmentCode,
		MunicipalityCode:           req.MunicipalityCode,
		AddressLine:                req.AddressLine,
		Email:                      req.Email,
		Phone:                      req.Phone,
		Environment:                issuers.Environment(req.Environment),
		EntityTypeCode:             req.EntityTypeCode,
		TaxSchemeCode:              req.TaxSchemeCode,
		TaxSchemeName:              req.TaxSchemeName,
		LiabilityCodes:             req.LiabilityCodes,
		MerchantRegistrationNumber: req.MerchantRegistrationNumber,
		SoftwareID:                 req.SoftwareID,
		SoftwarePIN:                req.SoftwarePIN,
		Certificate:                cert,
		CertificatePassword:        req.CertificatePassword,
	}
}

// handleGetMyIssuer devuelve el emisor del usuario autenticado — "un usuario = un emisor", no
// hace falta un {id} en el path: siempre es el propio (middleware.GetTenantID).
func (a *API) handleGetMyIssuer(w http.ResponseWriter, r *http.Request) {
	iss, err := a.issuers.GetIssuer(r.Context(), middleware.GetTenantID(r.Context()))
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, issuerToResponse(iss))
}

// createNumberingRangeRequest es el payload de registro de un rango de numeración para el
// emisor autenticado. TechnicalKey solo aplica a Invoice (CUFE); TestSetID solo aplica en
// habilitación.
//
// NextNumber es opcional — omitirlo significa "rango nunca usado", el primer documento
// emitido reclama RangeFrom. Si la resolución YA tiene números autorizados de verdad en la
// DIAN (ej. ya se usó antes directamente con cofacture, o se emitió manualmente fuera de
// api-dian), NextNumber le dice a la API exactamente qué número debe entregar el primer
// ClaimNext, para no reclamar uno que la DIAN ya vio y autorizó.
type createNumberingRangeRequest struct {
	DianDocumentTypeCode string `json:"dian_document_type_code"`
	Prefix               string `json:"prefix"`
	ResolutionNumber     string `json:"resolution_number"`
	ResolutionDate       string `json:"resolution_date"` // YYYY-MM-DD
	RangeFrom            int64  `json:"range_from"`
	RangeTo              *int64 `json:"range_to,omitempty"`
	ValidFrom            string `json:"valid_from"` // YYYY-MM-DD
	ValidTo              string `json:"valid_to"`   // YYYY-MM-DD
	Environment          string `json:"environment"`
	TechnicalKey         string `json:"technical_key,omitempty"`
	TestSetID            string `json:"test_set_id,omitempty"`
	NextNumber           *int64 `json:"next_number,omitempty"`
}

type numberingRangeResponse struct {
	ID                   uuid.UUID `json:"id"`
	IssuerID             uuid.UUID `json:"issuer_id"`
	DianDocumentTypeCode string    `json:"dian_document_type_code"`
	Prefix               string    `json:"prefix"`
	RangeFrom            int64     `json:"range_from"`
	RangeTo              *int64    `json:"range_to,omitempty"`
	CurrentNumber        int64     `json:"current_number"`
	Environment          string    `json:"environment"`
	IsActive             bool      `json:"is_active"`
}

func numberingRangeToResponse(nr *numbering.NumberingRange) numberingRangeResponse {
	return numberingRangeResponse{
		ID:                   nr.ID,
		IssuerID:             nr.IssuerID,
		DianDocumentTypeCode: nr.DianDocumentTypeCode,
		Prefix:               nr.Prefix,
		RangeFrom:            nr.RangeFrom,
		RangeTo:              nr.RangeTo,
		CurrentNumber:        nr.CurrentNumber,
		Environment:          string(nr.Environment),
		IsActive:             nr.IsActive,
	}
}

func (a *API) handleCreateNumberingRange(w http.ResponseWriter, r *http.Request) {
	issuerID := middleware.GetTenantID(r.Context())

	var req createNumberingRangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	resolutionDate, err := parseDate(req.ResolutionDate)
	if err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "resolution_date inválida, se espera YYYY-MM-DD"})
		return
	}
	validFrom, err := parseDate(req.ValidFrom)
	if err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "valid_from inválida, se espera YYYY-MM-DD"})
		return
	}
	validTo, err := parseDate(req.ValidTo)
	if err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "valid_to inválida, se espera YYYY-MM-DD"})
		return
	}

	nr, err := a.numbering.RegisterRange(r.Context(), numbering.NumberingRange{
		IssuerID:             issuerID,
		DianDocumentTypeCode: req.DianDocumentTypeCode,
		Prefix:               req.Prefix,
		ResolutionNumber:     req.ResolutionNumber,
		ResolutionDate:       resolutionDate,
		RangeFrom:            req.RangeFrom,
		RangeTo:              req.RangeTo,
		ValidFrom:            validFrom,
		ValidTo:              validTo,
		Environment:          numbering.Environment(req.Environment),
		TechnicalKey:         req.TechnicalKey,
		TestSetID:            req.TestSetID,
	}, req.NextNumber)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, numberingRangeToResponse(nr))
}

// handleListNumberingRanges devuelve los rangos del emisor autenticado, opcionalmente
// filtrados por ?dian_document_type_code=. Sin paginación a propósito — ver
// numbering.Repository.ListByIssuer.
func (a *API) handleListNumberingRanges(w http.ResponseWriter, r *http.Request) {
	issuerID := middleware.GetTenantID(r.Context())
	docType := r.URL.Query().Get("dian_document_type_code")

	ranges, err := a.numbering.ListRanges(r.Context(), issuerID, docType)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	out := make([]numberingRangeResponse, len(ranges))
	for i, nr := range ranges {
		out[i] = numberingRangeToResponse(nr)
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"numbering_ranges": out, "count": len(out)})
}

// handleGetNumberingRange exige que el rango pertenezca al emisor autenticado — si no, se
// responde el mismo 404 que un rango inexistente (numbering.ErrRangeNotFound), para no
// revelarle a un usuario que el ID que probó existe pero es de otro emisor.
func (a *API) handleGetNumberingRange(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	nr, err := a.numbering.GetRange(r.Context(), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	if nr.IssuerID != middleware.GetTenantID(r.Context()) {
		response.WriteError(w, numbering.ErrRangeNotFound)
		return
	}

	response.WriteJSON(w, http.StatusOK, numberingRangeToResponse(nr))
}

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}
