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

## Hallazgos pendientes para el dashboard definitivo (los 3 ya resueltos — ver "Factura Electrónica" más abajo)

Encontrados probando el ciclo completo con el dashboard improvisado (plano, JS sin build
step) — el usuario decidió explícitamente NO seguir parchando ese dashboard (quiere dedicar
el esfuerzo al dashboard definitivo en vez de invertir más en el improvisado), así que estos
quedan anotados para cuando se construya ese, no para arreglarse ahora. **Resueltos los 3 al
construir Factura Electrónica en el dashboard definitivo (2026-06-26)** — se deja el detalle
histórico de abajo (por qué importaba cada uno) y se anota dónde quedó resuelto.

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

## Clientes (2026-06-26)

Primera página real de "lista de registros" del frontend (las anteriores —`/issuers`— eran
como máximo 2-3 botones, no una tabla). Backend ya completo (`internal/customers`), sin huecos
que resolver antes — confirmado al revisar `handler_customers.go`/`service.go`/`errors.go`.

- **`lib/customers.ts`** (nuevo, sin memoización — datos del tenant, no catálogo estático):
  `listCustomers`/`createCustomer`/`updateCustomer`/`deleteCustomer`, mismo estilo que
  `lib/numberingRanges.ts`.
- **`components/customer-form/CustomerForm.tsx`** (nuevo): a diferencia de `CompanyForm`, un
  solo formulario sin pestañas — un cliente tiene menos campos que una empresa (sin software/
  certificado, sin CIIU, sin matrícula mercantil — `cofacture/domain/types.go` ya documenta que
  esos campos "solo aplican al emisor, nunca al receptor"/"siempre nil para el receptor", así
  que ni se piden). Misma grilla fija de 12 columnas y mismos hooks (`useCatalog`) que el resto
  del frontend. El campo `address` del payload (`partyDTO` en apidian) usa nomenclatura UBL
  genérica (`state_code`/`state_name`/`city_code`/`city_name`) en vez de
  `department_code`/`municipality_code` como en `Issuer` — el formulario sigue usando los
  mismos catálogos `listDepartments`/`listMunicipalities`, solo mapea código→nombre a mano al
  enviar (a diferencia de `issuers`, que hace ese join del lado del servidor).
- **`pages/CustomersPage.tsx`** (nuevo, ruta `/customers`, ítem de Sidebar habilitado): primera
  tabla real de datos del frontend — cabecera `bg-tertiary`, filas en zebra striping, acciones
  Editar/Eliminar por fila (iconos, sin confirmación con modal propio — `window.confirm` nativo
  por ahora, no existe un componente de diálogo de confirmación todavía). Sin búsqueda ni
  paginación a propósito (mismo criterio que `numbering.Repository.ListByIssuer`: no se agrega
  hasta que haya suficientes datos reales para necesitarlo). El formulario de creación/edición
  reemplaza la tabla mientras está abierto (mismo patrón que `IssuerManager`/
  `NumberingRangesPanel`).
- **Corrección de paso, encontrada al construir esto**: `CompanyForm`'s `TaxStep` nunca
  actualizaba `tax_scheme_name` cuando el usuario cambiaba `tax_scheme_code` — se quedaba en el
  default `"No aplica"` aunque el código elegido fuera, por ejemplo, `"01"` (IVA), mandando un
  par código/nombre inconsistente a la DIAN. Corregido ahí mismo y aplicado correctamente desde
  el principio en `CustomerForm`.

Verificado con un navegador real: crear cliente (con departamento→municipio dependiente) →
aparece en la tabla → editar (formulario prellenado) → cambia teléfono → se refleja en la tabla
→ eliminar (confirmación nativa) → tabla vacía de nuevo. También verificado el fix de
`tax_scheme_name` en `CompanyForm`: elegir "01 — IVA" y confirmar que la pestaña de Revisión
muestra "01 — IVA", no el default. Sin errores de consola. Datos de prueba limpiados de la base
real después.

## Productos (2026-06-26)

Mismo patrón que Clientes (ver sección anterior), backend ya completo
(`internal/products`) — confirmado revisando `handler_products.go`/`service.go`/`errors.go`,
sin huecos.

- **`lib/products.ts`** (nuevo, sin memoización): `listProducts`/`createProduct`/
  `updateProduct`/`deleteProduct`.
- **`lib/catalogs.ts`** ganó `listUnitMeasures` (`GET /catalogs/unit-measures`) — catálogo
  incompleto a propósito (11 de los cientos de códigos reales UN/ECE Rec. 20; por eso
  `documents.Service` nunca lo valida del lado del servidor), pero un select con 11 opciones
  reales sigue siendo mejor que un input de texto libre.
