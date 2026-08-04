# Diseño de un ERP en Go con Arquitectura Hexagonal

---

## Estado de implementación

### ✅ Módulos completos — API backend corriendo en Neon (`v2/db-architecture`)

| Módulo | Descripción |
|---|---|
| `catalog/` | Catálogos DIAN, países, monedas, unidades, tipos de documento |
| `security/` | Usuarios, JWT, invitaciones, perfiles, multi-empresa |
| `company/` | Empresa (NIT, configuración fiscal, certificado DIAN, logo) |
| `company/ → warehouses` | Bodegas: CRUD completo bajo `/companies/active/warehouses` |
| `customer/` | Clientes con datos tributarios |
| `supplier/` | Proveedores |
| `product/` | Productos con unidad, clasificación DIAN, precios |
| `inventory/` | Movimientos de inventario (entrada, salida, ajuste) + stock por bodega |
| `purchase/` | Órdenes de compra + recepción de mercancía (`POST /purchases/{id}/receive`) |
| `sales/` | Ventas (draft → confirmed → cancelled) |
| `sales/ → quotes` | Cotizaciones: draft → sent → accepted/rejected → convert-to-sale |
| `sales/ → payments` | Pagos recibidos y cartera (cuentas por cobrar) con saldo pendiente |
| `accounting/` | Plan de cuentas PUC, períodos, libro diario, balance general, estado de resultados |
| `electronic/` | FE, NC, ND, DS, NA — motor cofacture; PDF (`GET /documents/{id}/pdf`); factura desde venta (`POST /invoices/from-sale/{sale_id}`) |
| `payroll/` | Nómina colombiana: empleados, contratos, liquidaciones, seed SMMLV/ARL |
| `hr/` | Gestión de ausencias (vacaciones, incapacidades, licencias) |
| `shared/tenant/` | Multi-tenancy por contexto — `GetCompanyID(ctx)` |
| `shared/events/` | Bus de eventos en memoria |
| `shared/logger/` | Logger columnar estructurado (`slog.Handler`) + middleware HTTP con método/ruta/status/duración |
| `shared/money/` | Tipo Money (cents + currency) |
| `shared/cryptutil/` | AES-256-GCM para datos sensibles |
| `shared/timeutil/` | Zona horaria Colombia (America/Bogota) |
| `shared/email/` | Puerto `Sender` + implementación SMTP con TLS/STARTTLS |
| `shared/notification/` | Motor multicanal implementado: noop, SMTP, Resend |
| `shared/reports/` | Motor de documentos: HTML (html/template), PDF (chromedp), Excel (excelize), CSV — con templates para factura, nómina, cotización, etc. |
| `stats/` | Módulo transversal de lectura — `GET /api/v1/stats/billing`; queries directas sobre `electronic.documents`; diseñado para extenderse a ventas, compras, nómina sin importar módulos Go |
| `audit/` | Módulo transversal de escritura/lectura — `GET /api/v1/audit-events`; tabla `audit.events`; otros módulos lo consumen vía interfaz `AuditLogger` local (sin importar el paquete audit); `electronic` loguea create/confirm/delete |

### ✅ Eventos inter-módulo cableados

| Evento | Publicador | Suscriptores activos |
|---|---|---|
| `sale.confirmed` | `sales` | `inventory` — descuenta stock; `accounting` — asiento CxC/ingresos/IVA |
| `purchase.received` | `purchase` | `inventory` — entrada de mercancía; `accounting` — asiento inventario/proveedor/IVA descontable |
| `payroll.generated` | `payroll` | `accounting` — asiento gasto de nómina/deducciones/salarios por pagar |

### 🔲 Pendiente — segunda fase

#### Módulo `audit/` — integración progresiva en los demás módulos

`audit/` ya existe y está integrado en `electronic/` (create, confirm, delete). Cada módulo nuevo o existente que requiera trazabilidad debe recibir un `AuditLogger` en su handler HTTP e invocarlo después de cada operación relevante. La interfaz se declara localmente en el paquete consumidor para no crear acoplamiento de importación.

Módulos pendientes de integrar cuando se requiera:

| Módulo | Acciones sugeridas |
|---|---|
| `company/` | perfil actualizado, credenciales cambiadas, logo subido/eliminado |
| `security/` | login, registro, invitación aceptada, selección de empresa |
| `customer/` | creado, actualizado, eliminado |
| `supplier/` | creado, actualizado, eliminado |
| `product/` | creado, actualizado, eliminado |
| `sales/` | venta confirmada, cancelada, pago registrado |
| `purchase/` | orden confirmada, recepción registrada |
| `payroll/` | nómina generada, empleado creado/desvinculado |
| `hr/` | ausencia registrada, aprobada |

Patrón a seguir: ver `electronic/interfaces/http/handlers.go` — interfaz `AuditLogger` local + campo `audit` en `Handler` + helper `logDoc` o equivalente.

---

#### Módulo `stats/` — extensión a otros dominios

`stats/` hoy solo cubre `electronic.documents`. Cuando se requieran KPIs de otros módulos, agregar métodos al repositorio de stats con queries directas sobre los schemas correspondientes (sin importar módulos Go).

Dominios sugeridos:

| Endpoint futuro | Fuente de datos |
|---|---|
| `GET /api/v1/stats/sales` | `sales.sales`, `sales.payments` |
| `GET /api/v1/stats/inventory` | `inventory.movements` |
| `GET /api/v1/stats/payroll` | `payroll.payslips` |
| `GET /api/v1/stats/purchases` | `purchase.purchases` |

---

#### Módulo `security/` — gestión de usuarios, roles y seeds

**Estado actual del módulo:**

