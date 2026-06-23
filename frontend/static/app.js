// devui — probador manual de api-dian. JS plano, sin build step, sin dependencias.
// Cada "acción" es básicamente una petición de Postman convertida en un botón: un método +
// ruta fijos, un cuerpo JSON editable con plantilla precargada, y captura automática de
// variables ({{nombre}}) hacia las siguientes acciones — mismo patrón que ya validamos en
// api-dian/postman/api-dian.postman_collection.json.

const state = {
  baseUrl: localStorage.getItem("devui.baseUrl") || "http://localhost:8080/api/v1",
  token: localStorage.getItem("devui.token") || "",
  vars: JSON.parse(localStorage.getItem("devui.vars") || "{}"),
};

function saveState() {
  localStorage.setItem("devui.baseUrl", state.baseUrl);
  localStorage.setItem("devui.token", state.token);
  localStorage.setItem("devui.vars", JSON.stringify(state.vars));
}

// substitute reemplaza {{nombre}} por el valor capturado — si todavía no existe, lo deja tal
// cual (para que sea obvio en el textarea que falta correr la acción que lo captura).
function substitute(text) {
  return text.replace(/\{\{(\w+)\}\}/g, (m, name) => (name in state.vars ? state.vars[name] : m));
}

function pretty(obj) {
  return JSON.stringify(obj, null, 2);
}

// ── Catálogo de acciones ─────────────────────────────────────────────────────────────────────

