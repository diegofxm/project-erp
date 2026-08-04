package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	companyapp "github.com/diegofxm/erp/internal/company/application"
	companydomain "github.com/diegofxm/erp/internal/company/domain"
	thirdpartyapp "github.com/diegofxm/erp/internal/thirdparty/application"
	thirdpartydomain "github.com/diegofxm/erp/internal/thirdparty/domain"
)

type Handler struct {
	company  *companyapp.GetUseCase
	customer *thirdpartyapp.CreateUseCase
}

func NewHandler(company *companyapp.GetUseCase, customer *thirdpartyapp.CreateUseCase) *Handler {
	return &Handler{company: company, customer: customer}
}

// GET /api/v1/public/companies/{id}
func (h *Handler) handleGetCompany(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	c, err := h.company.ByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, companydomain.ErrCompanyNotFound) {
			respondError(w, http.StatusNotFound, "enlace de registro no válido")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !c.IsActive {
		respondError(w, http.StatusNotFound, "enlace de registro no válido")
		return
	}
	respond(w, http.StatusOK, map[string]any{
		"business_name": c.BusinessName,
		"has_logo":      len(c.Logo) > 0,
	})
}

// GET /api/v1/public/companies/{id}/logo
func (h *Handler) handleGetCompanyLogo(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	c, err := h.company.ByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, companydomain.ErrCompanyNotFound) {
			respondError(w, http.StatusNotFound, "enlace de registro no válido")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(c.Logo) == 0 {
		respondError(w, http.StatusNotFound, "sin logo")
		return
	}
	ct := c.LogoContentType
	if ct == "" {
		ct = "image/png"
	}
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(c.Logo)
}

// POST /api/v1/public/companies/{id}/customers
func (h *Handler) handleRegisterCustomer(w http.ResponseWriter, r *http.Request) {
	companyID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}

	c, err := h.company.ByID(r.Context(), companyID)
	if err != nil {
		if errors.Is(err, companydomain.ErrCompanyNotFound) {
			respondError(w, http.StatusNotFound, "enlace de registro no válido")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !c.IsActive {
		respondError(w, http.StatusNotFound, "enlace de registro no válido")
		return
	}

	var body struct {
		Identification struct {
			Number   string `json:"number"`
			TypeCode string `json:"type_code"`
		} `json:"identification"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	if body.Identification.Number == "" || body.Identification.TypeCode == "" || body.Name == "" {
		respondError(w, http.StatusBadRequest, "number, type_code y name son requeridos")
		return
	}

	cust, err := h.customer.Execute(r.Context(), companyID, thirdpartydomain.RoleCustomer, thirdpartyapp.SaveRequest{
		IdentificationTypeCode: body.Identification.TypeCode,
		IdentificationNumber:   body.Identification.Number,
		Name:                   body.Name,
		Email:                  body.Email,
		Phone:                  body.Phone,
		TaxSchemeCode:          "ZZ",
		TaxSchemeName:          "No aplica",
		LiabilityCodes:         []string{},
		AddressCountryCode:     "CO",
		AddressCountryName:     "Colombia",
	})
	if err != nil {
		if errors.Is(err, thirdpartydomain.ErrDuplicateCustomer) {
			respondError(w, http.StatusConflict, "ya existe un cliente con esta identificación")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusCreated, map[string]string{"name": cust.Name})
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}
