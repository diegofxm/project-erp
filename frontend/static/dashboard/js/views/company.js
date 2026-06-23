import { api, ApiError } from "../api.js";
import { session, setSession } from "../state.js";
import { el, clear, reportApiError, toast, formatDate, openModal } from "../ui.js";

const DOC_TYPES = [
  ["01", "Factura electrónica de Venta"],
  ["91", "Nota Crédito"],
  ["92", "Nota Débito"],
];

export async function renderCompany(root) {
  clear(root);
  root.appendChild(el("h1", { class: "view-title" }, "Mi empresa"));
  root.appendChild(el("p", { class: "view-sub" }, "Datos de tu empresa ante la DIAN y la configuración que vas completando con el tiempo."));

  root.appendChild(companyInfoCard());
  root.appendChild(softwareCard());
  root.appendChild(certificateCard());

  const rangesSection = el("div");
  root.appendChild(rangesSection);
  await renderRanges(rangesSection);
}

// ── Tarjeta: datos de la empresa (solo lectura — no hay endpoint para editarlos) ───────────
function companyInfoCard() {
  const iss = session.issuer || {};
  return el("div", { class: "card" }, [
    el("h3", {}, "Datos de la empresa"),
    el("dl", { class: "kv" }, [
      el("dt", {}, "NIT"), el("dd", {}, iss.nit || "—"),
      el("dt", {}, "Razón social"), el("dd", {}, iss.business_name || "—"),
      el("dt", {}, "Tipo de identificación"), el("dd", {}, iss.identification_type_code || "—"),
      el("dt", {}, "Ambiente"), el("dd", {}, iss.environment === "1" ? "Producción" : "Pruebas (habilitación)"),
      el("dt", {}, "Estado"), el("dd", {}, iss.is_active ? "Activo" : "Inactivo"),
    ]),
  ]);
}

// ── Tarjeta: software DIAN ───────────────────────────────────────────────────────────────────
function softwareCard() {
  const softwareId = el("input", { placeholder: "ID asignado por la DIAN" });
  const softwarePin = el("input", { type: "password", placeholder: "PIN del software" });
  const msg = el("div", { class: "form-note" });

  const form = el("form", { onsubmit: handleSubmit }, [
    field("ID del software", softwareId),
    field("PIN del software", softwarePin),
    msg,
    el("button", { type: "submit", class: "btn btn-primary" }, "Guardar software"),
  ]);

  async function handleSubmit(e) {
    e.preventDefault();
    msg.textContent = "";
    const body = {};
    if (softwareId.value.trim()) body.software_id = softwareId.value.trim();
    if (softwarePin.value.trim()) body.software_pin = softwarePin.value.trim();
    if (Object.keys(body).length === 0) {
      msg.textContent = "Completa al menos un campo.";
      return;
    }
    try {
      const updated = await api.updateMyIssuer(session.token, body);
      setSession(session.token, updated);
      toast("Software guardado.", "ok");
      softwareId.value = "";
      softwarePin.value = "";
    } catch (err) {
      msg.textContent = err instanceof ApiError ? err.message : "No se pudo guardar.";
    }
  }

  return el("div", { class: "card" }, [
    el("h3", {}, "Software DIAN"),
    el("p", { class: "card-help" }, "El ID y el PIN que la DIAN te asignó al registrar tu software de facturación. Puedes guardarlos por separado del certificado."),
    form,
  ]);
}

