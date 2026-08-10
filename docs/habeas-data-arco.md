# Habeas Data — flujo mínimo de cumplimiento (Ley 1581 de 2005)

> **Esto no es un concepto jurídico.** Es una guía operativa mínima para que el equipo tenga un
> procedimiento escrito al que remitirse cuando llegue una solicitud de un titular, alineada con
> el marco general de la Ley 1581 de 2005 y el Decreto 1377 de 2013 de Colombia. Antes de tratarla
> como política de cumplimiento definitiva, que la revise un abogado — en particular la base legal
> exacta para tratar los datos de terceros en el contexto de facturación/contabilidad (que suele
> no requerir autorización explícita bajo el literal de "relación contractual", artículo 10 del
> Decreto 1377/2013), y el cruce con las obligaciones de conservación documental de la DIAN.

Implementado 2026-08-10, punto 19 del plan de acción
(`docs/auditorias/2026-08-09/plan-de-accion.md`). Alcance: `erp/internal/thirdparty` (catálogo de
Clientes/Proveedores). No cubre datos de terceros que puedan aparecer en otros módulos (ver
"Limitaciones" abajo).

## ¿Quién es "titular" bajo esta ley?

**Solo personas naturales.** Un tercero es persona natural cuando `entity_type_code == "2"`
(campo "Tipo de persona" en el formulario de Cliente/Proveedor). Una persona jurídica
(`entity_type_code == "1"`, la mayoría de los NIT) **no es titular** de Habeas Data — es una
empresa, no un individuo. El sistema ya aplica esta distinción automáticamente: el campo de
consentimiento solo aparece en el formulario, y solo tiene efecto, cuando el tercero es persona
natural.

## Qué existe hoy en el sistema

- **Consentimiento**: al crear o editar un cliente/proveedor persona natural, hay un checkbox
  "El titular autorizó el tratamiento de sus datos personales...". Al marcarlo, se guarda
  `habeas_data_consent=true` y se estampa `habeas_data_consent_at` con la fecha (una sola vez —
  ediciones posteriores no mueven la fecha mientras el consentimiento siga activo). Al
  desmarcarlo, ambos campos se limpian.
- **Exportación (derecho de Acceso)**: botón de descarga (ícono ⬇) en la fila del cliente/proveedor
  en `/customers` y `/suppliers`, visible solo para personas naturales. Descarga un `.json` con
  todos los campos que el ERP guarda de ese titular. Equivale a `GET /api/v1/customers/{id}/export`
  o `GET /api/v1/suppliers/{id}/export`.
- **Auditoría**: cada exportación queda registrada en `audit.events` como
  `customer.data_exported` / `supplier.data_exported`, con usuario y fecha — sirve como evidencia
  de que una solicitud de acceso fue atendida en una fecha concreta.

## Procedimiento manual para atender una solicitud ARCO

No hay todavía un formulario público ni un sistema de tickets para que el titular radique su
solicitud — mientras eso no exista, así se atiende manualmente:

1. **Registrar la solicitud** (canal actual: correo o el que use la empresa) con fecha de
   recepción. Hasta que exista un sistema dedicado, llevar este registro en cualquier medio simple
   (hoja de cálculo, correo etiquetado) — lo importante es poder demostrar cuándo se recibió y
   cuándo se respondió.
2. **Verificar identidad** del solicitante antes de actuar (que quien pide los datos sea
   efectivamente el titular o su representante autorizado).
3. **Ejecutar según el derecho solicitado**:

   | Derecho | Cómo se atiende hoy |
   |---|---|
   | **Acceso** | Botón "Exportar datos personales" en `/customers` o `/suppliers` → entregar el `.json` descargado al titular. |
   | **Rectificación** | Editar el registro del tercero (botón lápiz) con el dato corregido y guardar. |
   | **Cancelación** (a veces llamado "derecho al olvido") | Botón eliminar (🗑) — quita el rol (Cliente/Proveedor). Si el tercero no tiene ningún otro rol, la fila se borra por completo. **Limitación importante**: esto no purga los documentos históricos (facturas, notas, órdenes) donde ese tercero ya aparece — ver "Limitaciones" abajo, esto es intencional y probablemente legalmente correcto, no un defecto. |
   | **Oposición** | No hay un mecanismo dedicado distinto a Cancelación hoy. Si el titular se opone a que se le siga contactando (no a que se conserven documentos legales ya emitidos), la acción práctica es quitarle el rol y no volver a usarlo en nuevas transacciones. |

4. **Responder al titular** dentro del plazo legal: **10 días hábiles** para consultas (acceso),
   **15 días hábiles** para reclamos (rectificación/cancelación/oposición), con posibilidad de una
   prórroga de 5 días hábiles adicionales si se le avisa al titular antes de que venza el plazo
   original. Confirmar estos plazos con el asesor legal antes de comunicarlos como definitivos.
5. **Dejar constancia** de la respuesta (qué se hizo, cuándo) en el mismo registro del paso 1. El
   evento de auditoría de exportación (`data_exported`) ya sirve como evidencia automática para el
   caso de Acceso; para Rectificación/Cancelación, la propia fecha de `updated_at`/la ausencia del
   registro después de eliminarlo cumplen el mismo propósito.

## Limitaciones conocidas (deliberadas, no descuidos)

- **La Cancelación no purga documentos históricos.** Un `DELETE` sobre un cliente/proveedor no
  toca `electronic.documents`, `sales.sales`, `purchase.orders` ni ninguna otra tabla que ya
  referencie a ese tercero por snapshot (nombre/identificación quedan congelados en el documento
  tal como se emitió). Esto es intencional por dos razones: (1) la DIAN exige conservar los
  documentos electrónicos y su información por años, así que "olvidar" a alguien borrando facturas
  ya emitidas probablemente violaría esa obligación distinta; (2) el propio Decreto 1377/2013
  reconoce excepciones al derecho de cancelación cuando existe un deber legal de conservación. No
  se implementó una purga en cascada — se documenta aquí para que quede explícito que es una
  decisión, no un hueco no evaluado.
- **Alcance limitado a `thirdparty`.** Si una solicitud de Acceso pide "todo lo que tengan sobre
  mí" en sentido amplio, técnicamente eso también podría incluir los documentos electrónicos,
  ventas u órdenes de compra donde el titular aparece como snapshot — el export de hoy no los
  incluye automáticamente. Si se necesita eso, hoy es un paso manual (buscar por número de
  identificación en Documentos/Ventas/Compras). Automatizarlo queda fuera de este alcance mínimo.
- **Sin formulario público de solicitud ni sistema de tickets.** El procedimiento de arriba es
  100% manual, tal como lo pedía el alcance de este punto ("aunque sea manual al inicio").
