package documents

import (
	"context"
	"fmt"
	"strings"

	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/diegofxm/cofacture/signer"
	"github.com/diegofxm/cofacture/soap"
	"github.com/google/uuid"
)

// DianRange es un rango de numeración tal como lo devuelve la DIAN vía GetNumberingRange —
// campos directos del webservice, sin DianDocumentTypeCode ni TestSetID que son conceptos
// propios de apidian. El caller agrega esos dos campos al importar.
type DianRange struct {
	ResolutionNumber     string `json:"resolution_number"`
	ResolutionDate       string `json:"resolution_date"` // YYYY-MM-DD (hora recortada)
	Prefix               string `json:"prefix"`
	RangeFrom            int64  `json:"range_from"`
	RangeTo              int64  `json:"range_to"`
	ValidFrom            string `json:"valid_from"` // YYYY-MM-DD
	ValidTo              string `json:"valid_to"`   // YYYY-MM-DD
	TechnicalKey         string `json:"technical_key,omitempty"`
	SuggestedDocTypeCode string `json:"suggested_doc_type_code,omitempty"`
}

// GetDianNumberingRanges consulta los rangos de numeración del emisor ante la DIAN vía
// GetNumberingRange. Mismo patrón que VerifyAcquirer: exige certificado + software_id
// configurados, usa el ambiente del emisor (HAB="2" → HabilitacionURL, PRD="1" → ProduccionURL),
// y devuelve los rangos tal como los reporta la DIAN — sin los campos que apidian agrega
// internamente (DianDocumentTypeCode, TestSetID).
func (s *Service) GetDianNumberingRanges(ctx context.Context, issuerID uuid.UUID) ([]DianRange, error) {
	iss, err := s.issuers.GetIssuer(ctx, issuerID)
	if err != nil {
		return nil, err
	}
	if len(iss.Certificate) == 0 || iss.SoftwareID == "" {
		return nil, ErrIssuerNotReadyToIssue
	}

	cert, key, err := signer.LoadPKCS12(iss.Certificate, iss.CertificatePassword)
	if err != nil {
		return nil, fmt.Errorf("cargar certificado del emisor: %w", err)
	}

	baseURL := soap.HabilitacionURL
	if iss.Environment == issuers.EnvironmentProduccion {
		baseURL = soap.ProduccionURL
	}

	client := soap.New(baseURL, cert, key)
	result, err := client.GetNumberingRange(iss.NIT, iss.NIT, iss.SoftwareID)
	if err != nil {
		return nil, fmt.Errorf("consultar GetNumberingRange en la DIAN: %w", err)
	}

	// OperationCode "100"/"0" = éxito; "301" = sin rangos — ambos son resultados normales,
	// no errores. Devolver lista vacía en lugar de error para que el frontend muestre el mensaje
	// adecuado.
	ranges := make([]DianRange, 0, len(result.ResponseList))
	for _, r := range result.ResponseList {
		ranges = append(ranges, DianRange{
			ResolutionNumber:     r.ResolutionNumber,
			ResolutionDate:       trimISODateTime(r.ResolutionDate),
			Prefix:               r.Prefix,
			RangeFrom:            r.FromNumber,
			RangeTo:              r.ToNumber,
			ValidFrom:            trimISODateTime(r.ValidDateFrom),
			ValidTo:              trimISODateTime(r.ValidDateTo),
			TechnicalKey:         r.TechnicalKey,
			SuggestedDocTypeCode: inferDocTypeFromPrefix(r.Prefix),
		})
	}
	return ranges, nil
}

// trimISODateTime recorta la parte de hora de un timestamp tipo "2019-01-19T00:00:00" → "2019-01-19".
func trimISODateTime(s string) string {
	if i := strings.Index(s, "T"); i > 0 {
		return s[:i]
	}
	return s
}

// inferDocTypeFromPrefix sugiere un tipo de documento DIAN a partir de los prefijos estándar
// de habilitación. En producción los prefijos son libres, así que devuelve "" si no reconoce.
func inferDocTypeFromPrefix(prefix string) string {
	switch strings.ToUpper(prefix) {
	case "SETP":
		return "01" // Factura Electrónica
	case "SEDS":
		return "05" // Documento Soporte
	default:
		return ""
	}
}
