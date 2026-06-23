import { api, ApiError } from "../api.js";
import { session } from "../state.js";
import { el, clear, reportApiError, toast, formatMoney, formatDate, statusBadge, pesosToCents, centsToPesos } from "../ui.js";

const INVOICE_TYPE = "01";

export async function renderInvoices(root) {
  await showList(root);
}

// ── Lista ────────────────────────────────────────────────────────────────────────────────────

async function showList(root) {
  clear(root);
  root.appendChild(
    el("div", { class: "view-head" }, [
      el("div", {}, [el("h1", { class: "view-title" }, "Facturación"), el("p", { class: "view-sub" }, "Crea el borrador, revísalo, y confirma cuando esté listo — confirmar es lo único que reclama un número real ante la DIAN.")]),
      el("button", { class: "btn btn-primary", onclick: () => showForm(root, null) }, "+ Nueva factura"),
    ])
  );

  const tableWrap = el("div");
  root.appendChild(tableWrap);

  try {
    const data = await api.listDocuments(session.token);
    const docs = (data.documents || []).filter((d) => d.dian_document_type_code === INVOICE_TYPE);
    if (docs.length === 0) {
      tableWrap.appendChild(el("p", { class: "empty-state" }, "Todavía no tienes facturas — crea la primera."));
      return;
    }
    docs.sort((a, b) => new Date(b.created_at) - new Date(a.created_at));
    tableWrap.appendChild(
      el("table", { class: "table" }, [
        el("thead", {}, el("tr", {}, [el("th", {}, "Cliente"), el("th", {}, "Número"), el("th", {}, "Total"), el("th", {}, "Estado"), el("th", {}, "Fecha"), el("th", {}, "")])),
        el(
          "tbody",
          {},
          docs.map((d) =>
            el("tr", {}, [
              el("td", {}, d.customer?.name || "—"),
              el("td", {}, d.prefix ? `${d.prefix}-${d.number}` : "Sin confirmar"),
              el("td", {}, formatMoney(d.totals?.payable_cents || 0)),
              el("td", {}, statusBadge(d.status)),
              el("td", {}, formatDate(d.issue_date || d.created_at)),
              el("td", { class: "table-actions" }, [el("button", { class: "btn btn-secondary btn-sm", onclick: () => showDetail(root, d.id) }, "Ver")]),
            ])
          )
        ),
      ])
    );
  } catch (err) {
    reportApiError(err);
  }
}

// ── Formulario crear/editar ──────────────────────────────────────────────────────────────────