El módulo existe y funciona para el flujo básico: registro, login, JWT, invitaciones, selección de empresa. Lo que falta es la capa de administración completa.

---

**Seeds de usuarios — no existen todavía**

Actualmente no hay ningún script de seed para `security.users`. El primer usuario se crea manualmente vía `POST /auth/register` desde el frontend. Esto debe cambiar:

| Seed | Descripción | Prioridad |
|---|---|---|
| `superadmin` | Dueño de la plataforma SaaS — acceso total, puede ver todas las empresas, gestionar suscripciones | Alta |
| `admin` (owner) | Dueño de una empresa tenant — crea la empresa, invita usuarios, configura credenciales DIAN | Alta |
| `counter` | Contador — acceso solo a módulos contables, electrónicos y nómina; sin acceso a configuración de la empresa ni al panel SaaS | Media |
| `user` | Empleado operativo — acceso solo a ventas, compras, inventario; sin módulos financieros ni configuración | Media |

Los seeds deben ir en `erp/internal/security/infrastructure/persistence/postgres/seed/` (mismo patrón que los otros módulos que ya tienen seed). El `superadmin` debe tener una contraseña tomada de una variable de entorno (`SUPERADMIN_PASSWORD`) para no hardcodearla.

---

**Endpoints pendientes del módulo `security/`:**

| Método | Path | Descripción | Estado |
|---|---|---|---|
| `GET` | `/api/v1/users` | Listar usuarios de la empresa activa | 🔲 Falta handler HTTP (el `repo.List()` existe en Go) |
| `GET` | `/api/v1/users/{id}` | Perfil de un usuario específico | 🔲 |
| `PUT` | `/api/v1/users/{id}/activate` | Activar/desactivar usuario | 🔲 |
| `DELETE` | `/api/v1/users/{id}` | Dar de baja usuario de la empresa | 🔲 |
| `GET` | `/api/v1/auth/companies` | Listar empresas del usuario actual | ⚠️ Es un stub — devuelve `{user_id, name}` en vez de la lista real |
| `POST` | `/api/v1/auth/refresh` | Renovar JWT sin re-login | 🔲 |

---

**Problemas conocidos:**

- `POST /auth/invite` — crea el usuario y genera el `invite_token` en DB, pero **no envía el correo** (TODO pendiente en el código). El token queda en la base de datos; para que funcione hoy hay que sacarlo manualmente de la DB o exponerlo temporalmente en la respuesta.
- `GET /auth/companies` — stub que no lista las empresas reales del usuario. Afecta el caso de un usuario con múltiples empresas: el login auto-selecciona la empresa solo cuando hay exactamente una; con más de una empresa el JWT queda con `cid=nil` y no hay pantalla de selección funcional.
- **Sin RBAC por módulo** — el campo `role` existe en `security.users` y `security.user_companies` (`owner/admin/member`) pero no hay middleware que lo aplique. Todos los usuarios autenticados acceden a todos los endpoints del tenant. La granularidad por módulo (counter solo ve contabilidad, user solo ve ventas) aún no está implementada.

---

**Diseño propuesto de roles:**

```text
superadmin     ← plataforma SaaS (transversal a todos los tenants)
  └─ puede listar/ver todos los tenants
  └─ puede gestionar suscripciones y planes
  └─ no tiene acceso a datos de negocio de cada tenant

owner          ← empresa (alias "admin" en el JWT actual)
  └─ configura la empresa: NIT, certificado DIAN, software DIAN, logo
  └─ invita y gestiona usuarios de su empresa
  └─ acceso total a todos los módulos del tenant

counter        ← contabilidad y documentos electrónicos
  └─ módulos: accounting/, electronic/, payroll/
  └─ sin acceso a: company/credentials, security/users

user           ← operativo
  └─ módulos: sales/, purchase/, inventory/, customers/, suppliers/, products/
  └─ sin acceso a: accounting/, payroll/, hr/, electronic/, configuración
```

El middleware de autorización por rol debe ir en `shared/tenant/` como una segunda capa después de la autenticación: `AuthMiddleware → RoleMiddleware(requiredRole)`.

---

#### Eventos aún no cableados

| Evento | Publicador | Suscriptor pendiente | Motivo |
|---|---|---|---|
| `sale.confirmed` | `sales` | `electronic` — generar FE automáticamente | Requiere que el usuario seleccione rango de numeración; hoy existe como endpoint explícito `POST /electronic/invoices/from-sale/{sale_id}` |
| `stock.moved` | `inventory` | `accounting` — asiento de movimiento de inventario | Requiere costo unitario del producto en el modelo de datos |

#### Sub-entidades pendientes por módulo

| Módulo | Falta |
|---|---|
| `payroll/` | Primas, cesantías, liquidación definitiva, vacaciones acumuladas |
| `hr/` | Control de asistencia, reclutamiento, hoja de vida del empleado |
| `company/` | Sucursales, parámetros de configuración por empresa |
| `inventory/` | Lotes/vencimientos, seriales |
| `electronic/` | Nómina electrónica DIAN (tipos 102/103) |

#### Módulo `accounting/` — pendientes de la fase de cierre contable

Contexto: en `v2/db-architecture` se completó de punta a punta pagos→contabilidad, retenciones en compras + certificados, conciliación bancaria, activos fijos + depreciación, presupuestos, y declaraciones de IVA/Renta/ICA (backend + frontend, commit `58c2cf1`). De esa fase quedaron los siguientes pendientes identificados, sin resolver todavía:

| Pendiente | Descripción | Prioridad |
|---|---|---|
| Import masivo de tarifas ICA | Hoy la carga de tarifas (`accounting.ica_tariffs`) es manual, una fila a la vez, por formulario. Cada contador conoce la tarifa de su propio municipio (+1100 en Colombia) — se necesita: botón "Descargar plantilla" (CSV con columnas `municipality_code,ciiu_code,fiscal_year,rate_bp,surcharge_bp` + fila de ejemplo), subida del archivo lleno, y un endpoint que valide cada fila (contra el catálogo CIIU real, ver siguiente ítem) y haga upsert por lote | Alta |
| Sincronizar CIIU de ICA con el catálogo ya existente | El sistema ya tiene el catálogo CIIU oficial completo: tabla `catalog.ciiu_codes` (508 códigos), servido en `GET /api/v1/catalogs/ciiu-codes`, ya usado por el frontend (`lib/ciiu.ts`, `TaxStep.tsx`) para el perfil de empresa. Los formularios de tarifa/declaración ICA (`AccountingTaxDeclarationsPage.tsx`) dejaron el campo CIIU como texto libre en vez de reusar ese mismo catálogo — no requiere cambio de esquema (accounting no cruza FKs con catalog, por diseño), solo cambiar el `<Input>` por el `Combobox` de `listCiiuOptions()` que ya existe | Alta |
| Import masivo de extracto bancario | La conciliación bancaria (`AccountingBankPage.tsx`) solo permite agregar movimientos del extracto uno por uno a mano. Falta poder subir el extracto real del banco (CSV/Excel/OFX) y parsearlo en lote | Media |
| Baja de activos fijos (disposal) | Se construyó alta + depreciación mensual por línea recta, pero no el flujo de "dar de baja" un activo. Los campos `gain_account`/`loss_account`/`disposed_at`/`disposal_amount`/`disposal_journal_id` ya existen en el esquema (agregados al unificar migraciones, commit pendiente) pero ningún caso de uso los postea todavía | Media |
| `voucher_types` no está conectado al posteo | El CRUD de tipos de comprobante (`POST/GET /accounting/voucher-types`) es administrativo nada más — verificado que `PostJournalUseCase` acepta cualquier string en `voucher_type` sin validarlo contra esa tabla. No rompe nada, pero hoy no sirve para nada tampoco | Baja |
| Tablas del esquema sin usar | `accounting.reconciliation_marks` y `accounting.exchange_rates` existen desde la migración original pero no tienen dominio/caso de uso/handler — confirmado con búsqueda exhaustiva. Eran para una conciliación genérica y multi-moneda que nunca se construyó (la conciliación bancaria real usa `bank_statement_lines`, diseño aparte que sí funciona). Decidir: implementarlas si se necesita multi-moneda, o limpiarlas del esquema | Baja |
| Nombre de cuenta `143505` | Se agregó `143505 Mercancías no fabricadas por la empresa` (repite el nombre del padre `1435`) porque no existía subcuenta de mercancías en el PUC real extraído de puc.com.co — hay 2 precedentes de ese patrón en el catálogo (`540505`, `590505`) pero no es lo común. Pendiente decidir si se le pone un nombre más específico | Baja |
| Verificación visual en navegador | Todo el frontend de esta fase (bancos, activos fijos, presupuestos, declaraciones, certificados, retenciones) se verificó por `npm run build` + pruebas curl contra el backend real, nunca abriendo un navegador de verdad | Media |

---

#### Menú (sidebar) — jerarquía pendiente de aplicar + catálogos vs. sub-entidades

Contexto: se diseñó y prototipó una jerarquía nueva para el sidebar (hoy es una lista plana de 11 ítems). El prototipo vive en `design/erp-ui-proposal/dashboard.html` + `shared.css` — **todavía no está aplicado al `frontend/src/components/Sidebar.tsx` real**, solo la parte de anclar Configuración al fondo (con `flex:1` en el nav de arriba + bloque separado abajo).

**Jerarquía acordada (3 grupos + Configuración anclada abajo, sin FKs entre módulos, pensada para poder prender/apagar un grupo completo según el plan SaaS que contrate cada empresa a futuro — solo Documentos Electrónicos / solo Nómina / ERP completo):**

```
Inicio
Comando                    ← solo superadmin, separado (es modo plataforma, no modo empresa)
─────────────
OPERACIÓN
  Documentos electrónicos
  Ventas
  Compras
  Inventario
FINANZAS
  Contabilidad
  Nómina                   ← módulo backend existe (payroll/), sin frontend
  RRHH                     ← módulo backend existe (hr/), sin frontend
CATÁLOGOS
  Clientes
  Proveedores
  Productos
─────────────
Configuración               ← anclada al fondo (ya aplicado en el Sidebar.tsx real)
```

Pendiente: aplicar los 3 grupos (`OPERACIÓN`/`FINANZAS`/`CATÁLOGOS`) con títulos de sección al `Sidebar.tsx` real — hoy solo tiene Configuración anclada, sin agrupar el resto.

**Regla para decidir si algo es un catálogo (`CATÁLOGOS`) o vive dentro de su módulo dueño:** un ítem va en `CATÁLOGOS` solo si lo necesitan *varios módulos no relacionados* al mismo tiempo (Clientes/Proveedores/Productos los usan Ventas, Compras, Inventario y Documentos). Si solo lo usa un módulo para su propio flujo interno, va como pestaña *dentro* de ese módulo — mismo patrón que Bodegas dentro de Inventario (`WarehousesPage.tsx`, subnav Existencias/Movimientos/Bodegas).

Aplicando esa regla, quedan pendientes (ninguno cambia la estructura del menú ya definida, solo dónde entra cada uno):

