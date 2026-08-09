# Agente Cofacture (Motor de Facturación Electrónica DIAN)

Auditoría de `cofacture/` (motor UBL 2.1 / firma XAdES / SOAP DIAN) y de su consumo desde
`erp/internal/electronic`. Metodología: lectura directa de código, no de documentación.

### ✅ Lo que está bien implementado

- **Fórmulas CUFE/CUDE/CUDS correctas y consistentes con la auditoría previa.**
  `cofacture/cufe/cufe.go:27` y `cofacture/cude/cude.go:23` comparten `dianhash.Seed` con los
  tres slots fijos de impuesto ("01"/"04"/"03" = IVA/INC/ICA). `cofacture/cuds/cuds.go:22-39`
  sigue usando **un único par `CodImp+ValImp` tomado de `HeaderTaxes[0]`** con SHA-384
  (`sha512.Sum384`), tal como se documentó antes — el comentario del archivo (líneas 4-8)
  explica explícitamente por qué difiere del CUFE/CUDE. Esta regla especial **sigue vigente**.
- **`schemeName="31"` sigue forzado para el SNO del Documento Soporte**, en dos capas
  independientes: `cofacture/builder/support_document.go:120` (hardcode al construir XML) y
  `erp/internal/electronic/application/confirm.go:527` (`supplierAsNIT` fuerza `TypeCode = "31"`
  antes de llegar al builder, y recalcula el dígito de verificación con `nit.ComputeCheckDigit`
  en vez de confiar en el valor guardado en BD).
- **`PostalZone` ya está implementado** como campo del dominio (`cofacture/domain/types.go:22`)
  y se serializa condicionalmente en `cofacture/builder/party.go:96-98`. `confirm.go:515-518` y
  `confirm.go:528-530` rellenan `"000000"` por defecto si el tercero no tiene dirección postal,
  para los casos DS/NA donde antes faltaba.
- **Firma XAdES-EPES bien fundamentada**: `cofacture/signer/signer.go` construye las 3
  `ds:Reference` (documento, KeyInfo, SignedProperties) con C14N 1.0 inclusivo
  (`dsig.MakeC14N10RecCanonicalizer()`, línea 33) y RSA-SHA256, con comentarios que documentan
  verificación byte a byte contra facturas reales de dos proveedores distintos.
- **Manejo de matices reales del servicio DIAN, no solo del anexo técnico.** `cofacture/soap/client.go:106-116`
  documenta y maneja explícitamente que la DIAN devuelve HTTP 404 con un body SOAP válido (no
  un fault) en casos normales (`GetAcquirer`). `cofacture/dian/parser.go:90-100`
  (`IsTestSetClosed`) distingue un cierre de set de pruebas de un rechazo real de contenido, y
  `HasRejections()` distingue "Rechazo" real de simples notificaciones dentro de
  `ErrorMessage`.
- **Notas Crédito, Notas Débito, Documento Soporte y Nota de Ajuste tienen código funcional
  real**, no solo estructura: `cofacture/builder/credit_note.go`, `debit_note.go`,
  `support_document.go`, `adjustment_note.go` construyen árboles XML completos con roles de
  partes, referencias de facturación (`BillingReference`), `DiscrepancyResponse`, impuestos y
  totales — no son stubs. `erp/internal/electronic/application/confirm.go` tiene un caso
  `confirmXxx` end-to-end para cada uno de los 5 tipos de documento (FE, NC, ND, DS, NA).
- **Distinción real de ambiente habilitación/producción**: URLs SOAP separadas y correctas
  (`cofacture/soap/client.go:19-20`, verificadas contra el WSDL real según el comentario), y
  `erp/internal/electronic/infrastructure/cofacture/adapter.go:232-237` (`soapURL`) resuelve
  por `environmentCode`. `TestSetID` se maneja como campo propio de `NumberingRange` y solo se
  usa la ruta asíncrona (`SendTestSetAsync` + `GetStatusZip`, sondeo acotado de 6 intentos/5s en
  `confirm.go:399-413`) cuando el ambiente es habilitación y hay `TestSetID`; producción siempre
  usa `SendBillSync` síncrono (`confirm.go:372-373`).
