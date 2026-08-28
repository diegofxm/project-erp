package payroll

import (
	"crypto/sha512"
	"fmt"
)

// Cune computes the Unique Electronic Payroll Code (Código Único de Nómina Electrónica, CUNE)
// per section 8.1 of DIAN's Technical Annex:
//
//	SHA-384(NumNE + FecNE + HorNE + ValDev + ValDed + ValTolNE + NitNE + DocEmp + TipoXML + SoftwarePIN + TipAmb)
//
// All monetary values must be formatted with two decimals before calling this function (e.g.
// "3500000.00", not "3500000"). The result is the 96-character hex SHA-384 that goes into
// InformacionGeneral/@CUNE.
func Cune(numNE, fecNE, horNE, valDev, valDed, valTolNE, nitNE, docEmp, tipoXML, softwarePIN, tipAmb string) string {
	input := numNE + fecNE + horNE + valDev + valDed + valTolNE + nitNE + docEmp + tipoXML + softwarePIN + tipAmb
	sum := sha512.Sum384([]byte(input))
	return fmt.Sprintf("%x", sum)
}
