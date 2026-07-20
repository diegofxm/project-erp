package documents

import (
	"context"

	"github.com/diegofxm/apidian/internal/customers"
	"github.com/diegofxm/apidian/internal/email"
	"github.com/diegofxm/apidian/internal/issuers"
	"github.com/diegofxm/apidian/internal/numbering"
	"github.com/diegofxm/apidian/internal/vendors"
	"github.com/google/uuid"
)

// IssuerPort define lo que documents necesita del dominio de emisores. Es una interfaz
// angosta (no *issuers.Service directamente) para poder probar Service con un fake en tests,
// igual patrón que transfers.AccountPort en core-bank.
type IssuerPort interface {
	GetIssuer(ctx context.Context, id uuid.UUID) (*issuers.Issuer, error)
}

// NumberingPort define lo que documents necesita del dominio de numeración.
type NumberingPort interface {
	GetRange(ctx context.Context, id uuid.UUID) (*numbering.NumberingRange, error)
	ClaimNext(ctx context.Context, id uuid.UUID) (int64, error)

	// ReleaseIfCurrent — ver numbering.Service.ReleaseIfCurrent. Se llama cuando un número
	// recién reclamado termina en rejected/send_error, o cuando confirmar falla ANTES de
	// llegar a intentar el envío (ej. construir o firmar el XML) — en ambos casos el número
	// nunca quedó realmente ante la DIAN, así que el siguiente intento lo puede reclamar de
	// nuevo en vez de avanzar y dejar un hueco (ver sección 9.33).
	ReleaseIfCurrent(ctx context.Context, id uuid.UUID, number int64) error

	// ClearTestSetID — ver numbering.Service.ClearTestSetID. Se llama cuando la DIAN responde
	// que el set de pruebas de SendTestSetAsync ya quedó certificado/cerrado de su lado (ver
	// dian.Result.IsTestSetClosed) — no es un rechazo de contenido del documento, es un
	// detalle de certificación que el sistema resuelve solo, sin intervención manual (ver
	// docs/apidian-architecture.md sección 9.43).
	ClearTestSetID(ctx context.Context, id uuid.UUID) error
}

// CustomerPort define lo que documents necesita del catálogo de clientes — solo para
// verificar, cuando el llamador manda un CustomerID opcional, que ese cliente pertenece al
// mismo emisor (mismo criterio que NumberingPort con el rango: nunca se confía en que el
// cliente referenciado sea del emisor correcto sin comprobarlo). El catálogo en sí no
// participa en construir el XML — eso sigue siendo pass-through puro (ver model.go).
type CustomerPort interface {
	GetCustomer(ctx context.Context, id uuid.UUID) (*customers.Customer, error)
}

// VendorPort define lo que documents necesita del catálogo de proveedores — misma función que
// CustomerPort pero para el VendorID opcional de Documento Soporte.
type VendorPort interface {
	GetVendor(ctx context.Context, id uuid.UUID) (*vendors.Vendor, error)
}

// CatalogPort valida payment_means/liability_codes contra catálogos DIAN que viven en JSONB
// (payment_terms/payment_methods) o en un TEXT[] sin FK posible (liability_codes) — ver
// auditoría de catálogos huérfanos, docs/apidian-architecture.md. Sin esto, un código
// inválido ahí solo se detectaba al confirmar, con la DIAN rechazando el documento — con esto
// se detecta antes, al crear/editar el borrador.
//
// unit_measures (lines[].UnitCode) NO se valida aquí a propósito: ese catálogo sigue
// incompleto (11 códigos de muestra frente al estándar UN/ECE Rec. 20 completo, ver
// migrations/000006_products.up.sql) — validar contra un catálogo incompleto rechazaría
// códigos legítimos que la DIAN sí aceptaría. Se agrega cuando se complete ese catálogo.
//
// GetTaxTypeName resuelve customer.tax_scheme_name/lines[].tax_type_name a partir de sus
// códigos — ninguno de los dos se acepta ya del cliente (ver applyCustomerDefaults/
// linesFromInput), así un documento nunca puede guardar un código y un nombre que no
// correspondan entre sí.
type CatalogPort interface {
	IsValidPaymentTerm(ctx context.Context, code string) (bool, error)
	IsValidPaymentMethod(ctx context.Context, code string) (bool, error)
	IsValidLiabilityCode(ctx context.Context, code string) (bool, error)
	GetTaxTypeName(ctx context.Context, code string) (name string, found bool, err error)

	// GetPaymentTermName/GetPaymentMethodName/GetIdentificationTypeName — para
	// RenderInvoicePDF (ver docs/apidian-architecture.md sección 9.39): la representación
	// gráfica muestra "Contado"/"Transferencia.../"Cédula de Ciudadanía", no el código crudo.
	GetPaymentTermName(ctx context.Context, code string) (name string, found bool, err error)
	GetPaymentMethodName(ctx context.Context, code string) (name string, found bool, err error)
	GetIdentificationTypeName(ctx context.Context, code string) (name string, found bool, err error)

	// GetItemStandardName/GetItemStandardAgencyID resuelven lines[].item_type_name/
	// item_type_agency_id a partir de item_type_code — el cliente ya no los manda (ver
	// docs/apidian-architecture.md sección 9.45), así un documento nunca puede guardar un
	// código y un nombre/agencia que no correspondan entre sí (mismo motivo que
	// GetTaxTypeName). AgencyID separado de Name porque puede ser "" con found=true.
	GetItemStandardName(ctx context.Context, code string) (name string, found bool, err error)
	GetItemStandardAgencyID(ctx context.Context, code string) (agencyID string, found bool, err error)

	// GetTaxRegimeName/GetLiabilityCodeName — para el pie del PDF: por norma debe aparecer el
	// régimen tributario y las responsabilidades fiscales del emisor (Resolución DIAN 042).
	GetTaxRegimeName(ctx context.Context, code string) (name string, found bool, err error)
	GetLiabilityCodeName(ctx context.Context, code string) (name string, found bool, err error)

	// GetCiiuDescription — descripción de un código CIIU para el pie del PDF (actividad
	// económica del emisor con nombre legible en vez del código crudo).
	GetCiiuDescription(ctx context.Context, code string) (description string, found bool, err error)
}

// SettingsPort devuelve el color de marca del emisor para el PDF. La interfaz es angosta a
// propósito: documents no necesita leer ni escribir configuraciones completas.
type SettingsPort interface {
	GetBrandColor(ctx context.Context, issuerID uuid.UUID) string
}

// EmailPort define lo que documents necesita para enviar correo — ver
// docs/apidian-architecture.md sección 9.42. Angosta a propósito (no *email.SMTPSender
// directamente) para poder fakear el envío en tests, mismo motivo que el resto de los puertos
// de este archivo, no porque haya más de una implementación real todavía.
type EmailPort interface {
	Send(ctx context.Context, msg email.Message) error
}
