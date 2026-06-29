package api

import (
	"encoding/json"
	"net/http"

	"github.com/diegofxm/apidian/internal/api/response"
	"github.com/diegofxm/cofacture/domain"
)

// Este archivo implementa el autorregistro de clientes por QR (patrón D1 y similares, ver
// docs/apidian-architecture.md sección 9.41) — las únicas dos rutas de todo apidian, además de
// auth/register y auth/login, que se registran SIN pasar por middleware.Auth (ver api.go). El
// emisor publica el link/QR de GET /public/issuers/{id} en su mostrador; cualquiera que lo
// escanee puede autorregistrarse como adquiriente sin sesión.
//
// Deliberadamente angosto: no reusa partyDTO completo. Un cliente de mostrador llenando esto
// desde el celular no debe tener que elegir departamento/municipio/régimen tributario/
// responsabilidades fiscales — esos campos quedan vacíos y el emisor los completa después
// desde CustomerForm/CustomerSection si los necesita. customers.Service.CreateCustomer ya
// acepta un domain.Party con solo Name+Identification (el resto es opcional), así que no hizo
// falta tocar ese servicio para esto.

// publicIssuerResponse es deliberadamente angosto — nunca reusar issuerResponse aquí, que
// expone has_software_credentials/has_certificate/etc. a cualquiera en internet. HasLogo sigue
// el mismo criterio que issuerResponse.HasLogo: solo presencia, el logo en sí se sirve aparte
// (GET .../logo) para no inflar esta respuesta con bytes de imagen.
type publicIssuerResponse struct {
	BusinessName string `json:"business_name"`
	HasLogo      bool   `json:"has_logo"`
}

// handleGetPublicIssuer sirve el nombre del emisor para el encabezado del formulario público
// ("Regístrate como cliente de {business_name}") — el id del emisor es público a propósito
// (es lo que va impreso en el QR), así que no hay nada que ocultar revelando si existe o no.
func (a *API) handleGetPublicIssuer(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	iss, err := a.issuers.GetIssuer(r.Context(), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	if !iss.IsActive {
		response.WriteJSON(w, http.StatusNotFound, response.Error{Error: "empresa no encontrada"})
		return
	}

	response.WriteJSON(w, http.StatusOK, publicIssuerResponse{BusinessName: iss.BusinessName, HasLogo: len(iss.Logo) > 0})
}

// handleGetPublicIssuerLogo sirve el logo del emisor en crudo, sin sesión — mismo patrón
// binario que handleGetMyIssuerLogo, pero el emisor viene del path (es público), no del token.
func (a *API) handleGetPublicIssuerLogo(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	iss, err := a.issuers.GetIssuer(r.Context(), id)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	if !iss.IsActive || len(iss.Logo) == 0 {
		response.WriteJSON(w, http.StatusNotFound, response.Error{Error: "la empresa no tiene un logo configurado"})
		return
	}

	w.Header().Set("Content-Type", "image/"+iss.LogoContentType)
	_, _ = w.Write(iss.Logo)
}

// publicCustomerRequest es el subconjunto mínimo de partyDTO que tiene sentido pedirle a un
// visitante anónimo desde su celular.
type publicCustomerRequest struct {
	Identification identificationDTO `json:"identification"`
	Name           string            `json:"name"`
	Email          string            `json:"email,omitempty"`
	Phone          string            `json:"phone,omitempty"`
}

type publicCustomerResponse struct {
	Name string `json:"name"`
}

// handleCreatePublicCustomer crea un Customer del emisor dado sin autenticación — el dígito de
// verificación (si Identification.TypeCode es NIT) se deriva dentro de
// customers.Service.CreateCustomer igual que en el flujo autenticado (ver internal/nit).
func (a *API) handleCreatePublicCustomer(w http.ResponseWriter, r *http.Request) {
	issuerID, ok := parseUUID(w, r.PathValue("id"))
	if !ok {
		return
	}

	iss, err := a.issuers.GetIssuer(r.Context(), issuerID)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	if !iss.IsActive {
		response.WriteJSON(w, http.StatusNotFound, response.Error{Error: "empresa no encontrada"})
		return
	}

	var req publicCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteJSON(w, http.StatusBadRequest, response.Error{Error: "JSON inválido"})
		return
	}

	party := domain.Party{
		Identification: domain.Identification{
			Number:   req.Identification.Number,
			TypeCode: req.Identification.TypeCode,
		},
		Name:  req.Name,
		Email: req.Email,
		Phone: req.Phone,
	}

	c, err := a.customers.CreateCustomer(r.Context(), issuerID, party)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.WriteJSON(w, http.StatusCreated, publicCustomerResponse{Name: c.Party.Name})
}
