package catalogs

import "context"

// MemoryRepository implementa Repository en memoria para tests.
// Precargada con un subconjunto mínimo pero real de cada catálogo.
type MemoryRepository struct {
	departments         []Entry
	municipalities      []Municipality
	identificationTypes []Entry
	taxTypes            []Entry
	paymentMethods      []Entry
	paymentTerms        []Entry
	unitMeasures        []Entry
	taxRegimes          []Entry
	liabilityCodes      []Entry
	dianDocumentTypes   []Entry
	currencies          []Currency
	itemStandards       []ItemStandard
	ciiuCodes           []CiiuCode
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		departments:         []Entry{{Code: "11", Name: "Bogotá D.C.", Description: "Distrito Capital de Bogotá"}},
		municipalities:      []Municipality{{Code: "11001", Name: "Bogotá D.C.", DepartmentCode: "11", Description: "Bogotá Distrito Capital"}},
		identificationTypes: []Entry{{Code: "13", Name: "Cédula de Ciudadanía"}, {Code: "31", Name: "NIT"}},
		taxTypes:            []Entry{{Code: "01", Name: "IVA"}, {Code: "ZZ", Name: "No aplica"}},
		paymentMethods:      []Entry{{Code: "10", Name: "Efectivo"}, {Code: "47", Name: "Transferencia Débito Bancaria"}},
		paymentTerms:        []Entry{{Code: "1", Name: "Contado"}, {Code: "2", Name: "Crédito"}},
		unitMeasures:        []Entry{{Code: "94", Name: "Unidad de servicio"}},
		taxRegimes:          []Entry{{Code: "00", Name: "Régimen 00"}},
		liabilityCodes:      []Entry{{Code: "R-99-PN", Name: "No aplica (persona natural)"}},
		dianDocumentTypes:   []Entry{{Code: "01", Name: "Factura electrónica de Venta"}},
		currencies:          []Currency{{Code: "COP", Name: "Peso Colombiano", Symbol: "$"}},
		itemStandards: []ItemStandard{
			{Code: "001", Name: "UNSPSC", AgencyID: "10", Description: "Código Estándar de Productos y Servicios de Naciones Unidas"},
			{Code: "010", Name: "GTIN", AgencyID: "9", Description: "Números Globales de Identificación de Productos"},
			{Code: "020", Name: "Partida Arancelaria", AgencyID: "195", Description: "Partida arancelaria según estatuto tributario"},
			{Code: "999", Name: "Estándar de adopción del contribuyente", AgencyID: "", Description: "Código propio del contribuyente, sin estándar externo"},
		},
		ciiuCodes: []CiiuCode{
			{Code: "6201", Description: "Desarrollo de sistemas informáticos (planificación, análisis, diseño, programación, pruebas)"},
			{Code: "6202", Description: "Consultoría informática y gerencia de instalaciones informáticas"},
		},
	}
}

func (r *MemoryRepository) ListDepartments(_ context.Context) ([]Entry, error) {
	return r.departments, nil
}
func (r *MemoryRepository) ListMunicipalities(_ context.Context, departmentCode string) ([]Municipality, error) {
	if departmentCode == "" {
		return r.municipalities, nil
	}
	var out []Municipality
	for _, m := range r.municipalities {
		if m.DepartmentCode == departmentCode {
			out = append(out, m)
		}
	}
	return out, nil
}
func (r *MemoryRepository) ListIdentificationTypes(_ context.Context) ([]Entry, error) {
	return r.identificationTypes, nil
}
func (r *MemoryRepository) ListTaxTypes(_ context.Context) ([]Entry, error) { return r.taxTypes, nil }
func (r *MemoryRepository) ListPaymentMethods(_ context.Context) ([]Entry, error) {
	return r.paymentMethods, nil
}
func (r *MemoryRepository) ListPaymentTerms(_ context.Context) ([]Entry, error) {
	return r.paymentTerms, nil
}
func (r *MemoryRepository) ListUnitMeasures(_ context.Context) ([]Entry, error) {
	return r.unitMeasures, nil
}
func (r *MemoryRepository) ListTaxRegimes(_ context.Context) ([]Entry, error) {
	return r.taxRegimes, nil
}
func (r *MemoryRepository) ListLiabilityCodes(_ context.Context) ([]Entry, error) {
	return r.liabilityCodes, nil
}
func (r *MemoryRepository) ListDianDocumentTypes(_ context.Context) ([]Entry, error) {
	return r.dianDocumentTypes, nil
}
func (r *MemoryRepository) ListCurrencies(_ context.Context) ([]Currency, error) {
	return r.currencies, nil
}
func (r *MemoryRepository) ListItemStandards(_ context.Context) ([]ItemStandard, error) {
	return r.itemStandards, nil
}
func (r *MemoryRepository) ListCiiuCodes(_ context.Context) ([]CiiuCode, error) {
	return r.ciiuCodes, nil
}
func (r *MemoryRepository) IsValidPaymentTerm(_ context.Context, code string) (bool, error) {
	return containsCode(r.paymentTerms, code), nil
}
func (r *MemoryRepository) IsValidPaymentMethod(_ context.Context, code string) (bool, error) {
	return containsCode(r.paymentMethods, code), nil
}
func (r *MemoryRepository) IsValidLiabilityCode(_ context.Context, code string) (bool, error) {
	return containsCode(r.liabilityCodes, code), nil
}
func (r *MemoryRepository) GetTaxTypeName(_ context.Context, code string) (string, bool, error) {
	n := getEntryName(r.taxTypes, code)
	return n, n != "", nil
}
func (r *MemoryRepository) GetPaymentTermName(_ context.Context, code string) (string, bool, error) {
	n := getEntryName(r.paymentTerms, code)
	return n, n != "", nil
}
func (r *MemoryRepository) GetPaymentMethodName(_ context.Context, code string) (string, bool, error) {
	n := getEntryName(r.paymentMethods, code)
	return n, n != "", nil
}
func (r *MemoryRepository) GetIdentificationTypeName(_ context.Context, code string) (string, bool, error) {
	n := getEntryName(r.identificationTypes, code)
	return n, n != "", nil
}
func (r *MemoryRepository) GetTaxRegimeName(_ context.Context, code string) (string, bool, error) {
	n := getEntryName(r.taxRegimes, code)
	return n, n != "", nil
}
func (r *MemoryRepository) GetLiabilityCodeName(_ context.Context, code string) (string, bool, error) {
	n := getEntryName(r.liabilityCodes, code)
	return n, n != "", nil
}
func (r *MemoryRepository) GetItemStandardName(_ context.Context, code string) (string, bool, error) {
	for _, s := range r.itemStandards {
		if s.Code == code {
			return s.Name, true, nil
		}
	}
	return "", false, nil
}
func (r *MemoryRepository) GetItemStandardAgencyID(_ context.Context, code string) (string, bool, error) {
	for _, s := range r.itemStandards {
		if s.Code == code {
			return s.AgencyID, true, nil
		}
	}
	return "", false, nil
}
func (r *MemoryRepository) GetCiiuDescription(_ context.Context, code string) (string, bool, error) {
	for _, c := range r.ciiuCodes {
		if c.Code == code {
			return c.Description, true, nil
		}
	}
	return "", false, nil
}

func getEntryName(entries []Entry, code string) string {
	for _, e := range entries {
		if e.Code == code {
			return e.Name
		}
	}
	return ""
}

func containsCode(entries []Entry, code string) bool {
	for _, e := range entries {
		if e.Code == code {
			return true
		}
	}
	return false
}