- **Secretos DIAN cifrados en reposo (parcialmente, ver riesgos)**: `SoftwarePIN`,
  `CertificatePassword` y `NeSoftwarePIN` se cifran con AES-256-GCM
  (`erp/internal/shared/cryptutil/aes.go`) antes de persistirse
  (`erp/internal/company/infrastructure/persistence/postgres/repository.go:31-42`), igual que
  `TechnicalKey` del rango de numeración
  (`erp/internal/electronic/infrastructure/persistence/postgres/numbering_repository.go:38`).
  Las respuestas HTTP de `company` solo exponen `has_certificate` (bool) y metadatos
  (subject/issuer/expiración), nunca los bytes crudos
  (`erp/internal/company/interfaces/http/handlers.go:573-576`), y la actualización de
  credenciales está protegida por `requireManage` (`handlers.go:150`).
- **Tests "reales" contra la DIAN están correctamente aislados**: los `*_real_test.go` /
  `realsend_*_test.go` hacen `t.Skip` si no existe `COFACTURE_TEST_FIXTURES_DIR`
  (`cofacture/soap/realsend_test.go:53`), así que no rompen CI ni ejecutan contra DIAN por
  accidente.
- **Desacoplamiento correcto vía puertos**: `erp/internal/electronic/domain/ports.go` define
  `BuilderSignerPort`, `ZipperPort`, `SenderPort`, `DianRangesFetcherPort`, `EmailZipPort` como
  interfaces del dominio; `erp/internal/electronic/infrastructure/cofacture/adapter.go` es el
  único archivo de `erp` que importa tipos concretos de `cofacture` (builder/signer/soap/zip),
  y el dominio/aplicación de `electronic` solo ve `cofdom.Invoice` y los puertos propios.

### ⚠️ Lo que existe pero tiene problemas / riesgos

- **[RIESGO ALTO — bloqueante de producción] Ambigüedad de timeout no reconciliada: puede
  producir doble facturación o números atascados.** En `confirm.go:378-388` (`sendSync`), si
  `sender.SendBillSync` devuelve error (incluido timeout HTTP tras 60s,
  `cofacture/soap/client.go:39`), se llama `markError` → `finish(..., StatusSendError, ...)`
  **sin `trackID`** (`confirm.go:431-433`, el 3er argumento de `finish` es `""`) y **sin
  consultar `GetStatus`/`GetStatusZip` para verificar si la DIAN sí procesó el documento**. Acto
  seguido, `finish` (línea 445-447) libera el consecutivo con `ReleaseIfCurrent`. Si el timeout
  fue solo de red (la DIAN sí recibió y validó/aceptó el documento del lado de ellos), el
  sistema: (a) no tiene forma de saberlo porque no guardó ningún identificador de rastreo, y
  (b) libera el número para que un reintento manual (`CloneDraft`,
  `erp/internal/electronic/application/create_draft.go:361`) lo reutilice en un documento
  *distinto*, con una fecha/hora distinta y por tanto un CUFE distinto — dos documentos
  jurídicamente independientes para la misma operación de negocio (riesgo de doble
  facturación/doble IVA declarado) o, si el número queda reutilizado antes que el primero llegue
  a "Aceptado", un rechazo DIAN por consecutivo duplicado. No existe ningún paso de
  reconciliación (`GetStatus` con el número de radicado) antes de liberar el consecutivo o antes
  de permitir el reintento.

  **✅ Resuelto (2026-08-09) — parcialmente, ver limitación anotada.** Se confirmó que una
  reconciliación literal vía `GetStatus`/`GetStatusZip` no es mecánicamente posible para
  `SendBillSync`: la DIAN no entrega ningún identificador hasta responder completo, así que un
  timeout no deja nada que consultar después (`GetStatus`/`GetStatusZip` solo aplican a las
  rutas asíncronas, que sí devuelven un ZipKey de entrada). La corrección real implementada:
  `erp/internal/electronic/infrastructure/cofacture/adapter.go` (`SendBillSync`,
  `SendTestSetAsync`) ahora distingue con `errors.As` si el error es un `*soap.Fault` explícito
  (la DIAN respondió y rechazó a nivel de protocolo, sin ambigüedad) — lo envuelve con el nuevo
  `domain.ErrDianRejectedSync`. `ConfirmUseCase.markError` (`confirm.go`) usa
  `errors.Is(sendErr, domain.ErrDianRejectedSync)`: si es un fault explícito, comportamiento de
  siempre (`StatusSendError`, libera el consecutivo); para cualquier otro error (timeout,
  conexión, respuesta ilegible) pasa al nuevo estado `domain.StatusSendUnknown`, que **no libera
  el consecutivo** en `finish()` — cierra el escenario concreto de doble facturación descrito
  arriba. El error de empaquetado ZIP (antes de cualquier llamada a la DIAN) se separó para ir
  directo a `StatusSendError`, ya que ahí sí hay certeza de que nunca se envió nada. Nuevo
  estado propagado a frontend (`StatusBadge` con tono *warning*, filtros de las 5 páginas de
  documentos) y a `stats` (cuenta junto a `rejected`/`send_error`). Verificado con
  `go build/vet/test` (limpio) y `npm run build` (sin errores de tipos).

  **Limitación conocida, que queda para el punto 03 del plan de acción**: hoy no existe ningún
  camino automático (ni manual vía endpoint) para resolver un documento en `StatusSendUnknown` —
  el consecutivo queda bloqueado indefinidamente hasta una verificación manual en el portal de
  la DIAN o una intervención directa en base de datos. Tampoco se implementó todavía el
  mecanismo de reintento/contingencia en sí (mover a la ruta asíncrona o agregar backoff) — eso
  sigue siendo el punto 03, que ahora sí puede apoyarse en la distinción de estados que quedó
  lista aquí.
