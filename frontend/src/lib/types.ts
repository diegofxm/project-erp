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
  tax_scheme_name?: string;
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
