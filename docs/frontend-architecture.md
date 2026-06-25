# Arquitectura Profesional frontend DIAN en React TS (FRONTEND)

> Este documento es la bitácora del frontend — separado a propósito de
> `docs/apidian-architecture.md` (decisión explícita del usuario, 2026-06-23): los hallazgos
> sobre el backend van allá; todo lo que tenga que ver con cómo se ve/usa/construye el
> frontend (el explorador tipo Postman en `frontend/static/`, el dashboard improvisado en
> `frontend/static/dashboard/`, y el dashboard definitivo que se construirá después en
> React+TS) va aquí.

## Historial: `devui` + dashboard improvisado (2026-06-22/23)

El usuario pidió pasar de probar con Postman a probar con un frontend real en el navegador,
para simular un usuario real — razonando que eso revela huecos ocultos en la API que Postman/
curl nunca muestran porque nunca necesitan presentar nada, solo enviar y leer un código de
estado. Esto se confirmó de inmediato (ver hallazgos abajo y en `apidian-architecture.md`
secciones 9.28-9.30).

`frontend/` es un módulo Go **independiente y separado de `apidian`** (decisión explícita del
usuario: no quería el frontend de pruebas viviendo dentro del módulo del backend) — agregado a
`go.work` pero no a ningún `cmd/` de `apidian`. Un servidor estático de ~25 líneas
(`//go:embed`) sirve dos herramientas sin build step (HTML+CSS+JS plano, sin npm, sin
frameworks):

- **`frontend/static/`** (`/`) — un explorador tipo Postman: un botón por endpoint, JSON
  editable con plantilla precargada, variables capturadas automáticamente entre pasos
  (`{{invoiceRangeId}}`, `{{invoiceCufe}}`, etc.). El usuario decidió mantener este tal cual
  está, sin más cambios — le sirve como herramienta técnica rápida y le gustó el resultado.
- **`frontend/static/dashboard/`** (`/dashboard/`) — un dashboard "improvisado" (login/
  registro con formularios, sidebar, configuración del emisor con certificado por
  drag-and-drop, catálogos de clientes/productos, y el flujo completo de facturación: crear
  borrador → editar → ver → confirmar y enviar). ES modules nativos del navegador, sin
  bundler. El usuario decidió (2026-06-23) dejar de invertir en arreglarle bugs — lo usa tal
  cual está para terminar el ciclo de pruebas actual, y dedicará el esfuerzo de ahí en
  adelante al dashboard definitivo en React+TS (título de este documento) en vez de seguir
  parchando este.

## Hallazgos pendientes para el dashboard definitivo

Encontrados probando el ciclo completo con el dashboard improvisado (plano, JS sin build
step) — el usuario decidió explícitamente NO seguir parchando ese dashboard (quiere dedicar
el esfuerzo al dashboard definitivo en vez de invertir más en el improvisado), así que estos
quedan anotados para cuando se construya ese, no para arreglarse ahora.

### 1. El snapshot de cliente en una factura nueva solo copia 4 campos del cliente guardado

`frontend/static/dashboard/js/views/invoices.js`, el listener de `customerSelect` solo copia
`identification.type_code`, `identification.number`, `name`, `email` del cliente seleccionado
— nunca `address`, `phone`, `tax_scheme_code`, `tax_scheme_name`, `liability_codes`,
`entity_type_code`, `merchant_registration_number`. El formulario tampoco tiene campos para
casi nada de eso.

El usuario fue explícito sobre el porqué importa: aunque `customers` es un catálogo de
conveniencia separado de la fuente de verdad del documento (correcto, ver
`apidian-architecture.md` sección 4.2), el snapshot que se persiste en el documento SÍ debe
ser un reflejo completo del cliente en ese momento — por motivos históricos, si el cliente se
elimina después, la factura ya emitida no debe perder datos que sí tenía disponibles al
crearla.

**Para el dashboard definitivo**: al seleccionar un cliente guardado, copiar el objeto
completo que devuelve `GET /customers` (todos los campos del `partyDTO`) al construir el
`customer` del borrador — no reconstruirlo a mano desde 4 inputs visibles. El backend
(`apidian`) ya expone todo lo necesario vía `customerResponse`/`partyDTO`; esto es
puramente un hueco del frontend, no de la API.

### 2. `cac:PaymentMeans` (forma de pago) — el dashboard nunca lo pide ni lo manda

Confirmado contra el Anexo Técnico (`docs/reference/anexo-tecnico-1.9.txt`, FAN01/CAN01/DAN01):
`cac:PaymentMeans` es obligatorio (cardinalidad `1..N`) para Invoice/CreditNote/DebitNote. El
formulario de "Nueva factura" del dashboard improvisado nunca pide forma de pago, así que
nunca se manda — esto causó rechazos reales de la DIAN ("errores en campos mandatorios") en
3+ facturas de prueba del usuario.

