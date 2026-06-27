package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/diegofxm/apidian/internal/api/middleware"
	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/diegofxm/apidian/internal/numbering"
	"github.com/google/uuid"

	"github.com/diegofxm/apidian/internal/api/response"
)

// createIssuerRequest es el payload de datos de una empresa nueva — POST /api/v1/issuers
// (autenticado, pero no embebido en el registro: desde la Fase 9.32 el registro de usuario y
// la creación de empresa están desacoplados, ver sección 9.32 y handler_auth.go).
// CertificateBase64/CertificatePassword/SoftwarePIN son secretos: se cifran en
// internal/issuers antes de guardar (AES-256-GCM) y NUNCA se devuelven en la respuesta — ver
// issuerResponse.
type createIssuerRequest struct {
	NIT                    string `json:"nit"`
	CheckDigit             string `json:"check_digit"`
	BusinessName           string `json:"business_name"`
	TradeName              string `json:"trade_name,omitempty"`
	IdentificationTypeCode string `json:"identification_type_code"`
	DepartmentCode         string `json:"department_code"`
	MunicipalityCode       string `json:"municipality_code"`
	AddressLine            string `json:"address_line"`
	Email                  string `json:"email"`
	Phone                  string `json:"phone,omitempty"`
	Environment            string `json:"environment"`
	EntityTypeCode         string `json:"entity_type_code,omitempty"`
	// TaxSchemeCode es el único campo que se manda — TaxSchemeName se deriva del catálogo
	// tax_types en issuers.Service, nunca se acepta del cliente (ver
	// docs/apidian-architecture.md).
	TaxSchemeCode  string   `json:"tax_scheme_code,omitempty"`
	LiabilityCodes []string `json:"liability_codes,omitempty"`
	TaxRegimeCode  *string  `json:"tax_regime_code,omitempty"`
	// IndustryClassificationCodes son los códigos CIIU (catálogo DANE, no DIAN) — máximo 4
	// (1 actividad principal + hasta 3 secundarias, ver issuers.ErrTooManyIndustryClassificationCodes).
	IndustryClassificationCodes []string `json:"industry_classification_codes,omitempty"`
	MerchantRegistrationNumber  *string  `json:"merchant_registration_number,omitempty"`
	SoftwareID                  string   `json:"software_id"`
	SoftwarePIN                 string   `json:"software_pin"`
	CertificateBase64           string   `json:"certificate_base64"`
	CertificatePassword         string   `json:"certificate_password"`
}

// issuerResponse es deliberadamente angosto — nunca incluye Certificate/SoftwarePIN/
// CertificatePassword, ni siquiera cifrados: una vez guardados, esta API no los expone más.
// HasSoftwareCredentials/HasCertificate son solo presencia (true/false) — para que el
// frontend pueda mostrar "ya configurado" sin que el secreto en sí viaje de vuelta nunca.
type issuerResponse struct {
	ID                          uuid.UUID `json:"id"`
	NIT                         string    `json:"nit"`
	BusinessName                string    `json:"business_name"`
	IdentificationTypeCode      string    `json:"identification_type_code"`
	Environment                 string    `json:"environment"`
	TaxRegimeCode               *string   `json:"tax_regime_code,omitempty"`
	IndustryClassificationCodes []string  `json:"industry_classification_codes,omitempty"`
	HasSoftwareCredentials      bool      `json:"has_software_credentials"`
	HasCertificate              bool      `json:"has_certificate"`
	IsActive                    bool      `json:"is_active"`
	CreatedAt                   time.Time `json:"created_at"`
}

func issuerToResponse(iss *issuers.Issuer) issuerResponse {
	return issuerResponse{
		ID:                          iss.ID,
		NIT:                         iss.NIT,
		BusinessName:                iss.BusinessName,
		IdentificationTypeCode:      iss.IdentificationTypeCode,
		Environment:                 string(iss.Environment),
		TaxRegimeCode:               iss.TaxRegimeCode,
		IndustryClassificationCodes: iss.IndustryClassificationCodes,
		HasSoftwareCredentials:      iss.SoftwareID != "" && iss.SoftwarePIN != "",
		HasCertificate:              len(iss.Certificate) > 0 && iss.CertificatePassword != "",
		IsActive:                    iss.IsActive,
		CreatedAt:                   iss.CreatedAt,
	}
}

// issuerFromRequest convierte el DTO público en el issuers.Issuer de dominio — único lugar
// donde se hace esta conversión, usado por handleRegister (handler_auth.go).
func issuerFromRequest(req createIssuerRequest, cert []byte) issuers.Issuer {
	return issuers.Issuer{
		NIT:                         req.NIT,
		CheckDigit:                  req.CheckDigit,
		BusinessName:                req.BusinessName,
		TradeName:                   req.TradeName,
		IdentificationTypeCode:      req.IdentificationTypeCode,
		DepartmentCode:              req.DepartmentCode,
		MunicipalityCode:            req.MunicipalityCode,
		AddressLine:                 req.AddressLine,
		Email:                       req.Email,
		Phone:                       req.Phone,
		Environment:                 issuers.Environment(req.Environment),
		EntityTypeCode:              req.EntityTypeCode,
		TaxSchemeCode:               req.TaxSchemeCode,
		LiabilityCodes:              req.LiabilityCodes,
		TaxRegimeCode:               req.TaxRegimeCode,
		IndustryClassificationCodes: req.IndustryClassificationCodes,
		MerchantRegistrationNumber:  req.MerchantRegistrationNumber,
		SoftwareID:                  req.SoftwareID,
		SoftwarePIN:                 req.SoftwarePIN,
		Certificate:                 cert,
		CertificatePassword:         req.CertificatePassword,
	}
}

