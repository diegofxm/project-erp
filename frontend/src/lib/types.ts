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

// Módulo del sidebar que un plan puede desbloquear — catálogo fijo, GET /admin/modules.
export interface SaasModule {
  code: string;
  name: string;
  description: string;
}

export type BillingCycle = "monthly" | "annual" | "none";

// Plan de suscripción — GET /admin/plans. Precios en centavos (igual que accounting/electronic).
export interface Plan {
  id: string;
  code: string;
  name: string;
  description: string;
  billing_cycle: BillingCycle;
  price_cents: number;
  included_documents: number | null; // null = ilimitado
  price_per_extra_document_cents: number;
  requires_certificate: boolean;
  certificate_price_cents: number; // se suma a price_cents si la empresa no trae su propio certificado
  annual_increment_pct: number; // ej. 5.5 = 5.5%
  is_internal: boolean; // uso interno de la plataforma, excluido del catálogo público
  is_active: boolean;
  modules: string[]; // códigos de SaasModule que este plan desbloquea
  created_at: string;
  updated_at: string;
}

// Suscripción activa de una empresa — GET /admin/companies/{id}/subscription.
export interface Subscription {
  id: string;
  company_id: string;
  plan_id: string;
  has_own_certificate: boolean;
  status: "active" | "cancelled" | "suspended";
  contracted_price_cents: number;
  current_period_start: string;
  current_period_end: string;
  cert_expires_at?: string;
}

// Configuración global de la plataforma — hoy solo la tasa de IVA. GET/PATCH /admin/settings.
export interface SaasSettings {
  iva_rate_bp: number; // puntos básicos: 1900 = 19%
  updated_at: string;
}

// Configuración de personalización de LA empresa activa (no de la plataforma) — GET/PATCH
// /companies/active/settings, ver erp/internal/company/interfaces/http/handlers.go.
export interface CompanySettings {
  brand_color: string;
}

// Entrada del resumen de facturación del período vigente — GET /admin/billing/summary.
export interface BillingEntry {
  company_id: string;
  business_name: string;
  nit: string;
  plan_name: string;
  documents_included: number | null;
  documents_used: number;
  overage_documents: number;
  base_cents: number;
  overage_cents: number;
  iva_cents: number;
  total_cents: number;
}

// Entrada de renovaciones próximas — GET /admin/billing/renewals.
export interface RenewalEntry {
  company_id: string;
  business_name: string;
  nit: string;
  plan_name: string;
  current_period_end: string;
  days_until_renewal: number; // negativo = ya venció
  renewal_cents: number;
}

export interface Payment {
  id: string;
  company_id: string;
  subscription_id?: string;
  type: "plan" | "certificate" | "overage";
  amount_cents: number;
  note: string;
  paid_at: string;
}

// Usuario de la plataforma (todas las empresas) — GET /admin/users, solo lectura.
// invite_accepted_at vacío = usuario invitado que aún no ha configurado su contraseña.
export interface AdminUser {
  id: string;
  email: string;
  name: string;
  role: string;
  is_superadmin: boolean;
  is_active: boolean;
  invite_accepted_at?: string;
  created_at: string;
}

// Foto mínima de una empresa para el panel superadmin — GET /admin/companies/{id}.
export interface CompanyInfo {
  id: string;
  business_name: string;
  trade_name: string;
  nit: string;
}

