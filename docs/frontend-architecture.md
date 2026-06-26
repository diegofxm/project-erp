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

## Fase 1.6 — rutas en inglés, multi-empresa desde el dashboard, iconos de Navbar (2026-06-26)

Continuación pedida por el usuario, antes de seguir con Customers/Products/Documents. La hoja
de ruta de roles (superadmin/dueño de empresa/contador) que se discutió en esta misma fase
quedó documentada aparte en `docs/apidian-architecture.md` secc. 9.35 — no aquí, porque es un
diseño que afecta el modelo de datos y el JWT del backend, no solo el frontend.

1. **Rutas en inglés** (solo los *paths*, las etiquetas visibles siguen en español):
   `/configuracion`→`/settings`, `/facturas`→`/invoices`, `/clientes`→`/customers`,
   `/productos`→`/products`, `/numeracion`→`/numbering`, nueva `/issuers`.
2. **`components/IssuerManager.tsx`** (nuevo): la lógica de listar/seleccionar/crear empresa se
   extrajo de `OnboardingPage` para poder reusarla también dentro del dashboard ya con una
   empresa activa. `pages/IssuersPage.tsx` (nueva, ruta `/issuers`) la envuelve con el chrome de
   una página de dashboard normal; `OnboardingPage` sigue envolviéndola con su Card centrada de
   antes (gate previo al login, sin cambios de comportamiento ahí). Bug encontrado y corregido
   durante la verificación: crear una empresa desde `/issuers` no refrescaba la lista (se
   quedaba con los datos de antes de crear, la empresa nueva no aparecía) —
   `IssuerManager.handleCreate` ahora vuelve a pedir `listIssuers()` después de crear en vez de
   confiar en el estado ya cargado.
3. **Navbar**: dos iconos nuevos a la izquierda del avatar — uno funcional (`Building2`, enlaza
   a `/issuers`) y uno reservado/deshabilitado (`Bell`, notificaciones — no existe backend de
   notificaciones todavía, se deja el lugar reservado en vez de inventar contenido).

Verificado con un navegador real: registro → crear Empresa A → icono de empresas del Navbar →
`/issuers` → crear Empresa B desde ahí mismo → el Navbar refleja el cambio a B → volver a
`/issuers` y cambiar de vuelta a A → el Navbar lo refleja. Cero errores de consola. Datos de
prueba limpiados de la base real después.

## Fase 1.7 — Configuración → Empresa (software/certificado/numeración) y regla de ancho de página (2026-06-26)

A pedido explícito del usuario, la pestaña "Empresa" de `SettingsPage` (antes deshabilitada,
"próximamente") se construyó — completa exactamente lo que `CompanyForm` deja fuera a propósito
desde la Fase 1.5 (configuración técnica), más el registro de resoluciones:

- **`components/issuer-settings/SoftwareCertificateForm.tsx`**: completa `software_id`/
  `software_pin`/certificado/`certificate_password` vía `PUT /issuers/me`
  (`AuthContext.updateIssuer`, nuevo — a diferencia de `login`/`createIssuer`/`selectIssuer`,
  este NO reemite el token, solo actualiza `activeIssuer` y la sesión persistida, porque sigue
  siendo la misma empresa activa). El certificado `.p12` se sube como archivo y se convierte a
  base64 en el navegador (`lib/fileToBase64.ts`, nuevo) antes de mandarlo — nunca como
  multipart. Cada campo es independiente (omitir = no tocar, mismo contrato que
  `issuers.UpdateIssuerRequest`); badges "✓ Software"/"✓ Certificado" muestran si ya está
  configurado sin que el secreto viaje nunca de vuelta — requirió agregar
  `has_software_credentials`/`has_certificate` (solo presencia, nunca el valor) a
  `issuerResponse` en el backend, ver `docs/apidian-architecture.md` (es un cambio de apidian,
  documentado allá).
