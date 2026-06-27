// Formas de los DTOs que expone apidian — ver apidian/internal/api/handler_auth.go y
// handler_issuers.go. Mantener en sincronía a mano (no hay generación automática todavía).

export interface User {
  id: string;
  email: string;
  name: string;
  role: string;
}

export type IssuerEnvironment = "1" | "2"; // 1 = producción, 2 = habilitación

export interface Issuer {
  id: string;
  nit: string;
  business_name: string;
  identification_type_code: string;
  environment: IssuerEnvironment;
  // Solo presencia (true/false) — el secreto en sí (software_pin/certificate/
  // certificate_password) nunca viaja de vuelta, ver issuerResponse en apidian.
  has_software_credentials: boolean;
  has_certificate: boolean;
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
  software_id?: string;
  software_pin?: string;
  certificate_base64?: string;
  certificate_password?: string;
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

// Espejo de numberingRangeResponse en apidian/internal/api/handler_issuers.go.
export interface NumberingRange {
  id: string;
  issuer_id: string;
  dian_document_type_code: string;
  prefix: string;
  range_from: number;
  range_to?: number;
  current_number: number;
  environment: IssuerEnvironment;
  is_active: boolean;
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

// Espejo de productRequest/productResponse (apidian/internal/api/handler_products.go).
// Deliberadamente sin quantity/line_extension_cents/taxes (plural) — eso es dato de USO al
// armar una línea de documento, no del catálogo (ver products.Product). tax_type_code/
// tax_type_name/tax_percent son un único impuesto por defecto, de conveniencia.
export interface ProductPayload {
  description: string;
  unit_code: string;
  unit_price_cents: number;
  item_code?: string;
  item_type_code?: string;
  item_type_name?: string;
  item_type_agency_id?: string;
  tax_type_code?: string;
  tax_type_name?: string;
  tax_percent?: number;
}

export interface Product extends ProductPayload {
  id: string;
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
  item_type_code?: string;
  item_type_name?: string;
  item_type_agency_id?: string;
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

// Espejo de documentResponse — solo los campos que aplican a Factura (Invoice). Nota Crédito/
// Nota Débito agregan billing_reference/discrepancy_response/note_type_code, que se quedan
// fuera de este tipo hasta que esos documentos tengan su propia UI.
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
  prefix?: string;
  number?: number;
  document_key?: string;
  issue_date?: string;
  qr_url?: string;
  signed_xml?: string;
  dian_track_id?: string;
  dian_status_code?: string;
  dian_status_description?: string;
  dian_status_message?: string;
  customer_id?: string;
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
}
