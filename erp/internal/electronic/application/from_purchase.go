package application

import (
	"context"
	"fmt"

	cofdom "github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/electronic/domain"
	productdomain "github.com/diegofxm/erp/internal/product/domain"
	purchasedomain "github.com/diegofxm/erp/internal/purchase/domain"
)

// CreateFromPurchaseUseCase genera un borrador de Documento Soporte (DS) a partir de una orden
// de compra recibida — espejo de CreateFromSaleUseCase (factura desde venta). El DS lo emite el
// propio comprador cuando el proveedor no está obligado a facturar electrónicamente.
type CreateFromPurchaseUseCase struct {
	draft     *CreateDraftUseCase
	purchases purchasedomain.Repository
	suppliers domain.SupplierPort
	products  productdomain.Repository
}

func NewCreateFromPurchaseUseCase(
	draft *CreateDraftUseCase,
	purchases purchasedomain.Repository,
	suppliers domain.SupplierPort,
	products productdomain.Repository,
) *CreateFromPurchaseUseCase {
	return &CreateFromPurchaseUseCase{draft: draft, purchases: purchases, suppliers: suppliers, products: products}
}

type FromPurchaseRequest struct {
	CompanyID         uuid.UUID
	PurchaseID        uuid.UUID
	NumberingRangeID  uuid.UUID
	OperationTypeCode string // "10" Residente, "11" No Residente — default "10" si viene vacío
}

func (uc *CreateFromPurchaseUseCase) Execute(ctx context.Context, req FromPurchaseRequest) (*domain.Document, error) {
	purchase, err := uc.purchases.GetByID(ctx, req.CompanyID, req.PurchaseID)
	if err != nil {
		return nil, fmt.Errorf("from-purchase: orden de compra: %w", err)
	}
	if purchase.Status != purchasedomain.StatusReceived {
		return nil, fmt.Errorf("from-purchase: la orden debe estar recibida")
	}
	if purchase.SupportDocumentID != nil {
		return nil, fmt.Errorf("from-purchase: esta orden ya generó el documento soporte %s", *purchase.SupportDocumentID)
	}

	supplier, err := uc.suppliers.GetByID(ctx, req.CompanyID, purchase.SupplierID)
	if err != nil {
		return nil, fmt.Errorf("from-purchase: proveedor: %w", err)
	}

	productMap := map[uuid.UUID]*productdomain.Product{}
	for _, l := range purchase.Lines {
		if _, ok := productMap[l.ProductID]; !ok {
			p, err := uc.products.GetByID(ctx, req.CompanyID, l.ProductID)
			if err != nil {
				return nil, fmt.Errorf("from-purchase: producto %s: %w", l.ProductID, err)
			}
			productMap[l.ProductID] = p
		}
	}

	operationType := req.OperationTypeCode
	if operationType == "" {
		operationType = "10"
	}

	doc, err := uc.draft.CreateSupportDocumentDraft(ctx, SupportDocumentDraftRequest{
		CompanyID:         req.CompanyID,
		NumberingRangeID:  req.NumberingRangeID,
		Supplier:          purchaseSupplierToParty(supplier),
		Lines:             purchaseLinesToCof(purchase.Lines, productMap),
		PaymentMeans:      purchasePaymentMeansOrDefault(purchase.PaymentMeans),
		Note:              purchase.Notes,
		CurrencyCode:      "COP",
		OperationTypeCode: operationType,
		SupplierID:        &purchase.SupplierID,
	})
	if err != nil {
		return nil, err
	}

	if err := uc.purchases.SetSupportDocumentID(ctx, req.CompanyID, req.PurchaseID, doc.ID); err != nil {
		// El borrador ya se creó y es válido — no revertirlo por un fallo al anotar el vínculo,
		// solo se pierde la protección contra doble-generación esta vez.
		return doc, fmt.Errorf("from-purchase: documento soporte creado pero no se pudo vincular a la orden: %w", err)
	}

	return doc, nil
}

func purchaseSupplierToParty(s *domain.Party) cofdom.Party {
	verif := ""
	if s.IdentificationTypeCode == "31" {
		verif = s.CheckDigit
	}
	regime := ""
	if s.TaxRegimeCode != nil {
		regime = *s.TaxRegimeCode
	}
	return cofdom.Party{
		Identification: cofdom.Identification{
			TypeCode:         s.IdentificationTypeCode,
			Number:           s.IdentificationNumber,
			VerificationCode: verif,
		},
		Name: s.Name,
		Address: cofdom.Address{
			Line:        s.AddressLine,
			CityCode:    s.MunicipalityCode,
			StateCode:   s.DepartmentCode,
			CountryCode: "CO",
			PostalZone:  s.MunicipalityCode,
		},
		TaxSchemeCode:  s.TaxSchemeCode,
		TaxSchemeName:  s.TaxSchemeName,
		TaxRegimeCode:  regime,
		LiabilityCodes: s.LiabilityCodes,
		Phone:          s.Phone,
		Email:          s.Email,
	}
}

// purchasePaymentMeansOrDefault -- ver salePaymentMeansOrDefault, mismo criterio para el
// documento soporte generado desde una orden de compra.
func purchasePaymentMeansOrDefault(pms []cofdom.PaymentMean) []cofdom.PaymentMean {
	if len(pms) == 0 {
		return []cofdom.PaymentMean{{Code: "1", PaymentMethodCode: "1"}}
	}
	return pms
}

func purchaseLinesToCof(lines []purchasedomain.PurchaseLine, products map[uuid.UUID]*productdomain.Product) []cofdom.Line {
	out := make([]cofdom.Line, len(lines))
	for i, l := range lines {
		p := products[l.ProductID]
		unitCode := "94"
		itemCode := ""
		itemTypeCode := ""
		itemTypeName := ""
		taxCode := "01"
		taxName := "IVA"
		if p != nil {
			if p.UnitMeasureCode != "" {
				unitCode = p.UnitMeasureCode
			}
			itemCode = p.Code
			itemTypeCode = p.StandardCodeType
			itemTypeName = p.StandardCode
			if p.TaxSchemeCode != "" {
				taxCode = p.TaxSchemeCode
				taxName = p.TaxSchemeName
			}
		}
		var taxes []cofdom.Tax
		if l.TaxRate > 0 {
			taxes = []cofdom.Tax{{
				TypeCode:           taxCode,
				TypeName:           taxName,
				Percent:            l.TaxRate,
				TaxableAmountCents: int64(l.Subtotal * 100),
				TaxAmountCents:     int64(l.TaxAmount * 100),
			}}
		}
		out[i] = cofdom.Line{
			ItemCode:           itemCode,
			ItemTypeCode:       itemTypeCode,
			ItemTypeName:       itemTypeName,
			Description:        l.Description,
			Quantity:           l.Quantity,
			UnitCode:           unitCode,
			UnitPriceCents:     int64(l.UnitPrice * 100),
			LineExtensionCents: int64(l.Subtotal * 100),
			Taxes:              taxes,
		}
	}
	return out
}