- **[RIESGO ALTO] No hay reintentos automáticos ni cola de contingencia para el envío
  síncrono.** `cofacture/soap/client.go` hace una única petición HTTP con `Timeout: 60 *
  time.Second` (línea 39) y no reintenta. El único mecanismo de "reintento" en todo el pipeline
  es el sondeo acotado de `GetStatusZip` (6 intentos × 5s, `confirm.go:399-413`), que es
  exclusivo de la ruta asíncrona de habilitación (`SendTestSetAsync`), no de `SendBillSync` (la
  ruta normal de producción). Si el servicio DIAN está caído o lento, la única salida es
  `StatusSendError` y una acción manual del usuario (`CloneDraft`) — no hay cola de reintento, ni
  backoff, ni job en segundo plano. No se encontró ningún mecanismo de "contingencia" (búsqueda
  de `contingenc` en todo el repo solo aparece en documentación, no en código).

  **✅ Resuelto (2026-08-09), parcialmente — ver limitación anotada.** Se agregó `SendBillAsync`
  al `SenderPort`/adaptador (mismo patrón que `SendTestSetAsync`, con la misma distinción
  `soap.Fault`/error de transporte del punto 02). `ConfirmUseCase.sendSync` ahora, ante un error
  ambiguo de `SendBillSync`, reintenta automáticamente reenviando el **mismo ZIP ya firmado**
  (mismo CUFE, no un documento nuevo) por `SendBillAsync` + el mismo sondeo acotado de
  `GetStatusZip` que ya usaba la ruta de habilitación (extraído a un helper compartido
  `pollZipKey`/`finishFromPoll`). Se optó por "reintentar en el momento del fallo" en vez de
  "mover toda la producción a async por defecto" para no añadir latencia al caso feliz (una
  confirmación exitosa sigue siendo tan rápida como siempre). Además se cerró un dead-end que ya
  existía también en la ruta de habilitación: si el sondeo de `GetStatusZip` se agotaba sin
  respuesta, el documento quedaba en `StatusSent` para siempre sin ningún código que lo
  volviera a consultar. Nuevo caso de uso `CheckPendingStatus` + endpoint
  `POST /api/v1/electronic/documents/{id}/check-status` + botón "Consultar estado" (visible solo
  con `status == "sent"`) en las 5 páginas de edición de documentos.

  **Limitación que queda abierta**: todo esto es a demanda — el reintento automático solo ocurre
  en el instante del fallo original; `CheckPendingStatus` requiere que un usuario haga clic. No
  hay ningún job en segundo plano que revise periódicamente, sin intervención humana, los
  documentos que quedaron en `StatusSent`. Sería la evolución natural para contingencia 100%
  automática, pero implica infraestructura nueva (un scheduler que recorra todas las empresas,
  no solo la actual) y se dejó fuera de este cambio para no mezclarla con el fix puntual.