// handleCreateIssuer crea una empresa nueva y la vincula al usuario autenticado como "owner"
// (auth.Service.CreateIssuerForUser) — el camino para completar el "espacio de empresa"
// después de un registro desacoplado (ver sección 9.32), o para agregar una empresa/sucursal
// adicional a un usuario que ya tiene otras (multi-empresa). NO usa middleware.RequireTenant
// a propósito: un usuario sin ninguna empresa todavía necesita poder llamar esto. Reutiliza
// createIssuerRequest/issuerFromRequest tal cual los usaba el registro antes de la 9.32.
func (a *API) handleCreateIssuer(w http.ResponseWriter, r *http.Request) {
	var req createIssuerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	cert, err := base64.StdEncoding.DecodeString(req.CertificateBase64)
	if err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "certificate_base64 no es base64 válido"})
		return
	}

	result, err := a.auth.CreateIssuerForUser(r.Context(), middleware.GetUserID(r.Context()), issuerFromRequest(req, cert))
	if err != nil {
		response.WriteError(w, err)
		return
	}

	writeAuthResponse(w, http.StatusCreated, result)
}

// handleListMyIssuers lista las empresas a las que el usuario autenticado tiene acceso (ver
// user_issuers) — para que el cliente muestre un selector cuando hay más de una. Tampoco usa
// middleware.RequireTenant: es justamente lo que un usuario sin empresa activa necesita
// poder consultar (para saber si tiene 0, debe crear una; o 2+, debe elegir).
func (a *API) handleListMyIssuers(w http.ResponseWriter, r *http.Request) {
	list, err := a.auth.ListUserIssuers(r.Context(), middleware.GetUserID(r.Context()))
	if err != nil {
		response.WriteError(w, err)
		return
	}

	out := make([]issuerResponse, len(list))
	for i, iss := range list {
		out[i] = issuerToResponse(iss)
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"issuers": out, "count": len(out)})
}

// handleSelectIssuer reemite el token con {id} como empresa activa — solo si el usuario
// autenticado de verdad tiene acceso a ella (auth.ErrIssuerAccessDenied si no).
func (a *API) handleSelectIssuer(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	result, err := a.auth.SelectIssuer(r.Context(), middleware.GetUserID(r.Context()), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	writeAuthResponse(w, http.StatusOK, result)
}

// handleGetMyIssuer devuelve la empresa ACTIVA del usuario autenticado (middleware.GetTenantID
// — exige middleware.RequireTenant en la ruta).
func (a *API) handleGetMyIssuer(w http.ResponseWriter, r *http.Request) {
	iss, err := a.issuers.GetIssuer(r.Context(), middleware.GetTenantID(r.Context()))
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, issuerToResponse(iss))
}

// updateIssuerRequest completa software/PIN/certificado DESPUÉS del registro — todos los
// campos son opcionales (punteros nil = "no tocar"), porque el usuario los va consiguiendo
// independientemente, no necesariamente el mismo día que se registra (ver
// docs/apidian-architecture.md sección 9.25). CertificateBase64 es *string, no string: hay
// que distinguir "no mandó este campo" (nil) de "lo mandó vacío" (string vacío, que
// issuers.Service.UpdateIssuer rechaza con ErrEmptyCertificate).
type updateIssuerRequest struct {
	SoftwareID          *string `json:"software_id,omitempty"`
	SoftwarePIN         *string `json:"software_pin,omitempty"`
	CertificateBase64   *string `json:"certificate_base64,omitempty"`
	CertificatePassword *string `json:"certificate_password,omitempty"`
}

// handleUpdateMyIssuer completa/reemplaza software/PIN/certificado del emisor autenticado —
// "un usuario = un emisor", igual que handleGetMyIssuer, nunca recibe un {id} en el path.
func (a *API) handleUpdateMyIssuer(w http.ResponseWriter, r *http.Request) {
	var req updateIssuerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	svcReq := issuers.UpdateIssuerRequest{
		SoftwareID:          req.SoftwareID,
		SoftwarePIN:         req.SoftwarePIN,
		CertificatePassword: req.CertificatePassword,
	}
	if req.CertificateBase64 != nil {
		cert, err := base64.StdEncoding.DecodeString(*req.CertificateBase64)
		if err != nil {
			response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "certificate_base64 no es base64 válido"})
			return
		}
		svcReq.Certificate = cert
	}

	iss, err := a.issuers.UpdateIssuer(r.Context(), middleware.GetTenantID(r.Context()), svcReq)
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
// apidian), NextNumber le dice a la API exactamente qué número debe entregar el primer
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