const sections = [
  {
    id: "auth",
    label: "0. Auth",
    desc: "El registro ya NO exige software/PIN/certificado — solo los datos que la DIAN pide del emisor. Eso se completa después, en la pestaña siguiente.",
    actions: [
      {
        id: "register",
        title: "Registrar emisor + usuario admin",
        method: "POST",
        path: "/auth/register",
        auth: false,
        body: pretty({
          issuer: {
            nit: "900373076",
            check_digit: "1",
            business_name: "Empresa de Prueba DevUI",
            identification_type_code: "31",
            department_code: "11",
            municipality_code: "11001",
            address_line: "Calle 1 # 2-3",
            email: "facturacion@empresa.test",
            environment: "1",
          },
          email: "admin@empresa.test",
          password: "contraseña-segura",
          name: "Admin de Prueba",
        }),
        capture: (b) => ({ token: b.token, issuerId: b.issuer?.id }),
      },
      {
        id: "login",
        title: "Iniciar sesión",
        method: "POST",
        path: "/auth/login",
        auth: false,
        body: pretty({ email: "admin@empresa.test", password: "contraseña-segura" }),
        capture: (b) => ({ token: b.token, issuerId: b.issuer?.id }),
      },
      {
        id: "get-my-issuer",
        title: "Consultar mi emisor",
        method: "GET",
        path: "/issuers/me",
      },
    ],
  },
  {
    id: "issuer-config",
    label: "0b. Configuración emisor",
    desc: "Completa software/PIN/certificado DESPUÉS del registro — cada campo es independiente (manda solo el que tengas a mano). Elige el archivo .p12 abajo: se convierte a base64 automáticamente, igual que lo haría un frontend real.",
    actions: [
      {
        id: "update-issuer",
        title: "Completar software + certificado",
        method: "PUT",
        path: "/issuers/me",
        certFileField: true,
        body: pretty({
          software_id: "software-id-de-prueba",
          software_pin: "12345",
          certificate_base64: "",
          certificate_password: "clave-de-prueba",
        }),
      },
    ],
  },
  {
    id: "numbering",
    label: "1. Resoluciones",
    desc: "numbering_ranges en el código — lo que en lenguaje de negocio se llama 'resolución de numeración'.",
    actions: [
      {
        id: "create-range-invoice",
        title: "Crear rango — Factura (01)",
        method: "POST",
        path: "/numbering-ranges",
        body: pretty({
          dian_document_type_code: "01",
          prefix: "SETP",
          resolution_number: "18760000001",
          resolution_date: "2024-01-01",
          range_from: 1,
          range_to: 5000,
          valid_from: "2024-01-01",
          valid_to: "2027-01-01",
          environment: "2",
          technical_key: "clave-tecnica-de-prueba",
        }),
        capture: (b) => ({ invoiceRangeId: b.id }),
      },
      {
        id: "create-range-creditnote",
        title: "Crear rango — Nota Crédito (91)",
        method: "POST",
        path: "/numbering-ranges",
        body: pretty({
          dian_document_type_code: "91",
          prefix: "SETPNC",
          resolution_number: "18760000002",
          resolution_date: "2024-01-01",
          range_from: 1,
          range_to: 5000,
          valid_from: "2024-01-01",
          valid_to: "2027-01-01",
          environment: "2",
        }),
        capture: (b) => ({ creditNoteRangeId: b.id }),
      },
      {
        id: "create-range-debitnote",
        title: "Crear rango — Nota Débito (92)",
        method: "POST",
        path: "/numbering-ranges",
        body: pretty({
          dian_document_type_code: "92",
          prefix: "SETPND",
          resolution_number: "18760000003",
          resolution_date: "2024-01-01",
          range_from: 1,
          range_to: 5000,
          valid_from: "2024-01-01",
          valid_to: "2027-01-01",
          environment: "2",
        }),
        capture: (b) => ({ debitNoteRangeId: b.id }),
      },
      { id: "list-ranges", title: "Listar mis rangos", method: "GET", path: "/numbering-ranges" },
    ],
  },
  {
    id: "documents",
    label: "2. Documentos",
    desc: "Crear ya no firma ni envía — solo persiste un borrador (status: draft). Confirmar es el único paso que reclama el número real, firma, y envía si el ambiente lo permite.",
    actions: [
      {
        id: "create-invoice-draft",
        title: "Crear borrador de Factura",
        method: "POST",
        path: "/invoices",
        body: pretty({
          numbering_range_id: "{{invoiceRangeId}}",
          customer: { identification: { number: "222222222222", type_code: "13" }, name: "Consumidor Final" },
          lines: [
            {
              description: "Servicio de prueba (devui)",
              quantity: 1,
              unit_code: "94",
              line_extension_cents: 10000,
              unit_price_cents: 10000,
              taxes: [{ taxable_amount_cents: 10000, tax_amount_cents: 0, percent: 0, type_code: "01", type_name: "IVA" }],
            },
          ],
        }),
        capture: (b) => ({ invoiceDocumentId: b.id }),
      },
      {
        id: "update-invoice-draft",
        title: "Editar borrador de Factura",
        method: "PUT",
        path: "/invoices/{{invoiceDocumentId}}",
        body: pretty({
          numbering_range_id: "{{invoiceRangeId}}",
          customer: { identification: { number: "222222222222", type_code: "13" }, name: "Consumidor Final" },
          lines: [
            {
              description: "Servicio de prueba (devui, corregido)",
              quantity: 1,
              unit_code: "94",
              line_extension_cents: 10000,
              unit_price_cents: 10000,
              taxes: [{ taxable_amount_cents: 10000, tax_amount_cents: 0, percent: 0, type_code: "01", type_name: "IVA" }],
            },
          ],
        }),
      },
      {
        id: "confirm-invoice",
        title: "Confirmar documento (Factura)",
        method: "POST",
        path: "/documents/{{invoiceDocumentId}}/confirm",
        capture: (b) => ({ invoicePrefix: b.prefix, invoiceCufe: b.document_key }),
      },
      {
        id: "create-creditnote-draft",
        title: "Crear borrador de Nota Crédito (referencia la Factura)",
        method: "POST",
        path: "/credit-notes",
        body: pretty({
          numbering_range_id: "{{creditNoteRangeId}}",
          customer: { identification: { number: "222222222222", type_code: "13" }, name: "Consumidor Final" },
          lines: [
            {
              description: "Anulación de servicio de prueba (devui)",
              quantity: 1,
              unit_code: "94",
              line_extension_cents: 10000,
              unit_price_cents: 10000,
              taxes: [{ taxable_amount_cents: 10000, tax_amount_cents: 0, percent: 0, type_code: "01", type_name: "IVA" }],
            },
          ],
          credit_note_type_code: "2",
          billing_reference: { prefix: "{{invoicePrefix}}", number: "1", cufe: "{{invoiceCufe}}", issue_date: "2026-06-20" },
          discrepancy_response: { reference_id: "{{invoicePrefix}}1", response_code: "2", description: "Anulación de factura electrónica" },
        }),
        capture: (b) => ({ creditNoteDocumentId: b.id }),
      },
      {
        id: "confirm-creditnote",
        title: "Confirmar documento (Nota Crédito)",
        method: "POST",
        path: "/documents/{{creditNoteDocumentId}}/confirm",
      },
      {
        id: "create-debitnote-draft",
        title: "Crear borrador de Nota Débito (referencia la Factura)",
        method: "POST",
        path: "/debit-notes",
        body: pretty({
          numbering_range_id: "{{debitNoteRangeId}}",
          customer: { identification: { number: "222222222222", type_code: "13" }, name: "Consumidor Final" },
          lines: [
            {
              description: "Intereses por mora (devui)",
              quantity: 1,
              unit_code: "94",
              line_extension_cents: 10000,
              unit_price_cents: 10000,
              taxes: [{ taxable_amount_cents: 10000, tax_amount_cents: 0, percent: 0, type_code: "01", type_name: "IVA" }],
            },
          ],
          billing_reference: { prefix: "{{invoicePrefix}}", number: "1", cufe: "{{invoiceCufe}}", issue_date: "2026-06-20" },
        }),
        capture: (b) => ({ debitNoteDocumentId: b.id }),
      },
      {
        id: "confirm-debitnote",
        title: "Confirmar documento (Nota Débito)",
        method: "POST",
        path: "/documents/{{debitNoteDocumentId}}/confirm",
      },
      { id: "get-invoice", title: "Consultar documento (Factura)", method: "GET", path: "/documents/{{invoiceDocumentId}}" },
      { id: "list-documents", title: "Listar mis documentos", method: "GET", path: "/documents" },
      {
        id: "create-scratch-draft",
        title: "Crear borrador descartable (para probar DELETE)",
        method: "POST",
        path: "/invoices",
        body: pretty({
          numbering_range_id: "{{invoiceRangeId}}",
          customer: { identification: { number: "222222222222", type_code: "13" }, name: "Consumidor Final" },
          lines: [{ description: "Borrador para eliminar", quantity: 1, unit_code: "94", line_extension_cents: 10000, unit_price_cents: 10000, taxes: [] }],
        }),
        capture: (b) => ({ scratchDocumentId: b.id }),
      },
      { id: "delete-draft", title: "Eliminar borrador", method: "DELETE", path: "/documents/{{scratchDocumentId}}" },
    ],
  },
  {
    id: "customers",
    label: "3. Customers",
    desc: "Catálogo de conveniencia — nunca es la fuente de verdad del documento (eso sigue siendo el snapshot dentro de cada factura).",
    actions: [
      {
        id: "create-customer",
        title: "Crear cliente",
        method: "POST",
        path: "/customers",
        body: pretty({
          identification: { number: "900111222", type_code: "31" },
          name: "Cliente Recurrente S.A.S.",
          address: { line: "Calle 50", city_code: "11001", city_name: "Bogotá" },
          email: "cliente@recurrente.test",
          phone: "3001234567",
        }),
        capture: (b) => ({ customerId: b.id }),
      },
      { id: "list-customers", title: "Listar mis clientes", method: "GET", path: "/customers" },
      { id: "get-customer", title: "Consultar cliente", method: "GET", path: "/customers/{{customerId}}" },
      {
        id: "update-customer",
        title: "Actualizar cliente",
        method: "PUT",
        path: "/customers/{{customerId}}",
        body: pretty({ identification: { number: "900111222", type_code: "31" }, name: "Cliente Recurrente S.A.S. (actualizado)", email: "cliente@recurrente.test" }),
      },
      { id: "delete-customer", title: "Eliminar cliente", method: "DELETE", path: "/customers/{{customerId}}" },
    ],
  },
  {
    id: "products",
    label: "4. Products",
    desc: "Catálogo de ítems/servicios reutilizables — sin inventario/stock, solo los datos DIAN del ítem.",
    actions: [
      {
        id: "create-product",
        title: "Crear producto",
        method: "POST",
        path: "/products",
        body: pretty({
          description: "Licencia mensual de software",
          unit_code: "94",
          unit_price_cents: 5000000,
          item_code: "LIC-001",
          item_type_code: "999",
          item_type_name: "Estándar de adopción del contribuyente",
          tax_type_code: "01",
          tax_type_name: "IVA",
          tax_percent: 19,
        }),
        capture: (b) => ({ productId: b.id }),
      },
      { id: "list-products", title: "Listar mis productos", method: "GET", path: "/products" },
      { id: "get-product", title: "Consultar producto", method: "GET", path: "/products/{{productId}}" },
      {
        id: "update-product",
        title: "Actualizar producto",
        method: "PUT",
        path: "/products/{{productId}}",
        body: pretty({ description: "Licencia mensual de software", unit_code: "94", unit_price_cents: 6000000, tax_type_code: "01", tax_type_name: "IVA", tax_percent: 19 }),
      },
      { id: "delete-product", title: "Eliminar producto", method: "DELETE", path: "/products/{{productId}}" },
    ],
  },
];

