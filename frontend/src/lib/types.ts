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

// Payload mínimo de creación de empresa para esta fase — software_id/software_pin/
// certificate_base64 se completan después en la configuración del emisor (otra fase), no son
// obligatorios para crear la empresa (ver issuers.validateIssuer: solo exige nit/business_name/
// environment).
export interface CreateIssuerPayload {
  nit: string;
  check_digit: string;
  business_name: string;
  identification_type_code: string;
  department_code: string;
  municipality_code: string;
  address_line: string;
  email: string;
  environment: IssuerEnvironment;
}

export interface ListIssuersResult {
  issuers: Issuer[];
  count: number;
}
