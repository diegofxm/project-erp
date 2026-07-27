// Package catalog expone los catálogos DIAN/DANE de solo lectura que usan todos los módulos.
package domain

import "context"

// Entry es la estructura genérica de la mayoría de catálogos (código + nombre).
type Entry struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Municipality extiende Entry con la referencia al departamento padre.
type Municipality struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	DepartmentCode string `json:"department_code"`
	Description    string `json:"description,omitempty"`
}

// Currency tiene símbolo además de código y nombre.
type Currency struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
}

// ItemStandard es el estándar de codificación de productos (UNSPSC, GTIN, etc.).
type ItemStandard struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	AgencyID    string `json:"agency_id"`
	Description string `json:"description,omitempty"`
}

// CiiuCode es la clasificación industrial internacional uniforme.
type CiiuCode struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// Repository es el puerto de acceso a los catálogos (solo lectura).
// Implementado en infrastructure/persistence/postgres/repository.go.
type Repository interface {
	// Listados completos
	ListDepartments(ctx context.Context) ([]Entry, error)
	ListMunicipalities(ctx context.Context, departmentCode string) ([]Municipality, error)
	ListIdentificationTypes(ctx context.Context) ([]Entry, error)
	ListTaxTypes(ctx context.Context) ([]Entry, error)
	ListPaymentMethods(ctx context.Context) ([]Entry, error)
	ListPaymentTerms(ctx context.Context) ([]Entry, error)
	ListUnitMeasures(ctx context.Context) ([]Entry, error)
	ListTaxRegimes(ctx context.Context) ([]Entry, error)
	ListLiabilityCodes(ctx context.Context) ([]Entry, error)
	ListDianDocumentTypes(ctx context.Context) ([]Entry, error)
	ListCurrencies(ctx context.Context) ([]Currency, error)
	ListItemStandards(ctx context.Context) ([]ItemStandard, error)
	ListCiiuCodes(ctx context.Context) ([]CiiuCode, error)

	// Validaciones puntuales (usadas por otros módulos vía CatalogPort)
	IsValidLiabilityCode(ctx context.Context, code string) (bool, error)
	IsValidPaymentMethod(ctx context.Context, code string) (bool, error)
	IsValidPaymentTerm(ctx context.Context, code string) (bool, error)

	// Lookups por código
	GetTaxTypeName(ctx context.Context, code string) (string, bool, error)
	GetPaymentTermName(ctx context.Context, code string) (string, bool, error)
	GetPaymentMethodName(ctx context.Context, code string) (string, bool, error)
	GetIdentificationTypeName(ctx context.Context, code string) (string, bool, error)
	GetTaxRegimeName(ctx context.Context, code string) (string, bool, error)
	GetLiabilityCodeName(ctx context.Context, code string) (string, bool, error)
	GetItemStandardName(ctx context.Context, code string) (string, bool, error)
	GetItemStandardAgencyID(ctx context.Context, code string) (string, bool, error)
	GetCiiuDescription(ctx context.Context, code string) (string, bool, error)
}
