package application

import (
	"context"
	"fmt"

	cofdom "github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"

	customerdomain "github.com/diegofxm/erp/internal/customer/domain"
	"github.com/diegofxm/erp/internal/electronic/domain"
	productdomain "github.com/diegofxm/erp/internal/product/domain"
	salesdomain "github.com/diegofxm/erp/internal/sales/domain"
)

// CreateFromSaleUseCase genera un borrador de factura DIAN a partir de una venta confirmada.
type CreateFromSaleUseCase struct {
	draft     *CreateDraftUseCase
	sales     salesdomain.Repository
	customers customerdomain.Repository
	products  productdomain.Repository
}

func NewCreateFromSaleUseCase(
	draft *CreateDraftUseCase,
	sales salesdomain.Repository,
	customers customerdomain.Repository,
	products productdomain.Repository,
) *CreateFromSaleUseCase {
	return &CreateFromSaleUseCase{draft: draft, sales: sales, customers: customers, products: products}
}

type FromSaleRequest struct {
	CompanyID        uuid.UUID
	SaleID           uuid.UUID
	NumberingRangeID uuid.UUID
}

func (uc *CreateFromSaleUseCase) Execute(ctx context.Context, req FromSaleRequest) (*domain.Document, error) {
	sale, err := uc.sales.GetByID(ctx, req.CompanyID, req.SaleID)
	if err != nil {
		return nil, fmt.Errorf("from-sale: venta: %w", err)
	}
	if sale.Status != salesdomain.StatusConfirmed {
		return nil, fmt.Errorf("from-sale: la venta debe estar confirmada")
	}

	customer, err := uc.customers.GetByID(ctx, req.CompanyID, sale.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("from-sale: cliente: %w", err)
	}

	// Cargar productos únicos de las líneas
	productMap := map[uuid.UUID]*productdomain.Product{}
	for _, l := range sale.Lines {
		if _, ok := productMap[l.ProductID]; !ok {
			p, err := uc.products.GetByID(ctx, req.CompanyID, l.ProductID)
			if err != nil {
				return nil, fmt.Errorf("from-sale: producto %s: %w", l.ProductID, err)
			}
			productMap[l.ProductID] = p
		}
	}

	return uc.draft.CreateInvoiceDraft(ctx, InvoiceDraftRequest{
		CompanyID:        req.CompanyID,
		NumberingRangeID: req.NumberingRangeID,
		Customer:         saleCustomerToParty(customer),
		Lines:            saleLinesТoСof(sale.Lines, productMap),
		PaymentMeans:     []cofdom.PaymentMean{{Code: "1", PaymentMethodCode: "1"}},
		Note:             sale.Notes,
		CurrencyCode:     "COP",
		CustomerID:       &sale.CustomerID,
	})
}

func saleCustomerToParty(c *customerdomain.Customer) cofdom.Party {
	verif := ""
	if c.IdentificationTypeCode == "31" {
		verif = c.CheckDigit
	}
	regime := ""
	if c.TaxRegimeCode != nil {
		regime = *c.TaxRegimeCode
	}
	return cofdom.Party{
		Identification: cofdom.Identification{
			TypeCode:         c.IdentificationTypeCode,
			Number:           c.IdentificationNumber,
			VerificationCode: verif,
		},
		Name: c.Name,
		Address: cofdom.Address{
			Line:        c.AddressLine,
			CityCode:    c.MunicipalityCode,
			StateCode:   c.DepartmentCode,
			CountryCode: "CO",
			PostalZone:  c.MunicipalityCode,
		},
		TaxSchemeCode:  c.TaxSchemeCode,
		TaxSchemeName:  c.TaxSchemeName,
		TaxRegimeCode:  regime,
		LiabilityCodes: c.LiabilityCodes,
		Phone:          c.Phone,
		Email:          c.Email,
	}
}

func saleLinesТoСof(lines []salesdomain.SaleLine, products map[uuid.UUID]*productdomain.Product) []cofdom.Line {
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
