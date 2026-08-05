package domain

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// Module es un grupo de funcionalidad que un Plan puede desbloquear — pensado para poder
// prender/apagar un grupo completo del sidebar según el plan que contrate cada empresa (ver
// docs/Diseno_ERP_Go_Arquitectura_Hexagonal.md, sección de jerarquía del sidebar). El catálogo es
// fijo (sembrado por seed), no CRUD de superadmin: agregar un módulo real siempre implica un
// cambio de código de todos modos.
type Module struct {
	ID          uuid.UUID
	Code        string // "electronic_invoicing" | "erp_core" | "payroll_hr"
	Name        string
	Description string
}

type ModuleRepository interface {
	List(ctx context.Context) ([]Module, error)
	GetByCode(ctx context.Context, code string) (*Module, error)
}

var ErrModuleNotFound = errors.New("módulo no encontrado")
