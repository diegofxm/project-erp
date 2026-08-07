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
| `thirdparty/` | Terceros unificados (Clientes + Proveedores, roles independientes `IsCustomer`/`IsSupplier` sobre la misma fila) — ver "Terceros — unificación de Cliente y Proveedor" más abajo |
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
| `thirdparty/` | tercero creado, rol agregado/quitado, actualizado |
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
- **RBAC — implementado en versión jerárquica simple, no granular por módulo.** El rol (`owner`/`admin`/`member`) viaja en el JWT (`security.user_companies` → login/select-company lo resuelven) y `shared/tenant.CanManage(ctx)` protege server-side las operaciones administrativas: invitar usuarios, y perfil/credenciales/logo/configuración de empresa. `member` opera el resto del ERP sin restricción — pero **no hay granularidad por módulo** (el `counter`-solo-ve-contabilidad / `user`-solo-ve-ventas del diseño de abajo no se construyó; eso sigue siendo aspiracional).
  - ⚠️ **Pendiente real:** el frontend no oculta ni deshabilita los botones de gestión para un usuario con rol `member` — el backend ya los rechaza con 403, pero el botón sigue visible y clicable, y el usuario solo se entera del rechazo al hacer clic. Falta leer el rol de la sesión en el frontend (ya viaja en la respuesta de login/`/auth/me`) y condicionar la UI de esos botones.

---

**Diseño propuesto de roles** (aspiracional — granularidad por módulo, no implementada; lo que sí existe hoy es el modelo jerárquico simple de arriba):

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
  └─ módulos: sales/, purchase/, inventory/, thirdparty/, products/
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

Contexto: en `v2/db-architecture` se completó de punta a punta pagos→contabilidad, retenciones en compras + certificados, conciliación bancaria, activos fijos + depreciación, presupuestos, y declaraciones de IVA/Renta/ICA (backend + frontend, commit `58c2cf1`). De esa fase quedaron pendientes identificados; `voucher_types` (ya valida contra el catálogo al postear) y las tablas `reconciliation_marks`/`exchange_rates` (ya tienen dominio/caso de uso/frontend — TRM real + conciliación de cuentas) se resolvieron después y se sacaron de esta lista. Lo que sigue sin resolver:

| Pendiente | Descripción | Prioridad |
|---|---|---|
| Import masivo de tarifas ICA | Hoy la carga de tarifas (`accounting.ica_tariffs`) es manual, una fila a la vez, por formulario. Cada contador conoce la tarifa de su propio municipio (+1100 en Colombia) — se necesita: botón "Descargar plantilla" (CSV con columnas `municipality_code,ciiu_code,fiscal_year,rate_bp,surcharge_bp` + fila de ejemplo), subida del archivo lleno, y un endpoint que valide cada fila (contra el catálogo CIIU real, ver siguiente ítem) y haga upsert por lote | Alta |
| Sincronizar CIIU de ICA con el catálogo ya existente | El sistema ya tiene el catálogo CIIU oficial completo: tabla `catalog.ciiu_codes` (508 códigos), servido en `GET /api/v1/catalogs/ciiu-codes`, ya usado por el frontend (`lib/ciiu.ts`, `TaxStep.tsx`) para el perfil de empresa. Los formularios de tarifa/declaración ICA (`AccountingTaxDeclarationsPage.tsx`) dejaron el campo CIIU como texto libre en vez de reusar ese mismo catálogo — no requiere cambio de esquema (accounting no cruza FKs con catalog, por diseño), solo cambiar el `<Input>` por el `Combobox` de `listCiiuOptions()` que ya existe | Alta |
| Import masivo de extracto bancario | La conciliación bancaria (`AccountingBankPage.tsx`) solo permite agregar movimientos del extracto uno por uno a mano. Falta poder subir el extracto real del banco (CSV/Excel/OFX) y parsearlo en lote | Media |
| Baja de activos fijos (disposal) | Se construyó alta + depreciación mensual por línea recta, pero no el flujo de "dar de baja" un activo. Los campos `gain_account`/`loss_account`/`disposed_at`/`disposal_amount`/`disposal_journal_id` ya existen en el esquema (agregados al unificar migraciones) pero ningún caso de uso los postea todavía | Media |
| Nombre de cuenta `143505` | Se agregó `143505 Mercancías no fabricadas por la empresa` (repite el nombre del padre `1435`) porque no existía subcuenta de mercancías en el PUC real extraído de puc.com.co — hay 2 precedentes de ese patrón en el catálogo (`540505`, `590505`) pero no es lo común. Pendiente decidir si se le pone un nombre más específico | Baja |
| Verificación visual en navegador | Todo el frontend de esta fase (bancos, activos fijos, presupuestos, declaraciones, certificados, retenciones, TRM, conciliación de cuentas) se verificó por `npm run build` + pruebas curl contra el backend real, nunca abriendo un navegador de verdad | Media |

