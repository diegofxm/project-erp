# Arquitectura Profesional DIAN en Go (UBL21-DIAN + API-DIAN)

> Verificado contra el Anexo Técnico de Factura Electrónica de Venta v1.9 (Resolución DIAN 000165/2023).

## 0. Principio rector

API-DIAN existe **únicamente** para emitir documentos electrónicos válidos ante la DIAN: construir, firmar, numerar, transmitir y rastrear el estado de Invoice / CreditNote / DebitNote. No es un CRM, no es un ERP, no es un sistema de reportes.

Todo lo que no sea estrictamente necesario para que un documento sea legalmente válido ante la DIAN se delega a otros servicios (ver sección 8). Mismo principio que el Ledger de `core-bank`: el núcleo no sabe nada de KYC ni de redes de pago externas — aquí, API-DIAN no sabe nada de CRM ni de catálogos de producto.

El proyecto hermano `api-dian` (Fiber + GORM, con CRUD de Company/Customer/Product) queda **retirado**. Todo se reconstruye aquí, delgado desde el día uno.

---

## 1. Visión General

El sistema se divide en dos proyectos independientes:

### 1.1 UBL21-DIAN (Core / Motor de Facturación)
Responsabilidad única:
- Construir documentos UBL 2.1: Invoice, CreditNote, DebitNote, **AttachedDocument**
- Canonicalización XML
- Firma digital del documento (XAdES-EPES) **y** firma WS-Security del sobre SOAP (son dos firmas distintas, mismo certificado)
- Cálculo CUFE (Invoice) / CUDE (CreditNote, DebitNote)
- Generación del contenido del código QR (obligatorio en la representación gráfica)
- Compresión ZIP
- Comunicación SOAP con DIAN (habilitación y producción)
- Interpretación de respuestas DIAN

No conoce HTTP, bases de datos ni reglas de negocio externas.

---

### 1.2 API-DIAN (Aplicación / Orquestador)
Responsabilidad:
- Exponer API REST
- Gestionar **solo** las entidades imprescindibles para emitir: Issuer (emisor/tenant), NumberingRange (resolución de numeración), Invoice/CreditNote/DebitNote y su estado
- Persistencia y retención legal (5 años) de los documentos emitidos y las respuestas de la DIAN
- Control atómico del consecutivo dentro del rango autorizado (igual invariante que una transferencia: nunca se repite, nunca se salta)
- Orquestar llamadas al motor UBL21-DIAN
- Manejar estados de documentos (DRAFT → SIGNED → SENT → ACCEPTED/REJECTED)

No implementa firma, XML ni lógica DIAN interna. No gestiona clientes, productos ni usuarios — eso vive en otros servicios (sección 8).

---

## 2. Estructura del Proyecto UBL21-DIAN (CORE)

> Esta es la estructura **real** (Fase 1 completa y validada contra la DIAN real, no el plan
> original — diverge en varios puntos, documentado en la sección 9).

