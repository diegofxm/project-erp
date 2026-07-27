package documents

import (
	"context"
	"fmt"
	"math"

	"github.com/diegofxm/cofacture/domain"
)

// LineInput es lo que el llamador (HTTP u otro consumidor de la API) provee para una línea —
// sin aritmética: linesFromInput calcula LineExtensionCents/TaxAmountCents a partir de
// Quantity/UnitPriceCents/TaxPercent, así ningún consumidor de la API tiene que reimplementar
// esa aritmética ni puede mandarla inconsistente con quantity×unit_price (hallazgo de la
// auditoría de catálogos huérfanos: antes de esto, computeTotals/aggregateTaxes solo sumaban
// lo que ya viniera calculado en la petición, sin verificarlo — ver
// docs/apidian-architecture.md). domain.Line sigue siendo la forma ya calculada que cofacture
// necesita para construir el XML — no cambió, y este archivo no la reemplaza, solo evita que
// el llamador tenga que producirla a mano.
type LineInput struct {
	Description    string
	Quantity       float64
	UnitCode       string
	UnitPriceCents int64
	ItemCode       string
	// ItemTypeCode es el selector de @schemeID de la tabla 13.3.5 ("001" UNSPSC, "010" GTIN,
	// "020" Partida Arancelaria, "999" estándar propio) — vacío defaultea a "999".
	// ItemTypeName/ItemTypeAgencyID ya NO son input: se derivan del catálogo (ver
	// linesFromInput/docs/apidian-architecture.md sección 9.45), mismo motivo que
	// TaxTypeName no existe como campo de LineInput.
	ItemTypeCode string
	// TaxTypeCode vacío defaultea a "01" (IVA) con TaxPercent forzado a 0 — la DIAN rechaza
	// (regla FAU04, "Base Imponible es distinto a la suma de los valores de las bases
	// imponibles de todas líneas de detalle") si una línea no declara ningún cac:TaxTotal: la
	// base imponible del encabezado deja de coincidir con la suma de las líneas. Mismo patrón
	// ya probado contra la DIAN real en cofacture/soap/realsend_test.go — "sin impuesto" se
	// modela como IVA al 0%, no como ausencia total de impuesto. Por eso siempre hay
	// exactamente 1 impuesto por línea, nunca 0 (domain.Line sigue soportando varios si algún
	// día hace falta un caso más avanzado, pero este cálculo automático por ahora solo cubre
	// uno).
	TaxTypeCode string
	TaxPercent  float64
}

// linesFromInput calcula cada domain.Line a partir de su LineInput — ver el comentario ahí.
// Es un método de Service (no función libre) porque resuelve TaxTypeName contra el catálogo
// en Postgres.
func (s *Service) linesFromInput(ctx context.Context, inputs []LineInput) ([]domain.Line, error) {
	lines := make([]domain.Line, len(inputs))
	for i, in := range inputs {
		lineExtension := roundCents(in.Quantity * float64(in.UnitPriceCents))
		itemTypeCode := in.ItemTypeCode
		if itemTypeCode == "" {
			itemTypeCode = "999"
		}
		itemTypeName, found, err := s.catalogs.GetItemStandardName(ctx, itemTypeCode)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("%w: %q", ErrInvalidItemStandardCode, itemTypeCode)
		}
		itemTypeAgencyID, _, err := s.catalogs.GetItemStandardAgencyID(ctx, itemTypeCode)
		if err != nil {
			return nil, err
		}
		line := domain.Line{
			Description:        in.Description,
			Quantity:           in.Quantity,
			UnitCode:           in.UnitCode,
			LineExtensionCents: lineExtension,
			UnitPriceCents:     in.UnitPriceCents,
			ItemCode:           in.ItemCode,
			ItemTypeCode:       itemTypeCode,
			ItemTypeName:       itemTypeName,
			ItemTypeAgencyID:   itemTypeAgencyID,
		}
		taxTypeCode := in.TaxTypeCode
		taxPercent := in.TaxPercent
		if taxTypeCode == "" {
			taxTypeCode = "01"
			taxPercent = 0
		}
		name, found, err := s.catalogs.GetTaxTypeName(ctx, taxTypeCode)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("%w: %q", ErrInvalidTaxTypeCode, taxTypeCode)
		}
		taxAmount := roundCents(float64(lineExtension) * taxPercent / 100)
		line.Taxes = []domain.Tax{{
			TaxableAmountCents: lineExtension,
			TaxAmountCents:     taxAmount,
			Percent:            taxPercent,
			TypeCode:           taxTypeCode,
			TypeName:           name,
		}}
		lines[i] = line
	}
	return lines, nil
}

func roundCents(v float64) int64 {
	return int64(math.Round(v))
}

// resolveCustomerTaxSchemeName deriva Customer.TaxSchemeName del catálogo tax_types a partir
// de Customer.TaxSchemeCode — el cliente ya no puede mandar el nombre (ver partyDTO en
// internal/api/dto.go), así ningún consumidor de la API puede guardar un código y un nombre
// que no correspondan entre sí. Se llama después de applyCustomerDefaults, así que
// TaxSchemeCode nunca está vacío en la práctica (default "ZZ") — el chequeo de vacío queda
// igual de defensivo que en customers.Service.resolveTaxSchemeName.
func (s *Service) resolveCustomerTaxSchemeName(ctx context.Context, p *domain.Party) error {
	if p.TaxSchemeCode == "" {
		p.TaxSchemeName = ""
		return nil
	}
	name, found, err := s.catalogs.GetTaxTypeName(ctx, p.TaxSchemeCode)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: %q", ErrInvalidTaxSchemeCode, p.TaxSchemeCode)
	}
	p.TaxSchemeName = name
	return nil
}
