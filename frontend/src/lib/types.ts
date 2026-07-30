// Formas de los DTOs que expone el ERP — ver erp/internal/*/interfaces/http/handlers.go.
// Mantener en sincronía a mano (no hay generación automática todavía).

export interface User {
  id: string;
  email: string;
  name: string;
  role: string;
  is_superadmin: boolean;
}

export type CompanyEnvironment = "1" | "2"; // 1 = producción, 2 = habilitación

// Company espeja safeCompany() de erp/internal/company/interfaces/http/handlers.go.
export interface Company {
  id: string;
  nit: string;
  check_digit: string;
  business_name: string;
  trade_name?: string;
  identification_type_code: string;
  environment: CompanyEnvironment;

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

  // Credenciales DIAN
  software_id?: string;
  has_software_credentials: boolean;
  ne_software_id?: string;
  has_ne_software_credentials: boolean;
  has_certificate: boolean;
  // Metadatos del certificado — solo presentes cuando has_certificate es true y el ERP los soporta.
  certificate_subject?: string;
  certificate_issuer_cn?: string;
  certificate_expires_at?: string; // ISO 8601
  has_logo: boolean;
  logo_content_type?: string;

  is_active: boolean;
  created_at: string;
  updated_at: string;
}


// AuthResult espeja la respuesta de /auth/login, /auth/register, /auth/select-company.
export interface AuthResult {
  token: string;
  company_id?: string;
  user: User;
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

// Payload de creación de empresa — espeja CreateRequest en
// erp/internal/company/application/create.go (POST /api/v1/companies).
export interface CreateCompanyPayload {
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
  environment: CompanyEnvironment;
  entity_type_code?: string;
  tax_scheme_code?: string;
  tax_scheme_name?: string;
  liability_codes?: string[];
  tax_regime_code?: string;
  industry_classification_codes?: string[];
  merchant_registration_number?: string;
}

export interface ListCompaniesResult {
  companies: Company[];
  count: number;
}

// Actualizar credenciales DIAN — PUT /api/v1/companies/active/credentials.
// Si logo_base64 está presente se enruta a PUT /api/v1/companies/active/logo.
export interface UpdateCompanyPayload {
  // FE y DS comparten el mismo software
  software_id?: string;
  software_pin?: string;
  // NE (Nómina Electrónica) — software independiente
  ne_software_id?: string;
  ne_software_pin?: string;
  // Certificado compartido entre FE/DS/NE
  certificate_base64?: string;
  certificate_password?: string;
  // logo: enruta a PUT /companies/active/logo
  logo_base64?: string;
  logo_content_type?: string;
}

// Editar perfil de empresa — PUT /api/v1/companies/active.
export interface UpdateCompanyProfilePayload {
  business_name: string;
  trade_name: string;
  identification_type_code: string;
  department_code: string;
  municipality_code: string;
  address_line: string;
  email: string;
  phone: string;
  entity_type_code: string;
  tax_scheme_code: string;
  tax_scheme_name?: string;
  liability_codes: string[];
  tax_regime_code: string | null;
  industry_classification_codes: string[];
  merchant_registration_number: string | null;
  environment: CompanyEnvironment;
}

// Plan de suscripción — GET /admin/plans.
export interface Plan {
  id: string;
  name: string;
  description: string;
  max_documents_per_month: number | null; // null = ilimitado
  max_issuers: number;
  price_cop: number;
  affiliation_fee_cop: number;  // tarifa única de afiliación inicial
  renewal_fee_cop: number;      // tarifa de renovación anual
  annual_increment_pct: number; // porcentaje de incremento anual (ej. 5.5 = 5.5%)
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

// Suscripción activa de una empresa — GET /admin/companies/{id}/subscription.
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

// Configuración de personalización y tarifas de la empresa — GET/PATCH /companies/me/settings.
export interface CompanySettings {
  issuer_id: string;
  brand_color: string;
  price_per_document_cop: number;
  affiliation_fee_cop: number;
  renewal_fee_cop: number;
  affiliated_at: string | null;   // ISO 8601 o null si aún no está afiliado
  renewal_due_at: string | null;  // ISO 8601 o null si no aplica
  updated_at: string;
}

// Entrada del resumen de facturación mensual — GET /admin/billing/summary.
export interface BillingEntry {
  IssuerID: string;
  BusinessName: string;
  NIT: string;
  DocsThisMonth: number;
  PricePerDocumentCOP: number;
  SubtotalCOP: number;
  IVA: number;
  TotalCOP: number;
}

// Entrada de renovaciones próximas — GET /admin/billing/renewals.
export interface RenewalEntry {
  IssuerID: string;
  BusinessName: string;
  NIT: string;
  AffiliatedAt: string | null;
  RenewalDueAt: string | null;
  RenewalFeeCOP: number;
  DaysUntilRenewal: number; // 0 si ya venció
}

// Espejo de payments.Payment (apidian/internal/payments/model.go).
export interface Payment {
  id: string;
  issuer_id: string;
  type: "affiliation" | "renewal" | "documents";
  amount_cop: number;
  note: string;
  paid_at: string;
  created_at: string;
}

// Espejo de adminUserResponse (apidian/internal/api/handler_admin.go).
// invite_accepted_at null = usuario invitado que aún no ha configurado su contraseña.
export interface AdminUser {
  id: string;
  email: string;
  name: string;
  role: string;
  is_superadmin: boolean;
  is_active: boolean;
  invite_accepted_at: string | null;
  created_at: string;
}

export interface Prospect {
  id: string;
  name: string;
  email: string;
  nit?: string;
  has_cedula: boolean;
  has_rut: boolean;
  status: "pending" | "approved" | "rejected";
  notes?: string;
  reviewed_at?: string | null;
  created_at: string;
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

// Espejo de numberingRangeResponse en apidian/internal/api/handler_companies.go. status resume
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
  environment: CompanyEnvironment;
  is_active: boolean;
  status: NumberingRangeStatus;
}

export interface ListNumberingRangesResult {
  numbering_ranges: NumberingRange[];
  count: number;
}

// Rango devuelto por GET /dian/numbering-ranges — datos tal como los reporta la DIAN,
// sin dian_document_type_code ni test_set_id que son campos propios de apidian.
export interface DianRange {
  resolution_number: string;
  resolution_date: string; // YYYY-MM-DD
  prefix: string;
  range_from: number;
  range_to: number;
  valid_from: string; // YYYY-MM-DD
  valid_to: string;   // YYYY-MM-DD
  technical_key?: string;
  suggested_doc_type_code?: string; // sugerido cuando el prefijo es SETP/SEDS; "" si no reconocible
}

export interface GetDianNumberingRangesResult {
  ranges: DianRange[];
}

// Espejo de createNumberingRangeRequest. technical_key solo aplica cuando
// dian_document_type_code es "01" (Factura, CUFE); test_set_id solo aplica en habilitación
// ("2") — es el "set de pruebas" que la DIAN asigna para poder confirmar documentos de prueba.
export interface CreateNumberingRangePayload {
  dian_document_type_code: string;
  prefix: string;
  // Solo requeridos para FE (01) y DS (05); NC/ND/NA no tienen resolución DIAN
  resolution_number?: string;
  resolution_date?: string; // YYYY-MM-DD
  range_from: number;
  range_to?: number;
  valid_from?: string; // YYYY-MM-DD
  valid_to?: string; // YYYY-MM-DD
  environment: CompanyEnvironment;
  technical_key?: string;
  test_set_id?: string;
  next_number?: number;
}

// Espejo de identificationDTO — usado en snapshots de documentos (CustomerPayload/SupplierPayload).
export interface Identification {
  number: string;
  type_code: string;
  verification_code?: string;
}

// Espejo de addressDTO — usado en snapshots de documentos.
export interface Address {
  line?: string;
  city_code?: string;
  city_name?: string;
  state_code?: string;
  state_name?: string;
  country_code?: string;
  country_name?: string;
}

// CustomerPayload — snapshot del cliente dentro de un documento electrónico (IssueInvoicePayload).
// Mantiene la estructura anidada legacy porque el módulo electronic la espera así en el JSONB.
// Para el catálogo de clientes, ver Customer (campos planos del ERP).
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

// Customer — espejo de domain.Customer del ERP (campos planos, snake_case).
// Catálogo de adquirientes; NO confundir con CustomerPayload (snapshot en documentos).
export interface Customer {
  id: string;
  company_id: string;
  identification_type_code: string;
  identification_number: string;
  check_digit: string;
  entity_type_code: string;
  merchant_registration_number: string;
  name: string;
  tax_scheme_code: string;
  tax_scheme_name: string;
  tax_regime_code: string | null;
  liability_codes: string[];
  department_code: string;
  municipality_code: string;
  address_line: string;
  address_city_name: string;
  address_state_name: string;
  address_country_code: string;
  address_country_name: string;
  email: string;
  phone: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ListCustomersResult {
  customers: Customer[];
  count: number;
}

// SupplierPayload — snapshot del proveedor dentro de un Documento Soporte.
// Mantiene la estructura anidada legacy por las mismas razones que CustomerPayload.
export interface SupplierPayload {
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

// Supplier — espejo de domain.Supplier del ERP (campos planos).
// Catálogo de proveedores/terceros no obligados; NO confundir con SupplierPayload.
export interface Supplier {
  id: string;
  company_id: string;
  identification_type_code: string;
  identification_number: string;
  check_digit: string;
  entity_type_code: string;
  merchant_registration_number: string;
  name: string;
  tax_scheme_code: string;
  tax_scheme_name: string;
  tax_regime_code: string | null;
  liability_codes: string[];
  department_code: string;
  municipality_code: string;
  address_line: string;
  address_city_name: string;
  address_state_name: string;
  address_country_code: string;
  address_country_name: string;
  email: string;
  phone: string;
  payment_terms_days: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ListSuppliersResult {
  suppliers: Supplier[];
  count: number;
}

// ProductPayload — payload de creación/edición de producto en el catálogo ERP.
export interface ProductPayload {
  code: string;
  name: string;
  description: string;
  unit_measure_code: string;
  standard_code: string;
  standard_code_type: string;
  standard_code_id: string;
  standard_code_agency_id: string;
  is_service: boolean;
  tax_scheme_code: string;
  tax_scheme_name: string;
  tax_rate: number;
  base_price: number;
}

// Product — espejo de domain.Product del ERP (campos planos).
export interface Product extends ProductPayload {
  id: string;
  company_id: string;
  is_active: boolean;
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
// Supplier es el tercero no obligado (AccountingSupplierParty); la empresa emisora actúa de
// compradora y se deriva del token — no se envía en el payload.
// operation_type_code: "10" Residente / "11" No Residente.
// withholding_taxes: retenciones calculadas (ReteIVA "05", ReteRenta "06").
export interface IssueSupportDocumentPayload {
  numbering_range_id: string;
  supplier_id?: string;
  supplier: SupplierPayload;
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
  supplier_id?: string;
  supplier: SupplierPayload;
  lines: DocumentLineInput[];
  payment_means?: PaymentMean[];
  note?: string;
  currency_code?: string;
  operation_type_code: string;
  withholding_taxes?: Tax[];
  billing_reference: BillingReference;
  discrepancy_response?: DiscrepancyResponse;
}

// Espejo de relatedNoteDTO — NC/ND relacionadas con una FE, o NA relacionada con un DS.
// Solo presente en GET /documents/{id} para documentos confirmados (FE y DS).
export interface RelatedNote {
  id: string;
  dian_document_type_code: string;
  prefix?: string;
  number?: number;
  payable_cents: number;
  status: DocumentStatus;
  issue_date?: string;
}

// Espejo de documentResponse — cubre Factura, NC, ND, DS y NA (Nota de Ajuste DS).
// billing_reference/discrepancy_response presentes en NC/ND/NA; supplier/operation_type_code/
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
  supplier?: CustomerPayload;
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
  supplier_id?: string; // trazabilidad DS — nil si no se creó desde supplier guardado
  nc_count?: number; // cuántas NC referencian esta factura — solo en el listado
  nd_count?: number; // cuántas ND referencian esta factura — solo en el listado
  na_count?: number; // cuántas NA referencian este DS — solo en el listado
  // Solo en GET /documents/{id} para FE (01) y DS (05) confirmados.
  related_notes?: RelatedNote[];
  net_payable_cents?: number; // saldo neto = total − NC aceptadas + ND aceptadas (FE) o − NA (DS)
  // Solo en GET /documents/{id} para NC (91), ND (92) y NA (95).
  source_document_id?: string; // ID del FE o DS al que esta nota hace referencia
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
  search?: string;             // búsqueda libre: nombre cliente/proveedor o número de documento
}

// Espejo de auditEventResponse (apidian/internal/api/handler_audit.go).
export interface AuditEvent {
  id: string;
  user_name?: string;
  user_email?: string;
  action: string;
  resource_type?: string;
  resource_id?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
}

export interface ListAuditEventsResult {
  events: AuditEvent[];
  count: number;
}

export interface ListAuditEventsFilter {
  resource_id?: string;
  limit?: number;
  offset?: number;
}
