# Plan de Acción — orden de ejecución recomendado

Este plan reordena los hallazgos de los 9 informes (no por severidad aislada, sino por **dependencia real entre componentes**), para que cada fase se construya sobre una base ya corregida en vez de tener que rehacerse después.

## Lógica de secuenciación

La relación de dependencia del sistema es `frontend → erp → cofacture`: el frontend consume la API de `erp`, y `erp` consume `cofacture` como motor de facturación. Eso significa que **`cofacture` es la base de todo lo financiero/legal** — si tiene un bug de reconciliación o un dato sin cifrar, cualquier trabajo que se haga en `erp` (auditoría, RBAC, reportes) hereda ese riesgo sin saberlo, y si `cofacture` cambia de comportamiento más adelante para corregirlo, ese trabajo en capas superiores puede tener que revisarse. Por eso el orden general es **de abajo hacia arriba: Cofacture → erp → frontend → expansión de producto**, con una única excepción:

- **La fuga de secretos en git (Fase 0) no respeta esta jerarquía** — es un problema de higiene de repositorio, no de arquitectura de capas, y una credencial de base de datos real expuesta hoy es explotable independientemente de en qué capa se esté trabajando. Se resuelve primero y en paralelo a todo lo demás, no como parte de la secuencia de capas.

Dentro de cada fase, el orden interno también importa: por ejemplo, en la Fase 2 la autorización (RBAC) va antes que la auditoría porque ambas tocan los mismos handlers — conviene modificarlos una sola vez — y el CI de tests va después de escribir los tests de `electronic` (Fase 1), porque un CI que no corre nada de valor todavía no aporta.

Cada ítem indica: archivos/módulo, de qué otro ítem depende (si depende de alguno), esfuerzo cualitativo, e informe de origen.

---

## Fase 0 — Emergencia inmediata (sin dependencias, hoy mismo)

| # | Tarea | Estado | Depende de | Esfuerzo | Informe |
|---|---|---|---|---|---|
| **01** | Rotar la contraseña de Postgres en alwaysdata.net, las credenciales del proyecto Neon, `AUTH_JWT_SECRET`, `ISSUER_SECRETS_KEY` y las credenciales SMTP de Mailtrap expuestas en `_legacy/apidian/.env`/`.env.old`. Borrar ambos archivos del working tree, purgar el historial completo de git con `git filter-repo`/BFG (un `git rm` normal no basta), y corregir `.gitignore` para que `_legacy/apidian/.env*` quede cubierto. | ⚠️ **Riesgo aceptado, NO resuelto** (ver nota) | — | Bajo | 06 |

**Nota sobre el estado del punto 01 (2026-08-09):** `_legacy/` se eliminó del árbol de trabajo (commit `b29cf77`) pero, por decisión explícita del usuario, **sin purgar el historial de git** — los archivos `.env`/`.env.old` con las credenciales reales siguen recuperables desde commits anteriores (`git show e646a51:_legacy/apidian/.env`). Se confirmó además, preguntando directamente, que **esas credenciales (Postgres alwaysdata.net, Postgres Neon, `AUTH_JWT_SECRET`, `ISSUER_SECRETS_KEY`, SMTP Mailtrap) siguen en uso en producción hoy**. Ante esto, el usuario decidió explícitamente **aceptar el riesgo residual sin rotarlas por ahora**, en vez de proceder con la rotación guiada que se le ofreció. Este punto queda cerrado como "riesgo aceptado", no como "resuelto" — sigue siendo una credencial de producción real, expuesta y sin rotar, recuperable por cualquiera con acceso de lectura al repositorio o a un clon/fork existente.

---

## Fase 1 — Cofacture: la base de la que depende todo lo financiero/legal