// Plan contratado por la empresa activa — GET /saas/my-plan. Alimenta la página "Mi plan" y el
// gating de módulos del Sidebar.
export interface MyPlan {
  plan_name: string;
  modules: string[];
  included_documents: number | null;
  documents_used: number;
  current_period_end: string;
  contracted_cents: number;
  has_own_certificate: boolean;
  cert_expires_at?: string;
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
  reviewed_at?: string;
  created_at: string;
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
  // Solo aplica al catálogo de clientes (CustomerForm) — ignorado cuando este payload se usa
  // para editar el cliente embebido en un documento electrónico.
  credit_limit?: number | null;
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
  // Cupo máximo de cartera (ventas confirmadas sin pagar) permitido — null = sin límite.
  // Se valida al confirmar una venta (ver sales ConfirmUseCase).
  credit_limit: number | null;
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
  // Punto de reorden ("stock mínimo") — 0 = sin umbral configurado. Solo aplica a productos
  // físicos (is_service=false); el módulo inventory lo usa para resaltar existencias bajas.
  min_stock: number;
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

export type DocumentStatus = "draft" | "built" | "sent" | "accepted" | "rejected" | "send_error" | "send_unknown" | "environment_mismatch";

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

// ── Ventas (erp/internal/sales) ────────────────────────────────────────────────
// A diferencia de DocumentLineInput (factura electrónica, montos en *_cents porque así lo
// exige el estándar DIAN), sales/ usa float64 en pesos directo — sin conversión en el borde,
// ver lib/currency.ts. Misma forma de línea para cotización y venta (SalesLine).

export interface SalesLineInput {
  product_id: string;
  description: string;
  quantity: number;
  unit_price: number;
  discount: number; // porcentaje 0-100, aplicado antes de impuestos
  tax_rate: number;
}

export interface SalesLine extends SalesLineInput {
  id: string;
  subtotal: number;
  tax_amount: number;
  total: number;
}

export type QuoteStatus = "draft" | "sent" | "accepted" | "rejected" | "expired";

export interface CreateQuotePayload {
  customer_id: string;
  lines: SalesLineInput[];
  valid_until?: string; // YYYY-MM-DD, opcional
  notes: string;
}

export interface Quote {
  id: string;
  company_id: string;
  customer_id: string;
  number: string;
  status: QuoteStatus;
  issue_date: string;
  valid_until?: string;
  notes: string;
  lines: SalesLine[];
  created_at: string;
  updated_at: string;
}

export type SaleStatus = "draft" | "confirmed" | "cancelled";

export interface CreateSalePayload {
  customer_id: string;
  number: string;
  issue_date: string; // YYYY-MM-DD
  due_date?: string;  // YYYY-MM-DD, vencimiento de cartera
  notes: string;
  lines: SalesLineInput[];
}

export interface Sale {
  id: string;
  company_id: string;
  customer_id: string;
  number: string;
  status: SaleStatus;
  issue_date: string;
  due_date?: string;
  notes: string;
  lines: SalesLine[];
  // Factura electrónica ya generada desde esta venta, si alguna — cuando está presente, no se
  // puede volver a generar (ver electronic CreateFromSaleUseCase).
  invoice_document_id?: string;
  created_at: string;
  updated_at: string;
}

export type PaymentMethod = "cash" | "transfer" | "check" | "card" | "other";

export interface RecordPaymentPayload {
  sale_id: string;
  payment_date?: string; // YYYY-MM-DD
  amount: number;
  payment_method: PaymentMethod;
  reference: string;
  notes: string;
}

export interface SalePayment {
  id: string;
  company_id: string;
  sale_id: string;
  payment_date: string;
  amount: number;
  payment_method: PaymentMethod;
  reference: string;
  notes: string;
  created_at: string;
}

export interface ReceivableBalance {
  sale_id: string;
  sale_number: string;
  customer_id: string;
  issue_date: string;
  due_date?: string;
  total: number;
  paid: number;
  balance: number;
}

// ── Compras (erp/internal/purchase) ────────────────────────────────────────────
// Mismo criterio que sales: float64 en pesos directo, sin conversión a centavos.

export interface PurchaseLineInput {
  product_id: string;
  description: string;
  quantity: number;
  unit_price: number;
  discount: number; // porcentaje 0-100, aplicado antes de impuestos
  tax_rate: number;
}

export interface PurchaseLine extends PurchaseLineInput {
  id: string;
  subtotal: number;
  tax_amount: number;
  total: number;
}

export type PurchaseStatus = "draft" | "confirmed" | "received" | "cancelled";

export interface CreatePurchasePayload {
  supplier_id: string;
  number: string;
  issue_date: string; // YYYY-MM-DD
  due_date?: string;  // YYYY-MM-DD, recepción esperada
  notes: string;
  lines: PurchaseLineInput[];
}

export interface Purchase {
  id: string;
  company_id: string;
  supplier_id: string;
  number: string;
  status: PurchaseStatus;
  issue_date: string;
  due_date?: string;
  notes: string;
  lines: PurchaseLine[];
  withholdings: PurchaseWithholding[];
  // Documento Soporte ya generado desde esta orden, si alguno — cuando está presente, no se
  // puede volver a generar (ver electronic CreateFromPurchaseUseCase).
  support_document_id?: string;
  created_at: string;
  updated_at: string;
}

export interface RecordPurchasePaymentPayload {
  purchase_id: string;
  payment_date?: string; // YYYY-MM-DD
  amount: number;
  payment_method: PaymentMethod;
  reference: string;
  notes: string;
}

export interface PurchasePayment {
  id: string;
  company_id: string;
  purchase_id: string;
  payment_date: string;
  amount: number;
  payment_method: PaymentMethod;
  reference: string;
  notes: string;
  created_at: string;
}

export interface PayableBalance {
  purchase_id: string;
  purchase_number: string;
  supplier_id: string;
  issue_date: string;
  due_date?: string;
  total: number;
  paid: number;
  balance: number;
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

// ── Bodegas (erp/internal/company, dueño del catálogo) ─────────────────────────

export interface Warehouse {
  id: string;
  company_id: string;
  code: string;
  name: string;
  address: string;
  // La bodega que usan ventas/compras cuando el documento no elige una explícitamente. Solo
  // una por empresa — ver WarehousesPage "Marcar por defecto".
  is_default: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface WarehousePayload {
  code: string;
  name: string;
  address: string;
}

// ── Inventario (erp/internal/inventory) ─────────────────────────────────────────

export type MovementType = "entry" | "exit" | "transfer" | "adjust";

export interface StockEntry {
  id: string;
  company_id: string;
  product_id: string;
  warehouse_id: string;
  quantity: number;
  updated_at: string;
}

export interface Movement {
  id: string;
  company_id: string;
  number: string;
  product_id: string;
  warehouse_id: string;
  type: MovementType;
  quantity: number;
  reference?: string;
  description?: string;
  // Solo en movimientos type=transfer — enlaza el par salida/entrada de un mismo traslado.
  transfer_group_id?: string;
  created_at: string;
}

export interface MovementPayload {
  product_id: string;
  warehouse_id: string;
  type: MovementType;
  quantity: number;
  reference?: string;
  description?: string;
  // Requerido cuando type="transfer" — bodega destino.
  to_warehouse_id?: string;
}

// ── Contabilidad (erp/internal/accounting) ──────────────────────────────────────
// Los montos van en centavos (int64), como en electronic/ — NO en pesos como sales/purchase.
// PUC = Plan Único de Cuentas (Decreto 2650), catálogo global compartido por todas las empresas.

export interface Account {
  id: string;
  code: string;
  name: string;
  parent_code: string;
  level: number;
  category: string;
  is_posting: boolean; // solo las cuentas "de movimiento" (nivel más detallado) aceptan asientos
  is_active: boolean;
}

export type PeriodStatus = "OPEN" | "CLOSED";

export interface AccountingPeriod {
  id: string;
  company_id: string;
  year: number;
  month: number;
  status: PeriodStatus;
  opened_at: string;
  closed_at?: string;
}

export type JournalStatus = "DRAFT" | "POSTED" | "VOID";
export type JournalBook = "BOTH" | "PCGA" | "NIIF";

export interface JournalLine {
  id: string;
  account_id: string;
  account_code: string;
  debit: number;  // centavos
  credit: number; // centavos
  third_party_nit: string;
  cost_center: string;
  description: string;
  foreign_amount: number;
  foreign_currency: string;
}

export interface JournalEntry {
  id: string;
  company_id: string;
  period_id: string;
  date: string;
  description: string;
  status: JournalStatus;
  source: string;
  entry_type: string;
  voucher_type: string;
  voucher_number: string;
  source_document_id?: string;
  source_document_type?: string;
  book: JournalBook;
  lines: JournalLine[];
  created_at: string;
}

export interface PostJournalLinePayload {
  account_code: string;
  debit_cents: number;
  credit_cents: number;
  third_party_nit?: string;
  cost_center?: string;
  description?: string;
}

export interface PostJournalPayload {
  date: string; // YYYY-MM-DD
  description: string;
  book?: JournalBook;
  lines: PostJournalLinePayload[];
}

// Balance neto de una cuenta (P&L o BS) — balance positivo = naturaleza débito.
export interface AccountBalance {
  account_id: string;
  account_code: string;
  account_name: string;
  category: string;
  balance: number;
}

export interface TrialBalanceRow {
  account_id: string;
  account_code: string;
  account_name: string;
  category: string;
  debit: number;
  credit: number;
  balance: number;
}

export interface LedgerLine {
  journal_id: string;
  date: string;
  description: string;
  voucher_type: string;
  voucher_number: string;
  debit: number;
  credit: number;
  running_balance: number;
}

export interface VoucherType {
  id: string;
  code: string;
  name: string;
  resets_annually: boolean;
  is_active: boolean;
}

// ── Retenciones (erp/internal/purchase + accounting) ────────────────────────────

export interface WithholdingConcept {
  id: string;
  code: string;
  name: string;
  type: string; // RETEFUENTE | RETEIVA | RETEICA
  rate_bp: number;
  min_base_uvt: number;
  account_payable: string;
  account_receivable: string;
  applicable_to: string; // JURIDICA | NATURAL | BOTH
}

export interface PurchaseWithholding {
  id: string;
  purchase_order_id: string;
  concept_code: string;
  concept_name: string;
  base: number;
  rate_bp: number;
  amount: number;
  account_payable: string;
  created_at: string;
}

export interface WithholdingCertificate {
  id: string;
  number: string;
  fiscal_year: number;
  third_party_nit: string;
  concept_code: string;
  concept_name: string;
  wh_type: string;
  // En pesos (no centavos) — espejo de PurchaseWithholding.Base/Amount, no de journal_lines.
  gross_amount: number;
  tax_withheld: number;
  status: string;
  issued_at?: string;
}

// ── Bancos (erp/internal/accounting) ─────────────────────────────────────────────

export interface BankAccount {
  id: string;
  name: string;
  bank_name: string;
  account_no: string;
  account_id: string;
  is_active: boolean;
}

export interface StatementLine {
  id: string;
  bank_account_id: string;
  date: string;
  description: string;
  debit: number;
  credit: number;
  reference: string;
  is_reconciled: boolean;
  journal_line_id?: string;
}

export interface ReconciliationCandidate {
  line_id: string;
  journal_id: string;
  date: string;
  description: string;
  voucher_type: string;
  voucher_number: string;
  debit: number;
  credit: number;
}

// ── Activos fijos (erp/internal/accounting) ──────────────────────────────────────

export type AssetStatus = "ACTIVE" | "FULLY_DEPRECIATED" | "DISPOSED";

export interface FixedAsset {
  id: string;
  code: string;
  name: string;
  description: string;
  asset_account: string;
  depreciation_account: string;
  accumulated_account: string;
  acquisition_date: string;
  acquisition_cost: number;
  salvage_value: number;
  useful_life_months: number;
  monthly_depreciation: number;
  accumulated: number;
  status: AssetStatus;
  third_party_nit?: string;
}

export interface DepreciationRun {
  id: string;
  run_date: string;
  status: string;
  journal_id?: string;
}

// ── Presupuestos (erp/internal/accounting) ───────────────────────────────────────

export type BudgetStatus = "DRAFT" | "APPROVED" | "CLOSED";

export interface Budget {
  id: string;
  year: number;
  name: string;
  status: BudgetStatus;
}

export interface BudgetLine {
  account_code: string;
  account_name: string;
  months: [number, number, number, number, number, number, number, number, number, number, number, number];
  total: number;
}

export interface BudgetActualRow {
  account_code: string;
  account_name: string;
  budgeted_months: [number, number, number, number, number, number, number, number, number, number, number, number];
  actual_months: [number, number, number, number, number, number, number, number, number, number, number, number];
}

// ── Declaraciones de impuestos (erp/internal/accounting) ─────────────────────────

export type DeclarationStatus = "DRAFT" | "FILED" | "PAID" | "CORRECTED";

export interface IVADeclaration {
  id: string;
  period_start: string;
  period_end: string;
  period_type: string;
  generated_iva: number;
  deductible_iva: number;
  net_iva: number;
  previous_balance: number;
  amount_to_pay: number;
  carry_forward: number;
  status: DeclarationStatus;
}

export interface IncomeTaxDeclaration {
  id: string;
  fiscal_year: number;
  taxable_income: number;
  tax_rate_bp: number;
  tax_computed: number;
  tax_to_pay: number;
  amount_due: number;
  status: DeclarationStatus;
}

export interface ICATariff {
  id: string;
  municipality_code: string;
  ciiu_code: string;
  fiscal_year: number;
  rate_bp: number;
  surcharge_bp: number;
}

export interface ICADeclaration {
  id: string;
  municipality_code: string;
  period_start: string;
  period_end: string;
  ciiu_code: string;
  gross_revenue: number;
  net_base: number;
  tax_computed: number;
  surcharge_amount: number;
  tax_to_pay: number;
  previous_balance: number;
  amount_due: number;
  carry_forward: number;
  status: DeclarationStatus;
}

// ExchangeRate — TRM diaria, no está ligada a la empresa (dato de mercado).
export interface ExchangeRate {
  rate_date: string;
  from_currency: string;
  to_currency: string;
  rate: number;
  source: string;
  description: string;
}

// OpenLine — línea de asiento sin conciliar de una cuenta, candidata a cruzarse contra otra
// (conciliación de cuentas / cruce de partidas, distinto de la conciliación bancaria).
export interface OpenLine {
  line_id: string;
  journal_id: string;
  date: string;
  description: string;
  voucher_number: string;
  third_party_nit: string;
  debit_cents: number;
  credit_cents: number;
}