**Ya corregido en el backend** (`apidian`, sección 9.30 de `apidian-architecture.md`):
`documents.Service.validateBase` ahora exige `payment_means` no vacío desde el borrador — así
que el dashboard improvisado, tal como está hoy, **ya no puede crear un borrador de factura**
(toda petición a `POST /invoices` sin `payment_means` responde 400). El usuario decidió no
parchar esto en el dashboard improvisado; queda bloqueado hasta que se use el dashboard
definitivo o se edite el JSON a mano (vía el explorador tipo Postman en `frontend/static/`,
que sí permite editar el cuerpo libremente).

**Para el dashboard definitivo**: agregar un selector de forma de pago al formulario de
factura — mínimo `payment_means: [{code: "1"|"2", payment_method_code: <de PAYMENT_METHODS>}]`
("1" contado / "2" crédito, con `cbc:PaymentDueDate` si es crédito, ver Anexo Técnico FAN03/
FAN04).

### 3. No hay claridad del número de consecutivo actual de una resolución

El usuario señaló que el dashboard improvisado no deja claro cuál es el `current_number` de
una resolución antes de confirmar un documento — importante porque ayuda a anticipar qué
número se va a reclamar. Diferido explícitamente para el dashboard definitivo (no es un bug
del backend: `GET /numbering-ranges` ya devuelve `current_number`, ver
`apidian-architecture.md` sección 9.19 — es una cuestión de cómo se presenta en la UI).

## Dashboard definitivo (React + TS) — Fase 1: login/register + primer pantallazo (2026-06-23)

Proyecto en `frontend/` (Vite + React 19 + TypeScript, módulo separado de `apidian`, sin
relación de dependencia — solo se comunican por HTTP). Design system aplicado tal cual
`docs/reference/DESIGN_SYSTEM.md` (extraído de un dashboard previo del usuario, estética
pgAdmin/DataGrip: denso, 13px base, un solo azul de acento, CSS variables para tokens).

**Stack**: Tailwind v4 (`@tailwindcss/vite`), `lucide-react`, `react-router` v7, `fetch` propio
(`src/lib/apiClient.ts`, sin librería de cache — no hace falta todavía), sesión en
`localStorage` vía `AuthContext` (Bearer JWT, igual contrato que `apidian`).

**Corregido desde el día 1** (vs. las inconsistencias que el propio design system señala en
su sección 15): un solo azul de acento en todos los CTAs, colores semánticos (éxito/peligro/
info) como variables CSS reales (no hex sueltos), un solo radio de borde por tamaño (`rounded`
chico, `rounded-lg` solo contenedores grandes), foco visible global (`:focus-visible` en
`index.css`), hover vía clases Tailwind (`hover:bg-(--bg-hover)`) en vez de
`onMouseEnter`/`onMouseLeave`, tema oscuro con toggle real funcionando desde ya (paleta VS Code
Dark+ ya estaba modelada en el design system, solo faltaba el control).

**Decisión de alcance** (gate de empresa activa): `apidian` soporta multi-empresa (0/1/N por
usuario, sección 9.32) — sin empresa activa, casi toda la API responde 409
(`middleware.RequireTenant`). Por eso el "primer pantallazo" de esta fase no es solo
login/register: incluye `OnboardingPage` (listar empresas existentes / crear una nueva) como
paso obligatorio antes de montar el shell del dashboard (`DashboardLayout`). El formulario de
creación de empresa es deliberadamente mínimo — solo lo que `issuers.validateIssuer` exige de
verdad (nit, business_name, environment) más los campos de identidad/dirección obvios
(check_digit, identification_type_code, department_code, municipality_code, address_line,
email) — **no** pide software/PIN/certificado, eso queda para una fase de "configuración del
emisor" aparte (igual que en el backend, completar el emisor es gradual).

**Construido en esta fase**: `lib/apiClient.ts` + `lib/types.ts`, `context/AuthContext.tsx`
(login/register/logout/listIssuers/createIssuer/selectIssuer, sesión persistida),
`context/ThemeContext.tsx` (toggle claro/oscuro persistido), componentes base `ui/Button.tsx`
(las 5 variantes de la sección 10 del design system, antes no existía un componente
reutilizable), `ui/Input.tsx`, `ui/Card.tsx`, `ui/Banner.tsx`, `Navbar.tsx` (h-10, oscuro,
indicador de empresa activa, menú de usuario), `Sidebar.tsx` (colapsable, persistida, items de
navegación futuros mostrados pero deshabilitados — todavía no tienen página), `LoginPage.tsx`,
`RegisterPage.tsx`, `OnboardingPage.tsx`, `DashboardPage.tsx` (placeholder con resumen de
sesión/empresa activa), `DashboardLayout.tsx` (decide Onboarding vs. shell del dashboard según
si hay empresa activa), `ProtectedRoute.tsx`.

