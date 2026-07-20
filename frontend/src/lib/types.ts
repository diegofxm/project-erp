// Formas de los DTOs que expone apidian — ver apidian/internal/api/handler_auth.go y
// handler_issuers.go. Mantener en sincronía a mano (no hay generación automática todavía).

export interface User {
  id: string;
  email: string;
  name: string;
  role: string;
  is_superadmin: boolean;
}

export type IssuerEnvironment = "1" | "2"; // 1 = producción, 2 = habilitación

export interface Issuer {
  id: string;
  nit: string;
  check_digit: string;
  business_name: string;
  trade_name?: string;
  identification_type_code: string;
  environment: IssuerEnvironment;

  // Dirección
  department_code: string;
  municipality_code: string;
  department_name?: string;
  municipality_name?: string;
  address_line: string;
  phone?: string;
  email: string;

  // Información fiscal
  entity_type_code?: string;
  tax_scheme_code?: string;
  tax_scheme_name?: string;
  liability_codes?: string[];
  tax_regime_code?: string;
  industry_classification_codes?: string[];
  merchant_registration_number?: string;

  // Credenciales DIAN — los IDs no son secretos; PINs/cert nunca viajan de vuelta.
  // FE y DS comparten el mismo software; NE tiene software independiente.
  software_id?: string;
  has_software_credentials: boolean;
  ne_software_id?: string;
  has_ne_software_credentials: boolean;
  // Certificado digital compartido entre FE/DS/NE:
  has_certificate: boolean;
  // Metadatos del certificado derivados en memoria por issuers.Service.enrichCertMetadata —
  // sin datos sensibles. Solo presentes cuando has_certificate es true.
  certificate_subject?: string;
  certificate_issuer_cn?: string;
  certificate_expires_at?: string; // ISO 8601

  // has_logo: el logo en sí se sirve aparte (GET /issuers/me/logo) — ver
  // docs/apidian-architecture.md sección 9.39.
  has_logo: boolean;

  is_active: boolean;
  created_at: string;
}

export interface AuthResult {
  token: string;
  user: User;
  issuer?: Issuer;
}

export interface RegisterPayload {
  email: string;
  password: string;
  name: string;
}

export interface LoginPayload {
  email: string;
  password: string;
}

// Payload de creación de empresa — todo lo que la DIAN exige del emisor mismo, completo
// salvo la configuración técnica (software_id/software_pin/certificate_base64), que se
// completa después en una fase de configuración aparte (PUT /issuers/me, ver
// issuers.UpdateIssuerRequest). Espejo de createIssuerRequest en
// apidian/internal/api/handler_issuers.go, sin esos 4 campos.
export interface CreateIssuerPayload {
  nit: string;
  check_digit: string;
  business_name: string;
  trade_name?: string;
  identification_type_code: string;
  department_code: string;
  municipality_code: string;
  address_line: string;
  email: string;
  phone?: string;
  environment: IssuerEnvironment;
  entity_type_code?: string;
  tax_scheme_code?: string;
  liability_codes?: string[];
  tax_regime_code?: string;
  industry_classification_codes?: string[];
  merchant_registration_number?: string;
}

export interface ListIssuersResult {
  issuers: Issuer[];
  count: number;
}

// Completar software/PIN/certificado DESPUÉS de creada la empresa — PUT /issuers/me. Cada
// campo es independiente (omitido = "no tocar"), espejo de issuers.UpdateIssuerRequest. Nunca
// se manda "" para un campo que el usuario no llenó — se omite la llave entera.
export interface UpdateIssuerPayload {
  // FE y DS comparten el mismo software
  software_id?: string;
  software_pin?: string;
  // NE (Nómina Electrónica) — software independiente
  ne_software_id?: string;
  ne_software_pin?: string;
  // Certificado compartido entre FE/DS/NE
  certificate_base64?: string;
  certificate_password?: string;
  // logo_base64/logo_content_type: para la representación gráfica en PDF.
  logo_base64?: string;
  logo_content_type?: string;
}

// Editar perfil de empresa (razón social, dirección, datos fiscales) — PATCH /issuers/me/profile.
// Separado de UpdateIssuerPayload que toca solo credenciales técnicas (PUT /issuers/me).
// TaxRegimeCode/MerchantRegistrationNumber son nullable: null = borrar, "code" = asignar.
export interface UpdateIssuerProfilePayload {
  business_name: string;
  trade_name: string;
  department_code: string;
  municipality_code: string;
  address_line: string;
  email: string;
  phone: string;
  entity_type_code: string;
  tax_scheme_code: string;
  liability_codes: string[];
  tax_regime_code: string | null;
  industry_classification_codes: string[];
  merchant_registration_number: string | null;
  environment: IssuerEnvironment;
}

