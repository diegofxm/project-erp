package application

import (
	"context"
	"fmt"

	cofdom "github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/electronic/domain"
	productdomain "github.com/diegofxm/erp/internal/product/domain"
	salesdomain "github.com/diegofxm/erp/internal/sales/domain"
)

// CreateFromSaleUseCase genera un borrador de factura DIAN a partir de una venta confirmada.
type CreateFromSaleUseCase struct {
	draft     *CreateDraftUseCase
	sales     salesdomain.Repository
	customers domain.CustomerPort
	products  productdomain.Repository
}

func NewCreateFromSaleUseCase(
	draft *CreateDraftUseCase,
	sales salesdomain.Repository,
	customers domain.CustomerPort,
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
	if sale.InvoiceDocumentID != nil {
		return nil, fmt.Errorf("from-sale: esta venta ya generó la factura %s", *sale.InvoiceDocumentID)
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

	doc, err := uc.draft.CreateInvoiceDraft(ctx, InvoiceDraftRequest{
		CompanyID:        req.CompanyID,
		NumberingRangeID: req.NumberingRangeID,
		Customer:         saleCustomerToParty(customer),
		Lines:            saleLinesТoСof(sale.Lines, productMap),
		PaymentMeans:     salePaymentMeansOrDefault(sale.PaymentMeans),
		Note:             sale.Notes,
		CurrencyCode:     "COP",
		CustomerID:       &sale.CustomerID,
	})
	if err != nil {
		return nil, err
	}

	if err := uc.sales.SetInvoiceDocumentID(ctx, req.CompanyID, req.SaleID, doc.ID); err != nil {
		// El borrador ya se creó y es válido — no revertirlo por un fallo al anotar el vínculo,
		// solo se pierde la protección contra doble-facturación esta vez.
		return doc, fmt.Errorf("from-sale: factura creada pero no se pudo vincular a la venta: %w", err)
	}

	return doc, nil
}

func saleCustomerToParty(c *domain.Party) cofdom.Party {
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

// salePaymentMeansOrDefault usa los medios de pago realmente pactados en la venta -- antes esta
// función siempre forzaba "Contado/Efectivo" (código 1/1) sin importar cómo se vendió. Si la venta
// no trae ninguno (ventas creadas antes de este cambio, o el vendedor dejó el campo vacío), se
// conserva ese mismo default como último recurso para no romper la generación del documento.
func salePaymentMeansOrDefault(pms []cofdom.PaymentMean) []cofdom.PaymentMean {
	if len(pms) == 0 {
		return []cofdom.PaymentMean{{Code: "1", PaymentMethodCode: "1"}}
	}
	return pms
}

func saleLinesТoСof(lines []salesdomain.SaleLine, products map[uuid.UUID]*productdomain.Product) []cofdom.Line {
	out := make([]cofdom.Line, len(lines))
	for i, l := range lines {
		p := products[l.ProductID]
		unitCode := "94"
		itemCode := ""
		itemTypeCode := ""
		itemTypeName := ""
		itemTypeAgencyID := ""
		taxCode := "01"
		taxName := "IVA"
		if p != nil {
			if p.UnitMeasureCode != "" {
				unitCode = p.UnitMeasureCode
			}
			// StandardCode/StandardCodeID van al <cbc:ID schemeID= schemeName=> de
			// StandardItemIdentification (ver cofacture/builder/line_items.go) -- StandardCodeID es
			// el CÓDIGO de catálogo DIAN (ej. "999"), no el nombre. Antes este mapeo mandaba
			// StandardCodeType (texto libre del producto, ej. "Estándar propio") como schemeID, y
			// la DIAN rechazaba con FAZ12 "Codigo informado en @schemID no es valido" (bug real
			// encontrado 2026-08-11). schemeName/schemeAgencyID se resuelven contra itemStandards
			// (tabla oficial 13.3.5 del Anexo Técnico, ver create_draft.go) en vez de confiar en el
			// texto libre que el producto tenga guardado -- así el XML siempre queda con el nombre
			// oficial exacto sin importar qué se haya escrito al crear el producto.
			itemCode = p.StandardCode
			if itemCode == "" {
				itemCode = p.Code // nunca dejar el <cbc:ID> vacío si el producto no definió StandardCode
			}
			itemTypeCode = p.StandardCodeID
			if itemTypeCode == "" {
				itemTypeCode = "999"
			}
			if std, ok := itemStandards[itemTypeCode]; ok {
				itemTypeName = std.name
				itemTypeAgencyID = std.agencyID
			} else {
				itemTypeName = p.StandardCodeType
				itemTypeAgencyID = p.StandardCodeAgencyID
			}
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
			ItemTypeAgencyID:   itemTypeAgencyID,
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