- **`components/product-form/ProductForm.tsx`** (nuevo): un producto tiene menos campos que un
  cliente o una empresa — sin identificación, sin dirección. Único punto delicado:
  `unit_price_cents` viaja en centavos (mismo criterio que `domain.Line` en cofacture) pero el
  formulario lo muestra y edita en pesos normales, convirtiendo al guardar/precargar
  (`centsToAmount`/`× 100` al enviar) — mostrarle centavos crudos a quien llena el formulario
  hubiera sido confuso. Mismo fix de `tax_type_name` derivado de la selección de
  `tax_type_code` que ya se corrigió en `CompanyForm`/`CustomerForm`, aplicado bien desde el
  principio aquí.
- **`pages/ProductsPage.tsx`** (nuevo, ruta `/products`, ítem de Sidebar habilitado): misma
  tabla con cabecera `bg-tertiary` y zebra striping que `CustomersPage`, precio formateado con
  `Intl.NumberFormat("es-CO", { style: "currency", currency: "COP" })`.

Verificado con un navegador real: crear producto (unidad de medida + impuesto por defecto) →
aparece en la tabla con el precio formateado en pesos → editar (precio prellenado
correctamente de vuelta en pesos, no en centavos) → cambia precio → eliminar → tabla vacía de
nuevo. Sin errores de consola. Datos de prueba limpiados de la base real después.

## Sidebar: grupo "Documentos" expandible, se quita "Numeración" (2026-06-26)

El usuario notó que el ítem "Numeración" del Sidebar ya no tenía sentido como entrada propia
— la creación de resoluciones vive en Configuración → Empresa desde la Fase 1.7, y nunca llegó
a tener una página dedicada. Se eliminó del todo (no solo deshabilitado).

Para Facturas/Notas, en vez de un ítem plano "Facturas" se decidió un grupo expandible
**"Documentos"** con tres hijos: Factura Electrónica, Nota Crédito, Nota Débito — razón: el
backend ya maneja los 3 bajo un único `documents.Service` (mismo ciclo borrador→confirmar,
mismo `dian_document_type_code` los distingue), así que la navegación los agrupa igual en vez
de inventar 3 ítems de primer nivel sueltos. El patrón de árbol expandible (chevron que rota
90° al expandir/contraer) no es nuevo — ya lo describe `docs/reference/DESIGN_SYSTEM.md` secc.
6 para el sidebar original (conexiones → tablas), solo que el dashboard nuevo nunca lo había
necesitado hasta ahora.

- `components/Sidebar.tsx`: `NAV_ITEMS` pasa de una lista plana a una mezcla de hojas
  (`NavLeaf`) y grupos (`NavGroup`, con `children: NavLeaf[]`). Estado de expansión por grupo
  persistido en `localStorage` (`apidian.sidebarExpandedGroups`), "Documentos" expandido por
  default. Si el Sidebar completo está colapsado (modo solo-iconos), los grupos no se pueden
  expandir — no hay espacio para mostrar hijos, mismo criterio que ya aplicaba a las etiquetas
  de las hojas.
- Los 3 hijos de "Documentos" siguen deshabilitados ("próximamente") — construir la página de
  Facturas es la pieza más grande que falta y necesita su propia sesión de planeación (ver
  "Pendiente" abajo), esta sesión solo resolvió la navegación.

Verificado con un navegador real: "Numeración" ya no aparece en ningún lado; "Documentos"
aparece expandido por default mostrando sus 3 hijos; clic en "Documentos" colapsa/expande
correctamente; colapsar el Sidebar completo oculta los hijos sin romper nada. Sin errores de
consola.

## Simplificación: `tax_scheme_name`/`tax_type_name` ya no se mandan (2026-06-26)

Al planear la construcción de Factura, se encontró (y se corrigió en el backend, ver
`docs/apidian-architecture.md` sección 9.37) que `tax_scheme_name` (empresa/cliente) y
`tax_type_name` (producto) ahora se derivan del catálogo `tax_types` a partir del código — el
backend ya no acepta ni confía en el nombre que mande el cliente. Esto vuelve innecesaria la
lógica que esta misma sesión había agregado un poco antes (`handleTaxSchemeChange`/
`handleTaxTypeChange`, que derivaban el nombre en el `<select>` para no mandarlo
desincronizado): se quitó por completo de `CompanyForm`/`TaxStep`, `CustomerForm` y
`ProductForm` — los `<select>` de tipo de impuesto vuelven a ser un simple `onChange` que solo
actualiza el código, sin estado ni derivación para el nombre. El `<select>` sigue mostrando
"código — nombre" para elegir (viene del catálogo), solo que ya no hace falta mandar el nombre
de vuelta al guardar.

