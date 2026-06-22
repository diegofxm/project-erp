package customers

import (
	"time"

	"github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"
)

// Customer es un adquiriente guardado para reutilizar al emitir documentos — conveniencia de
// catálogo para que el frontend no tenga que retipear los mismos datos en cada factura.
//
// NUNCA es la fuente de verdad de un documento ya emitido: cada documento sigue guardando su
// propio snapshot en documents.customer (JSONB, ver internal/documents/model.go), tal cual
// llegó en el momento de emitir. Borrar o editar un Customer no afecta documentos ya emitidos
// — es lo mismo que la DIAN exige (snapshot, no referencia viva), aplicado también al lado de
// conveniencia: si no fuera así, this no sería un catálogo opcional sino una dependencia legal.
//
// Party reutiliza el mismo tipo que ya usa internal/documents para el campo Customer de un
// IssueInvoiceRequest — no se duplica la lista de campos en un struct propio.
type Customer struct {
	ID        uuid.UUID
	IssuerID  uuid.UUID
	Party     domain.Party
	CreatedAt time.Time
	UpdatedAt time.Time
}
