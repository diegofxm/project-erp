package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type ProspectStatus string

const (
	ProspectPending  ProspectStatus = "pending"
	ProspectApproved ProspectStatus = "approved"
	ProspectRejected ProspectStatus = "rejected"
)

// Prospect es una solicitud de acceso a la plataforma hecha por alguien que todavía no tiene
// cuenta — sube cédula y RUT, un superadmin la revisa y aprueba/rechaza (ver
// _legacy/apidian/internal/prospects, flujo ya validado que se porta tal cual).
type Prospect struct {
	ID    uuid.UUID
	Name  string
	Email string
	NIT   string

	CedulaFile        []byte
	CedulaContentType string
	RUTFile           []byte
	RUTContentType    string

	Status     ProspectStatus
	Notes      string
	ReviewedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

type ProspectRepository interface {
	Create(ctx context.Context, p Prospect) (*Prospect, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Prospect, error)
	List(ctx context.Context) ([]Prospect, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status ProspectStatus, notes string) (*Prospect, error)
}

var (
	ErrProspectNotFound   = errors.New("solicitud no encontrada")
	ErrProspectNotPending = errors.New("la solicitud ya fue revisada")
	ErrEmailTaken         = errors.New("ya existe una solicitud con ese correo")
)