- **[RIESGO MEDIO] Certificado `.p12`/`.pfx` se almacena SIN cifrar en la base de datos.** En
  `erp/internal/company/infrastructure/persistence/postgres/repository.go:66` (`Save`) y
  `:199-200` (`UpdateCredentials`), la columna `certificate` guarda `c.Certificate`/`p.Certificate`
  como `BYTEA` crudo — a diferencia de `certificate_password`, `software_pin` y `ne_software_pin`,
  que sí pasan por `cryptutil.Encrypt`. El contenedor PKCS12 completo (que incluye la llave
  privada RSA del emisor, protegida solo por la contraseña del certificado) queda expuesto ante
  cualquier acceso de lectura a la base de datos (dump, backup, réplica, acceso de un DBA), sin
  necesitar la clave de aplicación (`ENCRYPTION_KEY`). Es más consistente cifrar también el blob
  del certificado con la misma clave AES-256-GCM ya usada para los demás secretos.
- **[RIESGO MEDIO] Envío silenciosamente omitido cuando el ambiente del rango no coincide con
  el de la empresa.** En `confirm.go:361-364`, si `p.company.Environment != p.nr.Environment`
  el documento se persiste en estado `StatusBuilt` (ya "gastó" el consecutivo y generó XML
  firmado) y la función retorna sin error y sin llamar a `finish` — el documento queda
  indefinidamente en `built` sin transicionar nunca a `sent`/`accepted`/`rejected`/`send_error`.
  No hay ningún estado de error visible para el usuario ni alerta; solo se detecta mirando que
  nunca cambia de estado.
- **[RIESGO MEDIO] `PostalZone` por defecto es un valor ficticio fijo (`"000000"`), no un dato
  real.** `confirm.go:516` y `confirm.go:529` insertan `"000000"` cuando el tercero no tiene
  dirección postal capturada, en vez de exigir o inferir un valor real. Esto satisface el
  esquema (campo presente) pero no necesariamente la intención de la regla (código postal real
  del tercero); es una solución de conveniencia, no una corrección de datos.
- **[RIESGO BAJO-MEDIO] Nómina electrónica: el builder DIAN existe pero no está conectado al
  ERP.** `cofacture/payroll/builder.go` y `cofacture/payroll/cune.go` construyen XML de
  `NominaIndividual` con CUNE real, y `cofacture/soap/operations.go:126-155` expone
  `SendNominaSync`/`SendNominaSyncTestSet`. Sin embargo, `erp/internal/payroll` (módulo de
  nómina/HR real del ERP) **no importa `cofacture` en ningún archivo** (grep sin resultados). Es
  decir, el ERP puede generar y pagar nóminas pero no genera ni envía el Documento Soporte de
  Pago de Nómina Electrónica a la DIAN — la pieza de cofacture existe pero está huérfana.

### ❌ Lo que falta por completo

- **❌ Eventos RADIAN no implementados.** No existe código para acuse de recibo, recibo del
  bien/servicio, ni aceptación/rechazo expreso de factura por parte del receptor, en ningún
  paquete de `cofacture` ni de `erp`. La búsqueda de términos RADIAN/acuse/receipt-of-goods solo
  aparece en documentación (`docs/apidian-architecture.md`, `docs/auditorias/`), nunca en `.go`.
  Si el negocio necesita soportar el ciclo RADIAN (obligatorio para que la factura sea título
  valor), esto es una laguna total, no una implementación parcial.