| Elemento | Estado | Dónde debería vivir |
|---|---|---|
| Empleados | Backend existe (`payroll.Employee`, `erp/internal/payroll/domain/employee.go`), sin página frontend | Pestaña dentro de **Nómina** — NO como catálogo aparte, y NO reusando las tablas de Clientes/Proveedores (retención salarial, aportes a seguridad social y cuenta contable 2505 son completamente distintos al tratamiento comercial de Cliente/Proveedor — son 3 structs Go independientes hoy, y así debe seguir) |
| Sucursales | No existe (ni backend ni frontend) — ya estaba anotado en "Sub-entidades pendientes por módulo" arriba (`company/`) | Dentro de **Configuración → Empresa**, no como catálogo — es sub-configuración de la empresa, no dato maestro compartido |
| Aviso de identificación duplicada entre catálogos | No existe — diseño detallado abajo (ver "Búsqueda cruzada de terceros") | Mejora de UX, no de esquema: una misma cédula/NIT puede existir legítimamente en más de un catálogo a la vez (ej. un empleado que también es cliente por compras con descuento de empleado). Al crear un Cliente/Proveedor/Empleado, avisar (sin bloquear ni fusionar) si esa identificación ya existe en otro catálogo — "ya existe como Empleado, ¿es la misma persona?" |

---

#### Búsqueda cruzada de terceros al crear Cliente/Proveedor/Empleado (para cuando se construya Información Exógena)

Contexto: surgió al comparar con Odoo (modelo `res.partner` unificado) y SIESA (catálogo de "Terceros" unificado, típico en ERPs colombianos por el reporte de Información Exógena DIAN, que agrega montos por NIT sin importar el rol). Se decidió **no** unificar Cliente/Proveedor/Empleado en una sola tabla — rompería el principio de módulos independientes sin FKs cruzadas que ya sigue todo el proyecto, y el reporte de exógena no lo necesita (se resuelve en la capa de lectura/CQRS cruzando por NIT al momento de generarse, no en el modelo de escritura). En cambio, se diseñó una asistencia de captura: buscar el NIT en los otros catálogos y ofrecer copiar los datos, sin fusionar ni crear relación estructural.

**1. Módulo nuevo, de solo lectura, sin tabla propia:**

```
shared/thirdparty/
    domain/
        port.go        ← CustomerPort, SupplierPort, EmployeePort
                          (cada uno: FindByIdentification(ctx, companyID, idNumber) → *Match)
        match.go        ← ThirdPartyMatch{Source, ID, Name, Email, Phone, Address...}
    application/
        lookup.go        ← LookupUseCase: llama a los 3 puertos, arma la lista de coincidencias
    infrastructure/
        customer/adapter.go   ← implementa CustomerPort envolviendo customer.Repository real
        supplier/adapter.go   ← igual, con supplier.Repository
        payroll/adapter.go    ← igual, con payroll.EmployeeRepository
    interfaces/http/
        handlers.go      ← GET /api/v1/third-parties/lookup?identification_number=XXX
```

Mismo patrón de "puerto local" que ya usa `shared/notification` para leer stock/numeración de otros módulos sin ser dueño de esos datos — nada de tabla nueva ni FK cruzado.

**2. Flujo en el frontend:**

```
Usuario escribe el NIT en el formulario de "Nuevo cliente"
    ↓ (onBlur del campo identificación)
GET /third-parties/lookup?identification_number=900123456
    ↓
¿Hay coincidencia en Proveedor o Empleado?
    ↓ sí
Banner: "Ya existe como Proveedor: ACME SAS  [Copiar datos →]"
    ↓ clic
Se rellenan nombre, dirección, email, teléfono, régimen tributario...
en el formulario — el usuario revisa y confirma como cualquier alta normal
    ↓
POST /customers  (crea un registro Cliente totalmente independiente, sin FK)
```

Mismo patrón al revés en el formulario de Proveedor (busca en Cliente/Empleado) y a futuro en el de Empleado.

**3. Detalle fino — los campos no calzan 1 a 1:** Cliente/Proveedor comparten casi todos los campos, pero Empleado tiene `FirstName`+`LastName` en vez de `Name`, y no tiene régimen tributario ni responsabilidades fiscales. Hace falta una función de mapeo explícita por par de orígenes (`Proveedor→Cliente` casi directo; `Empleado→Cliente` necesita unir nombre y dejar los campos fiscales en blanco para que el usuario los complete) — no asumir que todos los campos coinciden.

---

#### Reportería cruzada (CQRS)

| Query | Estado |
|---|---|
| Balance general | ✅ `GET /api/v1/accounting/reports/bs` |
| Estado de resultados | ✅ `GET /api/v1/accounting/reports/pl` |
| Cartera por vencer | ✅ `GET /api/v1/receivables` |
| Valoración de inventario | 🔲 Pendiente — necesita costo unitario en el modelo |
| Resumen de ventas por período | 🔲 Pendiente |
| Reporte de nómina consolidado | 🔲 Pendiente |

#### Frontend — integración completada (commit b634d18) y pendientes

**Completado:**

| Archivo | Cambio |
|---|---|
| `lib/types.ts` | Tipo `Company` (espeja `safeCompany`), `Issuer = Company` como alias, `AuthResult` coincide con respuesta ERP |
| `context/AuthContext.tsx` | Migración completa de modelo issuer/apidian a company/ERP; `verifySession` usa `GET /auth/me`; `createIssuer` hace `POST /companies` → `POST /auth/select-company` |
| `lib/documents.ts` | Prefijo `/electronic/` en todos los paths de documentos |
| `lib/numberingRanges.ts` | Prefijo `/electronic/` en rangos de numeración |
| `lib/suppliers.ts` | `/suppliers` → `/suppliers` |
| `lib/catalogs.ts` | `dian-document-types` → `document-types` |