| # | Tarea | Estado | Depende de | Esfuerzo | Informe |
|---|---|---|---|---|---|
| **02** | Reconciliar con `GetStatus`/`GetStatusZip` antes de liberar el consecutivo cuando `sendSync` falla por timeout/error de transporte (`confirm.go`, función `markError`/`finish`) — hoy se libera el número sin confirmar si la DIAN sí procesó el documento. | ✅ **Resuelto** (ver nota) | — | Medio | 04 |
| **03** | Mecanismo de reintento/contingencia real para el envío síncrono: mover a la ruta asíncrona (`SendBillAsync` + `GetStatusZip`, que ya tiene un identificador para consultar después) o agregar un job de reintento con backoff. | ✅ **Resuelto** (ver nota) | 02 (comparten la lógica de reconciliación) | Medio-Alto | 04 |
| **04** | Cifrar el blob del certificado `.p12` en BD con la misma capa AES-256-GCM (`cryptutil`) ya usada para `certificate_password`/PINes. | ✅ **Resuelto** (ver nota) | — | Bajo | 04, 06 |
| **05** | Registrar un estado dedicado (`StatusEnvironmentMismatch` o similar) cuando el ambiente del rango de numeración no coincide con el de la empresa, en vez de dejar el documento atascado en `built` sin ningún error visible. | ✅ **Resuelto** (ver nota) | — | Bajo | 04 |
| **06** | Reemplazar el `PostalZone` fijo (`"000000"`) por captura/validación del dato postal real en `thirdparty`. | ⏳ Pendiente | — | Medio (toca el formulario de terceros) | 04 |
| **07** | Tests de aplicación para `electronic/application/confirm.go`: decisión síncrona/asíncrona, manejo de `TestSetID`, y el nuevo comportamiento de reconciliación de 02/03/05. | ⏳ Pendiente | 02, 03, 05 (para testear el comportamiento correcto, no el que se está reemplazando) | Medio | 07 |

**Nota sobre el punto 02 (2026-08-09):** el título original decía "reconciliar con `GetStatus`/`GetStatusZip`", pero al implementarlo se confirmó que eso **no es mecánicamente posible para `SendBillSync`**: el endpoint síncrono de la DIAN no devuelve ningún identificador (trackID/ZipKey) hasta que responde completo, así que si el `http.Client` agota el timeout esperando esa respuesta, no queda nada que consultar después — `GetStatus`/`GetStatusZip` solo sirven para las rutas asíncronas (`SendBillAsync`/`SendTestSetAsync`), que sí entregan un ZipKey de entrada. Dado eso, la corrección real implementada fue:
- El adaptador (`erp/internal/electronic/infrastructure/cofacture/adapter.go`, funciones `SendBillSync`/`SendTestSetAsync`) ahora distingue si el error es un `soap.Fault` explícito (la DIAN sí respondió y rechazó a nivel de protocolo — sin ambigüedad) usando `errors.As`, y en ese caso lo envuelve con el nuevo sentinel `domain.ErrDianRejectedSync`.
- `ConfirmUseCase.markError` (`erp/internal/electronic/application/confirm.go`) usa `errors.Is(sendErr, domain.ErrDianRejectedSync)`: si es un fault explícito → `StatusSendError` (comportamiento de siempre, libera el consecutivo). Si es cualquier otro error (timeout, conexión, respuesta ilegible) → nuevo estado `domain.StatusSendUnknown`, que **no libera el consecutivo** en `finish()`.
- El error de empaquetado ZIP (`uc.zipper.Zip(...)`, que ocurre *antes* de cualquier llamada a la DIAN) se separó para ir directo a `StatusSendError` sin pasar por esta ambigüedad — ahí sí hay certeza absoluta de que nunca se envió nada.
- Se agregó el estado nuevo en el frontend (`types.ts`, `StatusBadge.tsx` con tono *warning*, filtros de las 5 páginas de documentos) y en las métricas de `stats` (se cuenta junto a `rejected`/`send_error` para no desaparecer del dashboard).
- Verificado con `go build ./... && go vet ./... && go test ./...` (limpio, sin `FAIL`) y `npm run build` en frontend (sin errores de tipos).
- **Limitación conocida, que queda para el punto 03**: un documento en `StatusSendUnknown` no tiene hoy ningún camino automático de resolución — el consecutivo queda bloqueado indefinidamente hasta una revisión manual (verificar en el portal de la DIAN) o una intervención directa en BD. No se construyó un endpoint de resolución manual en este punto para no exceder el alcance — es candidato natural para cuando se aborde el punto 03 (mecanismo de reintento/contingencia real).

