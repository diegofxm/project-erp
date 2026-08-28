# Plan de pruebas end-to-end — FACTORIA XYZ S.A.S.

## Cómo usar este documento

Empresa ficticia para las pruebas: **FACTORIA XYZ S.A.S.**, distribuidora B2B de insumos de aseo y cafetería para empresas (sector simple, genérico, y que obliga a usar casi todos los módulos: compras a proveedores formales e informales, inventario en más de una bodega, ventas con y sin IVA, cartera, cuentas por pagar, activos fijos). Ya tienes la compañía configurada en la DIAN (habilitación) con tus datos reales — este plan asume eso como punto de partida.

- Cada caso trae: **Dónde** (menú/página exacta), **Qué hacer** (pasos concretos con datos de ejemplo) y **Qué verificar** (la señal de que algo salió mal — ahí es donde vas a encontrar bugs).
- Usa las casillas `- [ ]` para ir marcando en tu editor a medida que avanzas — puedes hacerlo en varias sesiones a lo largo de 2-4 meses simulados.
- Las fechas son sugeridas (simulando un periodo real) — ajústalas si haces las pruebas más rápido o más lento en el calendario real. Lo importante es el ORDEN, no las fechas exactas.
- Cuando algo falle o se vea raro, anótalo con el número de caso (ej. "Caso 14 — bug: ...") y me lo compartes para revisarlo.
- Al final hay una sección de **funcionalidades que hoy NO están disponibles en la interfaz** — no pierdas tiempo buscándolas.

---

## Fase 0 — Configuración inicial (antes de operar, "día 1")

### Caso 1 — Verificar el perfil de la empresa ante la DIAN
**Dónde:** Configuración → Empresa
- [ ] Confirma que el RUT, razón social, responsabilidades fiscales y régimen están correctos.
- [ ] Confirma que el **Ambiente DIAN** está en **Habilitación** (no Producción) mientras hagas estas pruebas.
- [ ] Confirma que el certificado digital está cargado y vigente (fecha de expiración a futuro).
- [ ] Confirma que Software ID y PIN están guardados.

**Verificar:** si algo de esto falta, ningún documento electrónico va a poder confirmarse más adelante — mejor detectarlo aquí.

### Caso 2 — Rangos de numeración DIAN
**Dónde:** Configuración → Empresa → Rangos de numeración
- [ ] Verifica (o importa desde la DIAN con el botón de importar) que existan rangos **activos** en ambiente Habilitación para los 5 tipos de documento: Factura Electrónica (01), Nota Crédito (91), Nota Débito (92), Documento Soporte (05), Nota de Ajuste (95).
- [ ] Anota el prefijo y el consecutivo actual de cada uno — te va a servir para comparar más adelante si algún número "se salta" o queda huérfano.

**Verificar:** que cada rango tenga `current_number` coherente con lo que realmente está autorizado en el portal de habilitación de la DIAN (el problema real que ya diagnosticamos esta sesión).

### Caso 3 — Bodegas
**Dónde:** Inventario → Bodegas
- [ ] Crea **Bodega Principal** (si no existe ya) y **Bodega Norte** (una segunda, para poder probar traslados — se necesitan mínimo 2 bodegas activas).

**Verificar:** que ambas queden en estado activo y que el selector de bodega en Compras/Inventario las liste a ambas.

### Caso 4 — Plan de cuentas y tipos de comprobante
**Dónde:** Contabilidad → Plan de cuentas / Contabilidad → (tipos de comprobante, si el ERP trae un CRUD para ellos)
- [ ] Revisa que existan cuentas para: Inventario, Costo de ventas, Ingresos por ventas, IVA por pagar, IVA descontable, Proveedores, Clientes, Caja/Bancos, Activos fijos.
- [ ] Si falta alguna, créala.

**Verificar:** que los asientos automáticos de venta/compra (más adelante) no fallen por cuenta inexistente.

### Caso 5 — Terceros iniciales
**Dónde:** Clientes / Proveedores
- [ ] Crea **2 clientes**: uno persona jurídica ("Comercializadora ABC S.A.S.", NIT ficticio válido con dígito de verificación correcto) y uno persona natural.
- [ ] Crea **2 proveedores**:
  - "Distribuidora El Roble S.A.S." — proveedor **formal, obligado a facturar** (emite su propia factura electrónica fuera de este sistema; nosotros solo registramos la compra, sin generar Documento Soporte).
  - "Pedro Pérez" — proveedor **persona natural, NO obligado a facturar** (para él SÍ generamos Documento Soporte desde la orden de compra).

**Verificar:** validación del NIT/cédula y dígito de verificación; que ambos tipos de identificación (NIT vs cédula) se guarden bien.