**Pendiente frontend — segunda fase:**

| Feature | Descripción |
|---|---|
| Panel de usuarios | Página para listar, invitar y desactivar usuarios de la empresa (requiere los endpoints `GET/PUT/DELETE /users` en el ERP) |
| Selector de empresa | Cuando un usuario tiene >1 empresa, `GET /auth/companies` debe devolver la lista real para que el frontend muestre un selector |
| Módulos nuevos | Ventas, cotizaciones, pagos, compras, inventario, contabilidad, nómina, RRHH — no tienen páginas frontend aún |
| Actualización de borrador (PUT) | Los endpoints `PUT /electronic/invoices/{id}`, `/credit-notes/{id}`, etc. aún no existen en el ERP |
| XML / Clone / Send-email | `GET /documents/{id}/xml`, `POST /documents/{id}/clone`, `POST /documents/{id}/send-email` — pendientes en el ERP |
| Panel admin SaaS | Estadísticas, gestión de suscripciones, auditoría — requiere endpoints y páginas nuevas |

---

## Filosofía

Si fuera a diseñar un ERP moderno en Go pensando en que dure 15 o 20 años, **no haría un monolito tradicional**, pero tampoco empezaría con microservicios. Haría un **Monolito Modular (Modular Monolith)** usando **Arquitectura Hexagonal (Ports & Adapters)** y **DDD (Domain Driven Design)**.

¿Por qué?

Porque un ERP tiene cientos de relaciones entre módulos.

- Un empleado pertenece a una compañía.
- La nómina genera asientos contables.
- La factura electrónica afecta cartera.
- Inventario afecta contabilidad.
- Compras afectan proveedores.
- Ventas afectan clientes.

Separar todo en APIs desde el principio termina siendo un infierno de sincronización.

Yo haría un solo backend en Go y módulos completamente desacoplados.

---

## Un solo módulo Go

**Una sola `go.mod` en la raíz del proyecto.** Nada de go.work, nada de módulos separados por dominio. El monolito modular es un único binario y su código fuente vive en `internal/`. La separación es conceptual (packages, carpetas), no física (módulos Go independientes).

Las librerías que sí son reutilizables por terceros (como `cofacture` para la generación de documentos DIAN) viven en su propio repositorio con su propio `go.mod`. Todo lo demás: interno.

---

## Arquitectura General

```text
                 Frontend
          (Vue / React / Flutter)

                    │

             HTTP / REST

                    │

             API Gateway net/http

                    │
────────────────────────────────────────
                Application
────────────────────────────────────────

      Company Module
      Customer Module
      Supplier Module
      Product Module
      Inventory Module
      Purchase Module
      Sales Module
      Accounting Module
      Payroll Module
      HR Module
      Electronic Documents
      Catalog Module
      Security Module
      Shared Utilities

────────────────────────────────────────
                 Domain
────────────────────────────────────────

Entities
Aggregates
Value Objects
Domain Events
Repositories (Ports)

────────────────────────────────────────
            Infrastructure
────────────────────────────────────────

PostgreSQL
SMTP / Resend / SES
S3
DIAN (vía cofacture)
PDF
XML / SOAP
```

> **Nota v1:** Redis y RabbitMQ **no son parte del arranque**. Redis entra cuando haya un caso real de caché de alto costo o sesiones distribuidas. RabbitMQ/NATS entra cuando haya un caso real de comunicación asíncrona con sistemas externos. Hasta entonces, los eventos de dominio son llamadas Go en memoria dentro del mismo proceso — eso es suficiente y mucho más simple de depurar.

---

## Un solo Go module, organización por dominio

No se organiza el proyecto por capas (`handlers`, `models`, `repositories`, `services`), sino por dominio.

```text
erp/
    cmd/
        server/
            main.go         ← único punto de arranque y wire-up de todo el sistema
    internal/
        company/
        customer/
        supplier/
        product/
        inventory/
        purchase/
        sales/
        accounting/
        payroll/
        hr/
        electronic/
        security/
        catalog/
        shared/             ← utilidades transversales: dinero, fechas, paginación, tenant, eventos
    go.mod
```

Cada módulo es completamente independiente y se comunica con los demás solo a través de eventos de dominio o interfaces (ports) explícitas.

---

## Multi-tenancy como ciudadano de primera clase

Cada registro en la base de datos pertenece a una compañía (`company_id`). Esto no es una responsabilidad de cada módulo individual — es una preocupación transversal que vive en `shared/tenant/`.

```text
shared/
    tenant/
        context.go      ← GetCompanyID(ctx) uuid.UUID
        middleware.go   ← extrae el company_id del JWT y lo inyecta al ctx de cada request
```

El middleware corre antes que cualquier handler. Cada repositorio recibe el `ctx` y extrae el `company_id` de ahí, nunca de un parámetro suelto que alguien pueda olvidar. Si el `company_id` no está en el contexto, la request no llega al dominio.

```go
// shared/tenant/context.go
type contextKey struct{}

func WithCompanyID(ctx context.Context, id uuid.UUID) context.Context {
    return context.WithValue(ctx, contextKey{}, id)
}

func GetCompanyID(ctx context.Context) uuid.UUID {
    id, _ := ctx.Value(contextKey{}).(uuid.UUID)
    return id
}
```

---

## Estructura de un módulo