// ── Motor de peticiones ──────────────────────────────────────────────────────────────────────

async function runAction(action, resultEl) {
  const path = substitute(action.path);
  const url = state.baseUrl.replace(/\/$/, "") + path;
  const headers = {};
  if (action.body !== undefined) headers["Content-Type"] = "application/json";
  if (action.auth !== false && state.token) headers["Authorization"] = "Bearer " + state.token;

  const opts = { method: action.method, headers };
  if (action.bodyValue !== undefined) {
    opts.body = substitute(action.bodyValue);
  }

  resultEl.innerHTML = "<em>enviando…</em>";
  try {
    const res = await fetch(url, opts);
    const text = await res.text();
    let json = null;
    try {
      json = text ? JSON.parse(text) : null;
    } catch {
      /* respuesta no-JSON (204, etc.) */
    }

    const statusClass = res.ok ? "status-ok" : "status-err";
    resultEl.innerHTML =
      `<div class="${statusClass}">${action.method} ${path} → ${res.status}</div>` +
      `<pre>${escapeHtml(json ? pretty(json) : text || "(sin cuerpo)")}</pre>`;

    if (res.ok && action.capture && json) {
      const captured = action.capture(json) || {};
      Object.entries(captured).forEach(([k, v]) => {
        if (v !== undefined && v !== null) state.vars[k] = v;
      });
      saveState();
      renderVars();
      updateAuthStatus();
    }
  } catch (err) {
    resultEl.innerHTML = `<div class="status-err">error de red: ${escapeHtml(String(err))}</div>`;
  }
}