`CreateIssuerPayload` perdió el campo `tax_scheme_name` del todo (issuers nunca lo expuso en la
respuesta, era puramente de escritura). `CustomerPayload`/`ProductPayload` lo conservan en
`lib/types.ts`: la respuesta del backend sigue incluyéndolo (ahora siempre correcto), así que
sigue siendo un campo legítimo de lectura — solo que ningún formulario lo vuelve a escribir.

Verificado con un navegador real + inspección directa de Postgres: crear empresa/cliente/
producto eligiendo "01 — IVA" (no el default) y confirmar que `tax_scheme_name`/`tax_type_name`
quedan en `"IVA"` en la base, sin que el frontend lo haya mandado.

## Factura Electrónica — construcción del borrador y transmisión a la DIAN (2026-06-26)

Pedido explícito: usar todos los campos ya construidos de issuers/customers/products para
armar la factura, primero la construcción del borrador, después la transmisión — en ese orden
("primero la construcción de factura, luego la transmisión por favor"). Antes de empezar se
encontró (y se corrigió en el backend, ver `docs/apidian-architecture.md` sección 9.37) que
`computeTotals`/`aggregateTaxes` eran pass-through puro — esto determinó el contrato que el
formulario nuevo termina usando: cantidad/precio/% de impuesto, nunca montos ya calculados.

**Tipos y cliente HTTP**: `lib/types.ts` ganó `DocumentLineInput`/`DocumentLine`/`Tax`/
`PaymentMean`/`Totals`/`IssueInvoicePayload`/`Document` (espejo de `documentResponse`, sin
`billing_reference`/`discrepancy_response`/`note_type_code` — esos son solo de Nota Crédito/
Nota Débito, que no tienen UI todavía). `lib/documents.ts` (nuevo): CRUD de borradores +
`confirmDocument`; `lineToInput` es el inverso de lo que calcula el servidor, para poder
re-editar un borrador ya guardado sin mostrar campos calculados como si fueran editables.
`lib/catalogs.ts` ganó `listPaymentTerms`/`listPaymentMethods`; `listNumberingRanges` ganó un
filtro opcional por `dian_document_type_code`. `lib/invoiceMath.ts` (nuevo) replica la misma
fórmula que el servidor — **solo para la vista previa** mientras se escribe, nunca la fuente
de verdad final (eso siempre es la respuesta del servidor tras guardar). `lib/currency.ts`
(nuevo) — `formatCOP`/`centsToAmount`/`amountToCents` compartidos por los componentes nuevos
(no se tocó el `centsToAmount`/`formatCOP` propios que ya tenían `ProductForm`/`ProductsPage`,
para no arriesgar algo ya verificado sin necesidad).