- **✅ Resuelto (2026-08-09), con matiz — antes: sin cola/mecanismo de contingencia formal.**
  Ahora hay un reintento automático (reenvío del mismo ZIP firmado por `SendBillAsync`) ante un
  fallo ambiguo de `SendBillSync`, más `CheckPendingStatus` para resolver manualmente un
  documento que quedó en `StatusSent`. Sigue sin existir job en background/cola de mensajes que
  haga esto sin intervención humana — la revisión de `StatusSent` sigue siendo a demanda (un
  clic en "Consultar estado"), no periódica y automática.
- **✅ Resuelto (2026-08-09), con matiz.** Ya no se libera el consecutivo automáticamente ante un
  error ambiguo de `sendSync` — pasa a `StatusSendUnknown` en vez de `StatusSendError` (ver
  detalle en el hallazgo de arriba). Sigue siendo cierto que no hay ninguna llamada real a
  `GetStatus`/`GetStatusZip` porque, para `SendBillSync`, la DIAN no entrega ningún identificador
  que permitiera esa consulta — la reconciliación automática de verdad solo es viable si se migra
  a la ruta asíncrona (punto 03 del plan de acción).
- **❌ Nómina electrónica no integrada al flujo del ERP** (ver riesgo arriba) — no es solo una
  carpeta vacía, pero el 100% del trabajo de conexión (dominio `payroll` del ERP → cofacture →
  SOAP) está sin hacer.
- **❌ No hay endpoint de "reintentar envío" para un documento en `send_error`** — solo existe
  clonar el borrador desde cero, que no preserva ni referencia el intento original fallido más
  allá de crear un nuevo documento independiente.

### 🔧 Recomendaciones concretas y accionables

1. **✅ Implementado (2026-08-09).** Antes de liberar el consecutivo en `markError`/`finish` (`confirm.go:431-447`), reconciliar
   con la DIAN. Si el error fue de transporte (timeout, conexión rechazada) en vez de un
   `soap.Fault` explícito, intentar `GetStatus`/`GetStatusZip` con el mismo ZipKey/trackID antes
   de decidir `StatusSendError` + liberar el número. Si no hay forma de obtener un trackID (el
   envío nunca llegó a generarse), documentar explícitamente esa distinción en el estado
   guardado.
2. **✅ Implementado (2026-08-09), a demanda — no en background.** `SendBillSync` ahora reintenta
   automáticamente vía `SendBillAsync` + `GetStatusZip` en el momento del fallo, y
   `CheckPendingStatus` permite volver a consultar manualmente un documento en `StatusSent`.
   Pendiente si se quiere contingencia sin intervención humana: un job periódico en background
   que recorra `StatusSent` de todas las empresas (infraestructura nueva, no incluida aquí).
3. **Cifrar el blob del certificado `.p12` en la columna `certificate`** con
   `cryptutil.Encrypt`/`Decrypt`, igual que ya se hace con `certificate_password`, `software_pin`
   y `technical_key`.
4. **Dar visibilidad explícita a los documentos "atascados" en `StatusBuilt`** por descoordinación
   de ambiente rango/empresa (`confirm.go:361-364`): registrar un estado de error dedicado (p.
   ej. `StatusEnvironmentMismatch`) en vez de retornar silenciosamente sin cambiar el estado.
5. **Conectar `erp/internal/payroll` con `cofacture/payroll` + `cofacture/soap`** siguiendo el
   mismo patrón de puertos (`BuilderSignerPort`/`SenderPort`) ya usado en
   `erp/internal/electronic`, si la nómina electrónica DIAN es un requisito de negocio vigente.
6. **Planificar la implementación de eventos RADIAN** (acuse de recibo, recibo del bien,
   aceptación/rechazo) si el negocio requiere que las facturas de venta operen como título
   valor — actualmente no hay ningún punto de partida en el código.
7. **Reemplazar el `PostalZone` fijo `"000000"`** por una validación/captura obligatoria del
   dato real en el módulo `thirdparty`, en vez de un valor de relleno en `confirm.go`.
