package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/diegofxm/api-dian/internal/auth"
	"github.com/google/uuid"

	"github.com/diegofxm/api-dian/internal/api/response"
)

// registerRequest combina los datos del emisor nuevo (mismo contrato que antes tenía
// POST /api/v1/issuers, ya retirado) con los del primer usuario admin que lo administra —
// "un usuario = un emisor", se crean juntos en una sola llamada (ver
// docs/api-dian-architecture.md sección 9.17).
type registerRequest struct {
	Issuer   createIssuerRequest `json:"issuer"`
	Email    string              `json:"email"`
	Password string              `json:"password"`
	Name     string              `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// authResponse es la respuesta de /auth/register y /auth/login: el token de sesión, el
// usuario y el emisor que administra — para que el frontend no tenga que encadenar un GET
// /issuers/me inmediatamente después.
type authResponse struct {
	Token  string         `json:"token"`
	User   userResponse   `json:"user"`
	Issuer issuerResponse `json:"issuer"`
}

type userResponse struct {
	ID       uuid.UUID `json:"id"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Role     string    `json:"role"`
	IssuerID uuid.UUID `json:"issuer_id"`
}

func userToResponse(u *auth.User) userResponse {
	return userResponse{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role, IssuerID: u.IssuerID}
}

func (a *API) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	cert, err := base64.StdEncoding.DecodeString(req.Issuer.CertificateBase64)
	if err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "certificate_base64 no es base64 válido"})
		return
	}

	result, err := a.auth.Register(r.Context(), auth.RegisterRequest{
		Issuer:   issuerFromRequest(req.Issuer, cert),
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		response.WriteError(w, err)
		return
	}

	a.writeAuthResponse(w, r, http.StatusCreated, result)
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	result, err := a.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	a.writeAuthResponse(w, r, http.StatusOK, result)
}

func (a *API) writeAuthResponse(w http.ResponseWriter, r *http.Request, status int, result *auth.AuthResult) {
	iss, err := a.issuers.GetIssuer(r.Context(), result.User.IssuerID)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, status, authResponse{
		Token:  result.Token,
		User:   userToResponse(&result.User),
		Issuer: issuerToResponse(iss),
	})
}
