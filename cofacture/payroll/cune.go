package payroll

import (
	"crypto/sha512"
	"fmt"
)

// Cune calcula el Código Único de Nómina Electrónica (CUNE) según la sección 8.1
// del Anexo Técnico DIAN:
//
//	SHA-384(NumNE + FecNE + HorNE + ValDev + ValDed + ValTolNE + NitNE + DocEmp + TipoXML + SoftwarePIN + TipAmb)
//
// Todos los valores monetarios deben formatearse con dos decimales antes de llamar
// esta función (e.g. "3500000.00", no "3500000"). El resultado es el hex SHA-384
// de 96 caracteres que va en InformacionGeneral/@CUNE.
func Cune(numNE, fecNE, horNE, valDev, valDed, valTolNE, nitNE, docEmp, tipoXML, softwarePIN, tipAmb string) string {
	input := numNE + fecNE + horNE + valDev + valDed + valTolNE + nitNE + docEmp + tipoXML + softwarePIN + tipAmb
	sum := sha512.Sum384([]byte(input))
	return fmt.Sprintf("%x", sum)
}
