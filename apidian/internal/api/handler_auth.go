package api

import (
	"encoding/json"
	"net/http"

	"github.com/diegofxm/apidian/internal/auth"
	"github.com/google/uuid"

	"github.com/diegofxm/apidian/internal/api/response"
)

// registerRequest son los datos del usuario nuevo — desde la Fase 9.32 ya NO incluye una
// empresa: el registro queda desacoplado de crear/vincularse a una empresa (ver
// docs/apidian-architecture.md sección 9.32). Para crear la primera empresa, ver
// POST /api/v1/issuers en handler_issuers.go.
type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// authResponse es la respuesta de /auth/register, /auth/login, POST /issuers y
// POST /issuers/{id}/select: el token de sesión, el usuario, y la empresa ACTIVA en ese token
// — Issuer es nil mientras el usuario no tenga ninguna empresa vinculada, o tenga varias sin
// haber seleccionado cuál usar (ver auth.Service.Login).
type authResponse struct {
	Token  string          `json:"token"`
	User   userResponse    `json:"user"`
	Issuer *issuerResponse `json:"issuer,omitempty"`
}

type userResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
	Role  string    `json:"role"`
}

func userToResponse(u *auth.User) userResponse {
	return userResponse{ID: u.ID, Email: u.Email, Name: u.Name, Role: u.Role}
}

func (a *API) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	result, err := a.auth.Register(r.Context(), auth.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		response.WriteError(w, err)
		return
	}

	writeAuthResponse(w, http.StatusCreated, result)
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

	writeAuthResponse(w, http.StatusOK, result)
}

// writeAuthResponse arma la respuesta a partir de un auth.AuthResult ya resuelto — no vuelve a
// consultar el emisor (a.auth ya lo trae en ActiveIssuer, si hay uno activo), a diferencia de
// como funcionaba antes de la Fase 9.32 (un usuario siempre tenía exactamente un emisor fijo).
func writeAuthResponse(w http.ResponseWriter, status int, result *auth.AuthResult) {
	resp := authResponse{
		Token: result.Token,
		User:  userToResponse(&result.User),
	}
	if result.ActiveIssuer != nil {
		iss := issuerToResponse(result.ActiveIssuer)
		resp.Issuer = &iss
	}
	response.WriteJSON(w, status, resp)
}