**Verificado de punta a punta con un navegador real** (Playwright headless, no solo
`tsc`/build): registro → onboarding (sin empresa) → crear empresa → dashboard con datos
correctos → toggle de tema oscuro → colapso de sidebar → logout → login → auto-selección de
empresa (exactamente 1 vinculada) directo al dashboard. Sin errores de consola en ningún paso.
Datos de prueba limpiados de la base real después.

**Catálogos DIAN sin endpoint HTTP** — RESUELTO 2026-06-24: `apidian` ahora expone
`GET /api/v1/catalogs/{departments,municipalities,identification-types,tax-types,
payment-methods,payment-terms,unit-measures,tax-regimes,liability-codes,
dian-document-types,currencies}` (`internal/catalogs`, ver
`docs/apidian-architecture.md`). `OnboardingPage` (y cualquier formulario futuro de
cliente/producto/factura) ya puede construir selects reales en vez de inputs de texto libre —
queda como tarea de la Fase 2 del frontend, no un bloqueo de backend.

## Fase 1.5 — registro, empresa por pestañas, Navbar/Configuración (2026-06-25)

A pedido del usuario, tras un recorrido manual del dashboard de la Fase 1, tres mejoras antes
de construir páginas nuevas:

1. **`RegisterPage`**: campo "Confirmar contraseña", validado solo en el navegador (nunca llega
   al backend si no coincide) — no es una regla de `auth.Service`.
2. **`OnboardingPage`**: el formulario plano de creación de empresa se reemplazó por
   `components/company-form/CompanyForm.tsx` — pestañas (`ui/Tabs.tsx`, mismo patrón de la
   secc. 7 del design system) "Identificación" / "Ubicación y contacto" / "Información
   tributaria" / "Revisión". Ahora pide todo lo que `createIssuerRequest` acepta del emisor
   (antes solo 8 de ~17 campos) usando los catálogos reales de `internal/catalogs`
   (`lib/catalogs.ts`, memoizados a nivel de módulo): tipo de identificación, departamento→
   municipio dependiente, tipo de impuesto, tipo de régimen, responsabilidades fiscales
   (checkboxes), códigos CIIU (`ui/TagInput.tsx`, máx. 4, sin catálogo — DANE no DIAN),
   matrícula mercantil. La validación de obligatorios solo se exige en la pestaña de revisión
   (banner con enlaces a la pestaña correspondiente); software/PIN/certificado siguen fuera de
   alcance (fase de configuración técnica aparte). Esto resuelve el pendiente que quedó anotado
   en la Fase 1 ("reemplazar inputs de texto libre por selects reales").
3. **Navbar y Configuración**: el botón de usuario ya no muestra el nombre, solo un avatar de
   iniciales (`bg-(--accent-primary)`, patrón GitHub/Linear/Vercel: identidad completa vive en
   el desplegable, no en la barra fija) — el desplegable ganó enlaces "Mi cuenta"/
   "Configuración" antes de "Cerrar sesión". El icono de tema se sacó del Navbar y ahora vive en
   `pages/SettingsPage.tsx` (nueva, ruta `/configuracion`, ítem de `Sidebar` habilitado), con
   pestañas General (tema) / Mi cuenta (nombre, correo — de solo lectura, no hay endpoint de
   edición de perfil todavía) / Empresa (deshabilitada, "próximamente" — configuración de
   empresa explícitamente fuera de alcance por ahora). Se decidió NO mostrar `User.role` en el
   desplegable como "cargo": en el backend (`internal/auth/model.go`) ese campo es siempre
   `"admin"` hoy (no hay roles granulares de usuario), mostrarlo habría sido engañoso.

Verificado de punta a punta con un navegador real (Playwright vía `chromium`, no solo
`tsc`/build): registro con contraseñas distintas → error inline sin llamar a la API → registro
con contraseñas iguales → onboarding → las 4 pestañas con datos de catálogo reales (el filtro
departamento→municipio funcionó, el banner de campos faltantes desapareció al completarlos) →
crear empresa real → dashboard → Navbar con avatar → desplegable → Configuración → toggle de
tema → Mi cuenta. Sin errores de consola en ningún paso. Datos de prueba limpiados de la base
real después.

## Pendiente para próximas fases

- Páginas reales para Facturas/Clientes/Productos/Numeración (hoy son items de sidebar
  deshabilitados, sin ruta) y para Configuración → Empresa (deshabilitada a propósito, ver
  arriba).
- Cambiar de empresa activa DESPUÉS de ya estar en el dashboard (hoy `selectIssuer` solo se usa
  desde `OnboardingPage`; el menú de usuario del Navbar no lo expone todavía).
- Endpoint para editar perfil de usuario (nombre/correo) — hoy Configuración → Mi cuenta es de
  solo lectura porque no existe.
- Los 3 hallazgos ya logueados arriba (snapshot de cliente incompleto, payment_means en el
  formulario de factura, claridad del consecutivo actual) siguen pendientes para cuando se
  construya la página de Facturas real.