// Plan de suscripción — GET /admin/plans.
export interface Plan {
  id: string;
  name: string;
  description: string;
  max_documents_per_month: number | null; // null = ilimitado
  max_issuers: number;
  price_cop: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

// Suscripción activa de un emisor — GET /admin/issuers/{id}/subscription.
export interface Subscription {
  id: string;
  issuer_id: string;
  plan_id: string;
  plan_name: string;
  max_documents_per_month: number | null;
  status: "active" | "cancelled" | "suspended";
  started_at: string;
  ends_at: string | null;
  created_at: string;
  updated_at: string;
}

// Configuración de personalización del emisor — GET/PATCH /issuers/me/settings.
export interface IssuerSettings {
  issuer_id: string;
  brand_color: string; // hex #RRGGBB
  updated_at: string;
}

// Formas compartidas por los catálogos de solo lectura en apidian/internal/catalogs/model.go.
export interface CatalogEntry {
  code: string;
  name: string;
  description: string;
}

export interface Municipality extends CatalogEntry {
  department_code: string;
}

// Espejo de catalogs.Currency — sin description, a diferencia de CatalogEntry.
export interface Currency {
  code: string;
  name: string;
  symbol: string;
}

// Selector real de @schemeID/@schemeName/@schemeAgencyID (tabla 13.3.5 del Anexo Técnico, ver
// docs/apidian-architecture.md sección 9.45) — exactamente 4 filas fijas, no confundir con un
// catálogo completo de códigos UNSPSC/GTIN/Arancel (esos no se cargan).
export interface ItemStandard extends CatalogEntry {
  agency_id?: string;
}

// Espejo de numberingRangeResponse en apidian/internal/api/handler_issuers.go. status resume
// is_active/agotado/vencido en un solo valor (ver numbering.NumberingRange.Status) — el select
// de la factura solo ofrece los "active", el panel de administración los muestra todos con una
// insignia.
export type NumberingRangeStatus = "active" | "expired" | "exhausted" | "inactive";

export interface NumberingRange {
  id: string;
  issuer_id: string;
  dian_document_type_code: string;
  prefix: string;
  range_from: number;
  range_to?: number;
  current_number: number;
  valid_from: string;
  valid_to: string;
  environment: IssuerEnvironment;
  is_active: boolean;
  status: NumberingRangeStatus;
}

export interface ListNumberingRangesResult {
  numbering_ranges: NumberingRange[];
  count: number;
}

// Espejo de createNumberingRangeRequest. technical_key solo aplica cuando
// dian_document_type_code es "01" (Factura, CUFE); test_set_id solo aplica en habilitación
// ("2") — es el "set de pruebas" que la DIAN asigna para poder confirmar documentos de prueba.
export interface CreateNumberingRangePayload {
  dian_document_type_code: string;
  prefix: string;
  resolution_number: string;
  resolution_date: string; // YYYY-MM-DD
  range_from: number;
  range_to?: number;
  valid_from: string; // YYYY-MM-DD
  valid_to: string; // YYYY-MM-DD
  environment: IssuerEnvironment;
  technical_key?: string;
  test_set_id?: string;
  next_number?: number;
}

// Espejo de identificationDTO (apidian/internal/api/dto.go).
export interface Identification {
  number: string;
  type_code: string;
  verification_code?: string;
}

// Espejo de addressDTO.
export interface Address {
  line?: string;
  city_code?: string;
  city_name?: string;
  state_code?: string;
  state_name?: string;
  country_code?: string;
  country_name?: string;
}

// Espejo de partyDTO — usado tal cual por customers (catálogo de adquirientes) y, a futuro,
// como snapshot del cliente al emitir un documento. Deliberadamente SIN
// merchant_registration_number: ese campo "siempre es nil para el receptor"
// (cofacture/domain/types.go) — un cliente nunca lo tiene, no se pide en el formulario.
export interface CustomerPayload {
  entity_type_code?: string;
  identification: Identification;
  name: string;
  address?: Address;
  tax_scheme_code?: string;
  tax_scheme_name?: string;
  liability_codes?: string[];
  tax_regime_code?: string;
  phone?: string;
  email?: string;
}

export interface Customer extends CustomerPayload {
  id: string;
  created_at: string;
  updated_at: string;
}

export interface ListCustomersResult {
  customers: Customer[];
  count: number;
}

// Espejo de partyDTO para el catálogo de proveedores/terceros no obligados (Documento Soporte).
// Misma forma que CustomerPayload — los campos son idénticos porque ambos son domain.Party.
export interface VendorPayload {
  entity_type_code?: string;
  identification: Identification;
  name: string;
  address?: Address;
  tax_scheme_code?: string;
  tax_scheme_name?: string;
  liability_codes?: string[];
  tax_regime_code?: string;
  phone?: string;
  email?: string;
}

export interface Vendor extends VendorPayload {
  id: string;
  created_at: string;
  updated_at: string;
}

export interface ListVendorsResult {
  vendors: Vendor[];
  count: number;
}

// Espejo de productRequest/productResponse (apidian/internal/api/handler_products.go).
// Deliberadamente sin quantity/line_extension_cents/taxes (plural) — eso es dato de USO al
// armar una línea de documento, no del catálogo (ver products.Product). tax_type_code/
// tax_percent son un único impuesto por defecto, de conveniencia.
// item_type_code: selector de @schemeID (tabla 13.3.5, ver ItemStandard) — "001" UNSPSC/"010"
// GTIN/"020" Partida Arancelaria/"999" propio (vacío = "999"). item_type_name/
// item_type_agency_id ya NO se mandan: el backend los deriva del catálogo (ver
// docs/apidian-architecture.md sección 9.45) y solo aparecen en la respuesta (Product, abajo).
export interface ProductPayload {
  description: string;
  unit_code: string;
  unit_price_cents: number;
  item_code?: string;
  item_type_code?: string;
  tax_type_code?: string;
  tax_percent?: number;
}

export interface Product extends ProductPayload {
  id: string;
  item_type_name?: string;
  item_type_agency_id?: string;
  created_at: string;
  updated_at: string;
}

export interface ListProductsResult {
  products: Product[];
  count: number;
}

// Espejo de lineInputDTO (apidian/internal/api/dto.go) — sin line_extension_cents ni taxes[]:
// el backend calcula esos valores a partir de quantity/unit_price_cents/tax_percent (ver
// docs/apidian-architecture.md sección 9.37). tax_type_code vacío significa "sin impuesto en
// esta línea" — 0 o 1 impuesto por línea, el caso común.
export interface DocumentLineInput {
  description: string;
  quantity: number;
  unit_code: string;
  unit_price_cents: number;
  item_code?: string;
  // Selector de @schemeID (ver ProductPayload.item_type_code) — item_type_name/
  // item_type_agency_id ya no se mandan, se derivan en el servidor.
  item_type_code?: string;
  tax_type_code?: string;
  tax_percent?: number;
}

// Espejo de taxDTO — forma de SALIDA, ya calculada por el servidor.
export interface Tax {
  taxable_amount_cents: number;
  tax_amount_cents: number;
  percent: number;
  type_code: string;
  type_name: string;
}

// Espejo de lineDTO — forma de SALIDA de una línea ya guardada, con line_extension_cents/
// taxes[] ya calculados por el servidor. free_of_charge/reference_price (muestras
// comerciales) quedan fuera a propósito — no se construye UI para ese caso todavía.
export interface DocumentLine extends DocumentLineInput {
  line_extension_cents: number;
  item_type_name?: string;
  item_type_agency_id?: string;
  taxes?: Tax[];
}

// Espejo de paymentMeanDTO. code es la forma de pago (catálogo payment_terms, "1" contado/"2"
// crédito); payment_method_code es el medio de pago (catálogo payment_methods).
export interface PaymentMean {
  code: string;
  payment_method_code: string;
  due_date?: string;
  payment_reference?: string;
}

// Espejo de totalsDTO — siempre calculado por el servidor, nunca se manda en una petición.
export interface Totals {
  line_extension_cents: number;
  tax_exclusive_cents: number;
  tax_inclusive_cents: number;
  prepaid_cents?: number;
  payable_cents: number;
}

export type DocumentStatus = "draft" | "built" | "sent" | "accepted" | "rejected" | "send_error";

// Espejo de issueInvoiceRequest (apidian/internal/api/handler_documents.go). customer reusa
// CustomerPayload — partyDTO tiene exactamente esa forma, así que copiar un Customer guardado
// al armar una factura es directo (sin mapear campo por campo).
export interface IssueInvoicePayload {
  numbering_range_id: string;
  customer: CustomerPayload;
  lines: DocumentLineInput[];
  payment_means?: PaymentMean[];
  note?: string;
  currency_code?: string;
  customer_id?: string;
}

// Espejo de billingReferenceDTO — referencia a la factura que origina una Nota Crédito/Débito.
// Number es string (no number) porque el DTO lo serializa así aunque el documento fuente lo
// tenga como int64.
export interface BillingReference {
  prefix: string;
  number: string;
  cufe: string;
  issue_date: string;
}

// Espejo de discrepancyResponseDTO — opcional en Nota Crédito/Débito.
export interface DiscrepancyResponse {
  reference_id: string;
  response_code: string;
  description: string;
}

// Espejo de issueCreditNoteRequest — extiende issueNoteRequest con credit_note_type_code
// (DIAN List 22: 1=Devolución parcial, 2=Anulación, 3=Descuento, 4=Ajuste de precio, 5=Otros).
export interface IssueCreditNotePayload {
  numbering_range_id: string;
  customer: CustomerPayload;
  lines: DocumentLineInput[];
  payment_means?: PaymentMean[];
  note?: string;
  currency_code?: string;
  customer_id?: string;
  billing_reference: BillingReference;
  credit_note_type_code: string;
  discrepancy_response?: DiscrepancyResponse;
}

// Espejo de issueNoteRequest para Nota Débito — igual que IssueCreditNotePayload pero SIN
// credit_note_type_code (ND no tiene un código de concepto equivalente en el esquema DIAN).
// DiscrepancyResponse usa códigos de la misma List 22 pero específicos de ND:
// 1=Intereses, 2=Gastos por cobrar, 3=Cambio del valor.
export interface IssueDebitNotePayload {
  numbering_range_id: string;
  customer: CustomerPayload;
  lines: DocumentLineInput[];
  payment_means?: PaymentMean[];
  note?: string;
  currency_code?: string;
  customer_id?: string;
  billing_reference: BillingReference;
  discrepancy_response?: DiscrepancyResponse;
}

// Espejo de issueSupportDocumentRequest (apidian/internal/api/handler_support_documents.go).
// Vendor es el tercero no obligado (AccountingSupplierParty); la empresa emisora actúa de
// compradora y se deriva del token — no se envía en el payload.
// operation_type_code: "10" Residente / "11" No Residente.
// withholding_taxes: retenciones calculadas (ReteIVA "05", ReteRenta "06").
export interface IssueSupportDocumentPayload {
  numbering_range_id: string;
  vendor_id?: string;
  vendor: VendorPayload;
  lines: DocumentLineInput[];
  payment_means?: PaymentMean[];
  note?: string;
  currency_code?: string;
  operation_type_code: string;
  withholding_taxes?: Tax[];
}

// Espejo de issueAdjustmentNoteRequest (apidian/internal/api/handler_adjustment_notes.go).
// Igual que IssueSupportDocumentPayload pero con billing_reference obligatoria al DS original
// (usa CUDS en vez de CUFE) y discrepancy_response opcional para el motivo del ajuste.
export interface IssueAdjustmentNotePayload {
  numbering_range_id: string;
  vendor_id?: string;
  vendor: VendorPayload;
  lines: DocumentLineInput[];
  payment_means?: PaymentMean[];
  note?: string;
  currency_code?: string;
  operation_type_code: string;
  withholding_taxes?: Tax[];
  billing_reference: BillingReference;
  discrepancy_response?: DiscrepancyResponse;
}

// Espejo de documentResponse — cubre Factura, NC, ND, DS y NA (Nota de Ajuste DS).
// billing_reference/discrepancy_response presentes en NC/ND/NA; vendor/operation_type_code/
// withholding_taxes presentes en DS/NA.
export interface Document {
  id: string;
  issuer_id: string;
  numbering_range_id: string;
  dian_document_type_code: string;
  status: DocumentStatus;
  customer: CustomerPayload;
  lines: DocumentLine[];
  payment_means?: PaymentMean[];
  totals: Totals;
  note?: string;
  currency_code?: string;
  billing_reference?: BillingReference;
  discrepancy_response?: DiscrepancyResponse;
  note_type_code?: string;
  // Documento Soporte (dian_document_type_code "05")
  vendor?: CustomerPayload;
  operation_type_code?: string;
  withholding_taxes?: Tax[];
  prefix?: string;
  number?: number;
  document_key?: string;
  issue_date?: string;
  qr_url?: string;
  dian_track_id?: string;
  dian_status_code?: string;
  dian_status_description?: string;
  dian_status_message?: string;
  customer_id?: string;
  vendor_id?: string; // trazabilidad DS — nil si no se creó desde vendor guardado
  nc_count?: number; // cuántas NC referencian esta factura — solo en el listado
  nd_count?: number; // cuántas ND referencian esta factura — solo en el listado
  created_at: string;
  updated_at: string;
}

export interface ListDocumentsResult {
  documents: Document[];
  count: number;
}

export interface ListDocumentsFilter {
  dian_document_type_code?: string;
  status?: DocumentStatus;
  from?: string; // YYYY-MM-DD
  to?: string; // YYYY-MM-DD
  limit?: number;
  offset?: number;
  source_document_id?: string; // NC/ND que referencian la factura con este ID
}