- **`components/issuer-settings/NumberingRangesPanel.tsx` + `NumberingRangeForm.tsx`**:
  registra una resolución de numeración (`POST /numbering-ranges`, `lib/numberingRanges.ts`
  nuevo — a diferencia de `lib/catalogs.ts`, sin memoización: son datos propios del tenant, no
  catálogos estáticos). Dos campos condicionales: clave técnica (CUFE) solo si el tipo de
  documento es Factura ("01"); **set de pruebas** solo si el ambiente es Habilitación — pedido
  explícito del usuario ("no olvides la parte de set de pruebas para que pueda colocar los
  datos antes de hacer pruebas"), es el identificador que la DIAN exige para confirmar
  documentos durante el proceso de habilitación.
- Verificado con el certificado y las credenciales DIAN reales del usuario
  (`docs/reference/certificado.p12`/`credenciales.txt`, sobre una empresa de prueba, no la
  empresa real) contra Postgres real: guardar software+certificado → badges cambian a "✓";
  registrar un rango con set de pruebas → aparece en la lista con su número actual. Encontrado
  en el camino: el backend que estaba corriendo en la máquina era una compilación anterior a
  este cambio — los campos nuevos de `issuerResponse` no llegaban hasta reiniciarlo (no es un
  bug del código, solo un recordatorio de reiniciar `go run` tras tocar Go).

**Regla de ancho de página** (el usuario notó que `/issuers` mostraba la lista de empresas con
un límite de ancho raro a la derecha, y diagnosticó correctamente que era "por el formulario de
creación"). Pasó por dos iteraciones antes de quedar bien:

1. Primer intento: listados a todo el ancho, formularios acotados a `max-w-2xl` (igual que
   `CompanyForm` desde el principio). El usuario lo corrigió: quería que los formularios
   **también** usaran el espacio completo, no solo los listados.
2. Segundo intento: formularios a todo el ancho con grillas `grid-cols-[repeat(auto-fit,
   minmax(Npx,1fr))]` (cada campo se redistribuye según cuántos quepan). El usuario lo corrigió
   otra vez: con `auto-fit` la cantidad de columnas que caben cambia con el ancho disponible, así
   que al colapsar/expandir el Sidebar los campos **se reordenan/saltan de fila** — "poco
   profesional". También causaba que el texto de algunos `<select>` quedara escondido cuando le
   tocaba una columna muy angosta en el recálculo.

**Regla final**: grilla de **12 columnas fijas** (`grid grid-cols-12 gap-3`, sin `auto-fit` ni
`minmax`), cada campo con un `col-span-N` fijo elegido a mano según qué tan corto/largo es su
contenido esperado (ej. `DV` — antes "Dígito verificación" — es `col-span-1`; "Razón social" es
`col-span-4`; un `<select>` con opciones largas como "Tipo de identificación" recibe más
columnas que uno con opciones cortas como "Ambiente"). Con columnas fijas, redimensionar el
contenedor (colapsar el Sidebar, por ejemplo) solo cambia el ancho en píxeles de cada columna —
nunca cuántas hay ni en qué fila/posición cae cada campo. Aplica a `IdentificationStep`/
`LocationStep`/`TaxStep` (los 3 pasos de `CompanyForm`), `SoftwareCertificateForm`, y
`NumberingRangeForm`. Bloques que no son un campo de una sola línea (el cuadro de checkboxes de
responsabilidades fiscales, `TagInput` de CIIU) usan `col-span-12` (la fila completa) en vez de
competir por espacio con los campos cortos.

Etiquetas acortadas de paso, mismo pedido del usuario: "Dígito verificación" → "DV" (como en el
RUT real), "Próximo número a reclamar (opcional)" → "Próximo número (opcional)".

Los listados (`/issuers`, filas de `NumberingRangesPanel`) y los paneles sueltos de un solo
control (pestaña General de `SettingsPage`, una sola tarjeta) no necesitan esta grilla — un
listado ya se redistribuye bien con `flex items-center justify-between` por fila, y una tarjeta
suelta simplemente no debe estirarse a ocupar todo el ancho solo porque puede (se deja del
tamaño de su contenido).

Verificado con un navegador real a 1440px: comparé las coordenadas DOM de cada campo de
`NumberingRangeForm` antes y después de colapsar el Sidebar — mismo orden, misma fila para cada
uno, solo cambió el ancho en píxeles de cada columna. Sin errores de consola.

## Estados de carga (2026-06-26)

El usuario notó que, por ejemplo, al entrar a `/issuers`, la tarjeta no mostraba nada — ni
contenido ni ningún indicador — hasta que `listIssuers()` resolvía; mientras tanto se veía como
una tarjeta "encogida". Causa: varios componentes ya distinguían `null` ("todavía no llegó la
respuesta") de `[]` ("llegó y está vacío") en su estado, pero el JSX no renderizaba *nada*
mientras el valor era `null` — ni un spinner, ni un alto mínimo. Corregido en todo el frontend
con dos piezas nuevas y reutilizables:

- **`components/ui/Spinner.tsx`**: envuelve `Loader2` de lucide-react con `animate-spin` — es
  la convención que el design system ya documentaba (secc. 14, "Spinners de carga:
  RefreshCw/Loader2 con animate-spin") pero que **nunca se había implementado** en este
  proyecto. Sin color propio: hereda `currentColor` (blanco dentro de un botón primario,
  `--text-muted` cuando se centra solo dentro de una tarjeta).
- **`components/ui/Button.tsx`**: el prop `loading` ya existía pero **no hacía nada visible**
  — solo deshabilitaba el botón. Ahora reemplaza el ícono por `<Spinner>` mientras `loading` es
  `true`. Esto se corrigió en un solo lugar y se propaga automáticamente a todos los botones de
  envío que ya pasaban `loading` (Login, Register, los 3 pasos de `CompanyForm`,
  `SoftwareCertificateForm`, `NumberingRangeForm`).
- **`lib/useCatalog.ts`** (hook nuevo): envuelve cualquier fetcher de `lib/catalogs.ts` y
  devuelve `{ data, loading }` — antes cada paso de `CompanyForm` y `NumberingRangeForm`
  repetía el mismo `useState<T[]>([]) + useEffect(() => fetcher().then(setX))`, indistinguible
  de "cargó y está vacío". Con el hook, cada `<Select>` que depende de un catálogo muestra una
  única opción deshabilitada "Cargando…" en vez de un `<select>` vacío con cero opciones
  mientras llega la primera respuesta (memoizada después, así que solo se nota una vez por
  catálogo por sesión). Aplicado en `IdentificationStep`, `LocationStep` (departamentos;
  municipios mantiene su propio flag porque depende del departamento elegido),
  `TaxStep`, `NumberingRangeForm`.
- **Listas con estado `null`** (`IssuerManager.issuers`, `NumberingRangesPanel.ranges`): ahora
  muestran `<Spinner>` centrado con una altura mínima (`min-h-32`/`min-h-20`) mientras el valor
  sigue en `null`, en vez de no renderizar nada — la tarjeta que lo contiene ya no se ve
  "encogida" durante la carga.

Verificado con un navegador real interceptando todas las llamadas a `/api/v1/` con un retraso
artificial de 1.5s (en localhost todo responde casi instantáneo, sin esto no se vería el
estado de carga en absoluto): el botón "Crear cuenta" muestra el spinner mientras
`POST /auth/register` está en curso; `/issuers` muestra el spinner centrado en vez de una
tarjeta vacía; el `<select>` de "Tipo de identificación" muestra "Cargando…" hasta que el
catálogo llega. Sin errores de consola. Datos de prueba limpiados de la base real después.

## Pendiente para próximas fases

- Páginas reales para Facturas/Clientes/Productos (hoy son items de sidebar deshabilitados, sin
  ruta). El ítem "Numeración" del Sidebar sigue deshabilitado a propósito — sigue siendo una
  página distinta (pensada para historial/consumo de cada rango), separada de la creación
  básica de resoluciones ya disponible en Configuración → Empresa desde la Fase 1.7.
- Endpoint para editar perfil de usuario (nombre/correo) — hoy Configuración → Mi cuenta es de
  solo lectura porque no existe.
- Los 3 hallazgos ya logueados arriba (snapshot de cliente incompleto, payment_means en el
  formulario de factura, claridad del consecutivo actual) siguen pendientes para cuando se
  construya la página de Facturas real.