### Caso 6 — Catálogo de productos
**Dónde:** Productos
- [ ] Crea al menos 4 productos:
  1. "Jabón líquido para manos 1L" — IVA 19%, unidad de medida "Unidad".
  2. "Papel higiénico institucional x12" — IVA 19%, unidad de medida "Unidad" o "Paquete".
  3. "Café tostado 500g" — IVA **0%/excluido**, unidad de medida "Unidad" (necesario para volver a probar el caso de mezclar líneas con y sin IVA en una misma factura — el bug FAU04 que corregimos esta sesión).
  4. Un producto/servicio genérico marcado como "Es servicio" (ej. "Flete de entrega") — sin manejo de stock.
- [ ] Define stock mínimo en al menos uno, para poder probar la alerta de stock bajo más adelante.

**Verificar:** que el estándar de clasificación DIAN (schemeID/schemeName) quede bien resuelto para cada producto — repasamos esto a fondo esta sesión, es un buen lugar para que reaparezca un bug similar.

---

## Fase 1 — Mes 1: arranque de operaciones

*(fechas sugeridas: primeros días del mes 1 simulado)*

### Caso 7 — Compra a proveedor formal (sin Documento Soporte)
**Dónde:** Compras → Órdenes de compra → Nueva
- [ ] Proveedor: Distribuidora El Roble. Líneas: 50 jabones + 30 papel higiénico. Bodega destino: Principal.
- [ ] Guardar como borrador → **Confirmar** → **Recibir**.
- [ ] En este caso **NO** generes Documento Soporte (el proveedor factura por su cuenta).

**Verificar:** que al "Recibir" el inventario de Bodega Principal suba exactamente en las cantidades compradas; que se haya generado el asiento contable de la compra; que aparezca en Cuentas por pagar con el saldo correcto.

### Caso 8 — Compra a proveedor informal (con Documento Soporte)
**Dónde:** Compras → Órdenes de compra → Nueva
- [ ] Proveedor: Pedro Pérez. Línea: 20 unidades de café tostado. Bodega destino: Principal.
- [ ] Confirmar → Recibir.
- [ ] Desde la orden ya recibida, usa el selector de rango para **generar el Documento Soporte** → revisa el borrador en Documentos → Documentos Soporte → **Confirmar**.

**Verificar:** que la DIAN lo acepte (`status: accepted`); que el botón vuelva a mostrar "Ver Documento Soporte" y no el selector; que si intentas generar un segundo DS desde la misma orden, el sistema te lo impida con un mensaje claro.

### Caso 9 — Traslado entre bodegas
**Dónde:** Inventario → Movimientos → Nuevo traslado
- [ ] Traslada 15 jabones de Bodega Principal a Bodega Norte.

**Verificar:** que se generen las dos entradas del movimiento (salida en origen, entrada en destino); que el stock total (sumando ambas bodegas) no cambie, solo se redistribuya; que al eliminar un traslado se reviertan **ambas** entradas.

### Caso 10 — Venta directa (sin cotización)
**Dónde:** Ventas → Ventas → Nueva
- [ ] Cliente: Comercializadora ABC. Líneas: 10 jabones + 5 papel higiénico, bodega Principal.
- [ ] Confirmar la venta (descuenta inventario y contabiliza).
- [ ] Generar factura electrónica con el rango correspondiente → confirmar.

**Verificar:** que el inventario baje correctamente; que la factura quede aceptada por la DIAN; que en Cartera aparezca el saldo pendiente si la venta fue a crédito.

### Caso 11 — Cotización → Venta → Factura (flujo completo)
**Dónde:** Ventas → Cotizaciones → Nueva
- [ ] Cliente persona natural. Línea: mezcla jabón (19%) + café (0%) — **a propósito**, para reconfirmar el fix de hoy (FAU04).
- [ ] Enviar cotización → marcar como Aceptada → **Convertir en venta** → Confirmar venta → Generar factura electrónica → Confirmar.

**Verificar:** que la factura quede `accepted` (no rechazada por base imponible); que la unidad de medida, forma de pago y datos del cliente heredados de la cotización se vean igual que si los hubieras cargado directo en Documentos (lo que unificamos hoy).

### Caso 12 — Cuadre de mitad de mes 1
**Dónde:** Inventario → Stock / Contabilidad → Asientos / Ventas → Cartera / Compras → Cuentas por pagar
- [ ] Revisa que el stock por bodega cuadre con lo comprado/vendido/trasladado hasta ahora.
- [ ] Revisa que cada movimiento (compra, venta, traslado) tenga su asiento contable correspondiente.

---

## Fase 2 — Mes 2: ciclo completo + casos especiales

### Caso 13 — Nota crédito (devolución parcial)
**Dónde:** Documentos → Notas Crédito → Nueva (referenciando la factura del Caso 10)
- [ ] El cliente devuelve 2 jabones. Generar y confirmar la nota crédito.

