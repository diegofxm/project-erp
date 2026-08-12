package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AccountRepository interface {
	GetByCode(ctx context.Context, code string) (*Account, error)
	GetPostable(ctx context.Context, code string) (*Account, error)
	List(ctx context.Context) ([]Account, error)
}

type PeriodRepository interface {
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*AccountingPeriod, error)
	GetByYearMonth(ctx context.Context, companyID uuid.UUID, year, month int) (*AccountingPeriod, error)
	Create(ctx context.Context, p AccountingPeriod) (*AccountingPeriod, error)
	Close(ctx context.Context, companyID, id uuid.UUID) error
	// Reopen revierte un cierre — uso excepcional (ej. se cerró por error); no valida si ya se
	// generaron asientos de cierre/apertura sobre el período, eso queda a criterio de quien reabre.
	Reopen(ctx context.Context, companyID, id uuid.UUID) error
	CloseAllForYear(ctx context.Context, companyID uuid.UUID, year int) error
	List(ctx context.Context, companyID uuid.UUID) ([]AccountingPeriod, error)
}

type JournalRepository interface {
	Create(ctx context.Context, entry JournalEntry) (*JournalEntry, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*JournalEntry, error)
	Void(ctx context.Context, companyID, id uuid.UUID) error
	ListByCompany(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]*JournalEntry, error)
	GetBySourceDocument(ctx context.Context, companyID, sourceDocID uuid.UUID, sourceDocType string) ([]*JournalEntry, error)
	NextVoucherSeq(ctx context.Context, companyID uuid.UUID, code string, year int) (int, error)
	// SetVoucherCounter fija el consecutivo para que el próximo comprobante emitido de ese código
	// sea nextNumber — solo tiene efecto si nextNumber es mayor al último ya asignado (devuelve
	// ErrNumberCounterBackwards si no). Pensado para migrar una empresa que ya traía su propia
	// numeración de otro sistema.
	SetVoucherCounter(ctx context.Context, companyID uuid.UUID, code string, year, nextNumber int) (int, error)
	GetYearPLBalances(ctx context.Context, companyID uuid.UUID, year int) ([]PLBalance, error)
	GetBSBalances(ctx context.Context, companyID uuid.UUID, asOf time.Time) ([]PLBalance, error)
	GetTrialBalance(ctx context.Context, companyID uuid.UUID, from, to time.Time) ([]TrialBalanceRow, error)
	// GetIncomeInPeriod suma el movimiento (crédito-débito) de las cuentas categoría "Ingreso"
	// en un rango de fechas — base gravable bruta para la declaración de ICA.
	GetIncomeInPeriod(ctx context.Context, companyID uuid.UUID, from, to time.Time) (int64, error)
	GetAccountLedger(ctx context.Context, companyID uuid.UUID, accountCode string, from, to time.Time) ([]LedgerLine, error)
	RegisterVoucherType(ctx context.Context, cfg VoucherTypeConfig) (*VoucherTypeConfig, error)
	ListVoucherTypes(ctx context.Context, companyID uuid.UUID) ([]*VoucherTypeConfig, error)
	// GetVoucherType lee un tipo de comprobante puntual (incluso inactivo, a diferencia de
	// ListVoucherTypes) -- necesario para Deactivate, que reusa RegisterVoucherType (upsert) sin
	// perder el name/resets_annually ya guardados.
	GetVoucherType(ctx context.Context, companyID uuid.UUID, code string) (*VoucherTypeConfig, error)
	// IsRegisteredVoucherType indica si code está registrado y activo para la empresa — usado
	// por PostJournalUseCase para validar tipos personalizados (los estándar del sistema no
	// necesitan estar aquí, ver domain.IsStandardVoucherType).
	IsRegisteredVoucherType(ctx context.Context, companyID uuid.UUID, code string) (bool, error)
}
