package journals

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Códigos de comprobante estándar del sistema.
// La empresa puede definir códigos adicionales con RegisterVoucherType.
const (
	VoucherExpense  = "CE"  // Comprobante de Egreso — pagos a proveedores
	VoucherIncome   = "CI"  // Comprobante de Ingreso — recaudos de clientes
	VoucherNote     = "NC"  // Nota de Contabilidad — ajustes manuales
	VoucherPayroll  = "NI"  // Nómina — asientos de nómina
	VoucherClosing  = "CJ"  // Cierre de Ejercicio — generado por CloseYear
	VoucherOpening  = "AP"  // Apertura de Año — generado por OpenYear
)

// VoucherTypeConfig es la configuración de un tipo de comprobante por empresa.
type VoucherTypeConfig struct {
	ID             uuid.UUID
	CompanyID      uuid.UUID
	Code           string
	Name           string
	ResetsAnnually bool
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// formatVoucherNumber produce el número de comprobante legible.
// Formato: {CÓDIGO}-{AÑO}-{SECUENCIA con 5 dígitos} → CE-2025-00001
func formatVoucherNumber(code string, year, seq int) string {
	return fmt.Sprintf("%s-%d-%05d", code, year, seq)
}
