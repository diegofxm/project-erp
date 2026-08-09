# Resumen Ejecutivo — Auditoría Técnica Multi-Agente (2026-08-09)

Síntesis de los 9 informes de esta carpeta. Cada hallazgo cita el informe de origen entre paréntesis — para el detalle completo (archivo/línea exactos), abrir ese informe.

## Estado general del proyecto

| Componente | Estado estimado | Justificación |
|---|---|---|
| **`cofacture`** (motor DIAN) | **~65%** listo para producción | Funcionalmente maduro: FE/NC/ND/DS/NA con firma XAdES real, envío SOAP real, CUFE/CUDE/CUDS correctos, distinción habilitación/producción real (04). Pero tiene un riesgo ALTO no resuelto (reconciliación de timeout, ver abajo), certificado `.p12` sin cifrar, RADIAN sin implementar, y nómina electrónica huérfana (el builder existe pero `erp/internal/payroll` no lo usa). |
| **`erp`** (backend Go) | **~60%** listo para producción | Arquitectura hexagonal sólida y consistente (01), compila/vetea/testea limpio (03), módulos de negocio ricos (ventas, compras, contabilidad con reportes financieros reales — 09). Pero RBAC real casi no existe en el backend (solo se oculta en el frontend), auditoría con cobertura muy despareja (0% en `electronic` y `security`), y el bus de eventos traga errores de negocio silenciosamente (03, 09). |
| **`frontend`** | **~55%** listo para producción | Buen sistema de componentes reutilizado de forma consistente, responsivo, con cobertura amplia de los módulos que sí tienen backend (05). Pero cero tests, sesión JWT frágil (sin interceptor 401, sin timeout de red), y Nómina/RRHH sin ninguna vista (confirmado explícitamente en el propio código del sidebar). |

Ningún componente tiene bloqueantes que impidan seguir desarrollando — los tres son viables — pero los tres comparten el mismo patrón: **lo que se construyó está bien construido, lo que falta es la red de seguridad alrededor** (autorización real, trazabilidad, reconciliación de fallos, tests que corran solos, observabilidad).

## Los 5 riesgos más críticos que bloquean producción

1. **[CRÍTICO — seguridad] Credenciales de producción reales filtradas y todavía trackeadas en git.** `_legacy/apidian/.env` y `.env.old` siguen en el working tree y en el historial, con una `DATABASE_URL` real de alwaysdata.net, otra de un proyecto Neon real, `AUTH_JWT_SECRET`/`ISSUER_SECRETS_KEY` reales y credenciales SMTP reales. `.gitignore` no los cubre (la regla `apidian/.env*` no matchea `_legacy/apidian/...`). Cualquier clon del repo expone estas credenciales hoy mismo. (06-seguridad)

2. **[ALTO — legal/financiero, bloqueante] Riesgo de doble facturación o números de documento atascados ante timeout con la DIAN.** Si `SendBillSync` falla por timeout, el sistema marca `send_error` **sin guardar trackID y sin consultar `GetStatus`** para confirmar si la DIAN sí procesó el documento, y libera el consecutivo. Un reintento manual (`CloneDraft`) puede generar un segundo documento con CUFE distinto para la misma operación de negocio. No existe ningún mecanismo de contingencia/reintento automático tampoco. (04-cofacture-dian)

3. **[ALTO — autorización] El RBAC por rol casi no se aplica en el backend, solo en la UI.** `tenant.CanManage()` solo se usa una vez en `sales`, `purchase` y `accounting` (para fijar consecutivos) — confirmar/cancelar una venta, recibir una compra, etc. no tienen ningún chequeo de rol en el servidor. Un usuario `member` autenticado puede ejecutar esas acciones llamando la API directamente, sin pasar por el frontend que sí oculta los botones. `thirdparty`, `electronic`, `inventory`, `hr`, `payroll`, `product` y `catalog` no usan `CanManage` ni una vez. (09-roadmap-erp, consistente con 03-backend-go)

4. **[ALTO — trazabilidad legal] Auditoría con cobertura nula justo donde más se necesita.** El logger de auditoría está inyectado en 12 de 13 módulos, pero las llamadas reales son 0 en `electronic` (26 handlers — emisión/anulación de documentos DIAN) y 0 en `security` (10 handlers — login, cambio de contraseña, invitaciones). El sistema no deja rastro de quién emitió/anuló una factura ni de los eventos de sesión. (09-roadmap-erp)

5. **[ALTO — integridad operativa] Errores de negocio silenciados en el bus de eventos, y esa misma falsa sensación de seguridad se repite en los tests.** Si falla el asiento contable de una venta confirmada (o el registro de inventario), el error solo se hace `log.Printf` — nunca llega al usuario ni al handler HTTP; la venta queda "confirmada" sin contabilizar y sin auditoría, sin que nadie se entere. Relacionado: los `schema_test.go` de los 14 módulos requieren `//go:build integration` y **no corren nunca** con un `go test ./...` normal — la cobertura de esquema reportada como "100%" no se ejecuta en la práctica sin que alguien sepa pasar el flag explícito, y no hay CI que lo haga por nadie. (03-backend-go, 07-testing-calidad)

