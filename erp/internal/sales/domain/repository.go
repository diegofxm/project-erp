package domain

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Save(ctx context.Context, s Sale) (*Sale, error)
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*Sale, error)
	List(ctx context.Context, companyID uuid.UUID) ([]Sale, error)
	UpdateStatus(ctx context.Context, companyID, id uuid.UUID, status SaleStatus) error
	Delete(ctx context.Context, companyID, id uuid.UUID) error
	// SetInvoiceDocumentID registra qué factura electrónica se generó a partir de esta venta —
	// evita generar dos veces desde la misma venta (ver electronic CreateFromSaleUseCase).
	SetInvoiceDocumentID(ctx context.Context, companyID, id, documentID uuid.UUID) error
	// NextSaleNumber devuelve el siguiente consecutivo de venta para la empresa y el año dados
	// (arranca en 1 cada año). Mismo patrón que accounting.JournalRepository.NextVoucherSeq.
	NextSaleNumber(ctx context.Context, companyID uuid.UUID, year int) (int, error)
	// SetSaleNumberCounter fija el consecutivo para que el próximo emitido sea nextNumber — solo
	// tiene efecto si nextNumber es mayor al último ya asignado (devuelve ErrNumberCounterBackwards
	// si no), para evitar duplicar números ya emitidos. Pensado para migrar una empresa que ya
	// traía su propia numeración de otro sistema.
	SetSaleNumberCounter(ctx context.Context, companyID uuid.UUID, year, nextNumber int) (int, error)
}