---

#### Numeración consecutiva de documentos — pendientes (sesión 2026-08-04)

Contexto: se implementó numeración consecutiva real (`sales.number_counters`, `purchase.number_counters`, patrón `PREFIJO-AÑO-00001`) para ventas (`VTA-`), cotizaciones (`COT-`) y órdenes de compra (`OC-`), más la corrección de los asientos automáticos de `accounting` (`on_sale_confirmed`, `on_purchase_received`, `on_sale_payment_recorded`, `on_purchase_payment_recorded`) que antes posteaban sin `VoucherNumber` y con el UUID crudo en la descripción — commit `6d9e47e`. De ahí salieron dos frentes pendientes:

**1) ✅ Resuelto (parcial) — folio en documentos que solo se identificaban por UUID:**

| Documento | Folio | Prioridad |
|---|---|---|
| `accounting.WithholdingCertificate` | `CERT-AÑO-00001` — un contador por empresa/año (`accounting.certificate_counters`), asignado en `IssueWithholdingCertificatesUseCase.Execute` antes de cada `Create`. De paso se corrigió un bug preexistente nunca antes ejercitado en vivo: el código guardaba `Status: "issued"` en minúscula pero el `CHECK` de la tabla exige `'DRAFT'/'ISSUED'/'CORRECTED'` — toda emisión de certificados fallaba con violación de constraint | ✅ Hecho |
| `inventory.Movement` | Un contador por empresa/tipo/año (`inventory.number_counters`, `doc_type` = entry/exit/transfer/adjust): `ENT-`/`SAL-`/`TRA-`/`AJU-`. Asignado dentro de `SaveMovement`/`Transfer` (postgres), no en la capa de aplicación — hay 3 puntos de entrada (ajuste manual, traslado, posteos automáticos de venta/compra confirmada) y centralizarlo ahí evita que alguno se quede sin folio | ✅ Hecho |
| `payroll.Payslip` (desprendible de nómina) | `NOM-AÑO-00001` — un contador por empresa/año (`payroll.number_counters`), asignado dentro de `PayslipRepository.Create` en la misma transacción. Sin obligación legal DIAN, es folio interno | ✅ Hecho |
| `payroll.Contract`, `hr.Absence` | Sin obligación legal ni PDF que lo requiera todavía | Pendiente (baja) |

No hizo falta tocar `electronic.Document` (factura, NC, ND, documento soporte, nota de ajuste) — ya tiene `Number`/`Prefix` resueltos vía `NumberingRange` (rango de resolución DIAN), sin gap ahí.

**Frontend — ✅ Resuelto:** `AccountingCertificatesPage.tsx` e `InventoryMovementsPage.tsx` ya muestran la columna "Folio" (`number`) que el backend expone — `payroll.Payslip` no lo necesita porque el módulo de nómina no tiene frontend aún (deferred). De paso se agregó soporte en `Breadcrumbs.tsx` (`muted?: boolean`) para que el último ítem de la miga de pan muestre solo el número del documento (ej. `VTA-2026-00001`) en vez de repetir el título completo que ya muestra el H1 (ej. "Venta VTA-2026-00001") — aplicado en `SaleEditorPage`, `QuoteEditorPage`, `PurchaseOrderEditorPage`, `InvoiceEditorPage`, `CreditNoteEditorPage`, `DebitNoteEditorPage`, `AdjustmentNoteEditorPage`, `SupportDocumentEditorPage` y `AccountingJournalEditorPage`.

