import { api, ApiError } from "../api.js";
import { session } from "../state.js";
import { el, clear, reportApiError, toast, openModal } from "../ui.js";
import { IDENTIFICATION_TYPES, MUNICIPALITIES } from "../catalogs.js";

export async function renderCustomers(root) {
  clear(root);
  root.appendChild(el("div", { class: "view-head" }, [
    el("div", {}, [el("h1", { class: "view-title" }, "Clientes"), el("p", { class: "view-sub" }, "El catálogo de clientes solo es para no retipear datos al facturar — la factura siempre guarda su propia copia.")]),
    el("button", { class: "btn btn-primary", onclick: () => openCustomerForm(root, null) }, "+ Nuevo cliente"),
  ]));

  const tableWrap = el("div", { id: "customers-table-wrap" });
  root.appendChild(tableWrap);
  await loadTable(tableWrap, root);
}

async function loadTable(tableWrap, root) {
  clear(tableWrap);
  try {
    const data = await api.listCustomers(session.token);
    const customers = data.customers || [];
    if (customers.length === 0) {
      tableWrap.appendChild(el("p", { class: "empty-state" }, "Todavía no tienes clientes — crea el primero."));
      return;
    }
    tableWrap.appendChild(
      el("table", { class: "table" }, [
        el("thead", {}, el("tr", {}, [el("th", {}, "Identificación"), el("th", {}, "Nombre"), el("th", {}, "Correo"), el("th", {}, "Teléfono"), el("th", {}, "")])),
        el(
          "tbody",
          {},
          customers.map((c) =>
            el("tr", {}, [
              el("td", {}, `${c.identification?.type_code || ""} ${c.identification?.number || ""}`),
              el("td", {}, c.name),
              el("td", {}, c.email || "—"),
              el("td", {}, c.phone || "—"),
              el("td", { class: "table-actions" }, [
                el("button", { class: "btn btn-secondary btn-sm", onclick: () => openCustomerForm(root, c) }, "Editar"),
                el("button", { class: "btn btn-danger btn-sm", onclick: () => handleDelete(c, tableWrap, root) }, "Eliminar"),
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

async function handleDelete(customer, tableWrap, root) {
  if (!confirm(`¿Eliminar a "${customer.name}"? Esto no afecta facturas ya emitidas.`)) return;
  try {
    await api.deleteCustomer(session.token, customer.id);
    toast("Cliente eliminado.", "ok");
    await loadTable(tableWrap, root);
  } catch (err) {
    reportApiError(err);
  }
}

function openCustomerForm(root, existing) {
  const isEdit = !!existing;
  const idType = selectEl(IDENTIFICATION_TYPES.map((t) => [t.code, `${t.code} — ${t.name}`]), existing?.identification?.type_code || "31");
  const idNumber = el("input", { required: "true", value: existing?.identification?.number || "" });
  const name = el("input", { required: "true", value: existing?.name || "" });
  const city = selectEl(
    [["", "— sin especificar —"], ...MUNICIPALITIES.map((m) => [m.code, m.name])],
    existing?.address?.city_code || ""
  );
  const addressLine = el("input", { value: existing?.address?.line || "", placeholder: "Calle 50 # 10-20" });
  const email = el("input", { type: "email", value: existing?.email || "" });
  const phone = el("input", { value: existing?.phone || "" });
  const msg = el("div", { class: "form-error" });

  const form = el("form", { class: "modal-form", onsubmit: handleSubmit }, [
    el("h3", {}, isEdit ? "Editar cliente" : "Nuevo cliente"),
    el("div", { class: "form-grid" }, [
      field("Tipo de identificación", idType),
      field("Número", idNumber),
      field("Nombre / Razón social", name),
      field("Municipio", city),
      field("Dirección", addressLine),
      field("Correo", email),
      field("Teléfono", phone),
    ]),
    msg,
    el("div", { class: "modal-actions" }, [el("button", { type: "submit", class: "btn btn-primary" }, isEdit ? "Guardar cambios" : "Crear cliente")]),
  ]);

  const close = openModal(form);

  async function handleSubmit(e) {
    e.preventDefault();
    msg.textContent = "";
    const cityOpt = MUNICIPALITIES.find((m) => m.code === city.value);
    const body = {
      identification: { number: idNumber.value, type_code: idType.value },
      name: name.value,
      email: email.value || undefined,
      phone: phone.value || undefined,
      address: addressLine.value || cityOpt ? { line: addressLine.value, city_code: cityOpt?.code, city_name: cityOpt?.name } : undefined,
    };
    try {
      if (isEdit) await api.updateCustomer(session.token, existing.id, body);
      else await api.createCustomer(session.token, body);
      close();
      toast(isEdit ? "Cliente actualizado." : "Cliente creado.", "ok");
      await renderCustomers(root);
    } catch (err) {
      msg.textContent = err instanceof ApiError ? err.message : "No se pudo guardar.";
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