**Nota sobre el punto 03 (2026-08-09):** se eligió "agregar un reintento" en vez de "migrar producción entera a la ruta asíncrona", precisamente para no cambiar la latencia del caso feliz (una confirmación exitosa por `SendBillSync` sigue respondiendo igual de rápido que siempre). El reintento solo se activa en el caso de error:
- `SenderPort` ganó `SendBillAsync` (puerto + adaptador, mismo patrón que `SendTestSetAsync`, con la misma distinción `soap.Fault` vs. error de transporte del punto 02).
- `ConfirmUseCase.sendSync`: si `SendBillSync` falla con un error ambiguo (no `ErrDianRejectedSync`), ya no se rinde — reintenta reenviando el **mismo ZIP ya firmado** (mismo CUFE, no un documento nuevo) por `SendBillAsync`, y si eso da un `zipKey`, hace el mismo sondeo acotado (6×5s) que ya usaba la ruta de habilitación. Si el reintento también resulta ambiguo, el documento queda en `StatusSendUnknown` (no libera el número) con ambos mensajes de error conservados. Si el sondeo se agota sin respuesta, queda en `StatusSent` con el `zipKey` guardado — recuperable, a diferencia de `StatusSendUnknown`.
- Se refactorizó el sondeo (`pollZipKey`/`finishFromPoll`) para compartirlo entre `sendTestSet` y el nuevo reintento, sin duplicar la lógica.
- **Se cerró el dead-end que ya existía en `StatusSent`** (afectaba también a la ruta de habilitación, no solo a la nueva): antes, si el sondeo se agotaba, el documento quedaba ahí para siempre sin ningún código que lo volviera a consultar. Nuevo caso de uso `ConfirmUseCase.CheckPendingStatus(companyID, id)` + endpoint `POST /api/v1/electronic/documents/{id}/check-status` + botón "Consultar estado" en las 5 páginas de edición de documentos (visible solo cuando `status == "sent"`) — permite volver a sondear con el mismo `zipKey` sin generar ningún envío nuevo.
- Verificado con `go build/vet/test` (limpio) y `npm run build` (sin errores de tipos).
- **Limitación que queda abierta**: todo lo anterior es a demanda (el reintento automático ocurre en el momento del fallo; `CheckPendingStatus` requiere que un usuario haga clic). No hay ningún job en segundo plano que barra periódicamente los documentos en `StatusSent` sin intervención humana — sería la evolución natural si se quiere contingencia 100% automática, pero se dejó fuera de este punto para no sumar una pieza de infraestructura nueva (scheduler cross-tenant) al mismo cambio. (Nota: no quedó como ítem numerado del plan, ver conversación del 2026-08-09 — no se le asignó fecha de revisión, se recomendó revisarlo solo si en producción real empiezan a acumularse documentos en `StatusSent`.)

**Nota sobre el punto 04 (2026-08-09):** `erp/internal/company/infrastructure/persistence/postgres/repository.go` — `Save`, `UpdateCredentials` y `scanCompany` ahora cifran/descifran la columna `certificate` con `cryptutil.Encrypt`/`Decrypt` (misma clave `ENCRYPTION_KEY` ya usada para `certificate_password`/PINes), en vez de guardar el blob PKCS12 crudo. **Cambio no retrocompatible**: un certificado guardado bajo el esquema anterior fallaría al descifrarse (el blob no es un ciphertext AES-GCM real, la verificación de integridad de GCM lo rechaza) — se confirmó directamente con el responsable del proyecto que **hoy no hay ninguna empresa real (ni local ni en producción) con un certificado ya subido**, así que no hizo falta backfill ni migración de datos existentes. Si en el futuro se despliega esto sobre una base de datos con certificados ya cargados, hay que cifrarlos en el mismo momento del despliegue (o pedir que se vuelvan a subir) — no hacerlo así rompería la firma/envío de documentos para esas empresas. Verificado con `go build/vet/test` (limpio).

