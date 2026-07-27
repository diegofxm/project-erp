package iva

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository abstrae el acceso a datos de las declaraciones de IVA.
type Repository interface {
	// QueryIVAMovements devuelve los movimientos de cuentas 2408xx en el período,
	// excluyendo los asientos de pago de declaraciones previas.
	QueryIVAMovements(ctx context.Context, companyID uuid.UUID, from, to time.Time) ([]*IVALine, error)

	// QueryReteIVA devuelve los débitos en cuentas 1365xx (reteiva a favor) en el período.
	QueryReteIVA(ctx context.Context, companyID uuid.UUID, from, to time.Time) ([]*ReteIVALine, error)

	// Save persiste una declaración (nueva o actualización de DRAFT).
	Save(ctx context.Context, decl IVADeclaration) (*IVADeclaration, error)

	// GetByID devuelve una declaración por UUID.
	GetByID(ctx context.Context, id uuid.UUID) (*IVADeclaration, error)

	// ListByCompany devuelve todas las declaraciones de una empresa, orden desc por period_start.
	ListByCompany(ctx context.Context, companyID uuid.UUID) ([]*IVADeclaration, error)

	// UpdateStatus actualiza el estado y campos opcionales (filed_at, journal_id).
	UpdateStatus(ctx context.Context, id uuid.UUID, status DeclarationStatus, journalID *uuid.UUID, filedAt *time.Time) error
}