## Checklist priorizada para llegar a producción

| # | Pendiente | Esfuerzo | Informe |
|---|---|---|---|
| 1 | Rotar TODAS las credenciales filtradas (Postgres alwaysdata, Neon, JWT/encryption secrets, SMTP) y purgar `_legacy/apidian/.env*` del historial de git con `git filter-repo`/BFG; corregir `.gitignore` | **Bajo** (mecánico, pero urgente e innegociable) | 06 |
| 2 | Reconciliar con `GetStatus`/`GetStatusZip` antes de liberar el consecutivo en un `send_error` por timeout; evaluar mover a la ruta asíncrona con reintento | **Medio** | 04 |
| 3 | Aplicar `requireManage`/`CanManage` real (middleware, no llamada opcional) en las acciones administrativas de `sales`, `purchase`, `accounting`, `electronic`, `thirdparty`, `inventory` | **Alto** (transversal a casi todos los módulos) | 09, 03 |
| 4 | Cerrar el hueco de auditoría en `electronic` (emisión/anulación) y `security` (login/cambio de contraseña/invitación) | **Medio** | 09 |
| 5 | Hacer que los errores de los suscriptores del bus de eventos (`on_sale_confirmed`, `on_purchase_received`, etc.) se propaguen al caso de uso en vez de solo loguearse; registrar la falla en `audit.events` | **Medio** | 03 |
| 6 | Cifrar el blob del certificado `.p12` en BD con la misma capa AES-256-GCM ya usada para `certificate_password`/PINes | **Bajo** | 04, 06 |
| 7 | Rate limiting/bloqueo progresivo en login; endpoint de logout con invalidación de sesión (hoy cambiar contraseña no revoca tokens ya emitidos) | **Medio** | 06 |
| 8 | Introducir CI mínimo que corra `go test ./... -tags=integration` con Postgres de servicio, y `go vet`/`govulncheck`/`npm audit` en cada push | **Medio** | 07, 08 |
| 9 | Healthcheck real (`pool.Ping`) en `/health`; fijar `MaxConns` del pool; documentar y verificar la estrategia de backup real (la documentación menciona Railway, pero el `.env.example` ya apunta a Neon — hay que resolver esa discrepancia) | **Medio-Alto** | 08 |
| 10 | Actualizar `react-router` a `>=7.18.2` (CVE alto real) y correr `npm audit fix` en `frontend` | **Bajo** | 06 |
| 11 | Interceptor global de 401 + timeout de red en `apiClient.ts`; mover el JWT fuera de `localStorage` si se quiere reducir superficie de robo por XSS | **Medio** | 05, 06 |
| 12 | Cabeceras de seguridad HTTP (CSP, X-Frame-Options, X-Content-Type-Options) y restringir CORS explícitamente en producción (hoy es fail-open a wildcard con credenciales si se olvida la variable) | **Bajo-Medio** | 06, 03 |
| 13 | Flujo mínimo de Habeas Data (Ley 1581) en `thirdparty`: consentimiento, exportación de datos, atención de derechos ARCO | **Medio** | 06 |
| 14 | Decidir si nómina electrónica DIAN es requisito de negocio vigente; si lo es, conectar `erp/internal/payroll` con `cofacture/payroll` antes de destrabar el frontend de RRHH/Nómina | **Alto** | 04, 09 |
| 15 | Costeo de inventario (mínimo promedio ponderado) antes de prometer cualquier reporte de rentabilidad | **Alto** | 09 |
| 16 | CRM comercial como módulo nuevo (`erp/internal/crm`), reutilizando `thirdparty` como base de contacto — no bloqueante, pero de alto valor de negocio si se vende como ERP completo | **Alto** | 09 |

Los ítems 1-9 son los que yo consideraría innegociables antes de un primer cliente real de pago con datos financieros/fiscales reales; 10-13 son higiene de producción que debería resolverse en el mismo ciclo; 14-16 son roadmap de producto, no bloqueantes de "puede operar de forma segura".

---

**Plan de acción ordenado por dependencia real entre componentes (cofacture → erp → frontend → expansión)**: ver [plan-de-accion.md](plan-de-accion.md).

Informes completos: [01-arquitectura](01-arquitectura.md) · [02-base-de-datos](02-base-de-datos.md) · [03-backend-go](03-backend-go.md) · [04-cofacture-dian](04-cofacture-dian.md) · [05-frontend](05-frontend.md) · [06-seguridad](06-seguridad.md) · [07-testing-calidad](07-testing-calidad.md) · [08-devops-produccion](08-devops-produccion.md) · [09-roadmap-erp](09-roadmap-erp.md)
