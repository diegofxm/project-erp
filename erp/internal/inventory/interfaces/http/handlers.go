package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/inventory/application"
	"github.com/diegofxm/erp/internal/inventory/domain"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

// AuditLogger es la interfaz mínima para registrar eventos de auditoría.
// La implementa audit/application.UseCase sin requerir importar ese paquete.
type AuditLogger interface {
	Log(ctx context.Context, companyID uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]any)
}

// ── DTOs de salida ──────────────────────────────────────────────────────────────────────────
// domain.StockEntry/Movement no llevan tags json (no deben — el dominio no conoce HTTP) — acá
// se mapean a snake_case, igual que sales/purchase (mismo bug real que se encontró ahí: antes
// este handler mandaba el struct de dominio crudo, con campos en PascalCase).

type stockEntryDTO struct {
	ID          uuid.UUID `json:"id"`
	CompanyID   uuid.UUID `json:"company_id"`
	ProductID   uuid.UUID `json:"product_id"`
	WarehouseID uuid.UUID `json:"warehouse_id"`
	Quantity    float64   `json:"quantity"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toStockEntryDTO(e domain.StockEntry) stockEntryDTO {
	return stockEntryDTO{
		ID: e.ID, CompanyID: e.CompanyID, ProductID: e.ProductID,
		WarehouseID: e.WarehouseID, Quantity: e.Quantity, UpdatedAt: e.UpdatedAt,
	}
}

type movementDTO struct {
	ID              uuid.UUID  `json:"id"`
	CompanyID       uuid.UUID  `json:"company_id"`
	Number          string     `json:"number"`
	ProductID       uuid.UUID  `json:"product_id"`
	WarehouseID     uuid.UUID  `json:"warehouse_id"`
	Type            string     `json:"type"`
	Quantity        float64    `json:"quantity"`
	Reference       string     `json:"reference,omitempty"`
	Description     string     `json:"description,omitempty"`
	TransferGroupID *uuid.UUID `json:"transfer_group_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

func toMovementDTO(m domain.Movement) movementDTO {
	return movementDTO{
		ID: m.ID, CompanyID: m.CompanyID, Number: m.Number, ProductID: m.ProductID, WarehouseID: m.WarehouseID,
		Type: string(m.Type), Quantity: m.Quantity, Reference: m.Reference, Description: m.Description,
		TransferGroupID: m.TransferGroupID, CreatedAt: m.CreatedAt,
	}
}

type Handler struct {
	move  *application.MoveUseCase
	get   *application.GetUseCase
	audit AuditLogger
}

func NewHandler(move *application.MoveUseCase, get *application.GetUseCase, audit AuditLogger) *Handler {
	return &Handler{move: move, get: get, audit: audit}
}

func (h *Handler) handleMove(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var req application.MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	m, err := h.move.Execute(r.Context(), companyID, req)
	if err != nil {
		if errors.Is(err, domain.ErrInsufficientStock) {
			respondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if h.audit != nil {
		uid := tenant.GetUserID(r.Context())
		var userID *uuid.UUID
		if uid != uuid.Nil {
			userID = &uid
		}
		h.audit.Log(r.Context(), companyID, userID, "inventory.movement_created", "movement", &m.ID, map[string]any{
			"type": string(m.Type), "quantity": m.Quantity,
		})
	}
	respond(w, http.StatusCreated, toMovementDTO(*m))
}

func (h *Handler) handleListStock(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	list, err := h.get.Stock(r.Context(), companyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	dtos := make([]stockEntryDTO, len(list))
	for i, e := range list {
		dtos[i] = toStockEntryDTO(e)
	}
	respond(w, http.StatusOK, dtos)
}

func (h *Handler) handleListMovements(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireTenant(w, r)
	if !ok {
		return
	}
	var productID *uuid.UUID
	if raw := r.URL.Query().Get("product_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			respondError(w, http.StatusBadRequest, "product_id inválido")
			return
		}
		productID = &id
	}
	list, err := h.get.Movements(r.Context(), companyID, productID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	dtos := make([]movementDTO, len(list))
	for i, m := range list {
		dtos[i] = toMovementDTO(m)
	}
	respond(w, http.StatusOK, dtos)
}

func requireTenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	cid := tenant.GetCompanyID(r.Context())
	if cid == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "empresa activa requerida")
		return uuid.Nil, false
	}
	return cid, true
}

func respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}