**Nota sobre el punto 05 (2026-08-09):** `erp/internal/electronic/domain/document.go` gana `StatusEnvironmentMismatch`. En `finalizeAndSend` (`confirm.go`), cuando `p.company.Environment != p.nr.Environment`, antes se retornaba el documento en `StatusBuilt` silenciosamente (sin error, sin transición futura — quedaba atascado para siempre sin que nadie se enterara salvo notando que nunca cambiaba de estado). Ahora pasa por `finish(...)` con el nuevo estado y un mensaje explícito ("el ambiente del rango de numeración (X) no coincide con el de la empresa (Y)..."), y — a diferencia de `StatusSendUnknown` — **sí libera el consecutivo**, porque aquí no hay ninguna ambigüedad: el documento nunca llegó a transmitirse a la DIAN (el error se detecta antes de empaquetar/enviar), así que no hay riesgo de doble facturación al liberar el número. Estado propagado a frontend (`StatusBadge` con tono de error — a diferencia de `send_unknown` esto sí es un fallo cierto, no ambiguo — y filtros de las 5 páginas de documentos) y a `stats`. Verificado con `go build/vet/test` y `npm run build` (ambos limpios).

---

## Fase 2 — erp (backend): autorización, trazabilidad y confiabilidad de la plataforma

Una vez el motor de facturación es confiable, lo que bloquea producción es que **cualquier usuario autenticado puede saltarse permisos por API** y que **los fallos de negocio no dejan rastro** — esto es transversal a todos los módulos de `erp`, así que se resuelve antes de seguir puliendo módulos individuales.

| # | Tarea | Depende de | Esfuerzo | Informe |
|---|---|---|---|---|
| **08** | Aplicar autorización real (RBAC) de forma sistemática: definir qué acciones son "administrativas" por módulo y envolverlas con `requireManage`/`CanManage` — empezando por `sales.handleCancel`, recepción/pago en `purchase`, anulación en `electronic` (hoy solo se oculta en el frontend, el backend no valida nada en estos casos). | — | Alto (transversal) | 09, 03 |
| **09** | Cerrar el hueco de auditoría en `electronic` (emisión/anulación DIAN, 0 de 26 handlers) y `security` (login/cambio de contraseña/invitación, 0 de 10 handlers). | 08 (mismos handlers, conviene tocarlos una sola vez) | Medio | 09 |
| **10** | Propagar los errores del bus de eventos (`on_sale_confirmed`, `on_purchase_received`, etc.) al caso de uso que publica en vez de solo `log.Printf`; decidir por evento si debe abortar la operación o solo advertir; registrar la falla en `audit.events`. | 09 (usa la infraestructura de auditoría ya reforzada) | Medio | 03 |
| **11** | Reemplazar los 14 `log.Printf` de esos mismos handlers por el logger estructurado del proyecto, y pasar el `context.Context` real de la request en vez de `context.Background()`. | 10 (se tocan los mismos archivos) | Bajo | 03 |
| **12** | Middleware de `recover()` centralizado con request-ID en la cadena HTTP; envolver `RunTRMDailySync` con `defer recover()` (única goroutine de larga vida sin protección hoy). | — | Bajo-Medio | 03 |
| **13** | Corregir el CORS "fail-open" (no activar wildcard automáticamente si falta la variable de entorno) y agregar cabeceras de seguridad HTTP básicas (`X-Content-Type-Options`, `X-Frame-Options`, CSP). | — | Bajo-Medio | 03, 06 |
| **14** | Rate limiting/bloqueo progresivo en login; endpoint de logout con invalidación server-side; invalidar sesiones activas al cambiar contraseña. | — | Medio | 06 |
| **15** | Resolver las 3 violaciones de encapsulamiento de schema (`company`→`security.user_companies`, `audit`→`security.users`, `stats`→`electronic.documents`) creando puertos locales, siguiendo el mismo patrón que ya usan `sales`/`purchase`/`electronic`. | — | Medio | 01, 02 |
| **16** | Ajustes de base de datos: `.down.sql` faltantes (`electronic`, `hr`, `payroll`, `saas`), `UNIQUE(company_id, number)` en `sales.sales`/`purchase.orders`, índices en FKs de `electronic.documents`, `MaxConns`/`MinConns` explícitos en el pool. | — | Bajo-Medio | 02, 08 |
| **17** | CI mínimo: `go test ./... -tags=integration` con Postgres de servicio + `go vet` + `govulncheck` + `npm audit`, en cada push. | 07 (para que ya exista algo de valor que el CI corra en `electronic`) | Medio | 07, 08 |
| **18** | Healthcheck real (`pool.Ping` en `/health`); resolver la discrepancia de infraestructura documentada (Railway vs. Neon/systemd) y documentar/verificar la estrategia de backup real; versionar el script de deploy y el unit file de systemd. | 17 (mismo esfuerzo de "hacer reproducible el despliegue") | Medio-Alto | 08 |
| **19** | Flujo mínimo de Habeas Data (Ley 1581) en `thirdparty`: consentimiento, exportación de datos del titular, procedimiento de derechos ARCO. | — | Medio | 06 |
| **20** | Actualizar `react-router` a `>=7.18.2` (CVE alto real) y correr `npm audit fix` para `postcss`/`nanoid` en `frontend`. | — | Bajo | 06 |