```text
customer/
    domain/
        customer.go         ← entidad principal
        address.go
        contact.go
        repository.go       ← puerto (interface): Save, Update, Delete, GetByID, List
        errors.go
        events.go           ← CustomerCreated, CustomerUpdated
        notifier.go         ← puerto para notificaciones (email de bienvenida, etc.)
    application/
        create_customer.go  ← caso de uso: validar, guardar, emitir evento
        update_customer.go
        delete_customer.go
        get_customer.go
        search_customer.go
    infrastructure/
        persistence/
            postgres/
                repository.go       ← implementación concreta del puerto
                migrations/
                    000001_create_customers.up.sql
                    000001_create_customers.down.sql
    interfaces/
        http/
            handler.go      ← HTTP → DTO → UseCase → Respuesta. Solo transforma, no decide.
```

### Domain

No conoce HTTP, PostgreSQL, Fiber, ni JSON. Solo conoce el lenguaje del negocio.

### Application (casos de uso)

Flujo típico de un caso de uso:

```text
CreateCustomer
    ↓
Validar campos (dominio)
    ↓
Verificar que la empresa existe (via CompanyPort)
    ↓
Guardar (CustomerRepository)
    ↓
Emitir CustomerCreated (EventBus)
```

### Interfaces

El handler HTTP solo transforma — nunca decide:

```text
HTTP Request
    ↓
Parsear y validar DTO
    ↓
Llamar al caso de uso
    ↓
Serializar respuesta HTTP
```

### Infrastructure

Implementaciones concretas de los puertos. El dominio nunca importa un package de `infrastructure/`.

---

## Migraciones con el módulo que les corresponde

Las migraciones viven dentro del módulo que las necesita, no en una carpeta global:

```text
internal/
    customer/
        infrastructure/
            persistence/
                postgres/
                    migrations/
                        000001_create_customers.up.sql
    sales/
        infrastructure/
            persistence/
                postgres/
                    migrations/
                        000001_create_sales.up.sql
```

Cada módulo usa su propia tabla de tracking:

```
x-migrations-table=customer_schema_migrations
x-migrations-table=sales_schema_migrations
```

Ventaja: borrar el módulo `customer/` también borra su historia de migraciones. Sin artefactos huérfanos.

---

## Eventos de dominio (bus en memoria)

Para la comunicación entre módulos dentro del monolito, los eventos de dominio son llamadas Go en el mismo proceso. Simple, trazable, sin dependencias externas.

```text
shared/
    events/
        bus.go          ← Register(eventType, handler) / Publish(event)
        events.go       ← InvoiceConfirmed, PayrollGenerated, StockMoved, etc.
```

Ejemplo de flujo:

```text
Sales confirma una factura
    ↓
Publica InvoiceConfirmed{...}
    ↓
Accounting.OnInvoiceConfirmed → crea asiento contable
Inventory.OnInvoiceConfirmed  → descuenta stock
Electronic.OnInvoiceConfirmed → genera XML y envía a la DIAN
```

Ninguno de los suscriptores conoce a Sales. Sales no conoce a ninguno de ellos. Solo conoce el evento.

Cuando en el futuro se necesite garantía de entrega o comunicación con sistemas externos, el bus en memoria se reemplaza por uno sobre RabbitMQ/NATS/Kafka — sin tocar el dominio.

---

## Repositorios

```go
// customer/domain/repository.go
type Repository interface {
    Save(ctx context.Context, c Customer) (*Customer, error)
    Update(ctx context.Context, c Customer) (*Customer, error)
    Delete(ctx context.Context, id uuid.UUID) error
    GetByID(ctx context.Context, id uuid.UUID) (*Customer, error)
    List(ctx context.Context, filter Filter) ([]*Customer, error)
}
```

Implementación en infrastructure:

```go
// customer/infrastructure/persistence/postgres/repository.go
type PostgresRepository struct {
    pool *pgxpool.Pool
}

func (r *PostgresRepository) Save(ctx context.Context, c domain.Customer) (*domain.Customer, error) {
    companyID := tenant.GetCompanyID(ctx) // siempre del contexto, nunca de un parámetro suelto
    // ...
}
```

---

## Wire-up: cmd/server/main.go

Todo el cableado de dependencias vive en `main.go`. Sin framework de DI — Go no lo necesita. Un `main.go` largo y explícito es una virtud: se puede leer de arriba abajo y entender exactamente qué está conectado a qué.

```go
// cmd/server/main.go
func main() {
    pool := mustOpenDB(cfg.DatabaseURL)

    // Migraciones
    customer.Migrate(cfg.DatabaseURL)
    sales.Migrate(cfg.DatabaseURL)
    // ...

    // Repos
    customerRepo := customerPostgres.NewRepository(pool)
    salesRepo    := salesPostgres.NewRepository(pool)

    // Bus de eventos
    bus := events.NewBus()

    // Casos de uso
    createCustomer := customerApp.NewCreateCustomer(customerRepo, bus)
    // ...

    // Handlers HTTP
    mux := http.NewServeMux()
    customerHTTP.Register(mux, createCustomer, ...)
    salesHTTP.Register(mux, ...)

    http.ListenAndServe(cfg.Addr, mux)
}
```

---

## Procesos de negocio

Ejemplo del ciclo completo de ventas:

```text
Cotización
    ↓
Pedido
    ↓
Despacho
    ↓
Factura
    ↓ (evento InvoiceConfirmed)
DIAN (Electronic)
    ↓ (evento InvoiceConfirmed)
Contabilidad (asiento)
    ↓ (evento InvoiceConfirmed)
Inventario (descuento de stock)
    ↓ (evento InvoiceConfirmed)
Cartera (cuenta por cobrar)
```

---

## Organización de módulos

### Company

```text
company/
    company.go          ← entidad raíz: nombre, NIT, configuración fiscal
    branch.go           ← sucursales
    warehouse.go        ← bodegas (usa Inventory)
    configuration.go    ← parámetros ERP por empresa
```

### Customer

