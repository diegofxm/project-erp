package payments

import (
	"time"

	"github.com/google/uuid"
)

type PaymentType string

const (
	TypeAffiliation PaymentType = "affiliation"
	TypeRenewal     PaymentType = "renewal"
	TypeDocuments   PaymentType = "documents"
)

type Payment struct {
	ID        uuid.UUID
	IssuerID  uuid.UUID
	Type      PaymentType
	AmountCOP int
	Note      string
	PaidAt    time.Time
	CreatedAt time.Time
}
