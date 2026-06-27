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
	Description      string
	Quantity         float64
	UnitCode         string
	UnitPriceCents   int64
	ItemCode         string
	ItemTypeCode     string
	ItemTypeName     string
	ItemTypeAgencyID string
	// TaxTypeCode vacío significa "esta línea no lleva impuesto" — 0 o 1 impuesto por línea
	// (el caso común; domain.Line sigue soportando varios si algún día hace falta un caso más
	// avanzado, pero este cálculo automático por ahora solo cubre uno).
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
		line := domain.Line{
			Description:        in.Description,
			Quantity:           in.Quantity,
			UnitCode:           in.UnitCode,
			LineExtensionCents: lineExtension,
			UnitPriceCents:     in.UnitPriceCents,
			ItemCode:           in.ItemCode,
			ItemTypeCode:       in.ItemTypeCode,
			ItemTypeName:       in.ItemTypeName,
			ItemTypeAgencyID:   in.ItemTypeAgencyID,
		}
		if in.TaxTypeCode != "" {
			name, found, err := s.catalogs.GetTaxTypeName(ctx, in.TaxTypeCode)
			if err != nil {
				return nil, err
			}
			if !found {
				return nil, fmt.Errorf("%w: %q", ErrInvalidTaxTypeCode, in.TaxTypeCode)
			}
			taxAmount := roundCents(float64(lineExtension) * in.TaxPercent / 100)
			line.Taxes = []domain.Tax{{
				TaxableAmountCents: lineExtension,
				TaxAmountCents:     taxAmount,
				Percent:            in.TaxPercent,
				TypeCode:           in.TaxTypeCode,
				TypeName:           name,
			}}
		}
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
