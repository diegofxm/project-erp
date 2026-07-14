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

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers);
  headers.set("Content-Type", "application/json");
  if (authToken) headers.set("Authorization", `Bearer ${authToken}`);

  const res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers });

  // 204 No Content u otras respuestas sin body — evita que res.json() reviente.
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;

  if (!res.ok) {
    const message = (data && typeof data === "object" && "error" in data ? String(data.error) : null) ?? `Error ${res.status}`;
    throw new ApiError(res.status, message);
  }
  return data as T;
}

// getBlob es para respuestas binarias (ej. el PDF de una factura, ver lib/documents.ts) — la
// función request() de arriba siempre asume JSON (res.text() + JSON.parse), incompatible con
// bytes de un PDF.
async function getBlob(path: string): Promise<Blob> {
  const headers = new Headers();
  if (authToken) headers.set("Authorization", `Bearer ${authToken}`);

  const res = await fetch(`${API_BASE_URL}${path}`, { method: "GET", headers });
  if (!res.ok) {
    const text = await res.text();
    const data = text ? JSON.parse(text) : null;
    const message = (data && typeof data === "object" && "error" in data ? String(data.error) : null) ?? `Error ${res.status}`;
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
