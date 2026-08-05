package http

import "net/http"

func (h *Handler) handleGetMyPlan(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	p, err := h.myPlan.Get(r.Context(), companyID)
	if err != nil {
		writeErr(w, err)
		return
	}
	respond(w, http.StatusOK, toMyPlanDTO(p))
}
