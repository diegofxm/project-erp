# Arquitectura Profesional frontend DIAN en React TS (FRONTEND)

> Este documento es la bitácora del frontend — separado a propósito de
> `docs/api-dian-architecture.md` (decisión explícita del usuario, 2026-06-23): los hallazgos
> sobre el backend van allá; todo lo que tenga que ver con cómo se ve/usa/construye el
> frontend (el explorador tipo Postman en `frontend/static/`, el dashboard improvisado en
> `frontend/static/dashboard/`, y el dashboard definitivo que se construirá después en
> React+TS) va aquí.

## Historial: `devui` + dashboard improvisado (2026-06-22/23)

El usuario pidió pasar de probar con Postman a probar con un frontend real en el navegador,
para simular un usuario real — razonando que eso revela huecos ocultos en la API que Postman/
curl nunca muestran porque nunca necesitan presentar nada, solo enviar y leer un código de
estado. Esto se confirmó de inmediato (ver hallazgos abajo y en `api-dian-architecture.md`
secciones 9.28-9.30).

`frontend/` es un módulo Go **independiente y separado de `api-dian`** (decisión explícita del
usuario: no quería el frontend de pruebas viviendo dentro del módulo del backend) — agregado a
`go.work` pero no a ningún `cmd/` de `api-dian`. Un servidor estático de ~25 líneas
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
`api-dian-architecture.md` sección 4.2), el snapshot que se persiste en el documento SÍ debe
ser un reflejo completo del cliente en ese momento — por motivos históricos, si el cliente se
elimina después, la factura ya emitida no debe perder datos que sí tenía disponibles al
crearla.

**Para el dashboard definitivo**: al seleccionar un cliente guardado, copiar el objeto
completo que devuelve `GET /customers` (todos los campos del `partyDTO`) al construir el
`customer` del borrador — no reconstruirlo a mano desde 4 inputs visibles. El backend
(`api-dian`) ya expone todo lo necesario vía `customerResponse`/`partyDTO`; esto es
puramente un hueco del frontend, no de la API.

### 2. `cac:PaymentMeans` (forma de pago) — el dashboard nunca lo pide ni lo manda

Confirmado contra el Anexo Técnico (`docs/reference/anexo-tecnico-1.9.txt`, FAN01/CAN01/DAN01):
`cac:PaymentMeans` es obligatorio (cardinalidad `1..N`) para Invoice/CreditNote/DebitNote. El
formulario de "Nueva factura" del dashboard improvisado nunca pide forma de pago, así que
nunca se manda — esto causó rechazos reales de la DIAN ("errores en campos mandatorios") en
3+ facturas de prueba del usuario.

**Ya corregido en el backend** (`api-dian`, sección 9.30 de `api-dian-architecture.md`):
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
`api-dian-architecture.md` sección 9.19 — es una cuestión de cómo se presenta en la UI).

## Pendiente

El dashboard definitivo (React + TS) todavía no tiene instrucciones de diseño del usuario —
no asumir su forma todavía. Este documento se irá llenando con esas instrucciones cuando el
usuario las dé, y con cualquier hallazgo nuevo que aparezca mientras se sigue usando el
dashboard improvisado o el explorador tipo Postman mientras tanto.
