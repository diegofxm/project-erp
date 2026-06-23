package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/diegofxm/apidian/internal/api/middleware"
	"github.com/diegofxm/apidian/internal/api/response"
	"github.com/diegofxm/apidian/internal/products"
	"github.com/google/uuid"
)

// productRequest/productResponse son deliberadamente propios, no lineDTO: el catálogo no
// incluye quantity/line_extension_cents ni una lista de impuestos (eso es dato de USO, ver
// internal/products/model.go) — solo un impuesto por defecto, de conveniencia.
type productRequest struct {
	Description      string  `json:"description"`
	UnitCode         string  `json:"unit_code"`
	UnitPriceCents   int64   `json:"unit_price_cents"`
	ItemCode         string  `json:"item_code,omitempty"`
	ItemTypeCode     string  `json:"item_type_code,omitempty"`
	ItemTypeName     string  `json:"item_type_name,omitempty"`
	ItemTypeAgencyID string  `json:"item_type_agency_id,omitempty"`
	TaxTypeCode      string  `json:"tax_type_code,omitempty"`
	TaxTypeName      string  `json:"tax_type_name,omitempty"`
	TaxPercent       float64 `json:"tax_percent,omitempty"`
}

func (req productRequest) toDomain() products.Product {
	return products.Product{
		Description:      req.Description,
		UnitCode:         req.UnitCode,
		UnitPriceCents:   req.UnitPriceCents,
		ItemCode:         req.ItemCode,
		ItemTypeCode:     req.ItemTypeCode,
		ItemTypeName:     req.ItemTypeName,
		ItemTypeAgencyID: req.ItemTypeAgencyID,
		TaxTypeCode:      req.TaxTypeCode,
		TaxTypeName:      req.TaxTypeName,
		TaxPercent:       req.TaxPercent,
	}
}

type productResponse struct {
	ID               uuid.UUID `json:"id"`
	Description      string    `json:"description"`
	UnitCode         string    `json:"unit_code"`
	UnitPriceCents   int64     `json:"unit_price_cents"`
	ItemCode         string    `json:"item_code,omitempty"`
	ItemTypeCode     string    `json:"item_type_code,omitempty"`
	ItemTypeName     string    `json:"item_type_name,omitempty"`
	ItemTypeAgencyID string    `json:"item_type_agency_id,omitempty"`
	TaxTypeCode      string    `json:"tax_type_code,omitempty"`
	TaxTypeName      string    `json:"tax_type_name,omitempty"`
	TaxPercent       float64   `json:"tax_percent,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func productToResponse(p *products.Product) productResponse {
	return productResponse{
		ID:               p.ID,
		Description:      p.Description,
		UnitCode:         p.UnitCode,
		UnitPriceCents:   p.UnitPriceCents,
		ItemCode:         p.ItemCode,
		ItemTypeCode:     p.ItemTypeCode,
		ItemTypeName:     p.ItemTypeName,
		ItemTypeAgencyID: p.ItemTypeAgencyID,
		TaxTypeCode:      p.TaxTypeCode,
		TaxTypeName:      p.TaxTypeName,
		TaxPercent:       p.TaxPercent,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}

func (a *API) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var req productRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	p, err := a.products.CreateProduct(r.Context(), middleware.GetTenantID(r.Context()), req.toDomain())
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, productToResponse(p))
}

// handleGetProduct exige que el producto pertenezca al emisor autenticado — si no, el mismo
// 404 que un producto inexistente (products.ErrProductNotFound).
func (a *API) handleGetProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	p, err := a.products.GetProduct(r.Context(), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	if p.IssuerID != middleware.GetTenantID(r.Context()) {
		response.WriteError(w, products.ErrProductNotFound)
		return
	}

	response.WriteJSON(w, http.StatusOK, productToResponse(p))
}

func (a *API) handleListProducts(w http.ResponseWriter, r *http.Request) {
	list, err := a.products.ListProducts(r.Context(), middleware.GetTenantID(r.Context()))
	if err != nil {
		response.WriteError(w, err)
		return
	}

	out := make([]productResponse, len(list))
	for i, p := range list {
		out[i] = productToResponse(p)
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"products": out, "count": len(out)})
}

func (a *API) handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	var req productRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	p, err := a.products.UpdateProduct(r.Context(), middleware.GetTenantID(r.Context()), id, req.toDomain())
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusOK, productToResponse(p))
}

func (a *API) handleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	if err := a.products.DeleteProduct(r.Context(), middleware.GetTenantID(r.Context()), id); err != nil {
		response.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
