package cartera

import (
	"time"

	"github.com/google/uuid"
)

// AgingBucket define un rango de antigüedad en días.
// DaysMax = -1 significa sin límite superior.
type AgingBucket struct {
	Label   string
	DaysMin int
	DaysMax int
}

// StandardBuckets son los rangos de antigüedad estándar usados en Colombia
// para el análisis de cartera y el cálculo de provisiones (Art. 145 ET).
var StandardBuckets = []AgingBucket{
	{Label: "Corriente", DaysMin: 0, DaysMax: 30},
	{Label: "31-60 días", DaysMin: 31, DaysMax: 60},
	{Label: "61-90 días", DaysMin: 61, DaysMax: 90},
	{Label: "91-180 días", DaysMin: 91, DaysMax: 180},
	{Label: "181-360 días", DaysMin: 181, DaysMax: 360},
	{Label: "Más de 360 días", DaysMin: 361, DaysMax: -1},
}

// DefaultAccountPrefixes son los prefijos PUC que representan cartera de clientes.
var DefaultAccountPrefixes = []string{"1305", "1310"}

// Movement es un movimiento (débito o crédito) de una línea de asiento contable
// en una cuenta de cartera. Incluye si la línea ya fue conciliada.
type Movement struct {
	LineID        uuid.UUID
	JournalID     uuid.UUID
	Date          time.Time
	Description   string
	ThirdPartyNIT string
	Debit         int64
	Credit        int64
	Reconciled    bool
}

// OpenItem es un cargo (débito) que tiene saldo pendiente después de aplicar
// los pagos disponibles en orden FIFO.
type OpenItem struct {
	LineID      uuid.UUID
	JournalID   uuid.UUID
	Date        time.Time
	Description string
	Original    int64  // monto original del cargo
	Remaining   int64  // saldo sin aplicar
	DaysOld     int
	BucketLabel string
}

// CustomerAging resume la cartera de un cliente por rangos de antigüedad.
type CustomerAging struct {
	ThirdPartyNIT string
	Buckets       map[string]int64 // label → monto pendiente en centavos
	Total         int64
	OpenItems     []*OpenItem // detalle de los cargos abiertos
}

// AgingReport es el reporte completo de antigüedad de cartera a una fecha de corte.
type AgingReport struct {
	AsOf           time.Time
	AccountPrefixes []string
	Customers      []*CustomerAging
	Totals         map[string]int64
	GrandTotal     int64
}

// ReconciliationMark empareja una línea de débito (factura) con una de crédito (pago).
type ReconciliationMark struct {
	ID             uuid.UUID
	CompanyID      uuid.UUID
	JournalLineID  uuid.UUID  // la línea marcada (normalmente un débito = factura)
	ReconciledWith *uuid.UUID // la línea de pago emparejada (crédito)
	Note           string
	ReconciledAt   time.Time
}

// StatementLine es una fila del extracto de cuenta de un cliente.
type StatementLine struct {
	LineID      uuid.UUID
	JournalID   uuid.UUID
	Date        time.Time
	Description string
	Debit       int64
	Credit      int64
	RunningBal  int64
	Reconciled  bool
}

// CustomerStatement es el extracto de movimientos de un cliente para un rango de fechas.
type CustomerStatement struct {
	ThirdPartyNIT string
	From          time.Time
	To            time.Time
	Lines         []*StatementLine
	OpenBalance   int64 // saldo pendiente al final del período
}
