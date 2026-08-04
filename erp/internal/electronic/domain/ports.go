package domain

import (
	"context"
	"time"

	cofdom "github.com/diegofxm/cofacture/domain"
	"github.com/google/uuid"
)

// CompanyInfo es la vista del dominio Company que necesita electronic.
// La empresa en el ERP adopta el rol de "emisor" ante la DIAN.
type CompanyInfo struct {
	ID                          uuid.UUID
	NIT                         string
	CheckDigit                  string
	BusinessName                string
	TradeName                   string
	IdentificationTypeCode      string
	DepartmentCode              string
	MunicipalityCode            string
	AddressLine                 string
	Email                       string
	Phone                       string
	Environment                 Environment
	EntityTypeCode              string
	TaxSchemeCode               string
	TaxSchemeName               string
	LiabilityCodes              []string
	TaxRegimeCode               *string
	IndustryClassificationCodes []string
	MerchantRegistrationNumber  *string
	SoftwareID                  string
	SoftwarePIN                 string // en texto plano (ya descifrado por company repo)
	Certificate                 []byte // PKCS12
	CertificatePassword         string // en texto plano (ya descifrado)
	Logo                        []byte
	LogoContentType             string
}

// CompanyPort lee la empresa como emisor DIAN.
type CompanyPort interface {
	GetCompany(ctx context.Context, id uuid.UUID) (*CompanyInfo, error)
}

// Party es la vista de thirdparty.Party que necesita electronic para armar cofdom.Party — a
// diferencia de sales.Customer/purchase.Supplier (vistas reducidas para PDF/email), electronic
// sí necesita casi todos los campos tributarios DIAN porque los usa para el XML firmado.
type Party struct {
	IdentificationTypeCode string
	IdentificationNumber   string
	CheckDigit             string
	Name                   string
	AddressLine            string
	MunicipalityCode       string
	DepartmentCode         string
	TaxSchemeCode          string
	TaxSchemeName          string
	TaxRegimeCode          *string
	LiabilityCodes         []string
	Phone                  string
	Email                  string
}

// CustomerPort lee el cliente para armar la factura electrónica desde una venta.
type CustomerPort interface {
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*Party, error)
}

// SupplierPort lee el proveedor para armar el Documento Soporte desde una compra.
type SupplierPort interface {
	GetByID(ctx context.Context, companyID, id uuid.UUID) (*Party, error)
}

// BuilderSignerPort construye el XML según el tipo de documento y lo firma con XAdES-EPES.
// Devuelve el XML firmado listo para empaquetar en ZIP.
// Encapsula cofacture/builder + cofacture/signer — el dominio/aplicación no importan etree.
type BuilderSignerPort interface {
	SignedInvoiceXML(inv cofdom.Invoice, cert []byte, password string, signingTime time.Time) ([]byte, error)
	SignedCreditNoteXML(cn cofdom.CreditNote, cert []byte, password string, signingTime time.Time) ([]byte, error)
	SignedDebitNoteXML(dn cofdom.DebitNote, cert []byte, password string, signingTime time.Time) ([]byte, error)
	SignedSupportDocumentXML(inv cofdom.Invoice, cert []byte, password string, signingTime time.Time) ([]byte, error)
	SignedAdjustmentNoteXML(an cofdom.AdjustmentNote, cert []byte, password string, signingTime time.Time) ([]byte, error)
}

// ZipperPort empaqueta un documento XML en ZIP con el nombre de archivo requerido por la DIAN.
type ZipperPort interface {
	Zip(kind, nit string, year int, number int64, xmlBytes []byte) (fileName string, zipBytes []byte, err error)
}

// SendResult contiene el resultado del envío a la DIAN.
type SendResult struct {
	ZipKey                 string
	StatusCode             string
	StatusDescription      string
	StatusMessage          string
	IsValid                bool
	HasRejections          bool
	IsTestSetClosed        bool
	ApplicationResponseXML string
}

// SenderPort envía el ZIP a la DIAN y consulta el estado.
type SenderPort interface {
	SendBillSync(zipFileName string, zipBytes []byte, cert []byte, password string, environmentCode string) (*SendResult, error)
	SendTestSetAsync(zipFileName string, zipBytes []byte, testSetID string, cert []byte, password string, environmentCode string) (zipKey string, err error)
	PollStatusZip(zipKey string, cert []byte, password string, environmentCode string) (*SendResult, error)
}

// CatalogPort valida códigos de catálogo DIAN usados en los documentos.
type CatalogPort interface {
	IsValidPaymentTerm(ctx context.Context, code string) (bool, error)
	IsValidPaymentMethod(ctx context.Context, code string) (bool, error)
	IsValidLiabilityCode(ctx context.Context, code string) (bool, error)
	GetTaxTypeName(ctx context.Context, code string) (string, bool, error)
}

// DianRange es un rango de numeración tal como lo devuelve la DIAN vía GetNumberingRange.
type DianRange struct {
	ResolutionNumber     string `json:"resolution_number"`
	ResolutionDate       string `json:"resolution_date"`
	Prefix               string `json:"prefix"`
	RangeFrom            int64  `json:"range_from"`
	RangeTo              int64  `json:"range_to"`
	ValidFrom            string `json:"valid_from"`
	ValidTo              string `json:"valid_to"`
	TechnicalKey         string `json:"technical_key,omitempty"`
	SuggestedDocTypeCode string `json:"suggested_doc_type_code,omitempty"`
}

// DianRangesFetcherPort consulta los rangos autorizados del emisor ante la DIAN.
type DianRangesFetcherPort interface {
	GetNumberingRanges(nit, softwareID string, cert []byte, password, environmentCode string) ([]DianRange, error)
}

// EmailZipPort construye el ZIP que va adjunto al correo del destinatario.
// Contiene el AttachedDocument XML firmado (AT 1.9 §9.1) + el PDF.
// Si el documento no tiene ApplicationResponseXML (aceptación DIAN), cae en
// fallback: ZIP con el SignedXML crudo + PDF, sin construir AttachedDocument.
type EmailZipPort interface {
	BuildEmailZip(doc *Document, company *CompanyInfo, filename string, pdfBytes []byte, now time.Time) (zipBytes []byte, err error)
}
