// Package catalogs expone, de solo lectura, los catálogos de referencia DIAN/DANE que ya
// vive en Postgres (departments, municipalities, identification_types, tax_types,
// payment_methods, unit_measures, dian_document_types, payment_terms, tax_regimes,
// liability_codes). A diferencia de issuers/customers/products/documents, no hay Service:
// no hay lógica de negocio que orquestar, solo lecturas — el Repository es el contrato
// completo, y también implementa documents.CatalogPort (ver ports.go en internal/documents).
package catalogs

// Entry es la forma compartida por la mayoría de catálogos: código, nombre, descripción.
type Entry struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Municipality tiene una columna adicional (DepartmentCode) que ningún otro catálogo tiene.
type Municipality struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	DepartmentCode string `json:"department_code"`
	Description    string `json:"description"`
}

// Currency no tiene "description" — tiene "symbol" en su lugar.
type Currency struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}