function escapeHtml(s) {
  return s.replace(/[&<>]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" }[c]));
}

// ── Render ───────────────────────────────────────────────────────────────────────────────────

let activeSection = sections[0].id;

function renderVars() {
  const el = document.getElementById("vars-list");
  const entries = Object.entries(state.vars);
  el.innerHTML = entries.length
    ? entries.map(([k, v]) => `<span class="var">${k} = ${escapeHtml(String(v)).slice(0, 40)}</span>`).join("")
    : "<em>ninguna todavía — corre 'Registrar emisor' para empezar</em>";
}

function updateAuthStatus() {
  const el = document.getElementById("authStatus");
  if (state.token) {
    el.textContent = "sesión activa";
    el.className = "pill pill-on";
  } else {
    el.textContent = "sin sesión";
    el.className = "pill pill-off";
  }
}

function renderTabs() {
  const nav = document.getElementById("tabs");
  nav.innerHTML = "";
  sections.forEach((s) => {
    const btn = document.createElement("button");
    btn.textContent = s.label;
    btn.className = s.id === activeSection ? "active" : "";
    btn.onclick = () => {
      activeSection = s.id;
      renderTabs();
      renderContent();
    };
    nav.appendChild(btn);
  });
}

function renderContent() {
  const root = document.getElementById("content");
  root.innerHTML = "";
  const section = sections.find((s) => s.id === activeSection);

  const desc = document.createElement("p");
  desc.className = "section-desc";
  desc.textContent = section.desc;
  root.appendChild(desc);

  section.actions.forEach((action) => {
    const card = document.createElement("div");
    card.className = "card";

    const head = document.createElement("div");
    head.className = "card-head";
    head.innerHTML =
      `<span class="method method-${action.method}">${action.method}</span>` +
      `<h3>${action.title}</h3>` +
      `<span class="path">${action.path}</span>`;
    card.appendChild(head);

    let textarea = null;
    if (action.body !== undefined) {
      textarea = document.createElement("textarea");
      textarea.value = substitute(action.body);
      action.bodyValue = textarea.value;
      textarea.oninput = () => (action.bodyValue = textarea.value);
      card.appendChild(textarea);
    }

    if (action.certFileField) {
      const note = document.createElement("div");
      note.className = "card-note";
      note.textContent = "Elegí el archivo .p12/.pfx — se convierte a base64 y se inserta solo en certificate_base64:";
      card.appendChild(note);

      const fileInput = document.createElement("input");
      fileInput.type = "file";
      fileInput.accept = ".p12,.pfx";
      fileInput.onchange = async () => {
        const file = fileInput.files[0];
        if (!file) return;
        const buf = await file.arrayBuffer();
        const b64 = arrayBufferToBase64(buf);
        try {
          const parsed = JSON.parse(textarea.value);
          parsed.certificate_base64 = b64;
          textarea.value = pretty(parsed);
          action.bodyValue = textarea.value;
        } catch {
          alert("El JSON del cuerpo no es válido — corregilo antes de elegir el archivo.");
        }
      };
      card.appendChild(fileInput);
    }

    const row = document.createElement("div");
    row.className = "row";
    const sendBtn = document.createElement("button");
    sendBtn.textContent = "Enviar";
    const result = document.createElement("div");
    result.className = "result";
    sendBtn.onclick = () => runAction(action, result);
    row.appendChild(sendBtn);
    card.appendChild(row);
    card.appendChild(result);

    root.appendChild(card);
  });
}

function arrayBufferToBase64(buf) {
  let binary = "";
  const bytes = new Uint8Array(buf);
  for (let i = 0; i < bytes.byteLength; i++) binary += String.fromCharCode(bytes[i]);
  return btoa(binary);
}

// ── Topbar ───────────────────────────────────────────────────────────────────────────────────

function initTopbar() {
  const baseUrlInput = document.getElementById("baseUrl");
  baseUrlInput.value = state.baseUrl;
  baseUrlInput.onchange = () => {
    state.baseUrl = baseUrlInput.value;
    saveState();
  };

  document.getElementById("btnLogout").onclick = () => {
    state.token = "";
    state.vars = {};
    saveState();
    renderVars();
    updateAuthStatus();
    renderContent();
  };

  updateAuthStatus();
}

initTopbar();
renderVars();
renderTabs();
renderContent();
