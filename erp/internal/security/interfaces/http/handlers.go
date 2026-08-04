package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/security/application"
	"github.com/diegofxm/erp/internal/security/domain"
	"github.com/diegofxm/erp/internal/shared/tenant"
)

type Handler struct {
	register      *application.RegisterUseCase
	login         *application.LoginUseCase
	selectCompany *application.SelectCompanyUseCase
	inviteUser    *application.InviteUserUseCase
	acceptInvite  *application.AcceptInviteUseCase
	updateProfile *application.UpdateProfileUseCase
	getProfile    *application.GetProfileUseCase
}

func NewHandler(
	register *application.RegisterUseCase,
	login *application.LoginUseCase,
	selectCompany *application.SelectCompanyUseCase,
	inviteUser *application.InviteUserUseCase,
	acceptInvite *application.AcceptInviteUseCase,
	updateProfile *application.UpdateProfileUseCase,
	getProfile *application.GetProfileUseCase,
) *Handler {
	return &Handler{
		register:      register,
		login:         login,
		selectCompany: selectCompany,
		inviteUser:    inviteUser,
		acceptInvite:  acceptInvite,
		updateProfile: updateProfile,
		getProfile:    getProfile,
	}
}

// --- rutas públicas ---

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req application.RegisterRequest
	if !decode(w, r, &req) {
		return
	}
	result, err := h.register.Execute(r.Context(), req)
	respondAuth(w, result, err)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &body) {
		return
	}
	result, err := h.login.Execute(r.Context(), body.Email, body.Password)
	respondAuth(w, result, err)
}

func (h *Handler) handleAcceptInvite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if !decode(w, r, &body) {
		return
	}
	result, err := h.acceptInvite.Execute(r.Context(), body.Token, body.Password)
	respondAuth(w, result, err)
}

// --- rutas protegidas ---

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuth(w, r)
	if !ok {
		return
	}
	u, err := h.getProfile.Execute(r.Context(), userID)
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, safeUser(u), nil)
}

func (h *Handler) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuth(w, r)
	if !ok {
		return
	}
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if !decode(w, r, &body) {
		return
	}
	u, err := h.updateProfile.Execute(r.Context(), userID, body.Name, body.Email)
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, safeUser(u), nil)
}

func (h *Handler) handleSelectCompany(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuth(w, r)
	if !ok {
		return
	}
	var body struct {
		CompanyID uuid.UUID `json:"company_id"`
	}
	if !decode(w, r, &body) {
		return
	}
	result, err := h.selectCompany.Execute(r.Context(), userID, body.CompanyID)
	respondAuth(w, result, err)
}

func (h *Handler) handleInvite(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireManage(w, r)
	if !ok {
		return
	}
	var body struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Role  string `json:"role"`
	}
	if !decode(w, r, &body) {
		return
	}
	role := domain.Role(body.Role)
	if role != domain.RoleAdmin {
		role = domain.RoleMember // default — el owner solo existe por creación de empresa
	}
	u, err := h.inviteUser.Execute(r.Context(), companyID, body.Email, body.Name, role)
	respond(w, u, err)
}

func (h *Handler) handleListCompanies(w http.ResponseWriter, r *http.Request) {
	// Este handler simplemente devuelve los company IDs del token actual.
	// Cuando company/ esté implementado devolverá los objetos completos.
	userID, ok := requireAuth(w, r)
	if !ok {
		return
	}
	u, err := h.getProfile.Execute(r.Context(), userID)
	if err != nil {
		respondError(w, err)
		return
	}
	// Reutilizar la info del token: la lista real viene del repo a través del perfil.
	// Por ahora devolvemos user_id como confirmación; company/ ampliará esto.
	respond(w, map[string]any{"user_id": u.ID, "name": u.Name}, nil)
}

// --- helpers ---

func requireAuth(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id := tenant.GetUserID(r.Context())
	if id == uuid.Nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"no autenticado"}`))
		return uuid.Nil, false
	}
	return id, true
}

// requireManage exige, además de sesión con empresa activa, que el rol sea owner/admin — usado
// por operaciones administrativas (invitar usuarios, credenciales de empresa, etc.).
func requireManage(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	cid := tenant.GetCompanyID(r.Context())
	if cid == uuid.Nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"empresa activa requerida"}`))
		return uuid.Nil, false
	}
	if !tenant.CanManage(r.Context()) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"requiere rol de administrador"}`))
		return uuid.Nil, false
	}
	return cid, true
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		http.Error(w, `{"error":"cuerpo inválido"}`, http.StatusBadRequest)
		return false
	}
	return true
}

func respond(w http.ResponseWriter, data any, err error) {
	if err != nil {
		respondError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// safeUser convierte un User del dominio en un mapa seguro para la respuesta HTTP (sin campos sensibles).
func safeUser(u *domain.User) map[string]any {
	return map[string]any{
		"id":         u.ID,
		"email":      u.Email,
		"name":       u.Name,
		"role":       u.Role,
		"is_active":  u.IsActive,
		"created_at": u.CreatedAt,
		"updated_at": u.UpdatedAt,
	}
}

func respondAuth(w http.ResponseWriter, result *domain.AuthResult, err error) {
	if err != nil {
		respondError(w, err)
		return
	}
	var companyID any
	if result.CompanyID != uuid.Nil {
		companyID = result.CompanyID
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token":      result.Token,
		"company_id": companyID,
		"user": map[string]any{
			"id":    result.User.ID,
			"email": result.User.Email,
			"name":  result.User.Name,
			"role":  result.User.Role,
		},
	})
}

func respondError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	if errors.Is(err, domain.ErrInvalidPassword) ||
		errors.Is(err, domain.ErrInvalidToken) ||
		errors.Is(err, domain.ErrUserInactive) {
		code = http.StatusUnauthorized
	} else if errors.Is(err, domain.ErrEmailTaken) {
		code = http.StatusConflict
	} else if errors.Is(err, domain.ErrUserNotFound) ||
		errors.Is(err, domain.ErrNotAMember) {
		code = http.StatusNotFound
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
