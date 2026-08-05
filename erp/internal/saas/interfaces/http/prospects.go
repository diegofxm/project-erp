package http

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/saas/application"
)

// handleSubmitProspect — público, sin autenticación (POST /api/v1/public/prospects). Archivos
// vienen en base64 dentro del JSON, mismo criterio que company.handleUpdateLogo.
func (h *Handler) handleSubmitProspect(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name              string `json:"name"`
		Email             string `json:"email"`
		NIT               string `json:"nit"`
		CedulaBase64      string `json:"cedula_base64"`
		CedulaContentType string `json:"cedula_content_type"`
		RUTBase64         string `json:"rut_base64"`
		RUTContentType    string `json:"rut_content_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	if body.Name == "" || body.Email == "" {
		respondError(w, http.StatusBadRequest, "nombre y correo son requeridos")
		return
	}
	req := application.SubmitProspectRequest{
		Name: body.Name, Email: body.Email, NIT: body.NIT,
		CedulaContentType: body.CedulaContentType, RUTContentType: body.RUTContentType,
	}
	if body.CedulaBase64 != "" {
		data, err := base64.StdEncoding.DecodeString(body.CedulaBase64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "cedula_base64 inválido")
			return
		}
		req.CedulaFile = data
	}
	if body.RUTBase64 != "" {
		data, err := base64.StdEncoding.DecodeString(body.RUTBase64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "rut_base64 inválido")
			return
		}
		req.RUTFile = data
	}
	p, err := h.prospects.Submit(r.Context(), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond(w, http.StatusCreated, toProspectDTO(p))
}

func (h *Handler) handleListProspects(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	list, err := h.prospects.List(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	dtos := toProspectDTOs(list)
	respond(w, http.StatusOK, map[string]any{"prospects": dtos, "count": len(dtos)})
}

func (h *Handler) handleApproveProspect(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	p, err := h.prospects.Approve(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logAudit(r.Context(), "saas.prospect_approved", "prospect", p.ID, nil)
	respond(w, http.StatusOK, toProspectDTO(p))
}

func (h *Handler) handleRejectProspect(w http.ResponseWriter, r *http.Request) {
	if !requireSuperAdmin(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	var body struct {
		Notes string `json:"notes"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	p, err := h.prospects.Reject(r.Context(), id, body.Notes)
	if err != nil {
		writeErr(w, err)
		return
	}
	h.logAudit(r.Context(), "saas.prospect_rejected", "prospect", p.ID, map[string]any{"notes": body.Notes})
	respond(w, http.StatusOK, toProspectDTO(p))
}

func (h *Handler) handleDownloadCedula(w http.ResponseWriter, r *http.Request) {
	h.downloadProspectFile(w, r, true)
}

func (h *Handler) handleDownloadRUT(w http.ResponseWriter, r *http.Request) {
	h.downloadProspectFile(w, r, false)
}

func (h *Handler) downloadProspectFile(w http.ResponseWriter, r *http.Request, cedula bool) {
	if !requireSuperAdmin(w, r) {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "id inválido")
		return
	}
	p, err := h.prospects.GetByID(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	data, ct := p.RUTFile, p.RUTContentType
	if cedula {
		data, ct = p.CedulaFile, p.CedulaContentType
	}
	if len(data) == 0 {
		respondError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}
	if ct == "" {
		ct = "application/pdf"
	}
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