**Pendiente — sembrar consecutivo inicial en los 3 folios nuevos:** el mecanismo `next_number` (ver punto 2 abajo) solo se construyó para `sales`/`purchase`/`accounting.voucher_counters`. `accounting.certificate_counters`, `inventory.number_counters` y `payroll.number_counters` no tienen ese endpoint — si una empresa migra con certificados/movimientos/nóminas ya numerados en otro sistema, no hay forma de fijar el punto de partida salvo SQL directo. Tampoco existe forma de personalizar el *prefijo* (`VTA-`, `CI-`, `ENT-`, etc.) en ningún documento excepto los electrónicos DIAN (`electronic.NumberingRange.Prefix`, que siempre lo tuvo) — prioridad baja hasta que alguien lo pida.

**2) ✅ Resuelto — fijar un consecutivo inicial distinto de 1 (migración de empresas con numeración previa):**

Antes, `sales.number_counters`, `purchase.number_counters` y `accounting.voucher_counters` siempre nacían en `last_seq = 1` la primera vez que se pedían, sin ningún endpoint para sembrar un valor inicial. Se agregó, replicando el mecanismo `next_number` que ya existía en `electronic.NumberingRange`:

- `POST /api/v1/sales/number-counters` — body `{doc_type: "sale"|"quote", year, next_number}`.
- `POST /api/v1/purchase/number-counters` — body `{year, next_number}`.
- `POST /api/v1/accounting/voucher-counters` — body `{code, year, next_number}` (valida el código contra `IsStandardVoucherType`/`IsRegisteredVoucherType`, igual que `PostJournalUseCase`).

Los tres requieren rol `owner`/`admin` (`tenant.CanManage`) y rechazan con 422 (`ErrNumberCounterBackwards`) si `next_number` es menor o igual al último ya asignado — evita duplicar números ya emitidos; el `UPDATE` usa una condición (`WHERE EXCLUDED.last_seq > ...`) en vez de comparar en Go, así queda atómico. Verificado en vivo: sembrar next_number=100/200/300/400 y luego crear venta/cotización/orden/asiento reales produjo `VTA-2026-00100`, `COT-2026-00200`, `OC-2026-00300`, `CI-2026-00400` — y los intentos de retroceder fueron rechazados.

---

#### Hallazgos de la sesión 2026-08-05 (flujo de ventas/cotizaciones probado en vivo)

**1) ✅ Resuelto — bodega por defecto no se creaba al registrar una empresa nueva.** `GetOrCreateDefault` (crea la bodega "Principal" si la empresa no tiene ninguna) solo se llamaba de forma perezosa al confirmar la primera venta o recibir la primera compra (`sales.ConfirmUseCase`, `inventory.OnPurchaseReceived`). Si el primer movimiento de un usuario nuevo era un "Ajuste manual" en Inventario, el selector de bodega salía vacío y el botón "Registrar" quedaba deshabilitado sin ninguna explicación en pantalla. Se corrigió `company.CreateUseCase.Execute` para llamar `warehouses.GetOrCreateDefault` justo después de crear la empresa — verificado en vivo: empresa nueva → `GET /companies/active/warehouses` ya trae "Principal" de inmediato.

**2) Pendiente — el Panel de Control (dashboard de Inicio) no tiene ningún dato de `sales`/`purchase`.** `DashboardPage.tsx` + el módulo `stats` (`internal/stats`) están construidos enteramente sobre `electronic.documents` (factura/NC/ND/documento soporte DIAN) — cero consultas a `sales.quotes`, `sales.sales` o `purchase.orders`. Confirmar una venta o aceptar una cotización no cambia ninguna card del Inicio porque esas cards nunca leyeron esas tablas. No es un bug — nunca se conectó. Falta decidir qué widgets agregar (ej. "Cotizaciones pendientes/aceptadas", "Ventas del mes") y si van al mismo `stats` module o a uno nuevo.

**3) Pendiente — no existe respuesta de cotización por email (aceptar/rechazar sin login).** El correo que manda `SendQuoteEmailUseCase` (plantilla `quote_issued.html`) es de una sola vía: adjunta el PDF y termina con "Mensaje automático — por favor no respondas a este correo". No hay link de "Aceptar"/"Rechazar" para que el cliente responda directamente — hoy `handleAcceptQuote`/`handleRejectQuote` solo los puede ejecutar el vendedor autenticado desde el ERP, asumiendo que el cliente avisó por otro canal (llamada, WhatsApp, etc.). Si se quiere self-service real, hace falta: token público de un solo uso por cotización, endpoint público (sin auth) en `sales/interfaces/http` o `public/`, y botones en la plantilla de correo.

