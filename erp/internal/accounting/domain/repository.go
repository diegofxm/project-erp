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
	GetByID(ctx context.Context, id uuid.UUID) (*AccountingPeriod, error)
	GetByYearMonth(ctx context.Context, companyID uuid.UUID, year, month int) (*AccountingPeriod, error)
	Create(ctx context.Context, p AccountingPeriod) (*AccountingPeriod, error)
	Close(ctx context.Context, id uuid.UUID) error
	CloseAllForYear(ctx context.Context, companyID uuid.UUID, year int) error
	List(ctx context.Context, companyID uuid.UUID) ([]AccountingPeriod, error)
}

type JournalRepository interface {
	Create(ctx context.Context, entry JournalEntry) (*JournalEntry, error)
	GetByID(ctx context.Context, id uuid.UUID) (*JournalEntry, error)
	Void(ctx context.Context, id uuid.UUID) error
	ListByCompany(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]*JournalEntry, error)
	GetBySourceDocument(ctx context.Context, companyID, sourceDocID uuid.UUID, sourceDocType string) ([]*JournalEntry, error)
	NextVoucherSeq(ctx context.Context, companyID uuid.UUID, code string, year int) (int, error)
	GetYearPLBalances(ctx context.Context, companyID uuid.UUID, year int) ([]PLBalance, error)
	GetBSBalances(ctx context.Context, companyID uuid.UUID, asOf time.Time) ([]PLBalance, error)
	RegisterVoucherType(ctx context.Context, cfg VoucherTypeConfig) (*VoucherTypeConfig, error)
	ListVoucherTypes(ctx context.Context, companyID uuid.UUID) ([]*VoucherTypeConfig, error)
}