async function showForm(root, existingDoc) {
  clear(root);
  const isEdit = !!existingDoc;
  root.appendChild(el("h1", { class: "view-title" }, isEdit ? "Editar borrador" : "Nueva factura"));

  let ranges = [],
    customers = [],
    products = [];
  try {
    const [rangesRes, customersRes, productsRes] = await Promise.all([api.listNumberingRanges(session.token), api.listCustomers(session.token), api.listProducts(session.token)]);
    ranges = (rangesRes.numbering_ranges || []).filter((r) => r.dian_document_type_code === INVOICE_TYPE);
    customers = customersRes.customers || [];
    products = productsRes.products || [];
  } catch (err) {
    reportApiError(err);
  }

  if (ranges.length === 0) {
    root.appendChild(el("p", { class: "empty-state" }, 'Necesitas al menos una resolución de Factura antes de crear una. Ve a "Mi empresa" y crea una.'));
    return;
  }

  const rangeSelect = selectEl(ranges.map((r) => [r.id, `${r.prefix} (consecutivo actual: ${r.current_number})`]), existingDoc?.numbering_range_id);

  // ── Cliente ──
  const customerSelect = selectEl([["", "— cliente ocasional / escribir a mano —"], ...customers.map((c) => [c.id, c.name])], existingDoc?.customer_id || "");
  const custIdType = el("input", { value: existingDoc?.customer?.identification?.type_code || "13" });
  const custIdNumber = el("input", { value: existingDoc?.customer?.identification?.number || "222222222222" });
  const custName = el("input", { required: "true", value: existingDoc?.customer?.name || "Consumidor Final" });
  const custEmail = el("input", { type: "email", value: existingDoc?.customer?.email || "" });

  customerSelect.addEventListener("change", () => {
    const c = customers.find((x) => x.id === customerSelect.value);
    if (c) {
      custIdType.value = c.identification?.type_code || "";
      custIdNumber.value = c.identification?.number || "";
      custName.value = c.name || "";
      custEmail.value = c.email || "";
    }
  });

  // ── Líneas ──
  let lines = existingDoc
    ? existingDoc.lines.map((l) => ({
        description: l.description,
        quantity: l.quantity,
        unitCode: l.unit_code,
        unitPricePesos: centsToPesos(l.unit_price_cents),
        taxPercent: l.taxes?.[0]?.percent ?? 0,
        taxTypeCode: l.taxes?.[0]?.type_code || "01",
        taxTypeName: l.taxes?.[0]?.type_name || "IVA",
      }))
    : [{ description: "", quantity: 1, unitCode: "94", unitPricePesos: 0, taxPercent: 19, taxTypeCode: "01", taxTypeName: "IVA" }];

  const linesContainer = el("div", { class: "lines-editor" });
  const totalsBox = el("div", { class: "invoice-totals" });
  const note = el("textarea", { rows: "2", placeholder: "Nota (opcional)" });
  note.value = existingDoc?.note || "";

  function renderLines() {
    clear(linesContainer);
    linesContainer.appendChild(
      el("table", { class: "table lines-table" }, [
        el("thead", {}, el("tr", {}, [el("th", {}, "Producto"), el("th", {}, "Descripción"), el("th", {}, "Cant."), el("th", {}, "Precio unit."), el("th", {}, "IVA %"), el("th", {}, "Subtotal"), el("th", {}, "")])),
        el(
          "tbody",
          {},
          lines.map((line, idx) => renderLineRow(line, idx))
        ),
      ])
    );
    linesContainer.appendChild(el("button", { type: "button", class: "btn btn-secondary btn-sm", onclick: addLine }, "+ Agregar línea"));
    renderTotals();
  }

  function renderLineRow(line, idx) {
    const productSelect = selectEl([["", "— manual —"], ...products.map((p) => [p.id, p.description])], "");
    productSelect.addEventListener("change", () => {
      const p = products.find((x) => x.id === productSelect.value);
      if (p) {
        line.description = p.description;
        line.unitCode = p.unit_code;
        line.unitPricePesos = centsToPesos(p.unit_price_cents);
        line.taxPercent = p.tax_percent || 0;
        line.taxTypeCode = p.tax_type_code || "01";
        line.taxTypeName = p.tax_type_name || "IVA";
        renderLines();
      }
    });

    const desc = el("input", { value: line.description, oninput: (e) => (line.description = e.target.value) });
    const qty = el("input", { type: "number", min: "0", step: "1", value: line.quantity, oninput: (e) => { line.quantity = Number(e.target.value); renderTotals(); } });
    const price = el("input", { type: "number", min: "0", step: "0.01", value: line.unitPricePesos, oninput: (e) => { line.unitPricePesos = Number(e.target.value); renderTotals(); } });
    const tax = el("input", { type: "number", min: "0", step: "0.01", value: line.taxPercent, style: "width:70px", oninput: (e) => { line.taxPercent = Number(e.target.value); renderTotals(); } });
    const subtotal = el("span", {}, formatMoney(lineExtensionCents(line)));

    return el("tr", {}, [
      el("td", {}, productSelect),
      el("td", {}, desc),
      el("td", {}, qty),
      el("td", {}, price),
      el("td", {}, tax),
      el("td", {}, subtotal),
      el("td", {}, el("button", { type: "button", class: "btn btn-danger btn-sm", onclick: () => removeLine(idx) }, "✕")),
    ]);
  }

  function addLine() {
    lines.push({ description: "", quantity: 1, unitCode: "94", unitPricePesos: 0, taxPercent: 19, taxTypeCode: "01", taxTypeName: "IVA" });
    renderLines();
  }
  function removeLine(idx) {
    if (lines.length === 1) return;
    lines.splice(idx, 1);
    renderLines();
  }

  function lineExtensionCents(line) {
    return pesosToCents(line.quantity * line.unitPricePesos);
  }
  function lineTaxCents(line) {
    return Math.round((lineExtensionCents(line) * line.taxPercent) / 100);
  }
  function renderTotals() {
    const subtotal = lines.reduce((sum, l) => sum + lineExtensionCents(l), 0);
    const tax = lines.reduce((sum, l) => sum + lineTaxCents(l), 0);
    clear(totalsBox);
    totalsBox.appendChild(el("div", {}, ["Subtotal: ", el("strong", {}, formatMoney(subtotal))]));
    totalsBox.appendChild(el("div", {}, ["IVA: ", el("strong", {}, formatMoney(tax))]));
    totalsBox.appendChild(el("div", { class: "invoice-total-grand" }, ["Total: ", el("strong", {}, formatMoney(subtotal + tax))]));
    // refresca solo los subtotales de línea visibles, sin reconstruir toda la tabla
    linesContainer.querySelectorAll("tbody tr").forEach((row, i) => {
      const cell = row.children[5];
      if (cell && lines[i]) cell.textContent = formatMoney(lineExtensionCents(lines[i]));
    });
  }

  renderLines();

  const msg = el("div", { class: "form-error" });

  const form = el("form", { class: "invoice-form", onsubmit: handleSubmit }, [
    field("Resolución", rangeSelect),
    el("h4", {}, "Cliente"),
    field("Cliente guardado (opcional)", customerSelect),
    el("div", { class: "form-grid" }, [field("Tipo ID", custIdType), field("Número ID", custIdNumber), field("Nombre", custName), field("Correo", custEmail)]),
    el("h4", {}, "Líneas"),
    linesContainer,
    totalsBox,
    field("Nota", note),
    msg,
    el("div", { class: "modal-actions" }, [
      el("button", { type: "submit", class: "btn btn-primary" }, isEdit ? "Guardar cambios" : "Guardar borrador"),
      el("button", { type: "button", class: "btn btn-secondary", onclick: () => (isEdit ? showDetail(root, existingDoc.id) : showList(root)) }, "Cancelar"),
    ]),
  ]);
  root.appendChild(form);

  async function handleSubmit(e) {
    e.preventDefault();
    msg.textContent = "";
    const body = {
      numbering_range_id: rangeSelect.value,
      customer: { identification: { number: custIdNumber.value, type_code: custIdType.value }, name: custName.value, email: custEmail.value || undefined },
      lines: lines.map((l) => ({
        description: l.description,
        quantity: l.quantity,
        unit_code: l.unitCode,
        line_extension_cents: lineExtensionCents(l),
        unit_price_cents: pesosToCents(l.unitPricePesos),
        taxes: l.taxPercent > 0 || l.taxTypeCode !== "ZZ" ? [{ taxable_amount_cents: lineExtensionCents(l), tax_amount_cents: lineTaxCents(l), percent: l.taxPercent, type_code: l.taxTypeCode, type_name: l.taxTypeName }] : [],
      })),
      note: note.value || undefined,
      customer_id: customerSelect.value || undefined,
    };
    try {
      const saved = isEdit ? await api.updateInvoiceDraft(session.token, existingDoc.id, body) : await api.createInvoiceDraft(session.token, body);
      toast(isEdit ? "Borrador actualizado." : "Borrador creado.", "ok");
      await showDetail(root, saved.id);
    } catch (err) {
      msg.textContent = err instanceof ApiError ? err.message : "No se pudo guardar.";
    }
  }
}