---

#### TRM: servicio SOAP propio, disparador diario y búsqueda por fecha (sesión 2026-08-06)

Contexto: se reemplazó el espejo público `co.dolarapi.com` por un servicio propio (`https://co-trm.vercel.app`, ver `TRM_API_URL`) que consulta directamente el Web Service SOAP oficial de la Superintendencia Financiera — expone `/trm` y `/trm?date=YYYY-MM-DD`, así que ahora se puede pedir la TRM de cualquier fecha, no solo la de hoy.

- **`internal/accounting/infrastructure/trmapi`** (nuevo, reemplaza `infrastructure/dolarapi`, eliminado): implementa `domain.TRMFetcher`, que ahora recibe la fecha explícita a consultar (`FetchTRM(ctx, date)`) en vez de dejar que la fuente externa decida qué es "hoy" — evita ambigüedad de husos horarios. El `Source` guardado pasó de `"DOLARAPI"` a `"SUPERFINANCIERA"`.
- **Columna `description`** agregada a `accounting.exchange_rates` (migración editada in-place, sin datos reales que migrar) — antes el texto de la columna "Descripción" del frontend salía de un `switch` hardcodeado por `source`; ahora se guarda la descripción real que devuelve el servicio (`"TRM oficial publicada por la Superintendencia Financiera de Colombia"`) o la que escriba el usuario al capturar manualmente (default `"Editado manualmente"` si la deja vacía).
- **Disparador diario** (`accountingapp.RunTRMDailySync`, lanzado como goroutine en `main.go` si `TRM_API_URL` está configurada): sincroniza la TRM de hoy todos los días a la 1:00 a.m. hora Colombia (`America/Bogota`); si falla, reintenta una sola vez una hora después y si vuelve a fallar espera al día siguiente — sin insistir indefinidamente contra el servicio externo.
- **`ExchangeRateUseCase.GetOrFetch`** (nuevo) — herramienta para que el contador busque la TRM de cualquier fecha pasada: primero contra la base local (`GET /accounting/exchange-rates/lookup?date=...`), y solo si no está ahí consulta el servicio externo una única vez y la deja guardada (la TRM histórica no cambia, nunca se vuelve a pedir esa fecha).
- **Control de abuso del botón "Sincronizar"** (frontend, `AccountingExchangeRatesPage.tsx`): queda deshabilitado en cuanto ya exista un registro de hoy en la base de datos — sin importar si lo puso el disparador automático o el propio botón — y se reactiva solo, o si por alguna razón el disparador de la 1 a.m. no corrió. Decisión explícita: se mantuvo como respaldo manual en vez de quitarlo del todo, para no depender 100% de que el disparador nunca falle.
- Los únicos tres caminos que llegan a tocar el servicio SOAP: el disparador diario, el botón "Sincronizar" (autolimitado por lo anterior), y la búsqueda por fecha específica (autolimitada porque cachea para siempre). El listado del panel (`GET /accounting/exchange-rates`) siempre lee de la base de datos, nunca del servicio externo.

Verificado en vivo contra el servicio real: sincronizar hoy, buscar una fecha histórica (`2026-01-15`) trayéndola del servicio la primera vez y de la base la segunda, y registrar tasas manuales con y sin descripción personalizada.

**Correcciones de la misma sesión, después de probarlo en vivo:**

- **Paginación real en vez de ventana fija de 90 días.** La primera versión de la lista pedía `listExchangeRates(hace 90 días, hoy)` — un usuario con filas más viejas que 90 días (ej. TRM de marzo) las veía desaparecer de la tabla sin ningún aviso, ni forma de traerlas de vuelta salvo la búsqueda puntual. Se descartó además la idea de un filtro de rango editable (el buscador por fecha con el SOAP ya cubre el caso "quiero ver una fecha vieja puntual"; un filtro habría sido redundante). Se cambió `ExchangeRateRepository.List` para paginar de verdad: `List(ctx, limit, offset) ([]ExchangeRate, total int, error)`, con `COUNT(*) OVER()` en la misma consulta — `GET /accounting/exchange-rates?limit=7&offset=0`, con controles "Anteriores"/"Siguientes" en el frontend. Página por defecto: 7 filas.
- **Indicador "Hoy: `$valor`"** junto al título (mismo tamaño de fuente que el `<h1>`) — nuevo endpoint de solo lectura `GET /accounting/exchange-rates/today` (`ExchangeRateUseCase.GetToday`, calcula el día en `America/Bogota` y hace un `Get` puro contra la base, nunca toca el servicio externo) para que el valor de hoy se vea sin importar en qué página de la lista esté parado el usuario, y sin disparar ninguna sincronización solo por abrir la página. También reemplazó el cálculo de `hasTodayRate` que antes dependía de que la fila de hoy estuviera en la página visible (con paginación real eso ya no se puede garantizar).