```text
customer/
    customer.go
    contact.go
    address.go
    tax.go              ← datos tributarios del cliente
    credit.go           ← cupo de crédito, condiciones de pago
```

### Supplier

```text
supplier/
    supplier.go
    payment_terms.go
    contact.go
```

### Product

```text
product/
    product.go
    category.go
    unit.go
    barcode.go
    price.go
    tax.go
```

### Inventory

```text
inventory/
    warehouse.go
    movement.go         ← entrada, salida, traslado
    stock.go            ← stock actual por producto/bodega
    lot.go              ← lotes (vencimientos, trazabilidad)
    serial.go           ← seriales
```

### Sales

```text
sales/
    quotation/
    order/
    shipment/
    invoice/
    payment/
    receivable/         ← cartera
```

### Accounting

Es el corazón del ERP. Todo lo que genera dinero o lo mueve termina aquí.

```text
accounting/
    chart/              ← plan de cuentas
    journal/            ← libro diario
    ledger/             ← libro mayor
    fiscal/             ← períodos fiscales, cierre
    period/
    posting/            ← motor de posteo: PostDocument(event) → asiento
```

El motor de posteo recibe eventos de dominio y genera asientos contables:

- Factura de Venta → Débito Cartera / Crédito Ingresos / Crédito IVA por pagar
- Factura de Compra → Débito Inventario o Gasto / Crédito Cuentas por Pagar
- Nómina → Débito Gasto de Personal / Crédito Provisiones / Crédito Bancos
- Pago → Débito Bancos / Crédito Cartera

### Payroll

```text
payroll/
    employee/
    contract/
    payslip/            ← liquidación mensual
    deduction/
    earning/
    social_security/    ← salud, pensión, ARL, caja, SENA, ICBF
    vacation/
    severance/
    liquidation/        ← liquidación definitiva
```

### HR

```text
hr/
    employee/           ← ficha del empleado (datos personales, historia)
    attendance/         ← control de asistencia
    absence/
    leave/              ← vacaciones solicitadas vs. ejecutadas
    recruitment/
```

### Electronic Documents

Motor documental que orquesta el ciclo DIAN. Delega la construcción del XML a `cofacture` (librería externa).

```text
electronic/
    domain/
        document.go     ← Document: borrador → confirmado → enviado → aceptado/rechazado
        signer.go       ← puerto: Sign(xml) (implementado por cofacture)
        sender.go       ← puerto: Send(zip) a la DIAN (implementado por cofacture)
        validator.go
```

Reutilizado por:

- Factura Electrónica (tipo 01)
- Documento Soporte (tipo 05)
- Nota Crédito (tipo 91)
- Nota Débito (tipo 92)
- Nota de Ajuste (tipo 95)
- Nómina Electrónica (tipo 103/102)

### Catalog

```text
catalog/
    countries/
    cities/
    currencies/
    taxes/
    units/
    banks/
    eps/
    arl/
    pension/
    occupations/
    document_types/
```

### Notification (shared — motor multicanal)

El correo es solo un canal. El módulo `notification/` es el motor de comunicaciones del ERP: cualquier módulo emite un evento y `notification/` decide qué canal usar y qué implementación concreta despacharlo.

```text
shared/
    notification/
        domain/
            notifier.go         ← puerto: Send(ctx, Message) error
            message.go          ← Message{To, Subject, Body, Channel, TemplateID, Data map}
            channel.go          ← Channel: "email" | "sms" | "whatsapp" | "push" | "slack" | "teams"
        application/
            send.go             ← SendUseCase: resuelve canal → proveedor → despacha
        infrastructure/
            email/
                smtp/
                    sender.go
                mox/
                    sender.go   ← preferido para correo transaccional propio
                resend/
                    sender.go
                ses/
                    sender.go
            sms/
                twilio/
                    sender.go
            whatsapp/
                meta/
                    sender.go
            push/
                fcm/
                    sender.go
        templates/
            email/
                welcome.html
                invoice_issued.html
                payroll_slip.html
                password_reset.html
                purchase_order.html
                absence_approved.html
            sms/
                otp.txt
                invoice_notification.txt
            whatsapp/
                invoice_notification.txt
```

**Principio clave:** ningún módulo de negocio importa `smtp`, `resend`, ni nada de infraestructura de correo. Solo conocen el puerto `Notifier` y emiten eventos como `InvoiceIssued` o `PayrollProcessed`. El módulo `notification/` escucha esos eventos y decide canal + proveedor + template. Cambiar de Mox a Resend no toca una sola línea de negocio.

### Reports (shared — motor de documentos)

El PDF es solo uno de los formatos de salida. `reports/` es el motor de documentos compartido del ERP: recibe datos + template y produce el formato que se pida.

```text
shared/
    reports/
        domain/
            renderer.go         ← puerto: Render(tpl string, data any) ([]byte, error)
            exporter.go         ← puerto: Export(content []byte, format Format) ([]byte, error)
            format.go           ← Format: "pdf" | "excel" | "csv" | "html" | "word"
        application/
            generate.go         ← GenerateUseCase: cargar template → render → exportar formato
        infrastructure/
            pdf/
                generator.go    ← chromedp/wkhtmltopdf sobre HTML renderizado
                fonts/
                images/
            excel/
                generator.go    ← excelize
            csv/
                generator.go
            html/
                renderer.go
        templates/
            accounting/
                balance_sheet.html
                income_statement.html
                journal_entry.html
                ledger.html
            payroll/
                payslip.html
                liquidation.html
                labor_certificate.html
                income_certificate.html
            sales/
                invoice.html
                quotation.html
                remission.html
            purchase/
                purchase_order.html
                reception.html
            electronic/
                invoice_graphic.html  ← representación gráfica FE (exigida por DIAN)
                support_doc.html
            inventory/
                kardex.html
                physical_count.html
                label.html
            hr/
                contract.html
                absence_certificate.html
```