// ── Detalle / confirmar ──────────────────────────────────────────────────────────────────────

async function showDetail(root, id) {
  clear(root);
  let doc;
  try {
    doc = await api.getDocument(session.token, id);
  } catch (err) {
    reportApiError(err);
    return showList(root);
  }

  root.appendChild(
    el("div", { class: "view-head" }, [
      el("div", {}, [el("h1", { class: "view-title" }, doc.prefix ? `${doc.prefix}-${doc.number}` : "Borrador de factura"), el("p", { class: "view-sub" }, statusBadge(doc.status))]),
      el("button", { class: "btn btn-secondary", onclick: () => showList(root) }, "← Volver a la lista"),
    ])
  );

  const card = el("div", { class: "card" });
  card.appendChild(
    el("dl", { class: "kv" }, [
      el("dt", {}, "Cliente"), el("dd", {}, doc.customer?.name || "—"),
      el("dt", {}, "Total"), el("dd", {}, formatMoney(doc.totals?.payable_cents || 0)),
      el("dt", {}, "Nota"), el("dd", {}, doc.note || "—"),
    ])
  );
  card.appendChild(
    el(
      "table",
      { class: "table" },
      [
        el("thead", {}, el("tr", {}, [el("th", {}, "Descripción"), el("th", {}, "Cant."), el("th", {}, "Precio"), el("th", {}, "Subtotal")])),
        el("tbody", {}, doc.lines.map((l) => el("tr", {}, [el("td", {}, l.description), el("td", {}, String(l.quantity)), el("td", {}, formatMoney(l.unit_price_cents)), el("td", {}, formatMoney(l.line_extension_cents))]))),
      ]
    )
  );
  root.appendChild(card);

  if (doc.status === "draft") {
    root.appendChild(
      el("div", { class: "card-actions" }, [
        el("button", { class: "btn btn-secondary", onclick: () => showForm(root, doc) }, "Editar borrador"),
        el("button", { class: "btn btn-danger", onclick: () => handleDelete(root, doc) }, "Eliminar borrador"),
        el("button", { class: "btn btn-primary", onclick: () => handleConfirm(root, doc) }, "Confirmar y enviar"),
      ])
    );
    root.appendChild(el("p", { class: "card-help" }, "Confirmar reclama el número real ante la DIAN, firma el documento, y lo envía si el ambiente lo permite — no se puede deshacer."));
  } else {
    const dianCard = el("div", { class: "card" }, [
      el("h3", {}, "Estado DIAN"),
      el("dl", { class: "kv" }, [
        el("dt", {}, "CUFE"), el("dd", {}, el("code", {}, doc.document_key || "—")),
        el("dt", {}, "Estado"), el("dd", {}, doc.dian_status_description || doc.dian_status_message || "—"),
      ]),
      doc.qr_url ? el("a", { href: doc.qr_url, target: "_blank", class: "btn btn-secondary btn-sm" }, "Ver QR en la DIAN") : null,
      doc.signed_xml ? el("button", { class: "btn btn-secondary btn-sm", onclick: () => downloadXml(doc) }, "Descargar XML firmado") : null,
    ]);
    root.appendChild(dianCard);
  }
}

