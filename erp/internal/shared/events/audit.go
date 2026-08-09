package events

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// AuditLogger es la interfaz mínima para registrar en audit.events -- mismo shape que ya usa
// cada módulo en su capa HTTP (ver erp/internal/*/interfaces/http/handlers.go), aquí a nivel de
// aplicación para que el caso de uso que publica un evento pueda dejar constancia si algún
// suscriptor falla, en vez de que el error se pierda en un log.Printf que nadie mira.
type AuditLogger interface {
	Log(ctx context.Context, companyID uuid.UUID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]any)
}

// PublishAndAudit publica e y, si algún suscriptor falla, registra la falla en audit.events bajo
// action (ej. "sale.accounting_posting_failed") -- no la pierde en el log del servidor.
//
// No aborta la operación del llamador: los publicadores actuales (confirmar venta, recibir
// compra, registrar pago, aprobar nómina) ya persistieron el cambio principal ANTES de publicar,
// así que "abortar" implicaría una reversión que hoy no existe. "Advertir" (registrar y seguir)
// es la decisión tomada para los cuatro casos existentes -- ver plan de acción 2026-08-09,
// Fase 2 punto 10. Si audit es nil, equivale al comportamiento de antes (el subscriber ya
// hizo su propio log.Printf).
func PublishAndAudit(ctx context.Context, bus *Bus, e Event, audit AuditLogger, companyID uuid.UUID, action, resourceType string, resourceID uuid.UUID) {
	errs := bus.Publish(ctx, e)
	if len(errs) == 0 || audit == nil {
		return
	}
	audit.Log(ctx, companyID, nil, action, resourceType, &resourceID, map[string]any{
		"error": errors.Join(errs...).Error(),
	})
}
