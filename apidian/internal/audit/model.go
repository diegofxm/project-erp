package audit

import (
	"time"

	"github.com/google/uuid"
)

// Acciones registradas en audit_events.
const (
	ActionDocumentCreated   = "document.created"
	ActionDocumentUpdated   = "document.updated"
	ActionDocumentConfirmed = "document.confirmed"
	ActionDocumentDeleted   = "document.deleted"
	ActionDocumentEmailSent = "document.email_sent"
	ActionDocumentCloned    = "document.cloned"
)

// Event es un registro de auditoría de una acción realizada por un usuario sobre un recurso.
type Event struct {
	ID           uuid.UUID
	IssuerID     uuid.UUID
	UserID       *uuid.UUID
	UserName     string // del JOIN con users — vacío si el usuario fue eliminado
	UserEmail    string // del JOIN con users
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	Metadata     map[string]any
	CreatedAt    time.Time
}

// ListFilter filtra eventos para GET /audit-events.
type ListFilter struct {
	ResourceID *uuid.UUID
	Limit      int
	Offset     int
}
