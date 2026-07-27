package catalogs

// Entry es la forma compartida por la mayoría de catálogos DIAN/DANE: código, nombre,
// descripción. Se devuelve directamente como JSON en los endpoints de solo lectura.
type Entry struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Municipality agrega DepartmentCode que ningún otro catálogo tiene.
type Municipality struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	DepartmentCode string `json:"department_code"`
	Description    string `json:"description"`
}

// Currency tiene Symbol en lugar de Description.
type Currency struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

// ItemStandard es la tabla 13.3.5 del Anexo Técnico DIAN — 4 filas fijas
// (UNSPSC/GTIN/Partida Arancelaria/estándar propio). AgencyID vacío ("") en la fila 999
// significa que @schemeAgencyID no debe escribirse en el XML.
type ItemStandard struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	AgencyID    string `json:"agency_id,omitempty"`
	Description string `json:"description"`
}

// CiiuCode es una actividad económica del catálogo CIIU (revisión DANE).
type CiiuCode struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}
