package withholdings

import (
	"context"
	"errors"
	"fmt"
)

// Service expone el cálculo y consulta de retenciones.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Calculate calcula la retención que corresponde a un único concepto sobre la base
// indicada. Retorna nil sin error cuando el pago no supera la base mínima en UVT.
//
// base: monto sobre el que aplica la retención, en centavos.
// year: año fiscal para obtener el valor del UVT y convertir min_base_uvt a pesos.
func (s *Service) Calculate(ctx context.Context,
	code string,
	wType WithholdingType,
	base int64,
	vendorType VendorType,
	year int,
) (*Result, error) {
	concept, err := s.repo.GetConcept(ctx, code, wType, vendorType)
	if err != nil {
		if errors.Is(err, ErrConceptNotFound) {
			return nil, fmt.Errorf("concepto %q tipo %s para %s: %w", code, wType, vendorType, ErrConceptNotFound)
		}
		return nil, err
	}

	// Verificar base mínima en UVT cuando el concepto la requiere.
	if concept.MinBaseUVT > 0 {
		uvt, err := s.repo.GetUVT(ctx, year)
		if err != nil {
			return nil, err
		}
		// minBase en centavos = UVT × valor_UVT_centavos
		minBase := int64(concept.MinBaseUVT * float64(uvt.ValueCents))
		if base < minBase {
			return nil, nil // pago inferior a la base mínima — no hay retención
		}
	}

	// Calcular monto: base × rate_bp / 10 000
	amount := base * int64(concept.RateBP) / 10_000

	return &Result{
		ConceptCode:       concept.Code,
		ConceptName:       concept.Name,
		Type:              concept.Type,
		RateBP:            concept.RateBP,
		Base:              base,
		Amount:            amount,
		AccountPayable:    concept.AccountPayable,
		AccountReceivable: concept.AccountReceivable,
	}, nil
}

// CalculateMany calcula varias retenciones de distintos conceptos en una sola llamada.
// Omite silenciosamente los conceptos que no superan la base mínima.
func (s *Service) CalculateMany(ctx context.Context, items []CalculationItem, year int) ([]*Result, error) {
	var results []*Result
	for _, item := range items {
		r, err := s.Calculate(ctx, item.Code, item.Type, item.Base, item.VendorType, year)
		if err != nil {
			return nil, fmt.Errorf("concepto %q: %w", item.Code, err)
		}
		if r != nil {
			results = append(results, r)
		}
	}
	return results, nil
}

// ListConcepts devuelve los conceptos activos de un tipo, útil para poblar
// selectores en la interfaz o para exportar el catálogo.
func (s *Service) ListConcepts(ctx context.Context, wType WithholdingType) ([]*Concept, error) {
	return s.repo.ListConcepts(ctx, wType)
}

// ListUVT devuelve los valores del UVT registrados, ordenados del más reciente al más antiguo.
func (s *Service) ListUVT(ctx context.Context) ([]*UVTValue, error) {
	return s.repo.ListUVT(ctx)
}

// CalculationItem describe la entrada de un cálculo individual dentro de CalculateMany.
type CalculationItem struct {
	Code       string
	Type       WithholdingType
	Base       int64
	VendorType VendorType
}
