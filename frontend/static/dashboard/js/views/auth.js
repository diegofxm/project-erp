import { api, ApiError } from "../api.js";
import { setSession } from "../state.js";
import { el, clear, reportApiError } from "../ui.js";
import { IDENTIFICATION_TYPES, DEPARTMENTS, municipalitiesByDepartment } from "../catalogs.js";

export function renderLogin(root, { onSuccess, goRegister }) {
  clear(root);

  const emailInput = el("input", { type: "email", required: "true", placeholder: "tucorreo@empresa.com" });
  const passwordInput = el("input", { type: "password", required: "true", placeholder: "••••••••" });
  const errorBox = el("div", { class: "form-error" });

  const form = el("form", { class: "auth-form", onsubmit: handleSubmit }, [
    el("h2", {}, "Iniciar sesión"),
    el("p", { class: "auth-sub" }, "Entra a tu panel de facturación electrónica."),
    field("Correo", emailInput),
    field("Contraseña", passwordInput),
    errorBox,
    el("button", { type: "submit", class: "btn btn-primary btn-block" }, "Iniciar sesión"),
    el("p", { class: "auth-switch" }, [
      "¿No tienes cuenta? ",
      el("a", { href: "#", onclick: (e) => { e.preventDefault(); goRegister(); } }, "Regístrate aquí"),
    ]),
  ]);

  root.appendChild(el("div", { class: "auth-shell" }, [el("div", { class: "auth-card" }, [form])]));

  async function handleSubmit(e) {
    e.preventDefault();
    errorBox.textContent = "";
    try {
      const result = await api.login({ email: emailInput.value, password: passwordInput.value });
      setSession(result.token, result.issuer);
      onSuccess();
    } catch (err) {
      errorBox.textContent = err instanceof ApiError ? err.message : "No se pudo conectar con el servidor.";
    }
  }
}

export function renderRegister(root, { onSuccess, goLogin }) {
  clear(root);

  const nit = el("input", { required: "true", placeholder: "900373076" });
  const checkDigit = el("input", { required: "true", maxlength: "1", placeholder: "1", style: "width:60px" });
  const businessName = el("input", { required: "true", placeholder: "Mi Empresa S.A.S." });
  const idType = select(IDENTIFICATION_TYPES.map((t) => [t.code, `${t.code} — ${t.name}`]), "31");
  const department = select(DEPARTMENTS.map((d) => [d.code, d.name]), "11");
  const municipality = select([], "");
  const addressLine = el("input", { required: "true", placeholder: "Calle 1 # 2-3" });
  const email = el("input", { type: "email", required: "true", placeholder: "facturacion@empresa.com" });
  const environment = select(
    [
      ["2", "Pruebas (habilitación) — recomendado para empezar"],
      ["1", "Producción"],
    ],
    "2"
  );

  function refreshMunicipalities() {
    const opts = municipalitiesByDepartment(department.value);
    municipality.innerHTML = "";
    opts.forEach((m) => municipality.appendChild(el("option", { value: m.code }, m.name)));
    if (opts.length === 0) {
      municipality.appendChild(el("option", { value: "" }, "— sin municipios cargados para este departamento —"));
    }
  }
  department.addEventListener("change", refreshMunicipalities);
  refreshMunicipalities();

  const adminName = el("input", { required: "true", placeholder: "Tu nombre" });
  const adminEmail = el("input", { type: "email", required: "true", placeholder: "tucorreo@empresa.com" });
  const adminPassword = el("input", { type: "password", required: "true", minlength: "8", placeholder: "mínimo 8 caracteres" });

  const errorBox = el("div", { class: "form-error" });

  const form = el("form", { class: "auth-form auth-form-wide", onsubmit: handleSubmit }, [
    el("h2", {}, "Crea tu cuenta"),
    el("p", { class: "auth-sub" }, "Solo los datos que la DIAN exige de tu empresa. El software, la resolución y el certificado se configuran después, cuando los tengas a mano."),
    el("h4", {}, "Datos de tu empresa"),
    el("div", { class: "form-grid" }, [
      field("NIT", nit),
      field("Dígito de verificación", checkDigit),
      field("Razón social", businessName),
      field("Tipo de identificación", idType),
      field("Departamento", department),
      field("Municipio", municipality),
      field("Dirección", addressLine),
      field("Correo de facturación", email),
      field("Ambiente DIAN", environment),
    ]),
    el("h4", {}, "Tu cuenta"),
    el("div", { class: "form-grid" }, [field("Tu nombre", adminName), field("Tu correo", adminEmail), field("Contraseña", adminPassword)]),
    errorBox,
    el("button", { type: "submit", class: "btn btn-primary btn-block" }, "Crear cuenta"),
    el("p", { class: "auth-switch" }, [
      "¿Ya tienes cuenta? ",
      el("a", { href: "#", onclick: (e) => { e.preventDefault(); goLogin(); } }, "Inicia sesión"),
    ]),
  ]);

  root.appendChild(el("div", { class: "auth-shell" }, [el("div", { class: "auth-card auth-card-wide" }, [form])]));

  async function handleSubmit(e) {
    e.preventDefault();
    errorBox.textContent = "";
    try {
      const result = await api.register({
        issuer: {
          nit: nit.value,
          check_digit: checkDigit.value,
          business_name: businessName.value,
          identification_type_code: idType.value,
          department_code: department.value,
          municipality_code: municipality.value,
          address_line: addressLine.value,
          email: email.value,
          environment: environment.value,
        },
        email: adminEmail.value,
        password: adminPassword.value,
        name: adminName.value,
      });
      setSession(result.token, result.issuer);
      onSuccess();
    } catch (err) {
      errorBox.textContent = err instanceof ApiError ? err.message : "No se pudo conectar con el servidor.";
    }
  }
}

function field(labelText, inputNode) {
  return el("label", { class: "field" }, [el("span", {}, labelText), inputNode]);
}

function select(pairs, selected) {
  const node = el(
    "select",
    {},
    pairs.map(([value, label]) => el("option", { value }, label))
  );
  if (selected !== undefined) node.value = selected;
  return node;
}