**Verificar:** que el inventario suba de vuelta esas 2 unidades; que el saldo en Cartera de esa factura se reduzca; que el "neto por cobrar" (factura − NC) se calcule bien en el detalle del documento.

### Caso 14 — Nota débito (cobro adicional)
**Dónde:** Documentos → Notas Débito → Nueva (referenciando la misma factura)
- [ ] Cobro adicional de flete no facturado inicialmente.

**Verificar:** que el saldo en Cartera de esa factura AUMENTE en el valor de la nota débito.

### Caso 15 — Nota de ajuste al Documento Soporte
**Dónde:** Documentos → Notas de Ajuste → Nueva (referenciando el DS del Caso 8)
- [ ] Corrige el valor de una línea (ej. el precio real del café era distinto).

**Verificar:** que quede correctamente vinculada al DS original (billing reference/CUFE) y que la DIAN la acepte.

### Caso 16 — Provocar un rechazo real de la DIAN (a propósito)
**Dónde:** Ventas → Venta nueva → Generar factura, o directo en Documentos → Factura
- [ ] Deja el correo del cliente vacío, o usa un NIT que no corresponda a la razón social escrita, y confirma.

**Verificar:** que el mensaje de rechazo mostrado sea claro y específico (no genérico); que el **consecutivo se libere** (revisa en Configuración → Rangos que `current_number` retrocedió); que en la página de la venta/orden vuelva a aparecer el selector de rango con el aviso de rechazo (lo que corregimos hoy) en vez de quedar bloqueada.
- [ ] Corrige el dato que causó el rechazo y **reintenta** desde el mismo selector — debe generar un documento nuevo con un CUFE distinto.

### Caso 17 — Retención en una compra
**Dónde:** Compras → orden recibida → sección de retenciones
- [ ] Aplica un concepto de retención (ej. ReteFuente) sobre la compra del Caso 7.

**Verificar:** que el asiento contable de la retención sea correcto y que el valor neto a pagar al proveedor en Cuentas por pagar refleje la retención.

### Caso 18 — Pago parcial de una factura de venta
**Dónde:** Ventas → Cartera → Registrar pago
- [ ] Sobre la factura del Caso 10, registra un pago parcial (ej. 60% del valor).

**Verificar:** que el saldo pendiente se recalcule correctamente y que quede visible como "parcialmente pagada".

### Caso 19 — Pago total de una cuenta por pagar
**Dónde:** Compras → Cuentas por pagar → Registrar pago
- [ ] Paga completo el saldo de la compra del Caso 7.

**Verificar:** que desaparezca (o quede en 0) de las cuentas por pagar pendientes.

### Caso 20 — Movimiento bancario + conciliación
**Dónde:** Contabilidad → Bancos → Nuevo movimiento / Contabilidad → Conciliación
- [ ] Registra el ingreso del pago del Caso 18 y el egreso del pago del Caso 19 como movimientos bancarios.
- [ ] Concilia ambos.

**Verificar:** que el saldo bancario calculado coincida con los movimientos; que la conciliación marque correctamente lo conciliado vs. pendiente.

### Caso 21 — Activo fijo
**Dónde:** Contabilidad → Activos fijos → Nuevo activo
- [ ] Registra, por ejemplo, "Estantería metálica para Bodega Norte" con su valor y vida útil.

**Verificar:** que se genere el asiento de adquisición y que empiece a calcular depreciación si el módulo lo hace automáticamente.

### Caso 22 — Ajuste manual de inventario
**Dónde:** Inventario → Movimientos → Ajuste
- [ ] Registra una merma de 1 unidad de papel higiénico (dañado).

**Verificar:** que el stock baje y quede el motivo/descripción guardado para auditoría.

### Caso 23 — Cierre del periodo mes 2
**Dónde:** Contabilidad → Periodos
- [ ] Cierra el mes 2.
- [ ] Intenta registrar un asiento manual con fecha dentro del mes ya cerrado.

**Verificar:** que el sistema **bloquee** ese registro con un mensaje claro (no que falle silenciosamente o con un error genérico).

---

## Fase 3 — Mes 3: cartera vencida, presupuesto y cierre

### Caso 24 — Cartera vencida
**Dónde:** Ventas → Ventas → Nueva
- [ ] Crea una venta a crédito con fecha de vencimiento ya pasada respecto a "hoy".

**Verificar:** que en Cartera aparezca marcada como vencida (no solo "pendiente").

### Caso 25 — Presupuesto vs. ejecución
**Dónde:** Contabilidad → Presupuestos → Nuevo presupuesto
- [ ] Define un presupuesto mensual para la cuenta de ventas o de un gasto específico.