Verificado en vivo con datos reales del usuario (7 filas ya generadas usando el panel): página 1 trae las 7, página 2 viene vacía con `total: 7`.

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

Aplicando esa regla, queda pendiente (no cambia la estructura del menú ya definida, solo dónde entra):

| Elemento | Estado | Dónde debería vivir |
|---|---|---|
| Empleados | Backend existe (`payroll.Employee`, `erp/internal/payroll/domain/employee.go`), sin página frontend | Pestaña dentro de **Nómina** — NO como catálogo aparte. `payroll.Employee` sigue siendo un struct independiente de `thirdparty.Party` (ver sección siguiente): no tiene campos tributarios DIAN, usa `FirstName`/`LastName` en vez de `Name`, y su tratamiento (retención salarial, aportes a seguridad social, cuenta contable 2505) no tiene nada que ver con el comercial de Cliente/Proveedor. Unificarlo es un paso deliberadamente aplazado — ver siguiente sección |
| Sucursales | No existe (ni backend ni frontend) — ya estaba anotado en "Sub-entidades pendientes por módulo" arriba (`company/`) | Dentro de **Configuración → Empresa**, no como catálogo — es sub-configuración de la empresa, no dato maestro compartido |

---

#### Terceros — unificación de Cliente y Proveedor en `thirdparty/` (implementado)

Contexto: surgió al comparar con Odoo (modelo `res.partner` unificado) y SIESA (catálogo de "Terceros" unificado, típico en ERPs colombianos por el reporte de Información Exógena DIAN, que agrega montos por NIT sin importar el rol). La primera propuesta fue una simple "asistencia de captura" (buscar el NIT en el otro catálogo y ofrecer copiar los datos) manteniendo las tablas separadas — al revisarlo con más calma, y aprovechando que aún no había datos reales en producción, se decidió ir más allá: **unificar `customer/` y `supplier/` en un solo módulo `thirdparty/`**, porque:

- Los dos structs eran prácticamente el mismo dato: 24 de 26 campos idénticos, solo `Customer.CreditLimit` y `Supplier.PaymentTermsDays` los diferenciaban.
- El diseño "sin FKs cruzadas" ya protegía esta unificación: `customer_id`/`supplier_id` en otros módulos son `UUID` sueltos, resueltos vía puerto local — fusionar las tablas de origen no rompe nada en los consumidores, solo cambia qué implementa el puerto.
- Los consumidores (`sales`, `purchase`, `electronic`, `public`) ya importaban `customerdomain.Repository`/`supplierdomain.Repository` completos y directos, sin el patrón de puerto local que sí sigue correctamente todo lo que toca `company` (`CompanyPort` + adaptador por consumidor) — la "pureza" de módulos independientes que se quería proteger ya no existía en la práctica.

**Empleado se dejó explícitamente fuera** de esta unificación: no es mellizo de Cliente/Proveedor como estos lo son entre sí (sin campos DIAN, nombre partido en `FirstName`/`LastName`), y no existe frontend de Nómina/RRHH todavía contra el cual validar cómo debería verse esa fila unificada. Cuando se construya ese frontend, ahí se decide si `payroll.employees` pasa a referenciar `thirdparty.parties` en vez de duplicar identificación/nombre/dirección.

**Diseño implementado:**