**Componentes**: `components/party-fields/PartyFields.tsx` (nuevo, extraído de
`CustomerForm`) — los campos de un `CustomerPayload` como componente controlado
(`value`/`onChange`), reusado por `CustomerForm` y por la sección de cliente de Factura.
`components/invoice-form/`: `CustomerSection` (selector de cliente guardado — copia TODOS los
campos vía `customers.customerToPayload`, resuelve el hallazgo "snapshot incompleto" de la
Fase 1 — + `PartyFields` editable debajo), `LineItemsEditor` (selector de producto guardado o
entrada manual, vista previa de cada línea con `invoiceMath`), `PaymentMeansEditor` (resuelve
el hallazgo "PaymentMeans nunca se pide" de la Fase 1 — obligatorio, no se puede enviar el
formulario sin al menos una), `TotalsSummary`, `StatusBadge` (estados de `Document`, mismos
tokens pastel que `Banner`), `InvoiceForm` (junta todo + selector de rango de numeración
filtrado a Factura con "Próximo número: N" — resuelve el hallazgo "sin claridad del
consecutivo" de la Fase 1).

**Páginas y rutas**: `pages/InvoicesPage.tsx` (`/documents/invoices`, lista) y
`pages/InvoiceEditorPage.tsx` (`/documents/invoices/:id` — primer parámetro de ruta dinámico
de este frontend; `:id === "new"` crea, `draft` edita, cualquier otro estado es solo lectura
con el bloque de estado DIAN). `components/Sidebar.tsx`: "Factura Electrónica" pasó de
deshabilitada a su página real (Nota Crédito/Nota Débito siguen deshabilitadas, sin UI propia
todavía).

**Transmisión a la DIAN**: en `InvoiceEditorPage`, con el borrador en `status === "draft"`,
botón "Confirmar y enviar" — deshabilitado con aviso si el emisor activo no tiene
`has_software_credentials`/`has_certificate`. Al confirmar, recarga el documento y muestra
CUFE, enlace a la representación gráfica de la DIAN, y el bloque de estado (`dian_status_code`/
`dian_status_description`/`dian_status_message`).

**Limitación deliberada** (anotada, no se construye ahora): 0 o 1 impuesto por línea — la DIAN
permite más, pero es el caso avanzado que se agrega cuando haga falta de verdad. Tampoco hay
UI para líneas sin cargo (`free_of_charge`/`reference_price`, muestras comerciales).

**Verificado con un navegador real** (datos QA descartables): crear empresa/cliente/producto
→ crear factura (cliente guardado, línea con producto guardado, forma de pago) → el total
mostrado coincidió exactamente con el que devolvió el servidor tras guardar → editar (cambiar
cantidad, recalculó bien, `PUT` 200) → listar → eliminar. 0 errores de consola. Los dos
"bugs" que parecían reales en el camino resultaron ser artefactos del propio script de
verificación (navegar/cerrar el navegador antes de que la petición —de ~1.5-2s— terminara,
cancelando el contexto en el servidor), no del producto.

**Verificado contra la DIAN real** (sandbox de habilitación, NIT 6382356, ver
`docs/apidian-architecture.md` sección 9.38 para el detalle): confirmar una factura desde la
UI nueva construyó, firmó, envió y mostró la respuesta real de la DIAN de punta a punta —
**autorizada** (`StatusCode "00"`, `"La Factura electrónica SETP990000001, ha sido
autorizada."`). En el camino se encontró que el rango de numeración real seguía configurado
con un `test_set_id` obsoleto (de antes de que existiera `SendBillSync`), corregido
directamente en la base — ver el detalle en la sección 9.38, no es un hallazgo de frontend.

## Representación gráfica (PDF) — logo del emisor y botón "Ver PDF" (2026-06-27)

Siguiente pieza del ciclo de Factura — backend nuevo (`internal/pdf`, ver
`docs/apidian-architecture.md` sección 9.39), frontend mínimo para usarlo: subir el logo y
abrir el PDF, sin construir nada de la representación gráfica en sí (eso vive 100% en el
servidor).

- `lib/apiClient.ts` ganó `getBlob(path): Promise<Blob>` — la función `request()` existente
  siempre asume JSON (`res.text()` + `JSON.parse`), incompatible con bytes de un PDF/imagen.
- `lib/documents.ts` ganó `getInvoicePdfBlobUrl(id)` — trae el PDF como blob y devuelve un
  Object URL. Hace falta el blob (no un `<a href>` plano) porque el endpoint exige
  `Authorization: Bearer`, que un link estático no puede mandar.
- `components/issuer-settings/LogoForm.tsx` (nuevo, en Configuración → Empresa junto a
  `SoftwareCertificateForm`): sube el logo (reusa `fileToBase64`, ya existía) y muestra una
  vista previa en miniatura trayendo `GET /issuers/me/logo` con el mismo patrón de blob.
  `StatusBadge` (el "✓/—" de `SoftwareCertificateForm`) se exportó para reusarlo aquí — segundo
  caso real de uso, mismo criterio de extracción que el resto del proyecto.
- `InvoiceEditorPage`: botón "Ver PDF" junto al badge de estado — visible para cualquier
  factura ya creada (no en `:id === "new"`), funciona igual en borrador (muestra "BORRADOR" en
  vez de CUFE/QR/número) que confirmada. Abre el blob en una pestaña nueva
  (`window.open(objectUrl, "_blank")`).

**Verificado con un navegador real, de punta a punta**: subir un logo PNG real (se ve la vista
previa) → crear borrador de factura → "Ver PDF" muestra el logo + "BORRADOR" → confirmar
(habilitación, set de pruebas real ya cerrado — mismo resultado esperado de la sección 9.10 de
`apidian-architecture.md`, construye/firma/persiste con CUFE/QR/número reales antes de
intentar enviar) → "Ver PDF" de nuevo muestra el mismo logo + CUFE/QR/número reales. 0 errores
de consola. Se encontró y corrigió un bug real en el camino (`RangeTo` nulo mostrando un tope
inventado en el pie del PDF) — detalle completo en `apidian-architecture.md` sección 9.39, es
un hallazgo de backend, no de este frontend.

## Enviar la factura al cliente por correo (2026-06-27)

Cierre del ciclo completo (Factura → PDF → correo) — ver `apidian-architecture.md` sección
9.42 para el detalle de backend. Frontend mínimo, sin formulario propio (asunto/cuerpo son
fijos en el servidor):

- `lib/documents.ts` ganó `sendInvoiceEmail(id)` — `POST /documents/{id}/send-email`, sin body
  ni respuesta (204).
- `InvoiceEditorPage`: botón "Enviar al cliente" (icono `Mail`) junto a "Ver PDF" — visible
  **solo** cuando `doc.status === "accepted"` (nunca en borrador/rechazada/con error de envío,
  mismo criterio que el backend). `window.confirm` antes de enviar, mismo patrón que eliminar
  borrador/confirmar factura (es una acción visible para un tercero real, el cliente). Éxito
  muestra un `<Banner tone="success">` con el correo al que se envió; sin marca persistida de
  "ya enviado" — el botón sigue disponible para reenviar las veces que haga falta.

**Verificado de punta a punta contra Mailtrap real**: empresa/factura de prueba creadas vía API
contra Postgres real, documento marcado `accepted` directamente en la base (para no gastar un
consecutivo real de la DIAN solo para llegar a ese estado, ya probado de sobra en fases
anteriores) → en el navegador, el botón apareció solo por estar `accepted`, el diálogo de
confirmación mostró el correo correcto del cliente, y el banner de éxito apareció tras el envío
real. 0 errores de consola. Datos de prueba eliminados al terminar.

## Selector real de clasificación de ítems (2026-06-28)

`ProductForm.tsx` reemplaza los 3 campos de texto libre ("Código de estándar"/"Nombre del
estándar"/"ID de agencia" — mal diseñados, dejaban guardar cualquier valor) por un `<Select>`
real cargado de `listItemStandards()` (`lib/catalogs.ts`, nuevo) — ver
`apidian-architecture.md` sección 9.45 para el detalle completo del bug real que esto corrige
(un código UNSPSC real terminó guardado donde debía ir un selector de 4 valores fijos, causando
un rechazo real de la DIAN). El campo "Código del ítem" existente se reutiliza para el código
real dentro del estándar elegido, con placeholder contextual. `LineItemsEditor.tsx` no
necesitó UI propia — solo se le quitó estado muerto (`itemTypeName`/`itemTypeAgencyId` nunca
tuvieron input propio, solo se copiaban del producto sin mostrarse).

**Hallazgo de proceso**: durante esta fase se descubrió que `npx tsc --noEmit` no revisaba
ningún archivo de este proyecto (`tsconfig.json` es de solo-referencias) — el comando correcto
es `npx tsc -b` (o `npm run build`). Ver memoria `feedback-tsc-noemit-silently-checks-nothing`.

## Alertas/confirmaciones propias + sesión resiliente (2026-07-01)

Tres mejoras de confianza en la UI pedidas por el usuario tras el primer recorrido completo
del dashboard con datos reales:

1. **`context/ConfirmContext.tsx`** (nuevo): reemplaza los 7 `window.confirm` nativos.
   `useConfirm()` devuelve una función `confirm(message, options?) → Promise<boolean>` — mismo
   contrato que el nativo, migración mecánica en cada sitio. Internamente renderiza un modal
   centrado con `Card`/`Button` del design system; el tono `"danger"` colorea el botón de
   confirmar en rojo. Montado una vez en `App.tsx`.

2. **`context/ToastContext.tsx`** + **`components/ui/Toast.tsx`** (nuevos): reemplaza los
   `<Banner>` de resultado de acción que quedaban fijos hasta que otra cosa los pisara.
   `useToast()` devuelve `{ success, error }` — cada llamada agrega un toast con auto-dismiss
   (éxito 4 s, error 6 s) y botón × para cerrar antes. Contenedor `fixed top-3 right-3 z-50`.

3. **Sesión resiliente**: `AuthContext` ahora valida el JWT contra `GET /issuers` al arrancar.
   Si el servidor responde 401 → cierra sesión (token inválido/expirado). Si no hay respuesta
   de red → marca `connectionError = true` sin cerrar la sesión (las credenciales no tienen la
   culpa si el backend está apagado). `ConnectionBanner.tsx` (nuevo) muestra la alerta de "sin
   conexión" + botón Reintentar, montado globalmente en `App.tsx`. `isReady` solo pasa a
   `true` después de que ese chequeo termine — evita que el dashboard parpadee con datos
   obsoletos.

Migración completada: `window.confirm` → `useConfirm()` en `LogoForm`, `NumberingRangesPanel`,
`CustomersPage`, `ProductsPage`, `InvoiceEditorPage`. Resultados de acción → `useToast()` en
los mismos sitios + `SoftwareCertificateForm`, `CustomerSection`.

Verificado de punta a punta con navegador real: apagar el backend → el dashboard sigue visible
con aviso de sin-conexión en vez de mandar a login → encender y Reintentar → aviso desaparece
sin recargar. Confirmar/eliminar/enviar correo → modal propio + toast de éxito/error que
desaparece solo.

## Nota Crédito y Nota Débito — ciclo completo (2026-07-01 / 2026-07-07)

Mismo ciclo completo que Factura Electrónica — lista, editor/visor, formulario, rutas, sidebar,
botón de emisión desde la factura fuente.

### Nota Crédito (NC)

- **`components/invoice-form/CreditNoteForm.tsx`** (nuevo): formulario de borrador NC — igual
  que `InvoiceForm` pero sin selector de rango de numeración general (filtra automáticamente a
  tipo `"91"`), agrega selector de `CreditNoteTypeCode` (motivo de anulación/corrección), y
  prellenado automático de `BillingReference` desde la factura fuente vía `?from=<id>`.
- **`pages/CreditNotesPage.tsx`** + **`pages/CreditNoteEditorPage.tsx`** — mismos patrones que
  `InvoicesPage`/`InvoiceEditorPage`. El visor en solo lectura muestra el concepto del motivo
  (`note_type_code`) y el bloque de estado DIAN.
- **`InvoiceEditorPage`**: botón "Emitir Nota Crédito" (icono `FileMinus`) visible solo cuando
  `status === "accepted"` — navega a `/documents/credit-notes/new?from=<id>`.
- Rutas: `/documents/credit-notes` y `/documents/credit-notes/:id`.
- **Verificado contra la DIAN real**: NC enviada y aceptada (`StatusCode "00"`) referenciando
  la factura de habilitación. Bug encontrado y corregido en el camino: `CreditNoteTypeCode`
  estaba hardcodeado a `"91"` en el backend (mismo valor que `dian_document_type_code`) — debía
  ser el código del motivo de corrección (ej. `"1"`). Ver `apidian-architecture.md` sección
  9.48.

### Nota Débito (ND)

- **`components/invoice-form/DebitNoteForm.tsx`** (nuevo): igual que `CreditNoteForm` pero sin
  `CreditNoteTypeCode`; los códigos de concepto son de la Lista 22 del Anexo Técnico
  (1=Intereses, 2=Gastos por cobrar, 3=Cambio del valor), distintos a los de NC.
- **`pages/DebitNotesPage.tsx`** + **`pages/DebitNoteEditorPage.tsx`**.
- **`InvoiceEditorPage`**: botón "Emitir Nota Débito" (icono `FilePlus`) — mismo criterio de
  visibilidad que NC.
- Rutas: `/documents/debit-notes` y `/documents/debit-notes/:id`.
- **Verificado contra la DIAN real**: ND enviada y aceptada sin problemas.

### Bug crítico encontrado: JSON tags en structs de backend

`BillingReferenceInput` y `DiscrepancyResponseInput` (Go) carecían de tags `json:"..."` —
sin ellos, `json.Marshal` produce claves PascalCase (`Prefix`, `Number`...) pero PostgreSQL
`->>'prefix'` (JSONB) es case-sensitive y devuelve NULL. Esto hacía que:
- El filtro `source_document_id` en `GET /documents` no funcionaba.
- Las queries de conteo NC/ND nunca encontraban coincidencias.

Fix: se agregaron tags a ambos structs y se ejecutó un script one-time (`cmd/fixjsonb`) para
corregir los 5 documentos ya existentes en la base. Ver `apidian-architecture.md` sección 9.48.

## Columna "Referencias" condicional en tabla de facturas (2026-07-07)

La tabla de `InvoicesPage` tiene una columna "Referencias" que **solo aparece cuando al menos
una factura de la página actual tiene NC o ND asociadas**. Si ninguna las tiene, la columna
no existe (ni el `<th>` ni los `<td>`).

Implementación:
- Backend: una sola query adicional por llamada a `ListByIssuer` (nunca N+1) — cuenta NC/ND
  agrupadas por `(ref_prefix, ref_number)` para todos los documentos del emisor, y las anota en
  los resultados del listado como `NCCount`/`NDCount` (campos con `omitempty`, ausentes cuando
  son cero).
- Frontend: `const hasRefs = documents?.some((d) => (d.nc_count ?? 0) > 0 || (d.nd_count ?? 0) > 0)` —
  `<th>` y `<td>` condicionados a `{hasRefs && ...}`. Los badges usan `--color-warning-bg/text`
  (NC, naranja) e `--color-info-bg/text` (ND, azul) definidos como tokens CSS en `index.css`.
- `InvoiceEditorPage` muestra además una sección "Notas emitidas sobre esta factura" con links
  a cada NC/ND cuando el documento ya está aceptado.

## Rediseño de navegación: sidebar plano + SubNav de pestañas (2026-07-09)

El usuario pidió reemplazar el sidebar con grupos expandibles/flyouts (confuso al colapsar) por
un patrón más limpio y escalable:

**Sidebar plano** (`components/Sidebar.tsx`): solo 5 ítems de primer nivel — Inicio, Documentos,
Clientes, Productos, Configuración. Sin grupos, sin flyouts, sin chevrons, sin `expandedGroups`
en localStorage. "Documentos" apunta a `/documents/invoices` pero queda activo para cualquier
ruta `/documents/*` (campo `activePrefix` en la definición del ítem, leído desde `useLocation()`
en el render — no depende del `isActive` de NavLink). El toggle de colapso se mantiene igual.

**`components/SubNav.tsx`** (nuevo): barra de pestañas contextual, `h-10`, justo debajo del
navbar. Lee `useLocation().pathname`, busca el primer prefijo en un array de configuración, y si
hay match renderiza sus pestañas; si no, no renderiza nada. Tab activa: `border-b-2
border-(--accent-primary) text-(--accent-primary)` con `-mb-px` para que la línea de color
tape el borde inferior del contenedor. Tab inactiva: `border-transparent text-(--text-secondary)`.

Configuración de sub-navegación actual en `SUB_NAVS`:
```ts
{ prefix: "/documents", items: [Factura Electrónica, Nota Crédito, Nota Débito] }
{ prefix: "/settings",  items: [General, Mi cuenta, Empresa] }
```
Para agregar sub-páginas a una sección futura, solo se agrega una entrada a `SUB_NAVS` sin
tocar el Sidebar.

**`DashboardLayout.tsx`**: el espacio a la derecha del Sidebar ahora es `flex flex-col
overflow-hidden` → `<SubNav />` (altura fija) → `<main class="flex-1 overflow-auto">` (scrollea
independientemente). SubNav tiene la misma altura `h-10` que el header del sidebar (donde vive
el hamburger), formando una línea horizontal visual continua.

## Configuración con rutas reales (2026-07-10)

La pestaña anterior de `SettingsPage` usaba `useState` para recordar qué pestaña activa
estaba — al recargar siempre volvía a "General". Reemplazado por rutas propias:

| Ruta | Componente | Contenido |
|---|---|---|
| `/settings` | `<Navigate to="/settings/general" replace />` | Redirige |
| `/settings/general` | `SettingsGeneralPage` | Toggle de tema |
| `/settings/account` | `SettingsAccountPage` | Nombre y correo (solo lectura) |
| `/settings/company` | `SettingsCompanyPage` | Software, certificado, logo, rangos |

El SubNav maneja la navegación entre sub-páginas (mismo patrón que `/documents/*`). Recargar la
página en `/settings/company` mantiene la sub-página correcta. `SettingsPage.tsx` original queda
como archivo muerto (no importado); los 3 componentes nuevos viven en `pages/Settings*.tsx`.

## Estándar de títulos de página con icono (2026-07-10)

Todas las páginas del dashboard tienen `<h1>` con un icono de Lucide React a la izquierda:

```tsx
<h1 className="flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
  <Icon className="h-4 w-4 shrink-0 text-(--text-secondary)" />
  Título
</h1>
```

El icono es `text-(--text-secondary)` (muted) para no competir con el texto. Mapa completo:

| Página | Icono |
|---|---|
| Factura Electrónica (lista y editor) | `FileText` |
| Nota Crédito (lista y editor) | `FileMinus` |
| Nota Débito (lista y editor) | `FilePlus` |
| Clientes | `Users` |
| Productos | `Package` |
| Configuración › General | `SlidersHorizontal` |
| Configuración › Mi cuenta | `User` |
| Configuración › Empresa | `Building2` |
| Mis empresas | `Building2` |

## Estandarización de Card en paneles de configuración (2026-07-10)

Los paneles de Configuración › Empresa usaban `<div>` con `border-color` (borde más oscuro, sin
sombra, `rounded` pequeño, fondo heredado) — visualmente distintos a las cards de Clientes/
Productos (que usan el componente `Card`, `rounded-lg`, `shadow-xl`, `bg-secondary`). Unificado:
`SoftwareCertificateForm`, `LogoForm`, `PublicRegistrationPanel`, `NumberingRangesPanel` y las
dos páginas nuevas `SettingsGeneralPage`/`SettingsAccountPage` ahora usan `<Card className="...">`.

**Componente `Card`**: `rounded-lg border border-(--border-light) bg-(--bg-secondary) shadow-xl` —
este es el contenedor estándar para cualquier panel de contenido o sección de formulario del
dashboard. No sustituir por `<div>` con borde manual.

## Jerarquía visual de formularios (2026-07-10)

Dentro de `InvoiceForm`, `CreditNoteForm` y `DebitNoteForm`, las secciones "Cliente", "Líneas"
y "Forma de pago" (y "Respuesta de discrepancia" en NC/ND) estaban separadas solo por `gap-4` —
sin ninguna marca visual, todo fluía como un único bloque blanco.

**Solución adoptada**: línea divisoria entre secciones con `border-t border-(--border-color) pt-3`
en el `<section>`. Fondo blanco en todo el formulario, sin cajas ni headers grises — la línea
es el único separador.

**Estándar para fondos de contenedores internos**:
- **Paneles interactivos** (el usuario llena campos aquí — `LineItemsEditor` add-form,
  `PaymentMeansEditor` add-form): `bg-(--bg-secondary)` (blanco). El borde propio
  (`border border-(--border-color)`) los delimita sin el gris de fondo que confundía con
  los inputs y botones.
- **Paneles de solo lectura / resumen** (`TotalsSummary`, `CustomerSection` modo summary,
  bloque de estado DIAN en `InvoiceEditorPage`/`CreditNoteEditorPage`/`DebitNoteEditorPage`):
  `bg-(--bg-primary)` (#f5f5f5). El gris los distingue intencionalmente como información,
  no como áreas editables.
- **Striping de tablas**: `i % 2 === 0 → bg-(--bg-primary)`, `i % 2 === 1 → bg-(--bg-secondary)`.
  Patrón intocable, igual en toda la app.

## Estándar de color de icono en títulos de página (2026-07-10)

El icono en los `<h1>` de las páginas del dashboard debe usar `text-(--accent-primary)` (azul),
no `text-(--text-secondary)` (gris). La referencia correcta siempre fue `OnboardingPage`, que
ya usaba el azul desde el principio — los 12 archivos de páginas se corrigieron en bloque.

**Patrón canónico de título de página**:
```tsx
<h1 className="... flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
  <Icon className="h-4 w-4 shrink-0 text-(--accent-primary)" />
  Título visible
</h1>
```

Regla: icono `h-4 w-4 shrink-0 text-(--accent-primary)` siempre. Los iconos de nav del
sidebar son `h-3.5 w-3.5 text-(--text-secondary)` (más pequeños, más apagados — nunca
confundir los dos contextos).

## CIIU: selector con lista descriptiva, oculto cuando vacío (2026-07-10)

El `TagInput` que mostraba los códigos CIIU seleccionados como chips de código solo (`"A0111"`)
se reemplazó en `TaxStep.tsx` por un render condicional propio:

- **Vacío** → no se renderiza nada (el contenedor con borde ya no aparece).
- **Con selecciones** → lista vertical, una fila por código: `código` (monospace) + descripción
  (texto secundario) + botón × para quitar. La descripción viene de buscar el `label` del
  catálogo `ciiuOptions` — formato `"código — descripción"`. **Solo el código va a la base de
  datos** (`industry_classification_codes: string[]`), la descripción es puramente visual.

El componente `TagInput` sigue existiendo para otros usos futuros, pero en este caso concreto
no aplica porque necesitamos mostrar más información que el código solo.

## PDF y correo para NC/ND (2026-07-10)

Las funciones `getInvoicePdfBlobUrl` y `sendInvoiceEmail` de `lib/documents.ts` se renombraron
a `getDocumentPdfBlobUrl` y `sendDocumentEmail` — los endpoints `/documents/{id}/pdf` y
`/documents/{id}/send-email` son genéricos y funcionan para cualquier tipo de documento.

`CreditNoteEditorPage` y `DebitNoteEditorPage` reciben los mismos botones que `InvoiceEditorPage`:
- **"Ver PDF"** — visible siempre que el documento exista (borrador o confirmado).
- **"Enviar al cliente"** — visible solo cuando `status === "accepted"`.

El ciclo completo (borrador → confirmar → PDF → correo) queda parejo entre Factura, NC y ND.

## Pendiente para próximas fases

- Endpoint para editar perfil de usuario (nombre/correo) — `SettingsAccountPage` sigue siendo
  solo lectura porque no existe endpoint en el backend.
- Filtros en la lista de facturas (por estado, fecha, cliente).
