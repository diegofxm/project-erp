package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/diegofxm/apidian/internal/api/response"
	"github.com/diegofxm/apidian/internal/prospects"
	"github.com/google/uuid"
)

// ── DTOs ─────────────────────────────────────────────────────────────────────

type prospectResponse struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Email      string     `json:"email"`
	NIT        string     `json:"nit,omitempty"`
	HasCedula  bool       `json:"has_cedula"`
	HasRut     bool       `json:"has_rut"`
	Status     string     `json:"status"`
	Notes      string     `json:"notes,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func prospectToResponse(p *prospects.Prospect) prospectResponse {
	return prospectResponse{
		ID: p.ID, Name: p.Name, Email: p.Email, NIT: p.NIT,
		HasCedula: p.HasCedula, HasRut: p.HasRut,
		Status: string(p.Status), Notes: p.Notes, ReviewedAt: p.ReviewedAt,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

// ── Endpoint público (cofacture.co → apidian) ─────────────────────────────────

type submitProspectRequest struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	NIT            string `json:"nit"`
	CedulaPDFBase64 string `json:"cedula_pdf_base64"`
	RutPDFBase64   string `json:"rut_pdf_base64"`
}

func (a *API) handleSubmitProspect(w http.ResponseWriter, r *http.Request) {
	var req submitProspectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}
	p, err := a.prospects.Submit(r.Context(), req.Name, req.Email, req.NIT, req.CedulaPDFBase64, req.RutPDFBase64)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusCreated, prospectToResponse(p))
}

// ── Endpoints de superadmin ───────────────────────────────────────────────────

func (a *API) handleAdminListProspects(w http.ResponseWriter, r *http.Request) {
	list, err := a.prospects.List(r.Context())
	if err != nil {
		response.WriteError(w, err)
		return
	}
	out := make([]prospectResponse, len(list))
	for i := range list {
		out[i] = prospectToResponse(&list[i])
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"prospects": out, "count": len(out)})
}

func (a *API) handleAdminApproveProspect(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	p, err := a.prospects.GetByID(r.Context(), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	if _, err := a.auth.CreateInvited(r.Context(), p.Email, p.Name); err != nil {
		response.WriteError(w, err)
		return
	}
	updated, err := a.prospects.Approve(r.Context(), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, prospectToResponse(updated))
}

func (a *API) handleAdminRejectProspect(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var body struct {
		Notes string `json:"notes"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	updated, err := a.prospects.Reject(r.Context(), id, body.Notes)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, prospectToResponse(updated))
}

func (a *API) handleAdminGetProspectCedula(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	data, err := a.prospects.GetCedulaPDF(r.Context(), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	if len(data) == 0 {
		response.WriteJSON(w, http.StatusNotFound, response.Error{Error: "cédula no disponible"})
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	_, _ = w.Write(data)
}

func (a *API) handleAdminGetProspectRut(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}
	data, err := a.prospects.GetRutPDF(r.Context(), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	if len(data) == 0 {
		response.WriteJSON(w, http.StatusNotFound, response.Error{Error: "RUT no disponible"})
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	_, _ = w.Write(data)
}
