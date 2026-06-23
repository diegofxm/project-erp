import { session, clearSession, isAuthenticated } from "./state.js";
import { el, clear } from "./ui.js";
import { renderLogin, renderRegister } from "./views/auth.js";
import { renderHome } from "./views/home.js";
import { renderCompany } from "./views/company.js";
import { renderCustomers } from "./views/customers.js";
import { renderProducts } from "./views/products.js";
import { renderInvoices } from "./views/invoices.js";

const NAV_ITEMS = [
  { hash: "#/", label: "Inicio", icon: "🏠" },
  { hash: "#/company", label: "Mi empresa", icon: "🏢" },
  { hash: "#/customers", label: "Clientes", icon: "👥" },
  { hash: "#/products", label: "Productos", icon: "📦" },
  { hash: "#/invoices", label: "Facturación", icon: "🧾" },
];

const PUBLIC_ROUTES = ["#/login", "#/register"];

function navigate(hash) {
  if (location.hash === hash) {
    render();
  } else {
    location.hash = hash;
  }
}

function render() {
  const hash = location.hash || "#/";
  const app = document.getElementById("app");

  if (!isAuthenticated()) {
    clear(app);
    if (hash === "#/register") {
      renderRegister(app, { onSuccess: () => navigate("#/"), goLogin: () => navigate("#/login") });
    } else {
      renderLogin(app, { onSuccess: () => navigate("#/"), goRegister: () => navigate("#/register") });
    }
    return;
  }

  if (PUBLIC_ROUTES.includes(hash)) {
    navigate("#/");
    return;
  }

  renderShell(app, hash);
}

function renderShell(app, hash) {
  clear(app);

  const sidebar = el(
    "nav",
    { class: "sidebar" },
    NAV_ITEMS.map((item) =>
      el(
        "a",
        { href: item.hash, class: "sidebar-link" + (hash === item.hash ? " active" : "") },
        [el("span", { class: "sidebar-icon" }, item.icon), item.label]
      )
    )
  );

  const topbar = el("header", { class: "topbar" }, [
    el("div", { class: "topbar-title" }, session.issuer?.business_name || "Mi empresa"),
    el("button", { class: "btn btn-secondary btn-sm", onclick: handleLogout }, "Cerrar sesión"),
  ]);

  const content = el("main", { class: "dashboard-content" });

  app.appendChild(el("div", { class: "dashboard-shell" }, [sidebar, el("div", { class: "dashboard-main" }, [topbar, content])]));

  dispatch(hash, content);
}

function dispatch(hash, content) {
  switch (hash) {
    case "#/company":
      renderCompany(content);
      break;
    case "#/customers":
      renderCustomers(content);
      break;
    case "#/products":
      renderProducts(content);
      break;
    case "#/invoices":
      renderInvoices(content);
      break;
    default:
      renderHome(content, { navigate });
  }
}

function handleLogout() {
  clearSession();
  navigate("#/login");
}

window.addEventListener("hashchange", render);
window.addEventListener("DOMContentLoaded", render);
render();