// ── Tarjeta: certificado digital (drag & drop) ───────────────────────────────────────────────
function certificateCard() {
  let pendingBase64 = "";
  let pendingFileName = "";

  const dropzone = el("div", { class: "dropzone" }, [
    el("div", { class: "dropzone-text" }, "Arrastra aquí tu certificado .p12 / .pfx, o haz clic para elegirlo"),
  ]);
  const fileInput = el("input", { type: "file", accept: ".p12,.pfx", style: "display:none" });
  const fileName = el("div", { class: "form-note" });
  const password = el("input", { type: "password", placeholder: "Contraseña del certificado" });
  const msg = el("div", { class: "form-note" });

  dropzone.addEventListener("click", () => fileInput.click());
  dropzone.addEventListener("dragover", (e) => {
    e.preventDefault();
    dropzone.classList.add("dropzone-over");
  });
  dropzone.addEventListener("dragleave", () => dropzone.classList.remove("dropzone-over"));
  dropzone.addEventListener("drop", (e) => {
    e.preventDefault();
    dropzone.classList.remove("dropzone-over");
    if (e.dataTransfer.files[0]) handleFile(e.dataTransfer.files[0]);
  });
  fileInput.addEventListener("change", () => {
    if (fileInput.files[0]) handleFile(fileInput.files[0]);
  });

  async function handleFile(file) {
    const buf = await file.arrayBuffer();
    pendingBase64 = arrayBufferToBase64(buf);
    pendingFileName = file.name;
    fileName.textContent = `Archivo listo: ${pendingFileName} (${(buf.byteLength / 1024).toFixed(1)} KB)`;
  }

  const form = el("form", { onsubmit: handleSubmit }, [
    dropzone,
    fileInput,
    fileName,
    field("Contraseña del certificado", password),
    msg,
    el("button", { type: "submit", class: "btn btn-primary" }, "Guardar certificado"),
  ]);

  async function handleSubmit(e) {
    e.preventDefault();
    msg.textContent = "";
    const body = {};
    if (pendingBase64) body.certificate_base64 = pendingBase64;
    if (password.value.trim()) body.certificate_password = password.value.trim();
    if (Object.keys(body).length === 0) {
      msg.textContent = "Elige un archivo o escribe la contraseña.";
      return;
    }
    try {
      const updated = await api.updateMyIssuer(session.token, body);
      setSession(session.token, updated);
      toast("Certificado guardado.", "ok");
      pendingBase64 = "";
      password.value = "";
      fileName.textContent = "";
    } catch (err) {
      // El backend valida que el .p12 + contraseña sean legibles ANTES de guardar (ver
      // docs/api-dian-architecture.md sección 9.26-d) — si falla, el mensaje ya es claro.
      msg.textContent = err instanceof ApiError ? err.message : "No se pudo guardar.";
    }
  }

  return el("div", { class: "card" }, [
    el("h3", {}, "Certificado digital"),
    el("p", { class: "card-help" }, "Tu certificado .p12 se sube tal cual (sin convertir a otro formato) — se cifra antes de guardarlo y nunca se vuelve a mostrar."),
    form,
  ]);
}

function arrayBufferToBase64(buf) {
  let binary = "";
  const bytes = new Uint8Array(buf);
  for (let i = 0; i < bytes.byteLength; i++) binary += String.fromCharCode(bytes[i]);
  return btoa(binary);
}

// ── Resoluciones de numeración ───────────────────────────────────────────────────────────────
async function renderRanges(container) {
  clear(container);
  const card = el("div", { class: "card" });
  const head = el("div", { class: "card-head-row" }, [
    el("h3", {}, "Resoluciones de numeración"),
    el("button", { class: "btn btn-secondary", onclick: () => openRangeForm(container) }, "+ Nueva resolución"),
  ]);
  card.appendChild(head);
  card.appendChild(el("p", { class: "card-help" }, "Cada resolución autoriza un rango de números para un tipo de documento — la DIAN exige una por tipo, no se reutilizan entre Factura/Nota Crédito/Nota Débito."));

  const tableWrap = el("div");
  card.appendChild(tableWrap);
  container.appendChild(card);

  try {
    const data = await api.listNumberingRanges(session.token);
    const ranges = data.numbering_ranges || [];
    if (ranges.length === 0) {
      tableWrap.appendChild(el("p", { class: "empty-state" }, "Todavía no tienes resoluciones registradas."));
      return;
    }
    const table = el("table", { class: "table" }, [
      el("thead", {}, el("tr", {}, [el("th", {}, "Tipo"), el("th", {}, "Prefijo"), el("th", {}, "Rango"), el("th", {}, "Consecutivo actual"), el("th", {}, "Ambiente"), el("th", {}, "Estado")])),
      el(
        "tbody",
        {},
        ranges.map((r) =>
          el("tr", {}, [
            el("td", {}, docTypeLabel(r.dian_document_type_code)),
            el("td", {}, r.prefix),
            el("td", {}, `${r.range_from} – ${r.range_to ?? "sin tope"}`),
            el("td", {}, String(r.current_number)),
            el("td", {}, r.environment === "1" ? "Producción" : "Pruebas"),
            el("td", {}, r.is_active ? "Activa" : "Inactiva"),
          ])
        )
      ),
    ]);
    tableWrap.appendChild(table);
  } catch (err) {
    reportApiError(err);
  }
}