```
ubl21dian/
├── domain/                   # tipos puros, sin DB, sin tags db/gorm
│   ├── invoice.go            # Invoice
│   ├── notes.go              # CreditNote/DebitNote (embeben Invoice) + BillingReference/DiscrepancyResponse
│   ├── attached_document.go  # AttachedDocument + ValidationResult + AttachedPartyInfo
│   ├── types.go              # Party, Identification, Address, Tax, Line, PaymentMean, NumberingRange...
│   ├── format.go             # FormatCents (formato de montos, compartido por todo el core)
│   └── time.go               # Bogota (huso horario fijo UTC-5, sin horario de verano)
├── builder/                  # ensambla el XML (etree), no firma
│   ├── invoice_builder.go    # BuildInvoice
│   ├── credit_note.go        # BuildCreditNote
│   ├── debit_note.go         # BuildDebitNote
│   ├── attached_document.go  # BuildInvoiceAttachedDocument (solo Invoice por ahora, ver 9.2)
│   ├── extensions.go         # sts:DianExtensions (InvoiceControl, SoftwareProvider, QR...)
│   ├── party.go               # AccountingSupplierParty/AccountingCustomerParty (compartido)
│   ├── tax.go                  # TaxTotal (compartido)
│   ├── line_items.go           # InvoiceLine/CreditNoteLine/DebitNoteLine (compartido)
│   ├── payment_means.go        # PaymentMeans (compartido)
│   ├── monetary_total.go       # LegalMonetaryTotal/RequestedMonetaryTotal (compartido)
│   ├── notes.go                 # BillingReference/DiscrepancyResponse (solo notas)
│   └── format.go                # formato de montos/cantidades para texto XML
├── xml/
│   └── namespaces.go          # namespaces UBL + DIAN reales (no los de un generador de referencia, ver 9.4)
├── cufe/
│   └── cufe.go                 # CUFE (Invoice) — Anexo 11.2
├── cude/
│   └── cude.go                 # CUDE (CreditNote/DebitNote) — Anexo 11.4
├── securitycode/
│   └── securitycode.go         # SoftwareSecurityCode — Anexo 11.8 (no es CUFE ni CUDE, aplica a los tres documentos)
├── internal/dianhash/
│   └── dianhash.go             # fórmula compartida entre cufe y cude — no es API pública del módulo
├── qr/
│   └── qr.go                   # URL del QR, distinta por ambiente — Anexo 11.7.1
├── signer/
│   ├── certificate.go          # carga de certificado desde .p12 o PEM
│   ├── signer.go                # firma XAdES-EPES del documento (C14N inclusivo)
│   └── xades.go                 # SignedProperties + política de firma fija de la DIAN
├── zip/
│   ├── zip.go                   # Build — comprime documentos para SendBillSync/SendBillAsync
│   └── filename.go              # convención de nombres, secciones 6.5.7/6.5.8
├── soap/
│   ├── client.go                # Client + HTTP
│   ├── envelope.go               # WS-Addressing + WS-Security (ver 9.4, difiere de la política publicada)
│   ├── operations.go             # SendTestSetAsync, GetStatusZip, GetStatus
│   └── types.go                  # DianResponse, UploadDocumentResponse — esquema real del WSDL, no inferido
└── dian/
    └── parser.go                  # interpreta DianResponse y decodifica el ApplicationResponse embebido
```

Notas frente al anexo técnico:
- **AttachedDocument** es el contenedor que se entrega al adquiriente — pero **no** es lo que se envía a la DIAN (eso es el Invoice firmado, dentro de un ZIP). Solo tiene sentido construirlo después de recibir la respuesta de validación (exige `cac:ParentDocumentLineReference`, mínimo una vez).
- **QR** es obligatorio en la representación gráfica; su URL cambia entre habilitación y producción (`catalogo-vpfe-hab` vs `catalogo-vpfe`), confirmado contra facturas reales.
- Se eliminó toda referencia a **CUNE**: ese código pertenece a Nómina Electrónica, un esquema XML distinto (no UBL) con su propio webservice — y, como aclara la sección 9.3, ni siquiera es parte de este proyecto.
- El cliente SOAP distingue explícitamente las operaciones, en particular **SendTestSetAsync** (habilitación con el set de pruebas) — ya probado contra el servidor real de habilitación, no solo contra el anexo.

---

## 3. Flujo del CORE

```
Invoice Model (con Issuer + Party + Items embebidos)
   → Builder
   → XML
   → Canonicalización
   → Firma XAdES-EPES
   → AttachedDocument (firmado)
   → QR
   → ZIP
   → SOAP (firmado WS-Security)
   → DIAN
   → Parser de respuesta → CUFE/CUDE + estado
```

---

## 4. Estructura del Proyecto API-DIAN

```
api-dian/
├── cmd/server/main.go
├── internal/
│   ├── model/            # Issuer, NumberingRange, Invoice, CreditNote, DebitNote, DocumentStatus
│   ├── service/          # orquesta ubl21dian + control atómico de consecutivo
│   ├── repository/       # persistencia de lo anterior, nada más
│   ├── handler/
│   └── routes/
├── ubl21dian/
└── config/
```

