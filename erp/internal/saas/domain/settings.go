package domain

import (
	"context"
	"time"
)

// Settings es la fila única de configuración global de la plataforma SaaS — hoy solo la tasa de
// IVA aplicada a todos los servicios (planes, certificados, excedente de documentos), pensada
// para no quedar hardcodeada en ningún cálculo de facturación.
type Settings struct {
	IVARateBP int // puntos básicos: 1900 = 19%
	UpdatedAt time.Time
}

type SettingsRepository interface {
	Get(ctx context.Context) (*Settings, error)
	Update(ctx context.Context, s Settings) (*Settings, error)
}
