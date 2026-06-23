import { api } from "../api.js";
import { session } from "../state.js";
import { el, clear, reportApiError } from "../ui.js";

export async function renderHome(root, { navigate }) {
  clear(root);
  root.appendChild(el("h1", { class: "view-title" }, `Hola, bienvenido a ${session.issuer?.business_name || "tu empresa"}`));
  root.appendChild(el("p", { class: "view-sub" }, "Resumen rápido de tu cuenta."));

  const grid = el("div", { class: "stat-grid" }, [
    statCard("loading-customers", "Clientes", "…"),
    statCard("loading-products", "Productos", "…"),
    statCard("loading-drafts", "Borradores sin confirmar", "…"),
    statCard("loading-confirmed", "Documentos confirmados", "…"),
  ]);
  root.appendChild(grid);

  const issuerReady = !!(session.issuer && session.issuer.id);
  if (!issuerReady) return;

  try {
    const [customers, products, documents] = await Promise.all([
      api.listCustomers(session.token),
      api.listProducts(session.token),
      api.listDocuments(session.token),
    ]);
    const drafts = (documents.documents || []).filter((d) => d.status === "draft").length;
    const confirmed = (documents.documents || []).filter((d) => d.status !== "draft").length;

    grid.replaceWith(
      el("div", { class: "stat-grid" }, [
        statCard("c1", "Clientes", String((customers.customers || []).length), () => navigate("#/customers")),
        statCard("c2", "Productos", String((products.products || []).length), () => navigate("#/products")),
        statCard("c3", "Borradores sin confirmar", String(drafts), () => navigate("#/invoices")),
        statCard("c4", "Documentos confirmados", String(confirmed), () => navigate("#/invoices")),
      ])
    );
  } catch (err) {
    reportApiError(err);
  }
}

function statCard(id, label, value, onclick) {
  return el("div", { class: "stat-card", id, onclick }, [el("div", { class: "stat-value" }, value), el("div", { class: "stat-label" }, label)]);
}