`Issuer` aquí es configuración de tenant (NIT, razón social, régimen fiscal, referencia al certificado, Software ID/PIN, resoluciones de numeración) — **no** es un módulo de CRM. `Customer` y los `Items` de una factura **no son entidades propias**: llegan embebidos en el payload de creación de cada documento y se persisten como snapshot dentro del documento emitido (porque eso es lo que la ley exige conservar), sin CRUD ni reglas de negocio propias sobre ellos.

---

## 5. Flujo API-DIAN

```
HTTP → Handler → Service → Repository → UBL21-DIAN → DIAN
```

---

## 6. Endpoints

```
POST /issuers                          # alta de un emisor/tenant (config DIAN)
POST /issuers/{id}/numbering-ranges    # registrar resolución de numeración
POST /invoices                         # payload incluye receptor + items embebidos
POST /invoices/{id}/send
POST /credit-notes
POST /debit-notes
GET  /documents/status/{trackId}
GET  /numbering-ranges/{id}/consumption
GET  /health
```

---

## 7. Regla de oro

- UBL21-DIAN no conoce HTTP ni DB.
- API-DIAN no conoce firma ni XML.
- API-DIAN no es un CRM ni un ERP: no gestiona clientes, productos ni usuarios. Solo recibe lo necesario para emitir un documento válido y delega el resto.

---

## 8. Fuera de alcance (delegado a otros servicios)

Decisión consciente, no descuido — si se necesitan, se integran como servicios externos consumidos vía API, nunca como módulos internos:

| Función | Por qué no vive aquí |
|---|---|
| CRM de Companies/Customers (contactos, direcciones, KYC) | No lo exige la DIAN; el XML solo necesita un snapshot al momento de emitir |
| Catálogo de Productos / Inventario (precios, stock) | Mismo motivo; los items llegan en el payload de la factura |
| Usuarios, roles, multi-tenant auth (JWT propio) | Responsabilidad de un servicio de identidad / gateway externo |
| PDF / representación gráfica | No es parte del esquema XML del anexo técnico; servicio de render aparte si se necesita |
| Notificaciones (email/SMS al receptor) | Servicio de notificaciones externo |
| Reportes / Dashboard / Analítica | Servicio de BI que consume los datos emitidos, no API-DIAN |
| Documento Soporte (CUDS) | Anexo técnico distinto, familia de documento separada — candidato a fase futura |
| Eventos RADIAN (Acuse de recibo, Reclamo, ApplicationResponse) | Solo obligatorio si la factura se negocia como título valor — fase futura explícita |
| Nómina Electrónica (CUNE) | Esquema XML distinto al UBL, webservice distinto — proyecto separado, no este |

---

## 9. Estado actual y hoja de ruta

### 9.1 Fase 1 (motor `ubl21dian`) — completa y validada contra la DIAN real

| Documento DIAN | Construir | Firmar (XAdES) | CUFE/CUDE | Enviado y validado en habilitación real |
|---|---|---|---|---|
| Factura electrónica de venta (01) | ✅ | ✅ | ✅ | ✅ Autorizada (`SETP-990068706`, StatusCode 00) |
| Nota Crédito (91) | ✅ | ✅ | ✅ | ✅ Procesada (referenciando la factura real anterior) |
| Nota Débito (92) | ✅ | ✅ | ✅ | ⚠️ Construido y con golden test; no se ha enviado real todavía |
| AttachedDocument (contenedor para el adquiriente) | ✅ solo Invoice | ✅ (placeholder genérico) | — | ✅ Probado con el `ApplicationResponse` real de la factura autorizada |

El pipeline completo (build → CUFE/CUDE → `SoftwareSecurityCode` → QR → firma XAdES → ZIP → envío SOAP con WS-Security → lectura de respuesta) está probado de punta a punta, no solo contra los ejemplos del anexo.

### 9.2 Pendiente dentro de "Facturación Electrónica" (mismo Anexo 1.9)

