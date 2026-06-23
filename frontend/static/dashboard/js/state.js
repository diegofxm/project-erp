// state.js — sesión actual (token + emisor). Namespace "dash." separado del de devui
// (localStorage "devui.*") para no compartir ni pisar nada entre las dos herramientas.

const TOKEN_KEY = "dash.token";
const ISSUER_KEY = "dash.issuer";

export const session = {
  token: localStorage.getItem(TOKEN_KEY) || "",
  issuer: JSON.parse(localStorage.getItem(ISSUER_KEY) || "null"),
};

export function setSession(token, issuer) {
  session.token = token;
  session.issuer = issuer;
  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(ISSUER_KEY, JSON.stringify(issuer));
}

export function clearSession() {
  session.token = "";
  session.issuer = null;
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(ISSUER_KEY);
}

export function isAuthenticated() {
  return !!session.token;
}