**Quién lo usa:** prácticamente todos los módulos — ventas (facturas, cotizaciones), contabilidad (comprobantes, balance), nómina (desprendibles, certificados), RRHH (contratos, cartas), documentos electrónicos (representación gráfica exigida por DIAN), inventario (kardex, etiquetas).

**Separación de responsabilidades dentro de `reports/`:**
- `templates/` — solo administra las plantillas HTML/Markdown
- `infrastructure/pdf/` — renderiza HTML → PDF
- `infrastructure/excel/` — exporta datos estructurados → .xlsx
- `application/generate.go` — orquesta: carga template + data → render HTML → exporta al formato pedido

Un Balance General se genera una sola vez (render HTML) y se exporta en el formato que pida el cliente — sin duplicar lógica.

### Security

```text
security/
    user/
    role/
    permission/
    session/            ← JWT + refreshtoken
    audit/              ← log de acciones sensibles
```

---

## Reportería: capa de lectura (CQRS ligero)

Los reportes no son casos de uso del dominio — son consultas que cruzan múltiples schemas y necesitan joins específicos. Intentar implementarlos con los repositorios de dominio llena esos repositorios de métodos que no pertenecen ahí.

La solución: una capa de lectura separada en `shared/queries/`.

```text
shared/
    queries/
        sales_summary.go        ← SELECT directo multi-schema, sin pasar por repos de dominio
        payroll_report.go
        balance_sheet.go
        income_statement.go
        inventory_valuation.go
```

Las queries leen directamente de la base de datos con SQL optimizado para el caso específico. El dominio escribe a través de sus repositorios. Las queries solo leen. Eso es CQRS en su forma más simple y práctica.

---

## Shared: solo utilidades transversales

```text
shared/
    money/          ← tipo Money (cents int64, currency string) con operaciones seguras
    date/           ← helpers de fecha (colombiana, fiscal, etc.)
    pagination/     ← Cursor, Offset, Page
    uuid/           ← helpers
    validation/     ← validadores reutilizables
    tenant/         ← context.go, middleware.go (ver sección Multi-tenancy)
    events/         ← bus.go, events.go
    queries/        ← capa de lectura cross-module
    email/          ← shared porque TODOS los módulos lo usan
        domain/
            notifier.go     ← puerto (interface)
        infrastructure/
            smtp/
            resend/
            ses/
        templates/
```

**Regla de oro de `shared/`:** solo entra lo que es reutilizado por 3 o más módulos independientes y no contiene lógica de negocio propia. Las entidades de dominio nunca van aquí.

---

## Comunicación entre módulos

Los módulos se comunican exclusivamente a través de eventos de dominio. Ningún módulo importa directamente el package de dominio de otro.

```text
InvoiceConfirmed (publicado por Sales)
    ↓
Inventory.OnInvoiceConfirmed → DecreaseStock
    ↓
Accounting.OnInvoiceConfirmed → CreateJournalEntry
    ↓
Electronic.OnInvoiceConfirmed → GenerateXML → SendToDIAN
```

Eventos principales:

- `OrderConfirmed`
- `InvoiceConfirmed`
- `InvoiceCancelled`
- `PaymentReceived`
- `EmployeeCreated`
- `PayrollGenerated`
- `StockReserved`
- `StockMoved`
- `PurchaseReceived`

---

## Base de datos

Una sola base PostgreSQL. Un schema por módulo de dominio.

```text
company.*
customer.*
supplier.*
sales.*
inventory.*
payroll.*
hr.*
accounting.*
catalog.*
security.*
electronic.*
```

Cada módulo es responsable de sus propias migraciones y de no hacer queries directas a schemas de otros módulos. Si necesita datos de otro módulo, los obtiene a través de un evento o de un puerto explícito.

---

## Pila tecnológica

| Necesidad | Decisión | Alternativas / Cuándo cambiar |
|---|---|---|
| Base de datos | PostgreSQL (schemas por módulo) | No cambiar |
| HTTP | `net/http` estándar | chi/gorilla si se necesita routing avanzado |
| Eventos | Bus en memoria (`shared/events/`) | RabbitMQ/NATS cuando haya garantía de entrega o sistemas externos |
| Notificaciones | `shared/notification/` — Mox por defecto | Intercambiable: SMTP, Resend, SES, Twilio, Meta — sin tocar negocio |
| Documentos/reportes | `shared/reports/` — HTML → PDF vía chromedp | Excel: excelize; CSV: stdlib |
| Caché | Sin caché | Cuando haya consultas medibles y costosas |
| Cola | Sin cola | Cuando haya integración async con sistemas externos |
| ORM | Sin ORM — pgx directo | No cambiar |
| DI | Manual en main.go | No cambiar |
| gRPC | No por ahora | Si se expone API para terceros o se extrae algún módulo |

---

## Recomendación final

- Un único backend en Go, un único módulo Go (`go.mod`).
- Monolito Modular — nunca microservicios prematuros.
- Arquitectura Hexagonal (Ports & Adapters).
- DDD: organizado por dominio, no por capa técnica.
- CQRS ligero: repos de dominio para escritura, queries directas para lectura/reportería.
- Eventos de dominio en memoria para comunicación inter-módulo.
- PostgreSQL con schemas — uno por módulo.
- Multi-tenancy por contexto Go, no por parámetro.
- Motor contable centralizado que recibe eventos del resto de módulos.
- `cofacture` como librería externa (propio repo Go) — el módulo `electronic/` la consume.
- `cmd/server/main.go` como único punto de wire-up, explícito y legible.
