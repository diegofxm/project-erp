// ui.js — helpers de DOM compartidos por las vistas. Nada de framework: solo azúcar sobre
// document.createElement para no repetir lo mismo en cada vista.

export function el(tag, attrs = {}, children = []) {
  const node = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (k === "class") node.className = v;
    else if (k === "html") node.innerHTML = v;
    else if (k.startsWith("on") && typeof v === "function") node.addEventListener(k.slice(2), v);
    else if (v !== undefined && v !== null) node.setAttribute(k, v);
  }
  (Array.isArray(children) ? children : [children]).forEach((c) => {
    if (c === undefined || c === null) return;
    node.appendChild(typeof c === "string" ? document.createTextNode(c) : c);
  });
  return node;
}

export function clear(node) {
  node.innerHTML = "";
}

// centsToPesos/pesosToCents — la API siempre habla en centavos enteros (unit_price_cents,
// line_extension_cents...); el usuario piensa en pesos. La conversión vive en un solo lugar
// para no repetir "* 100" / "/ 100" sueltos por las vistas.
export function centsToPesos(cents) {
  return (Number(cents) || 0) / 100;
}
export function pesosToCents(pesos) {
  return Math.round((Number(pesos) || 0) * 100);
}

export function formatMoney(cents) {
  return centsToPesos(cents).toLocaleString("es-CO", { style: "currency", currency: "COP", maximumFractionDigits: 2 });
}

export function formatDate(value) {
  if (!value) return "—";
  const d = new Date(value);
  if (isNaN(d.getTime())) return value;
  return d.toLocaleDateString("es-CO");
}

const STATUS_LABELS = {
  draft: { label: "Borrador", cls: "badge-draft" },
  built: { label: "Confirmado (sin enviar)", cls: "badge-built" },
  sent: { label: "Enviado, esperando DIAN", cls: "badge-sent" },
  accepted: { label: "Aceptado por la DIAN", cls: "badge-accepted" },
  rejected: { label: "Rechazado por la DIAN", cls: "badge-rejected" },
  send_error: { label: "Error al enviar", cls: "badge-error" },
};

export function statusBadge(status) {
  const info = STATUS_LABELS[status] || { label: status, cls: "badge-draft" };
  return el("span", { class: "badge " + info.cls }, info.label);
}

let toastTimer = null;
export function toast(message, kind = "error") {
  let box = document.getElementById("toast");
  if (!box) {
    box = el("div", { id: "toast" });
    document.body.appendChild(box);
  }
  box.className = "toast toast-" + kind;
  box.textContent = message;
  box.style.display = "block";
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => (box.style.display = "none"), 4500);
}

export function reportApiError(err) {
  toast(err.message || String(err));
}

// modal: muestra contentNode dentro de un overlay; devuelve close().
export function openModal(contentNode) {
  const overlay = el("div", { class: "modal-overlay" });
  const box = el("div", { class: "modal-box" }, [contentNode]);
  overlay.appendChild(box);
  overlay.addEventListener("click", (e) => {
    if (e.target === overlay) close();
  });
  document.body.appendChild(overlay);
  function close() {
    overlay.remove();
  }
  return close;
}