```
internal/thirdparty/
    domain/
        party.go        ← Party{...24 campos comunes..., IsCustomer, IsSupplier,
                            CreditLimit *float64, PaymentTermsDays int}
                            Role = "customer" | "supplier"
        repository.go    ← Repository{Save, GetByID, GetByIdentification (sin filtrar
                            por rol), List(role), Update, Delete}
    application/
        create.go, get.go, update.go, delete.go  ← reciben Role como parámetro;
                            un mismo juego de casos de uso sirve a ambos catálogos
    infrastructure/persistence/postgres/
        repository.go, migrations/000001_thirdparty.up.sql  ← schema thirdparty,
                            tabla parties, UNIQUE(company_id, identification_type_code,
                            identification_number) — una identificación, una fila
    interfaces/http/
        handlers.go      ← NewCustomerHandler/NewSupplierHandler fijan el Role;
                            rutas y forma del JSON idénticas a los módulos viejos
                            (/customers, /suppliers, wrapper "customers"/"suppliers")
                            → CERO cambios en el frontend
```

Puertos locales nuevos (mismo patrón que `CompanyPort`, uno por consumidor):

| Consumidor | Puerto | Vista (`domain.Customer`/`domain.Supplier`/`domain.Party`) |
|---|---|---|
| `sales` | `CustomerPort` | Reducida: nombre, identificación, dirección, contacto, `CreditLimit` |
| `purchase` | `SupplierPort` | Reducida: nombre, identificación, dirección, contacto (sin plazo de pago, no se usa) |
| `electronic` | `CustomerPort` + `SupplierPort` (un solo `Adapter` implementa ambos) | Casi completa — necesita los campos tributarios DIAN para armar `cofdom.Party` |

**El efecto colateral más importante — reemplaza la idea original de "avisar y copiar datos":** al dar de alta un Cliente cuya identificación ya existe como Proveedor (o viceversa), `CreateUseCase.Execute` ya no crea una fila duplicada ni necesita avisar nada — encuentra el tercero existente por `GetByIdentification` (que busca sin filtrar por rol) y le agrega el rol nuevo directamente, conservando los datos del rol que ya tenía. Verificado en vivo: crear Cliente NIT 900999999, luego crear Proveedor con el mismo NIT → misma fila, `is_customer=true` y `is_supplier=true`, `credit_limit` y `payment_terms_days` conviven sin pisarse. Borrar el rol Cliente deja la fila viva como Proveedor (`is_customer=false`, `credit_limit=null`) en vez de borrar el tercero completo.

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
| Panel admin SaaS | ✅ Resuelto — ver sección "Módulo SaaS" más abajo |

---

## Módulo SaaS: planes, suscripciones y facturación de plataforma

Contexto: para poner en producción la facturación electrónica hacía falta el módulo que la vende como servicio — hasta ahora no existía en el backend hexagonal (`erp/internal`), solo en el backend legado (`_legacy/apidian/internal/{plans,subscriptions}`) contra el que el frontend `/admin/*` (`AdminPage.tsx`) seguía apuntando sin que esas rutas existieran del lado nuevo. Se construyó desde cero con un modelo más rico que el legado: variante de certificado DIAN (propio vs. vendido por nosotros), cupo de documentos con cobro de excedente (no bloquea), IVA configurable, y módulos habilitables por plan pensando en Nómina/ERP como productos futuros sin tener que tocar el esquema de nuevo.

**Modelo de datos** (`erp/internal/saas`, esquema `saas`):
- `saas.modules` — catálogo fijo (`electronic_invoicing`, `erp_core`, `payroll_hr`), sembrado por seed — coincide con la agrupación que ya estaba en el diseño del sidebar ("solo Documentos Electrónicos / solo Nómina / ERP completo").
- `saas.plans` — `billing_cycle` (mensual/anual/sin ciclo) por plan (no uno solo para todo el catálogo), cupo de documentos + precio de excedente, `requires_certificate` + `certificate_price_cents` (el certificado se renueva siempre anual, independiente del ciclo del plan), `annual_increment_pct` (aplicado manualmente vía `POST /admin/plans/{id}/apply-increment`, no retroactivo a suscripciones vigentes), `is_internal` (plan usado por la empresa operadora — Cofacture —, excluido del catálogo público).
- `saas.plan_modules` — qué módulos desbloquea cada plan (M2M).
- `saas.subscriptions` — una activa por empresa (índice único parcial), con `contracted_price_cents` como foto del precio al contratar/renovar y `cert_expires_at` propio.
- `saas.payments`, `saas.settings` (IVA, fila única), `saas.prospects` (solicitudes de acceso con cédula/RUT, portado del legado).