---

## Fase 3 — Frontend: confiabilidad de sesión y accesibilidad

Con el backend ya autorizando y auditando correctamente, el frontend deja de tener que compensar ocultando botones como única defensa, y puede enfocarse en su propia deuda.

| # | Tarea | Depende de | Esfuerzo | Informe |
|---|---|---|---|---|
| **21** | Interceptor global de 401 en `apiClient.ts` con logout + redirección automática (hoy cada página muestra un error genérico). | 14 (necesita que exista logout server-side) | Medio | 05 |
| **22** | Timeout de red (`AbortController`) en `fetch()` dentro de `request()`/`getBlob()`. | — | Bajo | 05 |
| **23** | Hook compartido `useApiResource` (loading/error/cancelado) para reemplazar el patrón duplicado y con variaciones entre páginas. | 21, 22 (se construye sobre el cliente ya corregido) | Medio | 05 |
| **24** | Accesibilidad: `aria-label` en botones icon-only e inputs de búsqueda crudos, semántica ARIA (`role="combobox"`, etc.) en `Combobox.tsx`. | — | Bajo-Medio | 05 |
| **25** | Smoke tests de frontend (Vitest + Testing Library) sobre los flujos críticos: login, confirmación de venta/factura. | 21-24 (para testear el comportamiento ya corregido) | Medio | 07 |

---

## Fase 4 — Expansión de producto (alcance nuevo, no bugs — solo después de cerrar 0-3)

Estos ítems no son correcciones de algo roto, son capacidades nuevas. Tiene sentido dejarlos al final: construirlos sobre una base sin RBAC/auditoría/reconciliación real solo multiplicaría el riesgo ya identificado.

| # | Tarea | Depende de | Esfuerzo | Informe |
|---|---|---|---|---|
| **26** | Decidir si la nómina electrónica DIAN es requisito de negocio vigente; si lo es, conectar `erp/internal/payroll` con `cofacture/payroll` + `cofacture/soap` (el builder y el CUNE ya existen, están huérfanos) antes de destrabar el frontend de RRHH/Nómina. | Fases 1-3 cerradas | Alto | 04, 09 |
| **27** | Planificar eventos RADIAN (acuse de recibo, recibo del bien, aceptación/rechazo) si el negocio necesita que las facturas operen como título valor — hoy no hay ningún punto de partida en el código. | Fase 1 cerrada | Alto | 04 |
| **28** | Costeo de inventario (mínimo promedio ponderado) — requisito previo para cualquier reporte de rentabilidad real. | — | Alto | 09 |
| **29** | CRM comercial como módulo nuevo (`erp/internal/crm`), reutilizando `thirdparty` como base de contacto y el mismo patrón de puertos que ya usan `sales`/`purchase`/`inventory`. | Fase 2 cerrada (para que nazca ya con el RBAC/auditoría correctos, no heredando el problema) | Alto | 09 |
| **30** | Ampliar `stats` con reportes operativos (top clientes/productos, rotación de inventario, comparativo ventas-compras) sobre la infraestructura de dashboard ya existente. | 28 (para que los reportes de rentabilidad sean correctos desde el día uno) | Alto | 09 |

---

Informes de origen: [01-arquitectura](01-arquitectura.md) · [02-base-de-datos](02-base-de-datos.md) · [03-backend-go](03-backend-go.md) · [04-cofacture-dian](04-cofacture-dian.md) · [05-frontend](05-frontend.md) · [06-seguridad](06-seguridad.md) · [07-testing-calidad](07-testing-calidad.md) · [08-devops-produccion](08-devops-produccion.md) · [09-roadmap-erp](09-roadmap-erp.md) · [00-resumen-ejecutivo](00-resumen-ejecutivo.md)
