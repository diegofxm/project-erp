const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "http://localhost:8080/api/v1";

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

let authToken: string | null = null;

export function setAuthToken(token: string | null) {
  authToken = token;
}

// onSessionExpired se registra desde AuthContext.tsx (interceptor global de 401, ver
// docs/auditorias/2026-08-09/05-frontend.md punto 21). apiClient no puede importar AuthContext
// directamente (sería una dependencia circular: AuthContext ya importa apiClient) así que la
// relación se invierte con este callback registrable.
let onSessionExpired: (() => void) | null = null;

export function setSessionExpiredHandler(handler: (() => void) | null) {
  onSessionExpired = handler;
}

// Rutas donde un 401 es una respuesta de negocio normal, no una señal de que el token dejó de ser
// válido -- no deben disparar el logout automático:
// - /auth/login: contraseña incorrecta (ErrInvalidPassword) — normalmente ni siquiera hay token
//   todavía, pero se excluye explícitamente por claridad.
// - /auth/password: contraseña ACTUAL incorrecta al cambiarla (ErrCurrentPasswordIncorrect) — el
//   usuario sigue autenticado, solo se equivocó tecleando; desconectarlo por eso sería un bug de
//   UX, no una medida de seguridad.
// - /auth/logout: si el logout mismo devuelve 401 (la sesión ya estaba muerta del lado del
//   servidor), no hay que disparar OTRO logout -- evita un loop entre logout() y el interceptor.
const SESSION_EXPIRY_EXEMPT_PATHS = ["/auth/login", "/auth/password", "/auth/logout"];

function isSessionExpired401(path: string, status: number, hadToken: boolean): boolean {
  if (status !== 401 || !hadToken) return false;
  return !SESSION_EXPIRY_EXEMPT_PATHS.some((p) => path.startsWith(p));
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers);
  headers.set("Content-Type", "application/json");
  const hadToken = authToken !== null;
  if (authToken) headers.set("Authorization", `Bearer ${authToken}`);

  const res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers });

  // 204 No Content u otras respuestas sin body — evita que res.json() reviente.
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;

  if (!res.ok) {
    const message = (data && typeof data === "object" && "error" in data ? String(data.error) : null) ?? `Error ${res.status}`;
    if (isSessionExpired401(path, res.status, hadToken)) onSessionExpired?.();
    throw new ApiError(res.status, message);
  }
  return data as T;
}

// getBlob es para respuestas binarias (ej. el PDF de una factura, ver lib/documents.ts) — la
// función request() de arriba siempre asume JSON (res.text() + JSON.parse), incompatible con
// bytes de un PDF.
async function getBlob(path: string): Promise<Blob> {
  const headers = new Headers();
  const hadToken = authToken !== null;
  if (authToken) headers.set("Authorization", `Bearer ${authToken}`);

  const res = await fetch(`${API_BASE_URL}${path}`, { method: "GET", headers });
  if (!res.ok) {
    const text = await res.text();
    const data = text ? JSON.parse(text) : null;
    const message = (data && typeof data === "object" && "error" in data ? String(data.error) : null) ?? `Error ${res.status}`;
    if (isSessionExpired401(path, res.status, hadToken)) onSessionExpired?.();
    throw new ApiError(res.status, message);
  }
  return res.blob();
}

export const apiClient = {
  get: <T>(path: string) => request<T>(path, { method: "GET" }),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "POST", body: body !== undefined ? JSON.stringify(body) : undefined }),
  put: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "PUT", body: body !== undefined ? JSON.stringify(body) : undefined }),
  patch: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: "PATCH", body: body !== undefined ? JSON.stringify(body) : undefined }),
  del: <T>(path: string) => request<T>(path, { method: "DELETE" }),
  getBlob,
};