function docTypeLabel(code) {
  return (DOC_TYPES.find((d) => d[0] === code) || [code, code])[1];
}

function openRangeForm(container) {
  const docType = selectEl(DOC_TYPES, "01");
  const prefix = el("input", { required: "true", placeholder: "SETP" });
  const resolutionNumber = el("input", { required: "true", placeholder: "18760000001" });
  const resolutionDate = el("input", { type: "date", required: "true" });
  const rangeFrom = el("input", { type: "number", required: "true", value: "1" });
  const rangeTo = el("input", { type: "number", placeholder: "(opcional, sin tope si se deja vacío)" });
  const validFrom = el("input", { type: "date", required: "true" });
  const validTo = el("input", { type: "date", required: "true" });
  const environment = selectEl([["2", "Pruebas (habilitación)"], ["1", "Producción"]], "2");
  const technicalKeyWrap = el("div");
  const technicalKey = el("input", { placeholder: "Clave técnica (solo Factura)" });
  technicalKeyWrap.appendChild(field("Clave técnica (solo Factura)", technicalKey));

  function syncTechnicalKeyVisibility() {
    technicalKeyWrap.style.display = docType.value === "01" ? "" : "none";
  }
  docType.addEventListener("change", syncTechnicalKeyVisibility);
  syncTechnicalKeyVisibility();

  const msg = el("div", { class: "form-error" });

  const form = el("form", { class: "modal-form", onsubmit: handleSubmit }, [
    el("h3", {}, "Nueva resolución"),
    el("div", { class: "form-grid" }, [
      field("Tipo de documento", docType),
      field("Prefijo", prefix),
      field("Número de resolución", resolutionNumber),
      field("Fecha de la resolución", resolutionDate),
      field("Número inicial", rangeFrom),
      field("Número final", rangeTo),
      field("Vigente desde", validFrom),
      field("Vigente hasta", validTo),
      field("Ambiente", environment),
    ]),
    technicalKeyWrap,
    msg,
    el("div", { class: "modal-actions" }, [el("button", { type: "submit", class: "btn btn-primary" }, "Guardar resolución")]),
  ]);

  const close = openModal(form);

  async function handleSubmit(e) {
    e.preventDefault();
    msg.textContent = "";
    const body = {
      dian_document_type_code: docType.value,
      prefix: prefix.value,
      resolution_number: resolutionNumber.value,
      resolution_date: resolutionDate.value,
      range_from: Number(rangeFrom.value),
      valid_from: validFrom.value,
      valid_to: validTo.value,
      environment: environment.value,
    };
    if (rangeTo.value.trim()) body.range_to = Number(rangeTo.value);
    if (docType.value === "01" && technicalKey.value.trim()) body.technical_key = technicalKey.value.trim();

    try {
      await api.createNumberingRange(session.token, body);
      close();
      toast("Resolución creada.", "ok");
      await renderRanges(container);
    } catch (err) {
      msg.textContent = err instanceof ApiError ? err.message : "No se pudo crear la resolución.";
    }
  }
}

function field(labelText, inputNode) {
  return el("label", { class: "field" }, [el("span", {}, labelText), inputNode]);
}

function selectEl(pairs, selected) {
  const node = el("select", {}, pairs.map(([value, label]) => el("option", { value }, label)));
  if (selected !== undefined) node.value = selected;
  return node;
}
