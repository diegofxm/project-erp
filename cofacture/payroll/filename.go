package payroll

import "fmt"

// XMLFileName construye el nombre del archivo XML de una NominaIndividual:
//
//	nie{NIT, 10 dígitos, relleno izquierda}{año, 2 dígitos}{consecutivo, 8 hex}.xml
//
// Sección 3.3 del Anexo Técnico de Nómina Electrónica. El consecutivo se reinicia
// el 1 de enero de cada año; llevar la cuenta es responsabilidad del orquestador.
func XMLFileName(nit string, year int, consecutive uint32) string {
	return fmt.Sprintf("nie%010s%02d%08X.xml", nit, year%100, consecutive)
}

// AdjustXMLFileName construye el nombre del archivo XML de una NominaIndividualDeAjuste:
//
//	niae{NIT, 10 dígitos}{año, 2 dígitos}{consecutivo, 8 hex}.xml
func AdjustXMLFileName(nit string, year int, consecutive uint32) string {
	return fmt.Sprintf("niae%010s%02d%08X.xml", nit, year%100, consecutive)
}

// ZIPFileName construye el nombre del ZIP que contiene el documento de nómina:
//
//	z{NIT, 10 dígitos}{año, 2 dígitos}{consecutivo, 8 hex}.zip
//
// Sección 3.5 del Anexo Técnico. El consecutivo es del paquete ZIP enviado,
// distinto del consecutivo de los archivos XML individuales.
func ZIPFileName(nit string, year int, consecutive uint32) string {
	return fmt.Sprintf("z%010s%02d%08X.zip", nit, year%100, consecutive)
}