async function handleDelete(root, doc) {
  if (!confirm("¿Eliminar este borrador? Esta acción no se puede deshacer.")) return;
  try {
    await api.deleteDocument(session.token, doc.id);
    toast("Borrador eliminado.", "ok");
    await showList(root);
  } catch (err) {
    reportApiError(err);
  }
}

async function handleConfirm(root, doc) {
  if (!confirm("¿Confirmar esta factura? Se reclamará un número real ante la DIAN y no se puede deshacer.")) return;
  try {
    const confirmed = await api.confirmDocument(session.token, doc.id);
    toast("Factura confirmada.", "ok");
    await showDetail(root, confirmed.id);
  } catch (err) {
    reportApiError(err);
  }
}

function downloadXml(doc) {
  const blob = new Blob([doc.signed_xml], { type: "application/xml" });
  const url = URL.createObjectURL(blob);
  const a = el("a", { href: url, download: `${doc.prefix}-${doc.number}.xml` });
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

function field(labelText, inputNode) {
  return el("label", { class: "field" }, [el("span", {}, labelText), inputNode]);
}

function selectEl(pairs, selected) {
  const node = el("select", {}, pairs.map(([value, label]) => el("option", { value }, label)));
  if (selected !== undefined) node.value = selected;
  return node;
}