**Verificar:** que el comparativo contra lo realmente ejecutado (asientos ya registrados en meses 1-2) se calcule correctamente.

### Caso 26 — Declaración de impuestos
**Dónde:** Contabilidad → Declaraciones
- [ ] Registra una tarifa/declaración de IVA del bimestre correspondiente a los meses probados.

**Verificar:** que los valores base coincidan con el IVA generado/descontable real de las ventas y compras registradas.

### Caso 27 — Certificados
**Dónde:** Contabilidad → Certificados
- [ ] Revisa/genera el certificado de retención aplicado en el Caso 17 para el proveedor.

**Verificar:** que los datos del certificado (tercero, periodo, valores) coincidan con la retención real.

### Caso 28 — Cancelaciones y eliminaciones
**Dónde:** Ventas / Compras / Cotizaciones
- [ ] Crea una cotización nueva y **cancélala** (rechazar).
- [ ] Crea un borrador de venta y **elimínalo** antes de confirmar.
- [ ] Intenta cancelar una venta ya confirmada — verifica qué te permite y qué no.

**Verificar:** que el estado y las reglas de negocio (qué se puede hacer en cada estado) sean consistentes con lo que el formulario permite/oculta.

### Caso 29 — Reapertura de periodo
**Dónde:** Contabilidad → Periodos
- [ ] Reabre el periodo cerrado en el Caso 23.
- [ ] Registra un asiento retroactivo de ajuste.
- [ ] Vuelve a cerrarlo.

**Verificar:** que la reapertura quede registrada en Actividad (bitácora) como una acción excepcional.

### Caso 30 — Clonar y reenviar un documento electrónico
**Dónde:** Documentos → cualquier factura ya confirmada → Clonar
- [ ] Clona una factura ya aceptada, ajusta algo, y confírmala como un documento nuevo.

**Verificar:** que el clon nazca en borrador, sin numeración ni firma propias, y que al confirmarlo tome un consecutivo distinto al original.

### Caso 31 — PDF, XML y envío por correo
**Dónde:** cualquier documento electrónico confirmado
- [ ] Descarga el PDF (formato completo y media página), descarga el XML, y envía el documento por correo al cliente/proveedor.

**Verificar:** que el PDF refleje exactamente los mismos datos que el XML firmado (no una reconstrucción distinta — este fue justamente un punto que revisamos hoy con el CUFE).

### Caso 32 — Bitácora de actividad
**Dónde:** Configuración → Actividad
- [ ] Revisa que las acciones críticas de todo el periodo (confirmaciones, cancelaciones, cierres/reaperturas de periodo, rechazos) queden con texto legible y fecha correcta.

---

## Cuadre final (checklist de cierre del periodo simulado)

- [ ] Total de ventas del periodo (Ventas) = ingresos reconocidos en Contabilidad para esas mismas fechas.
- [ ] Total de compras del periodo = costos/gastos reconocidos en Contabilidad.
- [ ] Inventario valorizado (Inventario → Stock) = saldo de la cuenta de inventario en el plan de cuentas.
- [ ] Cartera pendiente (Ventas → Cartera) = suma real de facturas no pagadas o parcialmente pagadas.
- [ ] Cuentas por pagar pendientes (Compras → Cuentas por pagar) = suma real de compras no pagadas.
- [ ] Cada documento electrónico `accepted` tiene su asiento contable correspondiente — y cada uno `rejected`/`send_error` **no** dejó huella contable ni de inventario colgada.
- [ ] Los consecutivos usados en los rangos de numeración (Configuración → Rangos) coinciden con los documentos realmente aceptados — sin huecos causados por rechazos que no liberaron el número.

---

## Funcionalidades que HOY NO están disponibles en la interfaz (no las busques)

- **Nómina, RRHH, Empleados** — están en el menú pero deshabilitados (`disabled: true`); el backend existe pero no hay pantallas conectadas todavía.
- **Invitar usuarios adicionales a tu empresa / gestión de roles por empresa** — no encontré ningún formulario en la interfaz para esto (existe el modelo de datos y el flujo de aceptar invitación, pero no el botón para generarla desde una empresa normal). Si lo necesitas, dímelo y lo armamos.
- **Recepción de facturas electrónicas de proveedores** — este ERP solo **emite** documentos electrónicos (Factura, NC, ND, DS, NA); no recibe ni valida las facturas que un proveedor formal te envíe a ti. Por eso el Caso 7 (proveedor formal) se registra como una compra interna simple, sin generar ningún documento DIAN de nuestro lado.
- **/admin (Comando)** — es la administración de la plataforma SaaS (planes, facturación de Cofacture a las empresas clientes), no operación de FACTORIA XYZ como empresa — solo entra ahí si quieres probar esa parte por separado.