**Catálogo semilla** (100% editable después desde `/admin/plans`, sin estudio de mercado — punto de partida): Gratis (10 docs/mes, $0), Emprendedor (100 docs/mes, $49.900+IVA), Ilimitado (sin límite, $499.900+IVA), Estrella (anual, documentos ilimitados + ERP completo, $1.990.000+IVA propio certificado / $2.390.000+IVA con certificado nuestro), Interno (los 3 módulos, $0, no aparece en catálogo público — el que usa Cofacture).

**Superadmin** — `security.users.is_superadmin` existía en dominio/BD pero no llegaba a ningún lado (JWT no lo llevaba, `safeUser()` no lo exponía, el guard `withSuperAdmin` del frontend siempre veía `undefined`). Se completó el plumbing: claim `isa` en el JWT (`security/infrastructure/jwt`), `tenant.WithSuperAdmin`/`IsSuperAdmin(ctx)` en `shared/tenant`, y `safeUser()` ahora incluye `is_superadmin`. Rutas `/api/v1/admin/*` usan `requireSuperAdmin` (local a `saas/interfaces/http`, inspirado en `_legacy/apidian/internal/api/handler_admin.go`) — no dependen de `company_id` activo, operan sobre todas las empresas.

**Endpoints**: `/api/v1/admin/{modules,plans,settings,companies/{id}/subscription,companies/{id}/payments,billing/summary,billing/renewals,users,prospects}` (superadmin) + `GET /api/v1/saas/my-plan` (empresa activa) + `POST /api/v1/public/prospects` (solicitud de acceso sin cuenta).

**Frontend**: `AdminPage.tsx` reescrito contra el contrato nuevo — la pestaña Planes ahora tiene el formulario completo de creación/edición que nunca existió (ciclo, precio, cupo, certificado, módulos), nueva pestaña Configuración (IVA). Página nueva "Mi plan" (`Configuración → Mi plan`) para el usuario normal: plan contratado, cupo usado/restante, módulos incluidos. `Sidebar.tsx` gana `moduleCode` opcional por ítem — oculta Contabilidad/Ventas/Compras/Inventario (`erp_core`) o Documentos (`electronic_invoicing`) si el plan de la empresa no los incluye (`useMyPlan()` falla abierto — sin dato todavía = no oculta nada, mismo criterio "sin suscripción = sin límite" que ya usaba el backend).

**Deliberadamente fuera de alcance**: no hay bloqueo duro a nivel de API cuando un módulo no está en el plan — el gating es solo de UI (ocultar/deshabilitar en el sidebar). El límite de documentos sí se aplica con dinero real (excedente facturado vía `saas/billing`), no bloqueando la emisión. `Renew` no extiende `cert_expires_at` automáticamente (queda con la fecha de cuando se vendió el certificado la primera vez) — pendiente si se necesita ese detalle más adelante. No hay pasarela de pago integrada — los pagos se registran manualmente desde `/admin/companies/{id}`.

### Alta de superadmin y aprovisionamiento real desde solicitudes de acceso

Contexto: `security.users.is_superadmin` nunca tuvo, ni antes ni después de la primera versión del módulo SaaS, una forma legítima de otorgarse — el único superadmin que existía en desarrollo se creó a mano por SQL directo. Además, `ProspectUseCase.Approve` solo cambiaba el estado a `approved` sin crear ninguna cuenta real, dejando el flujo de solicitudes de acceso incompleto. Se cerraron los dos huecos:

- **Primer superadmin por seed, no por API**: `erp/internal/security/infrastructure/persistence/postgres/seed/seed.go` crea (o promueve, si ya existe por correo) al superadmin inicial leyendo `SUPERADMIN_EMAIL`/`SUPERADMIN_PASSWORD` del entorno — nunca hardcodeado, y nunca pisa una contraseña que el usuario ya haya cambiado desde la app. Si esas variables no están definidas, no pasa nada (arrancar sin superadmin es válido).
- **Superadmins adicionales, promovidos por otro superadmin**: `PATCH /api/v1/admin/users/{id}/superadmin` (`{is_superadmin: bool}`), protegido por `requireSuperAdmin` — nunca por ningún flujo de alta de cliente. Frontend: botón "Hacer/Quitar superadmin" en `/admin/users`.
- **Aprobar un prospecto ahora aprovisiona la cuenta real**: `ProspectUseCase.Approve` (1) crea el usuario invitado sin contraseña (`security.InviteOwnerUseCase` — nuevo, distinto de `InviteUserUseCase` porque este NO vincula a una empresa existente ni acepta rol `owner`, que es exactamente lo que hace falta cuando la empresa todavía no existe), (2) crea su primera empresa con el NIT declarado usando ese usuario como `creatorID` de `company.CreateUseCase` (que ya vincula automáticamente como `owner` y crea la bodega por defecto), y (3) le envía un correo (`invite_owner.html`, nuevo) con el enlace a `/accept-invite?token=...` para poner contraseña — reutilizando el `POST /auth/accept-invite` que ya existía pero nunca tenía página propia en el frontend (`AcceptInvitePage.tsx`, nueva). El prospecto solo queda `approved` al final de las tres cosas — si algo falla a mitad de camino se queda `pending` y se puede reintentar sin duplicar nada, porque `InviteOwnerUseCase` es idempotente por correo (reutiliza al usuario ya creado si todavía no aceptó y no tiene ninguna empresa vinculada).

Verificado en vivo end-to-end: seed del superadmin desde variables de entorno + login, `POST /public/prospects` → `POST /admin/prospects/{id}/approve` → aparece en `GET /admin/users` → `POST /auth/accept-invite` con el token real de la fila → sesión válida → `GET /companies` confirma la empresa creada con el usuario como dueño.

**Pendiente, no resuelto en esta sesión**: `AcceptInviteUseCase` no auto-selecciona la empresa como sí hace `LoginUseCase` cuando el usuario tiene exactamente una — por eso el JWT que devuelve trae `company_id` nulo incluso cuando ya hay una empresa real vinculada, obligando a un paso extra de selección de empresa en el frontend. `InviteUserUseCase` (invitar compañeros a una empresa existente) sigue sin enviar el correo de invitación — el nuevo template/flujo solo se conectó para `InviteOwnerUseCase` (prospectos), no para ese caso.

### Pendiente — registro público desde una landing (`cofacture.co/planes`), alcance sin definir todavía

Contexto: lo que se construyó (`POST /public/prospects` + revisión en `/admin/prospects` + aprovisionamiento al aprobar) es la mitad "interna" del flujo de alta — recibir la solicitud y procesarla. Falta toda la mitad de cara al público para que un visitante real llegue desde una landing de planes y se registre solo. Diagnóstico completo (sesión donde se decidió dejarlo pendiente):

| Falta | Detalle |
|---|---|
| Página pública de registro | No existe ninguna pantalla que llame a `POST /public/prospects` — hoy solo se probó por curl. Sin esto no hay dónde subir cédula/RUT ni llenar el formulario. |
| Página pública de planes | El catálogo (`/admin/plans`) es privado para superadmin. No hay endpoint público que liste planes activos (excluyendo `is_internal`) para mostrar precios en una landing. |
| El prospecto no puede elegir plan | `domain.Prospect` no tiene campo de plan deseado. Aunque exista la landing, hoy no hay dónde guardar esa elección — y `ProspectUseCase.Approve` crea la empresa pero **no asigna ninguna suscripción**; el superadmin tendría que asignarla aparte en `/admin/company` después de cada aprobación. |
| Sin confirmación al prospecto | No se envía correo al recibir la solicitud ("la recibimos, te avisamos") ni al rechazarla — hoy el rechazo solo cambia un estado que ve el superadmin, el prospecto no se entera. |
| Sin cobro en el registro | Coherente con la decisión ya tomada de no integrar pasarela de pago, pero si el modelo final es "elige plan → paga → queda activo" es un tramo entero sin construir; si es "elige plan → se revisa → se factura aparte", es más simple pero de todos modos falta guardar la elección del plan. |

**Decisión explícita**: se deja así, sin desarrollar, hasta una próxima sesión donde se defina el alcance real (¿el pago va integrado al registro o queda para después de aprobar? ¿la landing es parte de este mismo frontend React o un sitio de marketing aparte que solo consume la API pública?) antes de construir nada.

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
