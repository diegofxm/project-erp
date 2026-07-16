package vendors

import (
	"time"

	"github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"
)

// Vendor es un tercero no obligado a facturar guardado en el catálogo del emisor. Misma
// estructura que customers.Customer — los roles están invertidos en el Documento Soporte
// (el vendor es AccountingSupplierParty; la empresa emisora es AccountingCustomerParty), pero
// los datos de Party son idénticos.
//
// Editar o borrar un vendor no afecta documentos ya emitidos: cada DS guarda su propio
// snapshot en documents.vendor (JSONB), igual que Invoice/customers.
type Vendor struct {
	ID        uuid.UUID
	IssuerID  uuid.UUID
	Party     domain.Party
	CreatedAt time.Time
	UpdatedAt time.Time
}
