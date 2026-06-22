// Package cufe calcula el Código Único de Factura Electrónica (CUFE) según el Anexo
// Técnico 1.9 de la DIAN (Resolución 000165/2023, sección 11.2).
//
// El CUDE (Notas Crédito/Débito) vive en el paquete cude, no aquí — son fórmulas casi
// idénticas, pero identificadores distintos para documentos distintos; mantenerlos en
// paquetes separados evita que alguien busque "cude" y solo encuentre "cufe".
package cufe

import (
	"crypto/sha512"
	"encoding/hex"

	"github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/internal/dianhash"
)

// Compute calcula el CUFE de una Factura Electrónica de Venta.
//
// technicalKey es la "clave técnica" (ClTec) del rango de numeración autorizado — se obtiene
// del webservice GetNumberingRange de la DIAN y nunca viaja dentro del XML, por lo que no es
// parte de domain.Invoice/domain.NumberingRange. Quien llame a Compute es responsable de
// obtenerla y guardarla de forma segura.
//
// Fórmula (sección 11.2): SHA-384(NumFac+FecFac+HorFac+ValFac+CodImp1+ValImp1+CodImp2+
// ValImp2+CodImp3+ValImp3+ValTot+NitOFE+NumAdq+ClTec+TipoAmbiente), donde CodImp1/2/3 son
// fijos "01"/"04"/"03" (IVA/INC/ICA) y ValImpN es "0.00" si ese impuesto no aplica.
func Compute(inv domain.Invoice, technicalKey string) string {
	sum := sha512.Sum384([]byte(dianhash.Seed(inv, technicalKey)))
	return hex.EncodeToString(sum[:])
}
