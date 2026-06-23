import { api, ApiError } from "../api.js";
import { session } from "../state.js";
import { el, clear, reportApiError, toast, openModal, formatMoney, pesosToCents, centsToPesos } from "../ui.js";
import { TAX_TYPES, UNIT_MEASURES } from "../catalogs.js";

export async function renderProducts(root) {
  clear(root);
  root.appendChild(el("div", { class: "view-head" }, [
    el("div", {}, [el("h1", { class: "view-title" }, "Productos"), el("p", { class: "view-sub" }, "Ítems/servicios reutilizables — sin inventario, solo los datos DIAN para no retipearlos al facturar.")]),
    el("button", { class: "btn btn-primary", onclick: () => openProductForm(root, null) }, "+ Nuevo producto"),
  ]));

  const tableWrap = el("div");
  root.appendChild(tableWrap);
  await loadTable(tableWrap, root);
}

async function loadTable(tableWrap, root) {
  clear(tableWrap);
  try {
    const data = await api.listProducts(session.token);
    const products = data.products || [];
    if (products.length === 0) {
      tableWrap.appendChild(el("p", { class: "empty-state" }, "Todavía no tienes productos — crea el primero."));
      return;
    }
    tableWrap.appendChild(
      el("table", { class: "table" }, [
        el("thead", {}, el("tr", {}, [el("th", {}, "Descripción"), el("th", {}, "Unidad"), el("th", {}, "Precio"), el("th", {}, "Impuesto"), el("th", {}, "")])),
        el(
          "tbody",
          {},
          products.map((p) =>
            el("tr", {}, [
              el("td", {}, p.description),
              el("td", {}, p.unit_code),
              el("td", {}, formatMoney(p.unit_price_cents)),
              el("td", {}, p.tax_type_code ? `${p.tax_type_name} (${p.tax_percent}%)` : "—"),
              el("td", { class: "table-actions" }, [
                el("button", { class: "btn btn-secondary btn-sm", onclick: () => openProductForm(root, p) }, "Editar"),
                el("button", { class: "btn btn-danger btn-sm", onclick: () => handleDelete(p, tableWrap, root) }, "Eliminar"),
              ]),
            ])
          )
        ),
      ])
    );
  } catch (err) {
    reportApiError(err);
  }
}

async function handleDelete(product, tableWrap, root) {
  if (!confirm(`¿Eliminar "${product.description}"?`)) return;
  try {
    await api.deleteProduct(session.token, product.id);
    toast("Producto eliminado.", "ok");
    await loadTable(tableWrap, root);
  } catch (err) {
    reportApiError(err);
  }
}

export function openProductForm(root, existing) {
  const isEdit = !!existing;
  const description = el("input", { required: "true", value: existing?.description || "" });
  const unitCode = unitDatalistInput(existing?.unit_code || "94");
  const unitPrice = el("input", { type: "number", step: "0.01", required: "true", value: existing ? centsToPesos(existing.unit_price_cents) : "" });
  const itemCode = el("input", { value: existing?.item_code || "" });
  const taxType = selectEl([["", "— sin impuesto —"], ...TAX_TYPES.map((t) => [t.code, t.name])], existing?.tax_type_code || "01");
  const taxPercent = el("input", { type: "number", step: "0.01", value: existing?.tax_percent ?? "19" });
  const msg = el("div", { class: "form-error" });

  const form = el("form", { class: "modal-form", onsubmit: handleSubmit }, [
    el("h3", {}, isEdit ? "Editar producto" : "Nuevo producto"),
    el("div", { class: "form-grid" }, [
      field("Descripción", description),
      field("Unidad de medida", unitCode.wrapper),
      field("Precio unitario (COP)", unitPrice),
      field("Código interno (opcional)", itemCode),
      field("Impuesto", taxType),
      field("Porcentaje del impuesto", taxPercent),
    ]),
    msg,
    el("div", { class: "modal-actions" }, [el("button", { type: "submit", class: "btn btn-primary" }, isEdit ? "Guardar cambios" : "Crear producto")]),
  ]);

  const close = openModal(form);

  async function handleSubmit(e) {
    e.preventDefault();
    msg.textContent = "";
    const taxName = TAX_TYPES.find((t) => t.code === taxType.value)?.name || "";
    const body = {
      description: description.value,
      unit_code: unitCode.input.value,
      unit_price_cents: pesosToCents(unitPrice.value),
      item_code: itemCode.value || undefined,
      tax_type_code: taxType.value || undefined,
      tax_type_name: taxType.value ? taxName : undefined,
      tax_percent: taxType.value ? Number(taxPercent.value) : undefined,
    };
    try {
      if (isEdit) await api.updateProduct(session.token, existing.id, body);
      else await api.createProduct(session.token, body);
      close();
      toast(isEdit ? "Producto actualizado." : "Producto creado.", "ok");
      await renderProducts(root);
    } catch (err) {
      msg.textContent = err instanceof ApiError ? err.message : "No se pudo guardar.";
    }
  }
}

function unitDatalistInput(value) {
  const listId = "unit-measures-list";
  let datalist = document.getElementById(listId);
  if (!datalist) {
    datalist = el("datalist", { id: listId }, UNIT_MEASURES.map((u) => el("option", { value: u.code }, u.name)));
    document.body.appendChild(datalist);
  }
  const input = el("input", { value, list: listId, required: "true" });
  return { input, wrapper: input };
}

function field(labelText, inputNode) {
  return el("label", { class: "field" }, [el("span", {}, labelText), inputNode]);
}

function selectEl(pairs, selected) {
  const node = el("select", {}, pairs.map(([value, label]) => el("option", { value }, label)));
  if (selected !== undefined) node.value = selected;
  return node;
}