| Pendiente | Por qué importa | Prioridad sugerida |
|---|---|---|
| `BuildCreditNoteAttachedDocument` / `BuildDebitNoteAttachedDocument` | Hoy solo Invoice tiene contenedor para entregarle al adquiriente | Alta — hueco pequeño, mismo patrón ya existente |
| Documento Soporte (05, CUDS) | Compras a no obligados a facturar — caso de uso frecuente | Media-alta, según necesidad real |
| Eventos RADIAN (Acuse de recibo, Reclamo) | Solo si la factura se negocia como título valor | Baja, opcional |
| Factura exportación (02) / importación (04) / contingencia (03) | Comercio exterior / caída de los sistemas DIAN | Baja, según necesidad |
| Documento Equivalente Electrónico (tiquete POS) | Ventas al detal de bajo valor | Baja, según necesidad |
| Documentos Equivalentes sectoriales (salud, transporte, energía...) | Solo aplica a esos sectores específicos | Fuera de alcance salvo que aplique |

### 9.3 Nómina Electrónica — no es una fase de este proyecto

Es un sistema DIAN aparte: otra resolución, otro anexo técnico, **otro esquema XML (no es UBL)**, otro webservice. No comparte código con Invoice/CreditNote/DebitNote más allá de la firma X.509 sobre el documento. Si alguna vez se necesita, debe ser un **proyecto independiente** — puede reutilizar `signer` y el patrón de `soap` como referencia de implementación, no como dependencia directa.

### 9.4 Hallazgos reales que vale la pena recordar

Verificados contra el servidor real de la DIAN, no solo contra el anexo técnico (documentados con más detalle en los comentarios del código correspondiente):

- El namespace real de `sts:DianExtensions` es `dian:gov:co:facturaelectronica:Structures-2-1`, no la URL que usan varios generadores de referencia open source.
- `cbc:TaxLevelCode` concatena varias responsabilidades con `;` en un único elemento, no un elemento por código.
- La política de WS-Security publicada en el WSDL (`RequireThumbprintReference`) no es la que acepta el servidor real: hay que usar `BinarySecurityToken` embebido + referencia directa.
- Nunca se puede reformatear (`Indent`) un documento después de firmarlo — invalida la firma porque cambia los bytes ya canonicalizados.
- El Anexo Técnico 1.9 tiene un error de transcripción real en su propio ejemplo de CUDE para Nota Débito (sección 11.4.5) — el hash publicado no corresponde a la cadena de composición que el mismo documento publica.

### 9.5 `ubl21dian` y `api-dian` siguen siendo dos módulos Go independientes

Cada uno con su propio `go.mod` y su propio repo git — `api-dian` consume a `ubl21dian` como
dependencia (`import "github.com/diegofxm/ubl21dian/..."`), nunca al revés. Es el mismo patrón
que usar cualquier paquete externo de Go, salvo que por ahora no está publicado en GitHub.

Mientras los dos se desarrollan en paralelo, `project-ubl/go.work` (no es un módulo Go en sí,
solo el archivo de workspace) le dice al compilador que resuelva `github.com/diegofxm/ubl21dian`
contra la carpeta local `./ubl21dian` en vez de ir a buscarlo a un remoto — así `api-dian` ve
los cambios de `ubl21dian` al instante, sin `git push`, sin tags, sin que `ubl21dian` necesite
siquiera tener un remoto configurado. El día que se quiera congelar una versión estable para
desplegar de verdad, se publica `ubl21dian` en un repo real con un tag (`v0.1.0`...), se
agrega `require github.com/diegofxm/ubl21dian v0.1.0` al `go.mod` de `api-dian`, y se quita (o
se deja de usar) el `go.work` — ahí es donde Go vuelve a resolver la dependencia "de verdad".

### 9.6 Próximo paso

Fase 2: el orquestador `api-dian` (persistencia, numeración atómica, API REST) sobre el motor ya construido. La hoja de ruta de la sección 9.2 se retoma después de tener Invoice/CreditNote/DebitNote funcionando de punta a punta a través del orquestador.
