// api.js — cliente HTTP mínimo para api-dian. Un solo punto de entrada (request) para que
// el manejo de Authorization/errores/JSON viva en un solo lugar.

const BASE_URL_KEY = "dash.baseUrl";

export function getBaseUrl() {
  return localStorage.getItem(BASE_URL_KEY) || "http://localhost:8080/api/v1";
}

export function setBaseUrl(url) {
  localStorage.setItem(BASE_URL_KEY, url);
}

export class ApiError extends Error {
  constructor(status, message) {
    super(message);
    this.status = status;
  }
}

async function request(method, path, { body, auth = true, token } = {}) {
  const headers = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (auth && token) headers["Authorization"] = "Bearer " + token;

  const res = await fetch(getBaseUrl() + path, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  const text = await res.text();
  let json = null;
  if (text) {
    try {
      json = JSON.parse(text);
    } catch {
      // respuesta no-JSON — se deja json en null, el caller decide qué hacer
    }
  }

  if (!res.ok) {
    throw new ApiError(res.status, (json && json.error) || `error HTTP ${res.status}`);
  }
  return json;
}

export const api = {
  register: (body) => request("POST", "/auth/register", { body, auth: false }),
  login: (body) => request("POST", "/auth/login", { body, auth: false }),

  getMyIssuer: (token) => request("GET", "/issuers/me", { token }),
  updateMyIssuer: (token, body) => request("PUT", "/issuers/me", { token, body }),

  createNumberingRange: (token, body) => request("POST", "/numbering-ranges", { token, body }),
  listNumberingRanges: (token) => request("GET", "/numbering-ranges", { token }),

  createCustomer: (token, body) => request("POST", "/customers", { token, body }),
  listCustomers: (token) => request("GET", "/customers", { token }),
  updateCustomer: (token, id, body) => request("PUT", `/customers/${id}`, { token, body }),
  deleteCustomer: (token, id) => request("DELETE", `/customers/${id}`, { token }),

  createProduct: (token, body) => request("POST", "/products", { token, body }),
  listProducts: (token) => request("GET", "/products", { token }),
  updateProduct: (token, id, body) => request("PUT", `/products/${id}`, { token, body }),
  deleteProduct: (token, id) => request("DELETE", `/products/${id}`, { token }),

  createInvoiceDraft: (token, body) => request("POST", "/invoices", { token, body }),
  updateInvoiceDraft: (token, id, body) => request("PUT", `/invoices/${id}`, { token, body }),
  confirmDocument: (token, id) => request("POST", `/documents/${id}/confirm`, { token }),
  deleteDocument: (token, id) => request("DELETE", `/documents/${id}`, { token }),
  listDocuments: (token) => request("GET", "/documents", { token }),
  getDocument: (token, id) => request("GET", `/documents/${id}`, { token }),
};
