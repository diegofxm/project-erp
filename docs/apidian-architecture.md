# Arquitectura Profesional DIAN en Go (COFACTURE + APIDIAN)

> Verificado contra el Anexo Técnico de Factura Electrónica de Venta v1.9 (Resolución DIAN 000165/2023).

## 0. Principio rector

APIDIAN existe **únicamente** para emitir documentos electrónicos válidos ante la DIAN: construir, firmar, numerar, transmitir y rastrear el estado de Invoice / CreditNote / DebitNote. No es un CRM, no es un ERP, no es un sistema de reportes.

Todo lo que no sea estrictamente necesario para que un documento sea legalmente válido ante la DIAN se delega a otros servicios (ver sección 8). Mismo principio que el Ledger de `core-bank`: el núcleo no sabe nada de KYC ni de redes de pago externas — aquí, APIDIAN no sabe nada de CRM ni de catálogos de producto.

El proyecto hermano `apidian` (Fiber + GORM, con CRUD de Company/Customer/Product) queda **retirado**. Todo se reconstruye aquí, delgado desde el día uno.

---

## 1. Visión General

El sistema se divide en dos proyectos independientes:

### 1.1 COFACTURE (Core / Motor de Facturación)
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

### 1.2 APIDIAN (Aplicación / Orquestador)
Responsabilidad:
- Exponer API REST
- Gestionar **solo** las entidades imprescindibles para emitir: Issuer (emisor/tenant), NumberingRange (resolución de numeración), Invoice/CreditNote/DebitNote y su estado
- Persistencia y retención legal (5 años) de los documentos emitidos y las respuestas de la DIAN
- Control atómico del consecutivo dentro del rango autorizado (igual invariante que una transferencia: nunca se repite, nunca se salta)
- Orquestar llamadas al motor COFACTURE
- Manejar estados de documentos (DRAFT → SIGNED → SENT → ACCEPTED/REJECTED)

No implementa firma, XML ni lógica DIAN interna. No gestiona clientes, productos ni usuarios — eso vive en otros servicios (sección 8).

---

## 2. Estructura del Proyecto COFACTURE (CORE)

> Esta es la estructura **real** (Fase 1 completa y validada contra la DIAN real, no el plan
> original — diverge en varios puntos, documentado en la sección 9).

```
cofacture/
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

## 4. Estructura del Proyecto APIDIAN

> Igual que en la sección 2: esto refleja lo que ya existe (Fase 2.1-2.2), no solo el plan
> original. Mismos patrones que `core-bank` (config/logger/database/server), sin `cache` ni
> `telemetry` — no hacen falta todavía aquí.

```
apidian/
├── cmd/server/main.go        # bootstrap: config → logger → database → seed → server
├── internal/
│   ├── config/                # Config + Load() desde variables de entorno
│   ├── logger/                 # Zap, igual patrón que core-bank
│   ├── database/
│   │   ├── database.go          # pgxpool + Migrate() (golang-migrate embebido)
│   │   ├── seed.go               # Seed() — catálogos DIAN desde seed/*.csv, idempotente
│   │   ├── migrations/           # *.sql — solo esquema (DDL), nunca datos de catálogo
│   │   └── seed/                 # *.csv — datos de catálogos, separados del esquema
│   ├── cryptutil/                # Encrypt/Decrypt AES-256-GCM — lo usan issuers y numbering,
│   │                              # vive aparte para que ninguno dependa del otro
│   ├── sqlutil/                    # Placeholders($1..$n) — evita desalinear columnas vs $N
│   │                                # a mano (causó 2 bugs reales en Fase 2.5/2.6)
│   ├── server/                      # http.Server, routes(), /health
│   ├── issuers/                      # emisor/tenant: datos + credenciales DIAN cifradas
│   ├── numbering/                      # rangos de numeración + ClaimNext atómico (UPDATE de una fila)
│   ├── documents/                      # orquesta cofacture (Invoice/CreditNote/DebitNote) —
│   │                                    # el único paquete que lo importa directamente
│   ├── auth/                            # usuarios + login + JWT — multi-empresa (ver 9.32):
│   │   │                                  # un usuario puede tener 0, 1, o varias empresas
│   │   ├── service.go                     # Register (solo usuario) / CreateIssuerForUser /
│   │   │                                    # ListUserIssuers / SelectIssuer / Login
│   │   ├── token.go                        # TokenIssuer: firma/valida JWT (HS256), tenant_id
│   │   │                                     # explícito por sesión, no fijo por usuario
│   │   └── password.go                      # hash/verify con bcrypt
│   ├── customers/                       # catálogo de adquirientes — conveniencia, no la
│   │                                      # fuente de verdad del documento (ver 4.2/9.21)
│   ├── products/                        # catálogo de ítems/servicios — misma lógica
│   └── api/                             # capa HTTP — primera vez que esto se expone por red
│       ├── api.go                        # New()/NewFromServices(), Handler(), rutas
│       ├── dto.go                         # contrato JSON, independiente de domain.* de cofacture
│       ├── handler_auth.go                 # register/login (únicas rutas públicas)
│       ├── handler_issuers.go              # issuers/me + numbering-ranges (protegidas)
│       ├── handler_documents.go             # invoices/credit-notes/debit-notes/documents
│       ├── handler_customers.go              # CRUD de customers
│       ├── handler_products.go                # CRUD de products
│       ├── middleware/                       # RequestID/Logging/Recovery/Auth
│       └── response/                          # WriteJSON/WriteError/classify()
```

Por qué `migrations/*.sql` y `seed/*.csv` están separados: una migración es esquema
versionado (no se vuelve a tocar una vez aplicada); un catálogo DIAN es dato de referencia que
puede necesitar refrescarse (la DIAN ajusta una descripción, o se completa un catálogo
parcial) sin que eso amerite escribir una migración nueva cada vez. Ambos viven dentro de
`internal/database/` porque `//go:embed` solo puede empotrar archivos del propio árbol de
paquete — por eso no es una carpeta `migrations/` suelta en la raíz del proyecto (error
inicial de la Fase 0, corregido en la Fase 2.2).

### 4.1 Mapa de dependencias internas

Por qué el orden de las sub-fases es ese (2.3 y 2.4 no están "sueltas" — son la base que
necesita 2.5, y por eso no dependen entre sí, pueden construirse en cualquier orden):

```
config, logger, cryptutil      (sin dependencias entre sí — cryptutil es la única utilidad
        ↓                       compartida por issuers/numbering, no es una dependencia
     database                   entre ellos, solo un utilitario común de cifrado)
        ↓
   ┌────┴────┐
 issuers   numbering            (ambos usan database+cryptutil; independientes entre sí)
   └────┬────┘
        ├──────────────┬──────────────┐
        ↓              ↓              ↓
       auth        customers       products    (customers/products solo usan database —
        ↓              ↓              ↓         ninguno depende de issuers en Go, solo
        │              ↓              │         issuer_id como FK a nivel de tabla)
        │          documents ←────────┘         (documents usa issuers+numbering+cofacture+
        │              ↓                         customers — CustomerPort, ver 9.23: solo para
        └──────┬───────┘                         validar que un CustomerID opcional pertenezca
               ↓                                 al mismo emisor, nunca para construir el XML)
              api          (usa los seis servicios; expone HTTP — middleware.Auth
                ↓           protege todo excepto /auth/register y /auth/login)
             server          (conecta todo + /health — ya existe desde la Fase 2.1)
```

**Regla de naming**: ninguna tabla ni paquete compartido por varios tipos de documento se
nombra según su primer caso de uso. Es la misma regla que ya aplica en `cofacture` para
`cufe`/`cude`/`securitycode` (sección 2) — se reforzó acá el 2026-06-20 al corregir
`invoice_type_codes` → `dian_document_types` (ver sección 9.6: ese catálogo es compartido por
Invoice/CreditNote/DebitNote y a futuro más documentos, no solo de la factura) y
`document_types` → `identification_types` (para no colisionar conceptualmente con el catálogo
anterior — dos cosas distintas no deberían competir por el mismo nombre genérico).

**Regla de columnas**: `created_at`/`updated_at` van siempre al final de cada tabla — nunca
intercalados entre columnas de negocio, ni siquiera cuando una columna nueva se agrega después
vía `ALTER TABLE` (que por defecto la pondría al final, después de `updated_at`). Reforzada el
2026-06-21 al unificar `000005_issuers_party_fields` dentro de `000003_issuers`: las columnas
de esa migración se insertaron en su lugar lógico (junto a las demás columnas de negocio de
`issuers`), no al final donde las había dejado el `ALTER TABLE` original.

### 4.2 Por qué `issuers` y no `companies`

UBL llama a quien emite el documento el *AccountingSupplierParty* — "Issuer" es la traducción
estándar de ese rol en terminología de facturación electrónica. Se descarta deliberadamente
"companies" porque es el nombre que tenía el proyecto legacy retirado, donde "company"
arrastraba CRM completo (contactos, KYC, etc.) — `issuers` aquí es la configuración mínima de
tenant para *emitir*: NIT, razón social, régimen fiscal, referencia al certificado, Software
ID/PIN, resoluciones de numeración. **No** es un módulo de CRM.

Si en el futuro existe un CRM real con la empresa completa, esa tabla `companies` vive en
**otro sistema/servicio**, no dentro de `apidian` — el mismo patrón que ya usa `core-bank`
(el ledger no tiene KYC completo de sus clientes, asume que vive afuera). `issuers` y un
eventual `companies` externo describirían la misma empresa real desde dos ángulos distintos
("bounded context"): no es duplicación accidental, es separación de responsabilidades a
propósito. `apidian` no debe crecer para absorber lo que le corresponde a un CRM.

`Customer` y los `Items` de una factura **no son entidades propias**: llegan embebidos en el payload de creación de cada documento y se persisten como snapshot dentro del documento emitido (porque eso es lo que la ley exige conservar), sin CRUD ni reglas de negocio propias sobre ellos.

---

## 5. Flujo APIDIAN

```
HTTP → Handler → Service → Repository → COFACTURE → DIAN
```

---

## 6. Endpoints

> Esto ya existe (Fase 2.7-2.25), no es solo el plan. Desde 9.25: `POST /invoices`/
> `credit-notes`/`debit-notes` crean un BORRADOR (sin reclamar número, sin firmar, sin
> enviar) — se puede editar (`PUT .../{id}`) o eliminar (`DELETE /documents/{id}`) libremente
> mientras siga en borrador. Construir, firmar, y enviar (si el ambiente lo permite) pasa todo
> junto en `POST /documents/{id}/confirm`, el único punto donde se "gasta" un número real de
> la DIAN — separar esto de la creación evita quemar un consecutivo por un error de captura.
>
> Todo excepto `/auth/*` y `/health` exige `Authorization: Bearer <token>`. Desde la sección
> 9.32, un usuario puede tener 0, 1, o varias empresas vinculadas — ningún endpoint recibe
> `issuer_id` del cliente, siempre se toma de la empresa ACTIVA en el token
> (`middleware.GetTenantID`), nunca de algo que el cliente pueda elegir directamente. Las tres
> rutas de gestión de empresas son las únicas que no exigen una empresa activa (justamente
> sirven para conseguir una); el resto responde `409` sin ella (`middleware.RequireTenant`).

```
POST /api/v1/auth/register                        # crea SOLO el usuario, sin empresa (público, ver 9.32)
POST /api/v1/auth/login                           # inicia sesión, devuelve el token (público) — autoselecciona
                                                   # la empresa activa solo si hay exactamente una vinculada

POST /api/v1/issuers                              # crear una empresa nueva y vincularla (rol owner), activa de una
GET  /api/v1/issuers                              # listar las empresas a las que el usuario tiene acceso
POST /api/v1/issuers/{id}/select                  # reemitir el token con esa empresa como activa

GET  /api/v1/issuers/me                           # consultar la empresa activa (nunca expone secretos)
PUT  /api/v1/issuers/me                           # completar software/PIN/certificado, parcial (ver 9.25)
POST /api/v1/numbering-ranges                     # registrar rango de la empresa activa
GET  /api/v1/numbering-ranges                     # listar mis rangos (?dian_document_type_code=, sin paginar)
GET  /api/v1/numbering-ranges/{id}                # consultar rango (debe ser del emisor propio; si no, 404)
POST /api/v1/invoices                             # crear borrador (sin reclamar número)
PUT  /api/v1/invoices/{id}                        # reemplazar un borrador
POST /api/v1/credit-notes
PUT  /api/v1/credit-notes/{id}
POST /api/v1/debit-notes
PUT  /api/v1/debit-notes/{id}
POST /api/v1/documents/{id}/confirm               # reclamar número + firmar + enviar si aplica (ver 9.25)
DELETE /api/v1/documents/{id}                     # eliminar un borrador (404 si no es del emisor, 409 si ya no es borrador)
GET  /api/v1/documents                            # listar mis documentos (filtros + ?limit=&offset=)
GET  /api/v1/documents/{id}                       # documento (debe ser del emisor propio; si no, 404)

POST/GET     /api/v1/customers[/{id}]             # catálogo de adquirientes (conveniencia, ver 9.21)
PUT/DELETE   /api/v1/customers/{id}
POST/GET     /api/v1/products[/{id}]              # catálogo de ítems/servicios (conveniencia, ver 9.21)
PUT/DELETE   /api/v1/products/{id}

GET  /health
```

---

## 7. Regla de oro

- COFACTURE no conoce HTTP ni DB.
- APIDIAN no conoce firma ni XML.
- APIDIAN no es un CRM ni un ERP: `customers`/`products` son catálogos de conveniencia, no
  perfiles completos (sin KYC, sin inventario/stock) — y NUNCA son la fuente de verdad de un
  documento ya emitido: cada factura sigue guardando su propio snapshot (`documents.customer`/
  `documents.lines`, JSONB), pass-through puro, exactamente igual que antes de que existieran
  estos catálogos (sección 9.21). Usuarios/auth sí viven aquí — replanteo explícito de
  2026-06-21 (sección 9.17): APIDIAN es el backend completo, no una pieza dentro de un
  ecosistema con un servicio de identidad aparte.

---

## 8. Fuera de alcance (delegado a otros servicios)

Decisión consciente, no descuido — si se necesitan, se integran como servicios externos consumidos vía API, nunca como módulos internos:

| Función | Por qué no vive aquí |
|---|---|
| ~~Catálogo de Customers (autocompletar, sin retipear)~~ | **Ya no aplica — construido en `internal/customers` (Fase 2.11, sección 9.21)**. Sigue NO siendo CRM (sin KYC) ni la fuente de verdad del documento (eso sigue siendo el snapshot pass-through) |
| ~~Catálogo de Products (ítems/servicios reutilizables)~~ | **Ya no aplica — construido en `internal/products` (Fase 2.11, sección 9.21)**. Sin inventario/stock — solo los datos DIAN del ítem para no retipearlos |
| ~~Usuarios, roles, multi-tenant auth~~ | **Ya no aplica — construido en `internal/auth` (Fase 2.9)**. Replanteo de 2026-06-21: APIDIAN es el backend completo, no hay servicio de identidad externo en esta topología |
| PDF / representación gráfica | No es parte del esquema XML del anexo técnico; servicio de render aparte si se necesita |
| Notificaciones (email/SMS al receptor) | Servicio de notificaciones externo |
| ~~Listados de documentos/rangos~~ | **Ya no aplica — `GET /numbering-ranges` y `GET /documents` construidos (Fase 2.9, sección 9.19)**. Esto es CRUD básico del propio orquestador, no analítica — no era delegable a otro servicio |
| Reportes / Dashboard / Analítica (agregaciones, gráficas) | Sigue siendo trabajo de un servicio de BI que consume los datos emitidos — los listados de arriba son consulta simple, no agregación |
| Documento Soporte (CUDS) | Anexo técnico distinto, familia de documento separada — candidato a fase futura |
| Eventos RADIAN (Acuse de recibo, Reclamo, ApplicationResponse) | Solo obligatorio si la factura se negocia como título valor — fase futura explícita |
| Nómina Electrónica (CUNE) | Esquema XML distinto al UBL, webservice distinto — proyecto separado, no este |

---

## 9. Estado actual y hoja de ruta

### 9.1 Fase 1 (motor `cofacture`) — completa y validada contra la DIAN real

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

### 9.5 `cofacture` y `apidian` siguen siendo dos módulos Go independientes

Cada uno con su propio `go.mod` y su propio repo git — `apidian` consume a `cofacture` como
dependencia (`import "github.com/diegofxm/cofacture/..."`), nunca al revés. Es el mismo patrón
que usar cualquier paquete externo de Go, salvo que por ahora no está publicado en GitHub.

Mientras los dos se desarrollan en paralelo, `project-ubl/go.work` (no es un módulo Go en sí,
solo el archivo de workspace) le dice al compilador que resuelva `github.com/diegofxm/cofacture`
contra la carpeta local `./cofacture` en vez de ir a buscarlo a un remoto — así `apidian` ve
los cambios de `cofacture` al instante, sin `git push`, sin tags, sin que `cofacture` necesite
siquiera tener un remoto configurado. El día que se quiera congelar una versión estable para
desplegar de verdad, se publica `cofacture` en un repo real con un tag (`v0.1.0`...), se
agrega `require github.com/diegofxm/cofacture v0.1.0` al `go.mod` de `apidian`, y se quita (o
se deja de usar) el `go.work` — ahí es donde Go vuelve a resolver la dependencia "de verdad".

### 9.6 Fase 2 (`apidian`) — en marcha

Framework HTTP: `net/http` nativo (ServeMux de Go 1.22+), no Fiber — decidido el 2026-06-20
para mantener consistencia con `core-bank`; el cuello de botella real de este sistema es la
DIAN/la base de datos, no el ruteo HTTP.

| Sub-fase | Contenido | Estado |
|---|---|---|
| 2.1 | Bootstrap: config/logger/database/server, `/health` | ✅ Verificado contra Postgres real |
| 2.2 | Esquema + seed de catálogos DIAN (8 catálogos, ver 9.4) | ✅ Verificado: seed idempotente, conteos estables tras re-ejecutar |
| 2.3 | `internal/issuers` — alta de emisor/tenant, credenciales cifradas | ✅ Verificado contra Postgres real (cifrado confirmado en crudo + roundtrip) |
| 2.4 | `internal/numbering` — claim atómico de consecutivo | ✅ Verificado: 300 reclamos concurrentes reales contra Postgres, exactamente {1..300} sin duplicados ni huecos |
| 2.5 | `internal/documents` — orquestar `cofacture` para Invoice | ✅ Verificado: pipeline completo contra la DIAN real (certificado y credenciales reales), ver 9.10 |
| 2.6 | Extender 2.5 a CreditNote/DebitNote | ✅ Verificado localmente contra Postgres real (sin red DIAN, ver 9.11) |
| 2.7 | `internal/api` — handlers/routes/middleware | ✅ Verificado con servidor real + curl (ver 9.12) |
| 2.8 | Prueba real end-to-end vía HTTP completo (`SendBillSync`) | ✅ Verificado con servidor real + curl, ver 9.16 |
| 2.9 | `internal/auth` — usuarios, login, JWT, aislamiento entre tenants | ✅ Verificado con servidor real + curl, ver 9.17 |
| 2.10 | Listados (`GET /numbering-ranges`, `GET /documents`) | ✅ Verificado con servidor real + curl, ver 9.19 |
| 2.11 | `internal/customers`/`internal/products` — catálogos de conveniencia | ✅ Verificado con servidor real + curl, ver 9.21 |

**Catálogos cargados en 2.2** (`internal/database/seed/*.csv`, idempotente vía `ON CONFLICT`):
currencies, departments, identification_types (códigos numéricos oficiales DIAN: 13 cédula,
31 NIT, etc. — no abreviaturas como "CC"/"NIT", corregido en la Fase 2.8 al fallar un envío
real con esos valores), municipalities, payment_methods, tax_types, unit_measures,
dian_document_types (solo 01/91/92 — lo que `cofacture` ya soporta; ver 4.1 sobre por qué no
se llama `invoice_type_codes`).

**Huecos conocidos de datos, no de arquitectura** — pendientes de una fuente oficial antes de
completarse:
- `departments`/`municipalities` están incompletos frente al catálogo real DANE/DIVIPOLA (24
  de 33 departamentos, 10 de ~1.102 municipios — solo lo que ya traía el proyecto legacy).
- `tax_level_codes`, `credit_note_concepts`, `debit_note_concepts`, `countries`,
  `type_organizations`, `type_regimes` no están cargados: el Anexo Técnico 1.9 remite esas
  tablas a la "Caja de Herramientas Factura Electrónica" (un `.xlsx` de la DIAN que no está en
  este repositorio, secciones 13.2.7.4/13.2.7.5) — no se inventaron códigos de cumplimiento
  tributario sin esa fuente.

### 9.7 Fase 2.3 (`internal/issuers`) — completa

Mismo patrón que `core-bank/internal/customers` (model/errors/repository/postgres/memory/
service, con `Service` validando antes de delegar al `Repository`). Diferencia clave: las
credenciales DIAN (`software_pin`, `certificate`, `certificate_password`) se cifran con
AES-256-GCM (`internal/issuers/secrets.go`) antes de tocar Postgres — la clave sale de
`ISSUER_SECRETS_KEY` (variable de entorno, 32 bytes en base64, `openssl rand -base64 32`),
nunca de la base de datos. `SoftwareID` no se cifra: es un identificador de registro ante la
DIAN, no una contraseña.

Verificado contra la base de datos real (no solo con el repositorio en memoria de los tests):
se registró un emisor de prueba, se confirmó por consulta directa que `software_pin` y
`certificate` NO son legibles en texto plano en las columnas crudas, se releyó a través del
servicio y los tres secretos descifraron exactamente igual al original, y se borró el
registro de prueba al terminar.

`technical_key` (la clave técnica de CUFE) **no** vive en `issuers`: es propia de cada
resolución/rango de numeración, no del emisor — se agrega en la Fase 2.4.

### 9.8 Fase 2.4 (`internal/numbering`) — completa

`numbering_ranges` es una sola tabla genérica (no una por tipo de documento), con
`dian_document_type_code` distinguiendo Invoice/CreditNote/DebitNote/futuro — misma regla de
naming de la sección 4.1. `range_to` es `NULL` cuando el tipo de documento no tiene tope
impuesto por una resolución DIAN (hoy todos lo tienen seteado, pero el esquema ya soporta el
caso sin tope sin necesitar otra migración).

`ClaimNext` reclama el siguiente consecutivo con un único `UPDATE ... RETURNING` — Postgres
serializa las escrituras concurrentes sobre la misma fila, así que no hace falta
`SELECT ... FOR UPDATE` explícito. Extraje el cifrado AES-256-GCM de `issuers` a
`internal/cryptutil` en este paso, porque `numbering` también necesita cifrar un secreto
(`technical_key`) y duplicar código de cifrado es exactamente el tipo de cosa que no se debe
duplicar.

**Verificado contra Postgres real, no solo el repositorio en memoria de los tests**: 300
goroutines reclamando al mismo tiempo sobre el mismo rango (conexiones reales del pool, no
simuladas) devolvieron exactamente `{1..300}`, sin duplicados ni huecos, 0 errores; el
siguiente reclamo tras agotar el rango falló con `ErrRangeExhausted` como se esperaba.

`technical_key` vive aquí, no en `issuers` — es propia de cada resolución, y solo la usa
`cufe.Compute` (Invoice); CreditNote/DebitNote usan `SoftwarePIN` de `issuers` vía
`cude.Compute`, no esto.

### 9.10 Fase 2.5 (`internal/documents`) — completa

El primer paquete de `apidian` que importa `cofacture` directamente. `Service.IssueInvoice`
reproduce exactamente el pipeline de `cofacture/soap/realsend_test.go` (Fase 1.7), pero
orquestado: carga el `Issuer` → carga y reclama el `NumberingRange` (`ClaimNext`) → construye
`domain.Invoice` → `cufe.Compute` → `securitycode.Compute` → `qr.URL` →
`builder.BuildInvoice` → `signer.Sign` → `zip.Build` → `soap.SendTestSetAsync` →
`dian.Interpret` → persiste. Usa el patrón de "ports" angostos (`IssuerPort`/
`NumberingPort`, mismo patrón que `core-bank/internal/transfers/ports.go`) en vez de depender
de los `*Service` concretos, para poder probarse con fakes sin necesitar Postgres real.

**Extensión a `issuers` descubierta en este paso**: faltaban campos para construir un
`domain.Party` completo del emisor (`EntityTypeCode`, `TaxSchemeCode`/`TaxSchemeName`,
`LiabilityCodes`, `MerchantRegistrationNumber`), con defaults confirmados contra la factura
real de la Fase 1.7 (`EntityTypeCode "1"`, `TaxSchemeCode "ZZ"`). Se agregaron originalmente
en una migración aparte (`000005_issuers_party_fields`) y se unificaron dentro de
`000003_issuers` el 2026-06-21, sin datos reales en juego todavía — ver sección 4.1.

`documents` es UNA tabla genérica (no una por tipo de documento), con `customer`/`lines`/
`payment_means` como snapshots JSONB pass-through — mismo principio que el resto del
proyecto. `Totals` y `HeaderTaxes` se calculan automáticamente a partir de `Lines` (no se le
pide al llamador que los calcule y posiblemente los deje inconsistentes).

**Limitación ya resuelta (ver sección 9.14)**: en su momento `cofacture/soap` solo exponía
`SendTestSetAsync` — `SendBillSync`/`SendBillAsync` se agregaron después, y resultaron ser la
forma de seguir probando contra habilitación incluso con el set de pruebas ya cerrado.
`IssueInvoice` por ahora solo intenta enviar si el emisor está en habilitación y el rango
tiene `TestSetID` (vía `SendTestSetAsync`) — integrar `SendBillSync` al orquestador queda
pendiente, ver 9.14.

**Verificado contra la DIAN real** (certificado, credenciales y rango de numeración reales de
`docs/reference/`, no simulados): se registró un emisor real, un rango real, y se emitió una
factura completa — construida, firmada (XML de 14.206 bytes con `<ds:Signature>` real),
enviada por SOAP, y se recibió un `ZipKey` real de la DIAN. El sondeo (`GetStatusZip`) devolvió
`StatusCode "2"`, `IsValid false`, con la descripción real:
*"Set de prueba con identificador ... se encuentra Aceptado."* — es decir, el set de pruebas
de habilitación de la Fase 1.7 ya está cerrado (esperado, no es un defecto de este código). Al
construir esta verificación se encontró y corrigió un bug real: las dos primeras versiones de
`Create` (en `issuers` y en `documents`) tenían menos placeholders SQL (`$1..$N`) que columnas
— typeo que ningún test con repositorio en memoria podía detectar, solo Postgres real. También
se descubrió que `dian.Result.StatusDescription` (el texto humano útil) se estaba perdiendo —
solo se persistía `StatusMessage` (vacío en la respuesta real) — corregido agregando
`dian_status_description` como columna propia.

### 9.11 Fase 2.6 (CreditNote/DebitNote en `internal/documents`) — completa

`IssueCreditNote`/`IssueDebitNote` reutilizan exactamente el mismo pipeline que `IssueInvoice`
(refactorizado en `preparedIssuance`/`buildNoteBase`/`finalizeAndSend`, compartido por los
tres) — la única diferencia real es `cude.Compute` en vez de `cufe.Compute` (usa
`SoftwarePIN`, no la clave técnica del rango) y `builder.BuildCreditNote`/`BuildDebitNote` en
vez de `BuildInvoice`. `documents` sigue siendo una sola tabla: `billing_reference` y
`discrepancy_response` (JSONB, nulos en Invoice) y `note_type_code` (solo CreditNote —
DebitNote no tiene ese campo en `cofacture`) son las únicas columnas nuevas.

**Decisión explícita del usuario**: verificación 100% local para esta fase, sin tocar la red
de la DIAN — el set de pruebas de habilitación de la Fase 1.7/1.9 ya está cerrado
("Aceptado"), y el portal ya ofrece el paso a producción pero el usuario decidió no activarlo
todavía. `IssueInvoice`/`IssueCreditNote`/`IssueDebitNote` se probaron con un emisor en
ambiente "producción" sin `TestSetID` — eso hace que `finalizeAndSend` construya, firme y
persista, pero nunca intente la rama de envío SOAP (mismo código de producción real, pero sin
ejecutar la parte que requeriría red). Verificado contra Postgres real (no solo el repo en
memoria): se crearon Invoice + CreditNote + DebitNote reales en la base, se releyeron desde
Postgres (no desde el objeto en memoria), y `billing_reference`/`discrepancy_response`/
`note_type_code` confirmaron el roundtrip JSONB correcto para los tres tipos.

**2 bugs reales más encontrados por esa verificación** (ningún test con repo en memoria los
detecta):
- `issuers.PostgresRepository.Create` insertaba `liability_codes` (columna `NOT NULL`) como
  `NULL` cuando el slice de Go era `nil` — el `DEFAULT '{}'` de la columna solo aplica si la
  columna se omite del INSERT, no si se manda `NULL` explícito.
- Se extrajo `internal/sqlutil.Placeholders(n)` — genera `$1,$2,...,$n` a partir de
  `len(args)`, así el conteo de placeholders nunca puede desalinearse de las columnas otra vez
  (la causa raíz de los 2 bugs de Fase 2.5). Tanto `issuers` como `documents` ya lo usan.

### 9.12 Fase 2.7 (`internal/api`) — completa

Mismo patrón que `core-bank/internal/api`: middleware (`RequestID` → `Logging` → `Recovery`,
de afuera hacia adentro), `response.WriteJSON`/`WriteError`/`classify()` mapeando los errores
de dominio de `issuers`/`numbering`/`documents` a códigos HTTP, y un único `API` struct que
agrupa los tres servicios. `internal/api/dto.go` define el contrato JSON **independiente**
de los tipos de `domain.*` de `cofacture` (que no tienen tags JSON y pueden cambiar libremente
— solo `internal/documents` debe conocer `cofacture` directamente, ver sección 4.1).

Decisiones de seguridad explícitas en los DTOs de respuesta: `issuerResponse` nunca incluye
`Certificate`/`SoftwarePIN`/`CertificatePassword` (ni cifrados); `numberingRangeResponse`
nunca incluye `TechnicalKey` — una vez guardados, la API no los vuelve a exponer.

No hay un `POST /invoices/{id}/send` separado del `POST /invoices` original (sí estaba en el
plan original, sección 6) — se eliminó porque ya no tenía un caso de uso real: `IssueInvoice`
ya construye, firma y envía (si aplica) en una sola llamada de servicio; separar "crear" de
"enviar" en la API hubiera sido una distinción sin contenido real detrás.

**Verificado con el servidor real, no solo `httptest`**: se levantó `cmd/server` contra
Postgres real, y con `curl` se creó un emisor (certificado autofirmado de prueba, igual que en
las fases anteriores), un rango de numeración, y se emitió una factura completa — la
respuesta trajo un CUFE real, una URL de QR real, y el XML firmado completo con
`<ds:Signature>`, certificado embebido y `SignedProperties` de XAdES. Se confirmó que la
respuesta del emisor NO contiene `certificate_base64` ni `software_pin`. Registros de prueba
eliminados al terminar.

### 9.14 `SendBillSync`/`SendBillAsync` agregados a `cofacture` — habilitación sigue disponible

Pregunta que se resolvió el 2026-06-21: una vez el set de pruebas oficial de la Fase 1.7/1.9
quedó "Aceptado" (cerrado, `SendTestSetAsync` ya no acepta más envíos para ese `TestSetID`),
¿la DIAN sigue aceptando envíos contra habilitación con las operaciones normales de envío
(`SendBillSync`/`SendBillAsync`), o bloquea todo envío adicional una vez completada la
certificación?

**Resultado: sigue disponible.** Se implementaron ambas operaciones en `cofacture/soap`
(`operations.go`), confirmadas contra el WSDL real (`docs/reference/wsdl/`):

- **`SendBillSync(fileName, content)`** — un solo documento, síncrono, devuelve `*DianResponse`
  de inmediato (mismo tipo que `GetStatus`/`GetStatusZip`, no `UploadDocumentResponse`). No
  lleva `testSetId`.
- **`SendBillAsync(fileName, content)`** — uno o varios documentos, asíncrono, devuelve
  `*UploadDocumentResponse` (`ZipKey`), el resultado real se consulta después con
  `GetStatusZip` — mismo patrón que `SendTestSetAsync`, también sin `testSetId`.

**Verificado contra la DIAN real** (`soap/realsend_sync_test.go`, `TestSendBillSync_Real`): un
primer intento falló por un error propio de la prueba (faltaban `StartDate`/`EndDate` del
rango de numeración — la DIAN lo señaló con precisión: reglas `FAB07a`/`FAB08a`/`ZB01`). Tras
corregirlo, la DIAN **autorizó la factura de verdad**: `StatusCode "00"`,
`"Procesado Correctamente."`, `"La Factura electrónica SETP990059896, ha sido autorizada."`
— con el set de pruebas oficial ya cerrado desde hace varias fases. Esto confirma que
habilitación seguirá disponible para pruebas de regresión durante el resto del proyecto, sin
necesidad de pedir un set de pruebas nuevo ni pasar a producción.

Queda una notificación no bloqueante (`FAJ43b`: el nombre informado no coincide exactamente
con el registrado en el RUT para ese NIT) — es solo "Notificación", no "Rechazo", no afectó el
`StatusCode 00`.

### 9.15 `SendBillSync` integrado a `internal/documents.Service` — primer envío real a través
del orquestador completo

`finalizeAndSend` (`internal/documents/service.go`) ahora enruta así:

- `iss.Environment != Habilitación` **o** `nr.Environment != Habilitación` → solo construye y
  firma, nunca envía (doble candado a propósito — antes de `SendBillSync` esto dependía solo
  de `TestSetID`, pero ahora habilitación envía de verdad incluso sin él, así que hacía falta
  un segundo seguro contra un envío accidental a producción).
- Habilitación + `TestSetID` presente → `sendAndUpdate` (camino viejo, `SendTestSetAsync`,
  asíncrono, sondea `GetStatusZip`).
- Habilitación + sin `TestSetID` → `sendSyncAndUpdate` (nuevo, `SendBillSync`, síncrono, la
  respuesta final llega en la misma llamada, sin `ZipKey` ni sondeo).

Las pruebas unitarias/`httptest` que antes usaban un emisor en habilitación sin `TestSetID`
para probar la ruta "no se envía" tuvieron que migrarse a `Environment: Producción` — ese es
ahora el único ambiente que nunca intenta red real (ver `internal/documents/service_test.go`,
`internal/api/api_test.go`).

**Verificado contra la DIAN real a través del orquestador completo** (no solo `cofacture`
directo) — `documents.Service.IssueInvoice` con un emisor/rango reales, sin `TestSetID`: el
primer intento fue rechazado (`StatusCode 99`, "errores en campos mandatorios") y reveló **dos
bugs reales, nunca antes detectados** porque ningún test previo había llegado a un envío real
a través de la API:

1. **`partyFromIssuer` nunca llenaba `Address.CityName`/`StateName`** — solo `CityCode`/
   `StateCode`. La DIAN exige el nombre, no solo el código. Fix: `issuers.Issuer` ahora tiene
   `DepartmentName`/`MunicipalityName`, poblados con un JOIN contra `departments`/
   `municipalities` en `GetByID`/`GetByNIT` (`internal/issuers/postgres.go`,
   `issuerSelectWithNames`) — deliberadamente fuera de `Create()`, para no duplicar el dato
   del catálogo.

2. **El catálogo `identification_types` (sembrado en la Fase 2.2) tenía los códigos
   equivocados** — abreviaturas legibles ("CC", "NIT", "CE"...) en vez de los códigos
   numéricos oficiales de la DIAN ("13", "31", "22"...) que `cbc:CompanyID.@schemeName` /
   `sts:ProviderID.@schemeName` esperan literalmente (`cofacture/builder/party.go`,
   `extensions.go`). Confirmado contra la factura real ya autorizada
   (`soap/realsend_sync_test.go`, que usa `"13"`/`"31"` directamente). Fix: CSV/migración
   corregidos a los 11 códigos numéricos reales (`11,12,13,21,22,31,41,42,47,50,91`); catálogo
   real reseembrado, el único emisor real existente (creado por Postman) migrado de `"NIT"` a
   `"31"`.

3. **Bug adicional, mismo origen** (rechazo `FAB23`+`FAB22b`): `softwareProviderFromIssuer`
   reutilizaba `iss.IdentificationTypeCode` para el proveedor de software — pero la DIAN
   registra a todo proveedor de software/facturador electrónico por NIT (código `"31"`)
   **sin importar el tipo de identificación personal del emisor** (un emisor persona natural
   se identifica como `"13"` en `Supplier.Identification`, pero siempre `"31"` en
   `SoftwareProvider.ProviderIdentification` — mismo número, distinto rol). Fix: `"31"`
   ahora fijo en `softwareProviderFromIssuer`, no heredado del emisor.

Tras los tres fixes: **`StatusCode "00"`, "Procesado Correctamente.", "La Factura electrónica
SETP990000000, ha sido autorizada."** — primera factura real autorizada a través de la cadena
completa `documents.Service` → `cofacture` → DIAN, sin `TestSetID`. Queda únicamente la
notificación no bloqueante `FAJ43b` (nombre no coincide con el RUT), ya documentada en 9.14.

### 9.16 Fase 2.8 verificada vía HTTP real (curl, no solo `documents.Service` directo)

Se repitió la verificación de 9.15 pasando por el servidor HTTP real (mismo patrón que la
Fase 2.7): registro → login → rechazo sin token (401) → `GET /issuers/me` con token (200) →
crear rango → emitir factura (firmada) → segundo emisor → acceso cruzado a documento ajeno
(404) → uso cruzado de rango ajeno (422) → acceso al propio documento (200). Los 10 pasos
pasaron contra Postgres real, sin tocar la red de la DIAN (mismo criterio de fases anteriores
— emisor en producción, sin `TestSetID`).

### 9.17 Replanteo de alcance: APIDIAN como backend completo + `internal/auth` propio

El usuario corrigió (2026-06-21) la aplicación estricta de la filosofía de `core-bank` a este
proyecto: `core-bank` es minimalista (sin "listar todo", sin auth, sin CRM completo) porque
asume que existen otros servicios alrededor para esas responsabilidades. `apidian` no tiene
ese ecosistema — **es el backend completo**, consumido directo por un frontend web
administrativo / app móvil, sin API puente intermedia. Esto invierte la lógica de la sección 8
para Auth/Usuarios (ya no es "responsabilidad de otro servicio" — se construye aquí), pero NO
cambia nada para Documento Soporte/RADIAN/Nómina (siguen diferidos por ser otra familia de
documento DIAN, razón no relacionada con la topología) ni para PDF/Notificaciones (razonable
seguir externalizándolos, no son el corazón del negocio).

Decisiones explícitas del usuario: autenticación propia (no Auth0/Clerk/proveedor externo);
"un usuario administra exactamente un emisor" (no hay tabla intermedia `user_issuers` para
multi-emisor por usuario — se agregaría después si hace falta, sin romper lo existente).

**`internal/auth` implementado y verificado real** (Fase 2.9):

- `users`: `issuer_id` (NOT NULL, fijo), `email` (UNIQUE global), `password_hash` (bcrypt,
  irreversible — a diferencia de `software_pin`/`certificate` de `issuers`, que sí se
  descifran para usarse con `cofacture`, una contraseña de login nunca necesita recuperarse).
- `POST /api/v1/auth/register` crea el emisor Y su primer usuario admin en una sola llamada
  (`auth.Service.Register` usa `IssuerPort.RegisterIssuer`, mismo patrón de "ports" angostos
  que `documents`). Valida que el correo esté libre ANTES de crear el emisor — evita un
  emisor "huérfano" sin usuario si el correo ya existía (verificado con un test que cuenta
  llamadas a `RegisterIssuer`: nunca se llama dos veces para el mismo correo duplicado).
- `POST /api/v1/auth/login` valida con bcrypt y devuelve un JWT (HS256, `AUTH_JWT_SECRET`,
  24h, sin refresh token todavía — proporcional al tamaño actual del proyecto). Las claims
  usan `user_id`/`tenant_id` propios, no `sub`/`iss` estándar de JWT, para no confundir
  "quién firmó el token" (apidian) con "qué emisor DIAN administra este usuario".
- `middleware.Auth` exige `Authorization: Bearer <token>` en todo excepto `/auth/*` y
  `/health`, e inyecta `UserID`/`TenantID` (= emisor DIAN) en el contexto.
- **Ningún endpoint vuelve a recibir `issuer_id` del cliente** — body, path y todo lo demás
  lo toman siempre de `middleware.GetTenantID(ctx)`. `GET /issuers/{id}` se simplificó a
  `GET /issuers/me` (ya no hace falta el path param: solo existe "el emisor propio").
  `POST /issuers/{id}/numbering-ranges` se simplificó a `POST /numbering-ranges`.
- **Aislamiento entre tenants verificado en dos capas**, no solo una:
  1. *Servicio* (`documents.Service.prepare`): un bug real preexistente — `nr.IssuerID` nunca
     se comparaba contra el `issuerID` de la petición — permitía emitir con el rango de
     numeración de OTRO emisor (drenando su consecutivo DIAN). Corregido con
     `ErrNumberingRangeIssuerMismatch` (422), con test (`TestIssueInvoice_
     NumberingRangeIssuerMismatch`) y verificado real vía curl (paso 9 de la secuencia
     anterior).
  2. *HTTP* (`handleGetNumberingRange`/`handleGetDocument`): si el recurso existe pero es de
     otro emisor, se responde el mismo 404 que si no existiera — nunca confirmarle a un
     usuario que el ID que probó existe pero es ajeno.
- `documents`/`numbering` no se tocaron en su lógica de negocio — solo `documents.Service.
  prepare` ganó el chequeo de pertenencia, que es un invariante de dominio independiente de
  auth (incluso un solo llamador confiable podría pasar IDs inconsistentes por error).

**Limpieza de esquema en el mismo paso** (sin datos reales en juego — ver
[[feedback-sql-schema-conventions]] en la memoria del proyecto): `000005_issuers_party_fields`
(un `ALTER TABLE` correctivo) se unificó dentro de `000003_issuers` (el `CREATE TABLE`
original), reordenando columnas para que `created_at`/`updated_at` queden al final — regla
ahora explícita en la sección 4.1. Las migraciones siguientes se renumeraron (`000006_documents`
→ `000005_documents`, `000007_users` → `000006_users`) para no dejar un hueco. La base de
datos real de desarrollo se reseteó (`DROP SCHEMA public CASCADE` + re-migrar + re-sembrar) y
se verificó limpio — seguro solo porque no había datos reales, confirmado por el usuario antes
de pedirlo.

### 9.18 Decisión de alcance: Listados sí, Customers/Products diferidos

Revisión pedida explícitamente por el usuario (sección 8): de las tres piezas pendientes
(Listados, Customers, Products), solo **Listados se construye ya** — es CRUD básico de
recursos que el orquestador ya tiene (rangos, documentos), no una funcionalidad nueva. Sin
listado, la única forma de ver un rango/documento es conocer su UUID exacto, lo que en la
práctica hace inutilizable la API para un admin real.

**Customers/Products se diferen, con criterio explícito de cuándo retomarlos**: ninguno de
los dos es necesario para que el orquestador emita documentos válidos (la DIAN exige un
snapshot al momento de emitir, no un catálogo vivo — Customer/Lines siguen llegando
pass-through, ver sección 4.2). Construirlos ahora significaría diseñar su forma (¿un cliente
es por emisor o compartido? ¿cómo versionar cuando cambian sus datos?) sin un frontend real
que valide ese diseño. Se retoman cuando la pantalla de "crear factura" del frontend necesite
de verdad un autocompletar — la forma de la tabla la dicta esa pantalla, no una suposición
hecha hoy.

### 9.19 Listados (`GET /numbering-ranges`, `GET /documents`) — completo y verificado real

- **`GET /api/v1/numbering-ranges`** — rangos del emisor autenticado, filtro opcional
  `?dian_document_type_code=`. **Sin paginación a propósito**: el volumen esperado por emisor
  es bajo (resoluciones de numeración, no documentos emitidos) — agregarla sería complejidad
  sin necesidad real todavía.
- **`GET /api/v1/documents`** — documentos del emisor autenticado, filtros opcionales
  `dian_document_type_code`, `status`, `from`/`to` (sobre `issue_date`, formato `YYYY-MM-DD`,
  mismo estilo de query string manual que usa `core-bank` para sus filtros por fecha — no
  existía un helper genérico de query params que reusar, se investigó primero). **Con
  paginación offset/limit** (`?limit=&offset=`): a diferencia de los rangos, los documentos
  crecen sin límite con el tiempo. `documents.Service.ListDocuments` normaliza
  `Limit`/`Offset` (nunca cero/negativo, tope `MaxListLimit=200`, default
  `DefaultListLimit=50`) — el repositorio nunca confía en que el llamador ya validó.
- Ambos devuelven `{"<recurso>": [...], "count": N}` — sin metadata de paginación
  (`total`/`has_more`): el cliente puede inferir si hay más páginas comparando
  `count == limit`. Deliberadamente simple para una primera versión; agregar un `total` real
  significaría una segunda consulta `COUNT(*)` por cada listado, que no se justificó todavía.
- Memory/Postgres implementan el mismo contrato (`Repository.ListByIssuer`) — `MemoryRepository`
  filtra y pagina en memoria con `sort.Slice`, `PostgresRepository` construye el `WHERE`
  dinámicamente (cada filtro opcional se agrega solo si aplica, con el placeholder numerado a
  partir de `len(args)` en el momento — mismo cuidado de no desalinear `$N` que motivó
  `internal/sqlutil.Placeholders` en fases anteriores).
- **Verificado contra Postgres real vía curl**: dos emisores de prueba, cada uno con sus
  propios rangos/documentos — `GET /numbering-ranges` y `GET /documents` de cada emisor nunca
  devuelven nada del otro; filtro por tipo de documento y por `limit` confirmados con
  resultados exactos; sin token, ambos devuelven 401.

### 9.20 Replanteo: Customers/Products se construyen ya, no se difieren más

El usuario retomó la decisión de 9.18 (2026-06-22) y pidió revisarla: con la disciplina de
`core-bank` ya aplicada a auth/listados, ¿tiene sentido seguir diferiendo Customers/Products
solo porque no hay frontend todavía? Conclusión, separando los dos casos:

- **Customers**: bajo riesgo (mismo patrón CRUD que el resto de `apidian`) y casi seguro
  necesario desde el primer día de cualquier frontend real — no hay forma de armar una
  pantalla de "crear factura" sin de dónde elegir el cliente. Se construye ya.
- **Products**: el usuario confirmó que sí necesita un catálogo de ítems con sus valores DIAN
  (precio, código de ítem, impuesto por defecto) para no retipearlos en cada factura — el
  patrón real que describió ("una tabla donde cada compañía coloca cada ítem con todos sus
  valores para insertarlos") es exactamente lo que un catálogo de conveniencia resuelve. Se
  construye ya también, aunque el frontend siga sin fecha definida.

Importante: esto NO cambia cómo se emiten documentos. `POST /invoices`/`credit-notes`/
`debit-notes` siguen recibiendo `customer`/`lines` pass-through, igual que siempre — el
catálogo es de dónde el frontend copia esos datos antes de armar la petición, no un nuevo
parámetro de la emisión. Ver 9.21 para el detalle de la implementación.

### 9.21 `internal/customers`/`internal/products` — completo y verificado real

CRUD completo (crear/consultar/listar/actualizar/eliminar) para ambos catálogos, acotados al
emisor autenticado — mismo patrón que el resto de `apidian` (repository/postgres/memory/
service), primeras operaciones de mutación-por-ID de todo el proyecto (hasta ahora solo había
creación y lectura). `Update`/`Delete` acotan directamente en el SQL
(`WHERE id = $1 AND issuer_id = $2`) en vez de depender de que cada handler futuro recuerde
comparar el dueño antes de mutar — más estricto que el patrón "leer y comparar" que usan
`GetNumberingRange`/`GetDocument`, justificado porque mutar es más riesgoso que leer.

- **`customers.Customer`** reutiliza `cofacture/domain.Party` tal cual (mismo tipo que ya usa
  `documents` para el campo `Customer` de una factura) — no se duplicó la lista de campos en
  un struct propio. Se agregó `partyFromDomain()` en `internal/api/dto.go` (inverso de
  `partyDTO.toDomain()`, que antes solo existía en una dirección porque ningún endpoint
  necesitaba devolver un `Party` en la respuesta).
- **`products.Product`** es un struct propio, deliberadamente angosto: sin `Quantity`,
  `LineExtensionCents` ni una lista de impuestos (eso es dato de USO, de una factura concreta,
  no de catálogo) — solo un impuesto por defecto (`TaxTypeCode`/`TaxTypeName`/`TaxPercent`),
  de conveniencia.
- **Dos bugs reales encontrados y corregidos** al verificar contra Postgres real (ninguno
  detectado por los tests con repos en memoria, igual patrón que bugs anteriores de esta
  fase):
  1. `customers`: `tax_scheme_code`/`tax_scheme_name` son FK a `tax_types(code)`, pero el
     código mandaba cadena vacía (no `NULL` de verdad) cuando el cliente no especificaba
     régimen tributario — violaba la FK. Fix: `nullableString()` antes del INSERT/UPDATE,
     mismo criterio que ya usan las columnas de dirección.
  2. `products`: la migración original FK-aba `unit_code` contra `unit_measures(code)` — pero
     ese catálogo solo tiene 10 códigos de muestra (NIU/KGM/LTR...) frente al estándar real
     completo (UN/ECE Rec. 20, cientos de códigos). El código `"94"` — el mismo que usan los
     tests reales contra la DIAN y que obtuvo `StatusCode 00` — no estaba en esas 10 filas: la
     FK habría bloqueado un producto válido. Fix: se quitó la FK (la migración y la BD real ya
     corregidas) — mismo criterio que `domain.Line.UnitCode`, que tampoco se valida contra
     catálogo por vivir en JSONB sin FK. Mismo hueco de datos ya conocido que
     `departments`/`municipalities` (sección 9.6) — no se inventó un catálogo completo sin la
     fuente oficial.
- **Verificado contra Postgres real vía curl** (17/17 pruebas): crear/consultar/listar/
  actualizar/eliminar para ambos catálogos, aislamiento completo entre dos emisores de prueba
  (404 al intentar leer o mutar el recurso de otro tenant), validaciones de entrada (400 sin
  nombre/identificación/descripción/unidad, precio negativo), 401 sin token.

### 9.22 `documents.customer_id` — trazabilidad opcional hacia el catálogo de clientes

El usuario notó que `customers`/`products`/`users`/`payment_methods`/`unit_measures` se veían
"sueltas" en el esquema y pidió revisar las relaciones reales contra el código. Conclusión
(separando dos motivos distintos):

- `users`/`customers`/`products` SÍ están conectadas (`issuer_id → issuers`) — son hojas: nada
  las referencia de vuelta, lo cual es correcto, no un descuido.
- `payment_methods`/`unit_measures` están huérfanas por un motivo previo a esta fase:
  `documents.lines`/`documents.payment_means` siempre han sido JSONB (snapshot pass-through),
  nunca columnas normalizadas, así que ningún código de esos catálogos vive en una columna con
  FK posible — viven dentro del blob JSON, sin validar contra catálogo (confirmado en
  `validateBase()`, que solo exige no-vacío, nunca cruza contra un catálogo).
- `documents.customer`/`documents.lines` NO referencian `customers`/`products` — **deliberado**:
  la DIAN exige que un documento conserve el snapshot exacto del momento de emitir, para
  siempre; una FK viva permitiría que editar/borrar un cliente cambiara retroactivamente
  facturas ya autorizadas, violando esa retención legal.

**Decisión y mejora aplicada**: agregar `documents.customer_id` como referencia OPCIONAL y de
**solo trazabilidad** (no para `products`/`lines`: una factura tiene varias líneas, hacerlo
bien ahí exigiría sacar `lines` de JSONB a una tabla propia, un cambio de fondo no justificado
todavía).

- **Migración**: el `ALTER TABLE documents ADD COLUMN customer_id ...` se agregó dentro de
  `000007_customers.up/down.sql` (no como archivo nuevo) — no puede ir en `000005_documents`
  porque en ese punto de la secuencia `customers` todavía no existe; la FK fallaría en una
  instalación nueva desde cero. `down.sql` deshace primero el `ALTER TABLE` y luego borra
  `customers`, en ese orden (la FK lo exige). `ON DELETE SET NULL`: borrar un cliente nunca
  rompe ni borra documentos ya emitidos, solo desvincula la trazabilidad. Nullable: una
  factura con datos sueltos simplemente no la tiene.
- **`documents.CustomerPort`** (nuevo, en `ports.go`): lo único que `documents` necesita de
  `customers` — verificar, cuando llega un `CustomerID` opcional, que pertenece al mismo
  emisor (`ErrCustomerIssuerMismatch`, 422 — mismo criterio que
  `ErrNumberingRangeIssuerMismatch`). Si el `CustomerID` no existe en absoluto, se propaga
  `customers.ErrCustomerNotFound` (404) sin necesidad de un caso nuevo en `classify()`.
  Esto agrega una arista nueva al grafo de dependencias: `documents → customers` (ver 4.1).
- **No se serializa en ningún XML ni viaja a la DIAN** — cero impacto en el cumplimiento del
  Anexo Técnico; el snapshot (`documents.customer`, JSONB) sigue siendo lo único que se firma
  y se envía, exactamente igual que antes de que existiera esta columna.
- **Verificado contra Postgres real vía curl** (11/11 pruebas, sin tocar los datos reales de
  pruebas manuales del usuario que ya existían en `issuers`/`users`): emitir con `customer_id`
  propio (200, lo persiste y lo devuelve), emitir sin `customer_id` (sigue funcionando igual
  que siempre, campo ausente por `omitempty`), `customer_id` de otro emisor (422),
  `customer_id` inexistente (404), y el caso crítico — **borrar el cliente después de emitir
  la factura no rompe la factura**: queda con `customer_id = NULL`, mismo `document_key`,
  recuperable normalmente.

### 9.23 `next_number` — registrar un rango que retoma una secuencia real ya usada

Antes de hacer pruebas reales de punta a punta vía Postman (registrar el emisor real, su
certificado/resolución real, clientes, productos, y emitir los tres documentos), el usuario
preguntó algo crítico: si su resolución REAL ya tiene números autorizados de verdad en la DIAN
(de las pruebas de Fase 1.7 con `cofacture` directo, ej. `SETP-990068706`), ¿el API puede
"continuar" esa secuencia, o reclamaría desde `range_from` otra vez — arriesgando un duplicado
real contra la propia DIAN?

**Respuesta antes del fix: no se podía.** `numbering.PostgresRepository.Create`/
`MemoryRepository.Create` forzaban `CurrentNumber = RangeFrom - 1` sin excepción — todo rango
nuevo se asumía completamente virgen, sin forma de decirle a la API "ya van usados hasta el
número X de verdad".

**Fix**: `numbering.Service.RegisterRange` ahora acepta un segundo parámetro opcional,
`nextNumber *int64` — expuesto en la API como `next_number` (opcional) en
`POST /api/v1/numbering-ranges`:

- `nil` (omitido): comportamiento de siempre, el primer `ClaimNext` entrega `range_from`.
- Con valor: debe caer dentro de `[range_from, range_to]` (si no,
  `ErrNextNumberOutOfRange`, 400) — `CurrentNumber` se fija en `next_number - 1`, así el
  primer `ClaimNext` entrega exactamente `next_number`, sin importar dónde empiece
  `range_from`.

La responsabilidad de decidir el punto de partida se movió del repositorio (que antes lo
forzaba) al `Service` (que tiene el contexto de negocio) — los repositorios ahora solo
persisten el `CurrentNumber` que ya viene decidido.

**Verificado contra Postgres real vía curl** (4/4): un rango sin `next_number` sigue
arrancando en `range_from` (comportamiento viejo intacto); un rango con
`next_number: 990068707` simulando una resolución real ya usada hasta el `...706` deja
`current_number = 990068706`, y el documento emitido con ese rango sale con
`number = 990068707` exactamente; `next_number` fuera del intervalo rechaza con 400. No se
tocaron los emisores/usuarios reales de pruebas manuales del usuario que ya existían en la
base.

**Guía para el frontend (cuando se construya)**: `next_number` NO debe ser un input visible
por defecto en el formulario de "agregar resolución" — la mayoría de resoluciones nuevas nunca
se han usado, así que mostrar un campo técnico de entrada confundiría el caso común. Patrón
recomendado:

1. Campo oculto detrás de un toggle/checkbox: *"¿Esta resolución ya tiene facturas emitidas
   fuera de este sistema?"*
2. Si se marca, revelar un input en términos de negocio (no el nombre técnico `next_number`):
   *"¿Cuál es el próximo número a emitir?"*
3. Validar en el cliente que el valor caiga dentro de `[range_from, range_to]` antes de
   enviar — mismo chequeo que ya hace el backend (`ErrNextNumberOutOfRange`, 400) — para dar
   feedback inmediato en vez de esperar la respuesta del servidor.

### 9.24 Próximo paso (histórico — superado por 9.25)

Sin tareas pendientes explícitas en este momento — Fase 2 (`apidian`) cubre hoy: bootstrap,
catálogos DIAN, emisores, numeración (incluyendo retomar una secuencia real ya usada),
documentos (Invoice/CreditNote/DebitNote, construir+firmar+enviar, con trazabilidad opcional a
un cliente guardado), auth con aislamiento entre tenants, listados, y los catálogos de
Customers/Products. Lo que sigue diferido (Documento Soporte/RADIAN/Nómina, PDF/
Notificaciones) tiene su razón documentada en la sección 8 — son otra familia de documento
DIAN o están bien delegados a un servicio externo, no son items "olvidados". Próximo paso
real: el usuario va a hacer pruebas de punta a punta con datos reales vía Postman (emisor real
+ certificado real + resolución real + cliente + productos + los tres documentos, incluyendo
envío real a la DIAN).

### 9.25 Borrador/confirmación de documentos + configuración gradual del emisor

El usuario describió el flujo de UX de un proyecto anterior (no DIAN, pero con el mismo patrón
de facturación electrónica) y pidió comparar si `apidian` ya soportaba ese flujo: (1) CRUD de
usuarios; (2) CRUD de "companies" con SOLO los datos que la DIAN exige, más un espacio
SEPARADO para ir cargando software/resolución/certificado independientemente, mientras se van
consiguiendo esos datos; (3)(4) CRUD completo de customers/productos; (5) CRUD de invoice, y
en un paso APARTE, firmar+enviar, verificando la respuesta para cerrar el ciclo.

**Comparación contra el código real** (grep, no memoria):

- Customers/Products: coinciden exactamente — CRUD completo ya existe (9.21).
- Users CRUD: NO existe más allá de register/login. Preguntado explícitamente — el usuario
  confirmó que **no lo necesita**: "un usuario por empresa basta por ahora". El modelo
  "un usuario = un emisor" se queda como está, a propósito.
- Configuración gradual del emisor: NO existía — `POST /auth/register` exigía
  software_id/software_pin/certificate_base64/certificate_password TODOS de una, sin forma de
  completarlos después.
- Invoice CRUD + firmar/enviar aparte: NO existía — `IssueInvoice`/`IssueCreditNote`/
  `IssueDebitNote` reclamaban número, construían, firmaban y enviaban en una sola llamada
  atómica (decisión explícita de una fase anterior, 9.15). Esto entraba en tensión directa con
  un riesgo real señalado en esta conversación: como reclamar el número pasa en el mismo
  instante que crear el documento, un error de captura (línea mal escrita, cliente
  equivocado) quema un consecutivo real de la DIAN sin forma de deshacerlo.

El usuario confirmó, vía pregunta explícita, las tres decisiones:

1. **Separar** "crear/revisar" de "confirmar → firmar y enviar" (recomendado y elegido).
2. **No** agregar Users CRUD — un usuario por emisor sigue siendo suficiente.
3. **Sí** permitir completar software/PIN/certificado después del registro, aclarando
   explícitamente: lo que en el código se llama `numbering` es lo que en lenguaje humano se
   lee como "resolución" — **no es un pedido de rename**, es el mismo patrón que ya existe con
   `issuers` representando "empresa" sin llamarse así en el código (9.7). El nombre de dominio
   (`numbering`, `issuers`) describe la responsabilidad técnica; la UI puede (y debe) usar el
   término que el usuario de negocio reconoce ("resolución", "empresa").

**Implementación — borrador/confirmación (`internal/documents`)**:

- `Document` gana `StatusDraft` (nuevo, antes del ciclo `built→sent→accepted/rejected/
  send_error`) y un campo `Note` persistido desde la creación del borrador (antes solo vivía
  en la petición HTTP, se perdía si la emisión no era atómica).
- `prefix`/`number`/`document_key`/`issue_date`/`issue_time`/`qr_url`/`signed_xml` pasan a ser
  NULLABLE en la tabla `documents` (editada en sitio en `000005_documents.up.sql` — la tabla
  estaba vacía de datos reales, mismo criterio que la convención de migraciones ya establecida)
  — `uq_documents_range_number` sigue siendo válida porque Postgres nunca considera dos NULL
  iguales entre sí, así que varios borradores sin número conviven sin chocar.
- `documents.Service` se reorganiza en dos fases:
  - `validateForIssuance` (emisor existe, rango pertenece al emisor y al tipo de documento
    correcto, cliente opcional pertenece al emisor) corre TANTO al crear/editar un borrador
    como, de nuevo, al confirmar — para fallar rápido sin costo en el borrador, y
    defensivamente otra vez al confirmar.
  - `claimAndLoadCert` (exige software/certificado listos, reclama el consecutivo real,
    carga el certificado) corre SOLO al confirmar — es el único punto de todo el servicio
    donde se "gasta" un número.
  - `CreateInvoiceDraft`/`CreateCreditNoteDraft`/`CreateDebitNoteDraft`,
    `UpdateInvoiceDraft`/`UpdateCreditNoteDraft`/`UpdateDebitNoteDraft`, `DeleteDraft` y
    `ConfirmDocument` reemplazan a los antiguos `IssueInvoice`/`IssueCreditNote`/
    `IssueDebitNote` (que hacían todo en una sola llamada).
- Nuevos errores: `ErrDocumentNotDraft` (editar/eliminar/confirmar algo que ya no es borrador
  — 409) y `ErrIssuerNotReadyToIssue` (confirmar sin software/certificado configurados — 422,
  mensaje de dominio claro en vez de un error de bajo nivel al parsear un certificado vacío).
- Nuevas rutas HTTP: `PUT /invoices/{id}` (y credit-notes/debit-notes), `DELETE
  /documents/{id}` (compartido entre los tres tipos — borrar no depende del tipo), `POST
  /documents/{id}/confirm` (compartido, despacha según `dian_document_type_code`).

**Implementación — configuración gradual del emisor (`internal/issuers`)**:

- `validateIssuer` ya NO exige `software_id`/`software_pin`/`certificate` al registrar — solo
  los datos que la DIAN pide del emisor mismo (NIT, razón social, ubicación, ambiente).
- `software_id TEXT`/`software_pin BYTEA`/`certificate BYTEA`/`certificate_password BYTEA` se
  vuelven NULLABLE en `000003_issuers.up.sql` (editado en sitio) — `cryptutil.Encrypt`/
  `Decrypt` ya manejaban un plaintext/ciphertext vacío como NULL simétrico, así que esto no
  rompía nada existente, solo destrababa el caso "todavía no configurado".
- `issuers.Service.UpdateIssuer` (nuevo) hace una actualización PARCIAL a propósito —
  cada puntero `nil` significa "no tocar este campo" — porque el usuario va completando
  software/PIN/certificado en el orden en que los consiga, sin tener que reenviar lo que ya
  cargó antes. Un valor explícitamente vacío (`""`) se rechaza (`ErrEmptySoftwareID`/
  `ErrEmptySoftwarePIN`/`ErrEmptyCertificate`) — casi siempre es un error de quien llama, nunca
  una forma válida de "borrar" la credencial.
- Nueva ruta `PUT /issuers/me` — mismo criterio "un usuario = un emisor" que
  `GET /issuers/me`, nunca recibe un `{id}` en el path.

**Verificado contra Postgres real** (vía curl, certificado autofirmado de prueba, sin tocar
los emisores/usuarios reales del usuario — 4 issuers/4 users intactos antes y después):
registro sin credenciales → crear borrador → editar borrador (PUT) → confirmar SIN
credenciales (422, `ErrIssuerNotReadyToIssue`, mensaje exacto) → `PUT /issuers/me` con
software/PIN/certificado → confirmar de nuevo (200, número real reclamado, CUFE calculado,
XML firmado con `<ds:Signature>`, intento de envío real a la DIAN habilitación que termina en
`send_error` porque el certificado es autofirmado, no uno real de la DIAN — comportamiento
esperado, no un bug) → intentar eliminar el documento ya confirmado (409) → intentar
confirmarlo otra vez (409). Los datos de prueba creados durante la verificación (1 issuer, 1
user, 1 numbering_range, 1 document) se eliminaron al cerrar la prueba.

### 9.26 Preguntas de configuración del emisor — multi-tenencia, vigencia de resoluciones, notificaciones

Tras construir 9.25, el usuario hizo pruebas en Postman y surgieron cuatro preguntas de
diseño. Quedan documentadas aquí como **decisiones de alcance explícitas, no huecos
olvidados** — mismo criterio que la sección 8.

**(a) Multi-empresa / multi-sucursal — qué ya funciona y qué no.**

- **Multi-sucursal por prefijo SÍ funciona hoy, sin cambios**: `numbering_ranges` no tiene
  ninguna restricción de unicidad sobre `(issuer_id, dian_document_type_code)` —
  un mismo emisor puede tener varios rangos de tipo "01" con prefijos distintos
  (ej. `SETP` para la sede principal, `SETB` para una sucursal), cada uno con su propia
  resolución/vigencia/clave técnica. `documents.Service.validateForIssuance` solo exige que
  el rango pertenezca al emisor y al tipo de documento — nunca exige que sea "el único" rango
  de ese tipo.
- **Multi-empresa (un usuario administrando varios NITs/emisores) NO funciona**: el modelo
  "un usuario = un emisor" (sección 9.17) fija un único `issuer_id` por usuario desde el
  registro, horneado en el JWT (`middleware.GetTenantID`) — no hay forma de que un mismo
  login cambie entre emisores ni de asociar un segundo NIT al mismo usuario.
- **Múltiples software_id/certificados por emisor NO funciona**: `issuers.Issuer` guarda
  `SoftwareID`/`Certificate`/`CertificatePassword` como columnas sueltas (no una tabla hija
  con una lista) — es una restricción estructural, no una validación que rechace un segundo
  valor. Cargar un certificado nuevo reemplaza al anterior, sin historial.

**Decisión (2026-06-22): documentar y diferir, no construir todavía.** No hay un caso de uso
concreto hoy que necesite multi-empresa ni multi-credencial por emisor — construirlo sin esa
necesidad sería diseñar para un futuro hipotético. Vale la pena anotar la asimetría de costo
para cuando haga falta decidir: hoy, con solo 4 issuers/4 users de prueba en la base real, el
cambio de modelo (de "un usuario tiene un `issuer_id`" a una relación muchos-a-muchos
usuario↔emisor) es barato — toca el JWT, `middleware`, `auth`, y una migración de esquema sin
datos de producción reales en juego. Una vez existan usuarios/emisores reales de producción,
el mismo cambio exige migrar datos existentes y pensar en sesiones/tokens ya emitidos — deja
de ser gratis. Si el usuario confirma que SÍ va a necesitar multi-empresa pronto, es mejor
señal para hacerlo ahora que esperar.

**(b) Vigencia de resoluciones — confirmado el hueco, queda pendiente a propósito.**

Verificado línea por línea: `numbering.PostgresRepository.ClaimNext` solo valida
`is_active = TRUE` y que no se haya agotado `range_to` — **nunca compara contra `valid_to`**.
Una resolución vencida sigue reclamando números para siempre, igual que en el sistema gratuito
de la DIAN que el usuario usaba antes. La diferencia es que ese sistema al menos permite que
quede vencida sin romperse; aquí tampoco se rompe, pero no hay ningún aviso: ni un estado
calculado (`expired`/`expiring_soon`), ni un endpoint para editar `valid_to` (no existe
`PUT /numbering-ranges/{id}`, solo `POST`/`GET`), ni infraestructura de notificación (no hay
SMTP/push/cron en todo el proyecto, verificado por grep). Si se confirma un documento con una
resolución vencida, hoy el error solo llegaría como un rechazo de la propia DIAN, no como un
aviso preventivo de `apidian`.

**Decisión: queda pendiente, sin tocar el esquema todavía.** A diferencia de (a), aquí no hay
urgencia de timing — `valid_to` YA existe como columna desde la Fase 2.4, así que calcular un
estado (`expired`/`expiring_soon`) o agregar un endpoint de actualización es lógica pura,
agregable en cualquier momento sin migración de datos. No hay costo por esperar.

**(c) Notificaciones (email/push) — reafirmada la decisión ya tomada en la sección 8.**

El usuario preguntó explícitamente si `apidian` debería tener un módulo de SMTP/email (a
futuro, para mandarle la factura al customer) o si eso debería vivir en el frontend —
señalando su preferencia general por mantener el menor número de responsabilidades posible
dentro de este servicio. La sección 8 ya había decidido esto antes de que existiera ninguna
necesidad concreta: *"Notificaciones (email/SMS al receptor) → Servicio de notificaciones
externo"*. Se reafirma esa decisión con el mismo razonamiento: `apidian` es el orquestador
DIAN, no un servicio de comunicaciones. Cuando haga falta enviar la factura al cliente, la
recomendación es que `apidian` solo **exponga** lo necesario (el XML firmado, el CUFE/QR, o
un endpoint para descargar el documento ya confirmado) y que el envío en sí lo haga otra
pieza — no necesariamente el frontend del navegador (un correo no se manda bien desde JS en el
cliente, por credenciales SMTP expuestas), sino preferiblemente un servicio/función pequeña y
aparte (lo que en core-bank se llamaría un "worker" de notificaciones), consumida vía su propia
API. Ni variables de entorno de SMTP ni plantillas de correo viven dentro de este repo.

**(d) Subida de certificado — flujo real explicado, y un hueco real corregido.**

El usuario preguntó cómo sería el flujo real (más allá de Postman) para subir un `.p12`:
input de archivo o drag-and-drop → `FileReader` del navegador lo convierte a base64,
client-side → el frontend manda el mismo JSON que hoy se manda a mano
(`certificate_base64`/`certificate_password`) por HTTPS → `apidian` decodifica esos bytes de
vuelta al `.p12` original, los cifra (AES-256-GCM) y los guarda tal cual → solo al confirmar
un documento, `cofacture/signer.LoadPKCS12` los decodifica de verdad (en memoria, sin PEM, sin
archivos temporales, sin `openssl` — confirmado, no hay ninguna conversión a PEM en el flujo
real, `LoadPEM` en `cofacture/signer` solo se usa en un test local de Fase 1).

Al explicar ese flujo se encontró un hueco real: `issuers.Service.UpdateIssuer` nunca
intentaba parsear el `.p12` que recibía — un archivo corrupto o una contraseña equivocada se
guardaban sin error, y la falla solo aparecía después, al confirmar el primer documento, como
un 500 genérico ("error interno del servidor"), no un mensaje claro.

**Fix (2026-06-22)**: `UpdateIssuer` ahora valida, cuando la llamada toca `Certificate` o
`CertificatePassword` y el emisor queda con AMBOS no vacíos después del merge, que de verdad
formen un `.p12` legible — devuelve `ErrInvalidCertificate` (400) si no. Si la combinación
sigue incompleta (ej. se sube el certificado pero la contraseña se va a completar después,
configuración gradual de 9.25), la validación se omite a propósito: no sería justo rechazar
algo que el usuario nunca pretendió que fuera completo todavía.

El reto de diseño fue mantener la regla de la sección 4.1 ("`documents` es el único paquete
que importa `cofacture` directamente") sin romperla: `internal/issuers` ganó un puerto angosto
nuevo, `CertificateValidator` (`func(certificate []byte, password string) error`, mismo
patrón que `documents.IssuerPort`/`NumberingPort`/`CustomerPort`), inyectado en `issuers.New`.
La implementación real (`documents.ValidateCertificate`, una función nueva y simple que envuelve
`signer.LoadPKCS12`) vive en `internal/documents` —el único sitio permitido— y se inyecta desde
`internal/api` al construir `issuerSvc`. `internal/issuers` sigue sin importar `cofacture` ni
`documents` en ningún archivo de producción.

Verificado contra Postgres real vía curl, sin tocar los issuers/users reales del usuario:
certificado base64 válido pero no-p12 → 400 con el mensaje de `ErrInvalidCertificate`
exacto; certificado real + contraseña correcta → 200; actualizar solo `software_id` sin tocar
el certificado → 200, sin disparar la validación (confirmado con un validador que falla la
prueba si se llama, en `TestUpdateIssuer_CertificateWithoutPasswordSkipsValidation`). Datos de
prueba eliminados al cerrar.

### 9.27 CORS configurable

El usuario pidió pasar de probar con Postman a probar con un frontend real en el navegador
(ver `docs/frontend-architecture.md` para todo lo relativo al frontend mismo — este documento
solo cubre lo que cambió en `apidian`). Eso expuso un hueco inmediato: `apidian` no tenía
CORS — cualquier llamada desde un origen distinto (otro puerto, `file://`) quedaba bloqueada
por el propio navegador aunque la petición sí le llegara al servidor.

**Fix**: `internal/api/middleware/cors.go`, nuevo. Lista explícita de orígenes permitidos vía
`CORS_ALLOWED_ORIGINS` (sin default implícito — mismo criterio que `ISSUER_SECRETS_KEY`/
`AUTH_JWT_SECRET`), preflight `OPTIONS` respondido de verdad (necesario porque casi toda la
API usa `Authorization: Bearer` + JSON + métodos no-simples, así que el navegador manda
preflight antes de cada petición real), `"*"` soportado solo para desarrollo local. Verificado
contra Postgres real: origen permitido recibe los headers correctos, origen no permitido
nunca recibe `Access-Control-Allow-Origin` (bloqueado del lado del navegador, no del
servidor), peticiones sin `Origin` (curl/Postman) no se ven afectadas.

### 9.28 `documentResponse` completo — faltaban `customer`/`lines`/`totals`/`created_at`

Construyendo la pantalla de "Facturación" del dashboard (lista + detalle de documentos) se
encontró que `GET /documents` y `GET /documents/{id}` (y por extensión las respuestas de
crear/editar/confirmar, que comparten el mismo `documentToResponse`) solo devolvían metadatos
— ID, estado, prefijo/número si ya estaba confirmado. Nunca el contenido capturado: ni el
cliente, ni las líneas, ni el total, ni la fecha de creación. Un frontend real no podía
mostrar "factura para Juan Pérez por $119.000" sin volver a pedirle los datos al usuario —
exactamente el tipo de hueco que probar la API como un usuario real revela y Postman/`devui`
nunca iban a exponer.

**Decisión del usuario**: ante la disyuntiva de resolverlo con una caché local en el
navegador (sin tocar `apidian`) o completar la API de verdad, el usuario eligió completar
`apidian` — razonando que el propósito del dashboard es justamente evitar huecos ocultos como
este, y que mejor corregirlo ahora (sin datos de producción reales en juego) que parchar el
síntoma en el frontend.

**Fix**: `documentResponse` ahora incluye `customer`/`lines`/`payment_means`/`totals`/`note`/
`currency_code` (siempre presentes, incluso en borrador), `billing_reference`/
`discrepancy_response`/`note_type_code` (solo CreditNote/DebitNote), y `created_at`/
`updated_at` (mismo criterio que `customerResponse`/`productResponse` — un borrador no tiene
`issue_date` todavía, así que es lo único con lo que un listado puede ordenar). Nuevas
funciones inversas en `dto.go` (`linesFromDomain`, `paymentMeansFromDomain`,
`totalsFromDomain`) — todos campos nuevos, aditivos, no rompen ningún consumidor existente
(Postman, `devui`). Verificado: build/test limpios + curl real confirmando que el borrador y
el documento confirmado devuelven `customer`/`lines`/`totals` completos.

### 9.29 Emisor persona natural — `EntityTypeCode`/`LiabilityCodes` rechazados por la DIAN real

Probando el ciclo completo desde el dashboard con su propia identidad real (cédula, no NIT —
la primera vez que alguien registra un emisor persona natural en todo este proyecto), el
usuario recibió **3 rechazos reales de la DIAN** (`StatusCode 99`, "Validación contiene
errores en campos mandatorios"). Inspeccionando la base de datos y el XML firmado guardado se
encontraron dos bugs reales en `issuers.applyDefaults`:

1. **`EntityTypeCode` defaulteaba a `"1"` (Persona Jurídica) sin importar el tipo de
   identificación** — confirmado contra una factura con NIT en la Fase 1, nunca ajustado para
   identificaciones personales. Un emisor con cédula (`identification_type_code: "13"`)
   terminaba mandando `<cbc:AdditionalAccountID>1</cbc:AdditionalAccountID>` — contradicción
   directa con su propia identificación.
2. **`LiabilityCodes` nunca tenía un default para el emisor** — a diferencia de
   `applyCustomerDefaults` (`documents/service.go`), que sí respalda al adquiriente con
   `["R-99-PN"]` cuando viene vacío. El XML del `AccountingSupplierParty` no traía NINGÚN
   `cbc:TaxLevelCode` — un campo mandatorio del Anexo Técnico, ausente por completo.

**Fix**: `defaultEntityTypeCode(identificationTypeCode)` deriva `"1"` solo para tipos de
identificación tributaria (`"31"` NIT, `"47"` NIT de otro país, `"50"` NIT de la DIAN) y `"2"`
para el resto (identificaciones personales); `LiabilityCodes` ahora respalda con `["R-99-PN"]`
igual que el adquiriente. Verificado con tests (`TestRegisterIssuer_NaturalPerson_
EntityTypeCode`, 7 combinaciones de tipo de identificación). El emisor real ya registrado del
usuario se corrigió directamente en la base de datos (mismos valores que el fix aplicaría) sin
necesidad de re-registrarse.

**Decisión del usuario sobre el alcance**: exponer `liability_codes` como campo configurable
en el dashboard queda explícitamente diferido — el usuario quiere primero completar el ciclo
con el dashboard actual ("improvisado"), juntar más hallazgos, y recién después construir el
dashboard definitivo con esos aprendizajes ya incorporados desde el diseño.

### 9.30 `PaymentMeans` obligatorio — tercera causa real de rechazo (`SETP-990000001`)

Después del fix de 9.29, el usuario probó una factura nueva sobre la resolución real (rango
`990000000`-`995000000`, confirmada para habilitación, no producción) y la DIAN la rechazó
otra vez con el mismo `StatusCode 99` genérico. Inspeccionando el XML firmado guardado:
`entity_type_code`/`liability_codes` ya estaban correctos en ese documento (el fix de 9.29 sí
se aplicó), pero **no aparecía ningún `cac:PaymentMeans`** — ni en la factura rechazada ni en
ninguna creada desde el dashboard, porque ese formulario nunca lo pide.

Confirmado contra el propio Anexo Técnico (`docs/reference/anexo-tecnico-1.9.txt`,
`FAN01`/`CAN01`/`DAN01`): `cac:PaymentMeans` tiene cardinalidad **`1..N`** — obligatorio, al
menos una ocurrencia — para Invoice, CreditNote y DebitNote. La factura real que SÍ fue
aceptada por la DIAN en la Fase 1 (`cofacture/soap/realsend_test.go`) lo incluía explícitamente
desde siempre; nunca se replicó esa exigencia en `apidian` ni en el dashboard.

**Fix**: `documents.Service.validateBase` ahora exige `len(paymentMeans) > 0`
(`ErrMissingPaymentMeans`, 400) para los tres tipos de documento, desde el borrador — no solo
al confirmar, mismo criterio que el resto de validaciones de esta función (fallar rápido, sin
gastar un número real). Se actualizaron todos los fixtures de test (`testRequest`/
`testNoteRequest` en `internal/documents`, `testPaymentMeans()` nuevo en `internal/api`) para
incluir una forma de pago por defecto. Build/vet/test limpios.

**Efecto colateral conocido y aceptado**: el dashboard improvisado (`frontend/static/
dashboard/`) nunca pide ni manda `payment_means` — con este fix, ya no puede crear ningún
borrador de factura nuevo (`POST /invoices` responde 400). El usuario decidió explícitamente
no parchar el dashboard improvisado para esto; queda documentado como pendiente para el
dashboard definitivo en `docs/frontend-architecture.md`. Mientras tanto, el explorador tipo
Postman (`frontend/static/`, sin tocar) sí permite agregar `payment_means` a mano en el JSON.

### 9.31 `documents.customer_id` físicamente fuera de orden — migraciones renumeradas

El usuario pidió revisar las 8 migraciones porque sospechaba que alguna no respetaba la regla
de `created_at`/`updated_at` siempre al final (sección 4.1). Los 8 archivos en sí estaban bien
— el problema estaba en la tabla **real**: `000007_customers.up.sql` agregaba `customer_id` a
`documents` con `ALTER TABLE ... ADD COLUMN` (a propósito, documentado: en la secuencia
original, `customers` todavía no existía en el punto donde se crea `documents`). Postgres
siempre añade columnas nuevas al final físicamente, sin importar dónde "debería" ir
lógicamente — así que `customer_id` terminaba en la posición 32 de 32, después de
`created_at`/`updated_at`. Esto afecta a **cualquier** instalación que corra las migraciones
en ese orden, no solo a la base de datos de desarrollo — no era un descuido de una sesión, era
estructural a la secuencia.

Mi primer intento de arreglo fue una migración nueva que reconstruía la tabla (`CREATE TABLE
... AS SELECT` + swap) preservando los datos reales — el usuario lo rechazó explícitamente:
"no quiero sino las migraciones sin alter ni nada por el estilo". Pidió en cambio **reordenar
las migraciones existentes de raíz**, aprovechando que en ese momento la base de datos no
tenía datos reales en juego (confirmado por el usuario: "ahora que estamos de cero, sin datos
reales").

**Fix real — renumeración de migraciones (2026-06-23)**: `customers` (sin dependencia de
`documents`) se adelantó de `000007` a `000005`; `products`/`users` se corrieron a
`000006`/`000007`; `documents` pasó de `000005` a `000008`. Con `customers` creándose antes,
`documents` ahora declara `customer_id UUID REFERENCES customers(id) ON DELETE SET NULL`
**directamente dentro de su propio `CREATE TABLE`**, en su posición lógica (justo después de
`customer`) — cero `ALTER TABLE` en toda la secuencia. Orden final de dependencias: catálogos
→ issuers → numbering_ranges → customers → products → users → documents.

**Aplicado real**: se vació la base de datos por completo (`DROP TABLE ... CASCADE` de las 14
tablas + la función `set_updated_at()` + la tabla de control `schema_migrations` de
golang-migrate) y se dejó que `cmd/server` reconstruyera todo desde cero con la secuencia
renumerada — `migrations applied` y `catalogs seeded` sin errores. Verificado con una consulta
directa a `information_schema.columns`: `customer_id` queda en la posición 12 (justo después
de `customer`), `created_at`/`updated_at` en las posiciones 31/32 (las últimas), como exige la
convención. `go build`/`vet`/`test` limpios en los tres módulos tras el cambio. El único dato
perdido en el vaciado fue el emisor de prueba con identidad real del usuario (`DIEGO FERNANDO
MONTOYA VALLEJO`, 2 resoluciones, 5 documentos, 1 cliente, 1 producto) — pérdida aceptada
explícitamente por el usuario, tendrá que volver a registrarse y configurar su emisor desde
el dashboard.

### 9.32 Multi-empresa — registro desacoplado de la empresa, `user_issuers` N:M

Hasta esta sección, la regla era rígida: "un usuario = un emisor" (`users.issuer_id NOT NULL`,
poblado en el mismo `POST /auth/register`). El usuario pidió explícitamente romper esa regla
para soportar el caso real de un contador o administrador que maneja varias empresas/
sucursales con el mismo login — sin saber todavía cuántas, ni si todas existen desde el primer
día.

**Diseño elegido** (de tres opciones presentadas, el usuario escogió en los tres casos la
recomendada):

1. **Login autoselecciona la empresa activa solo si el usuario tiene exactamente una vinculada**
   — preserva el comportamiento de hoy para el caso normal (un usuario, una empresa) sin que
   note el cambio. Con cero o varias, el token queda emitido sin empresa activa
   (`tenant_id = uuid.Nil`) y el cliente debe crear una (`POST /issuers`) o elegir una
   (`POST /issuers/{id}/select`) explícitamente.
2. Reordenar las migraciones para que `customer_id` quedara bien posicionado desde el inicio
   en vez de via `ALTER TABLE` — resuelto ya en la sección 9.31, antes de esta.
3. Numeración tras rechazo — ver 9.33, resuelto junto con esto.

**Esquema**: `users.issuer_id` se elimina por completo; nueva tabla `user_issuers(user_id,
issuer_id, role, created_at)` con PK compuesta `(user_id, issuer_id)` y `ON DELETE CASCADE` en
ambos lados — relación N:M pura, sin columnas adicionales que no se necesiten todavía (no hay
"empresa por defecto" persistida; cuál es "la activa" vive solo en el JWT de la sesión, nunca en
la base de datos). Migrada inline (sección 4.1: sin datos reales en juego todavía, ver 9.31) —
`000007_users.up.sql` ya no declara `issuer_id`; `000009_user_issuers.up.sql` es la tabla nueva.

**`auth.TokenIssuer.Issue` cambia de firma**: antes tomaba `tenant_id` de `u.IssuerID` (una
propiedad fija del usuario); ahora recibe `tenantID uuid.UUID` como parámetro explícito — la
empresa activa es contextual a la sesión, no una propiedad permanente del usuario.
`uuid.Nil` es un `tenant_id` válido en el token: significa "autenticado, sin empresa activa
todavía".

**`middleware.RequireTenant`** (nuevo, va después de `Auth` en la cadena): responde `409` con
un mensaje explícito si `GetTenantID(ctx) == uuid.Nil` — evita que cada handler tenga que
repetir esa comprobación o, peor, devuelva silenciosamente listas vacías sin explicar por qué.
Deliberadamente NO se aplica a las tres rutas de gestión de empresas (`POST /issuers`,
`GET /issuers`, `POST /issuers/{id}/select`) — son justamente las que un usuario sin empresa
activa necesita poder llamar.

**Endpoints nuevos**:
- `POST /api/v1/issuers` — crea una empresa nueva, la vincula al usuario autenticado como
  `owner`, y reemite el token con esa empresa ya activa. Reutiliza el mismo DTO/validaciones
  que antes vivían en el registro (`createIssuerRequest`/`issuerFromRequest`) — software/PIN/
  certificado siguen siendo opcionales aquí, igual que en la sección 9.25.
- `GET /api/v1/issuers` — lista las empresas a las que el usuario tiene acceso.
- `POST /api/v1/issuers/{id}/select` — reemite el token con `{id}` como empresa activa, solo si
  el usuario de verdad está vinculado a ella (`auth.ErrIssuerAccessDenied`, mapeado a `404` —
  mismo criterio de "indistinguible de no existe" que el resto de la API entre tenants).

**`auth.Service`** gana `CreateIssuerForUser`, `ListUserIssuers`, `SelectIssuer` — los tres
devuelven el mismo `AuthResult{Token, User, ActiveIssuer}` que `Register`/`Login`, así que
`handler_auth.go`/`handler_issuers.go` comparten una sola función de respuesta
(`writeAuthResponse`). `auth.IssuerPort` gana `GetIssuer` (antes solo tenía `RegisterIssuer`) —
necesario para devolver la empresa activa resuelta en la respuesta.

**Verificado contra Postgres real vía curl** (flujo completo): registrar usuario sin empresa
→ `GET /issuers/me` sin empresa activa responde `409` → crear primera empresa (token queda con
ella activa) → crear segunda empresa con el token original (sigue sin empresa "fija") → login
de nuevo con dos empresas vinculadas: `body.issuer` ausente, no autoselecciona ninguna →
`GET /issuers` lista las dos → `POST /issuers/{id}/select` activa la segunda, `GET /issuers/me`
confirma → intentar seleccionar una empresa de otro usuario responde `404`. `go build`/`vet`/
`test` limpios en los tres módulos.

### 9.33 Numeración: reutilizar el número tras un rechazo o error de envío

Pregunta abierta desde hace varias secciones: si la DIAN rechaza un documento (`StatusRejected`)
o ni siquiera se logra transmitir (`StatusSendError`), ese número de la resolución queda
permanentemente "quemado" — nunca existió de verdad ante la DIAN, pero el siguiente documento
reclama el que sigue, dejando un hueco para siempre en la numeración. El usuario preguntó
explícitamente si esto se podía resolver, sin saber de antemano cómo ("no sé cómo se pueda
resolver esto").

**Diseño elegido**: si el último número reclamado en un rango termina en rechazado o
`send_error`, el SIGUIENTE intento de confirmación sobre ese rango reclama ese mismo número de
nuevo, en vez de avanzar — nunca arriesgando un duplicado real.

**Mecanismo, en dos partes que se necesitan mutuamente**:

1. **`numbering.Service.ReleaseIfCurrent(ctx, rangeID, number)`** — revierte el último
   `ClaimNext` de un rango, pero SOLO si `current_number` sigue siendo exactamente `number`
   (nadie reclamó otro número desde entonces). Implementado como un único `UPDATE ...
   WHERE id = $1 AND current_number = $2` (mismo patrón atómico que `ClaimNext`): si la
   condición no se cumple, el `UPDATE` simplemente no afecta ninguna fila — no es un error, es
   la señal de "ya no es seguro retroceder, alguien más avanzó". Se llama desde
   `documents.Service.finish()` cuando el estado final es `StatusRejected`/`StatusSendError`, y
   también de forma defensiva en cualquier punto de `confirmInvoice`/`confirmCreditNote`/
   `confirmDebitNote`/`claimAndLoadCert`/`finalizeAndSend` donde se falla DESPUÉS de reclamar
   el número pero ANTES de que el documento llegue a existir de verdad (certificado corrupto,
   fallo al construir o firmar el XML, fallo al persistir) — mismo razonamiento: ese número
   nunca existió ante la DIAN. Siempre *best-effort*: si `ReleaseIfCurrent` falla o no aplica,
   el documento ya quedó con el estado correcto de todas formas, así que el error nunca se
   propaga — es una optimización, no una garantía de corrección.

2. **`uq_documents_range_number` pasa de constraint plana a índice ÚNICO PARCIAL**:
   `CREATE UNIQUE INDEX ... ON documents(numbering_range_id, number) WHERE status NOT IN
   ('rejected', 'send_error')`. Sin este cambio, el mecanismo de arriba sería inútil: el
   documento rechazado SIGUE existiendo en la tabla con ese número (es el registro histórico,
   nunca se borra ni se muta) — un segundo documento con el mismo número chocaría contra él. El
   índice parcial permite que cualquier cantidad de documentos rechazados/con error de envío
   compartan un número con el documento que SÍ lo reclama de verdad después; pero sigue siendo
   imposible que dos documentos `built`/`sent`/`accepted` (los estados que de verdad cuentan
   ante la DIAN) compartan uno — la base de datos sigue siendo la última línea de defensa
   contra un duplicado real, no solo la lógica de aplicación.

**Verificado real, de punta a punta, contra Postgres y la DIAN real** (no solo simulado): se
registró un emisor con un certificado autofirmado de prueba (no autorizado por la DIAN a
propósito, para forzar un rechazo real del SOAP) en habilitación, se confirmó una factura — la
DIAN respondió con un fallo SOAP real, el documento quedó en `send_error` con `number: 1`, y
`numbering_ranges.current_number` volvió a `0` (como si nunca se hubiera reclamado). Una
segunda factura sobre el mismo rango reclamó `number: 1` de nuevo, no `2` — ambos documentos
(`send_error` + el segundo) coexisten en la tabla gracias al índice parcial. Limpiado después
de verificar.

Cobertura de tests: `numbering.TestReleaseIfCurrent_RevertsLastClaim`/
`_NoOpIfNotCurrent` (la condición atómica en sí), `documents.
TestConfirmDocument_ClaimLoadFailure_ReleasesNumberForRetry` (un fallo de certificado libera el
número y el reintento sobre el mismo borrador lo reclama de nuevo).

### 9.34 Auditoría completa de catálogos — `internal/catalogs`, Formas de Pago/Tipo de
Régimen/Responsabilidades fiscales, CIIU

El usuario pidió una auditoría profunda de todos los catálogos DIAN/DANE del sistema. Hallazgo
central: desde la versión 1.9 del Anexo Técnico, la DIAN sacó **todas** las tablas de
catálogo del PDF y las movió a un archivo separado ("Caja de Herramientas Factura
Electrónica") — el PDF que ya teníamos en `docs/reference/` solo tiene las *reglas* de cada
campo (cardinalidad, formato), nunca los *valores*. El usuario encontró y agregó esa caja de
herramientas en `docs/reference/Caja de herramientas FE_V19_(v2026)/` — de ahí salen todos los
datos reales de esta sección.

**Fuentes usadas, en orden de confianza**:
1. `Listas de valores/*.gc` (formato Genericode XML, code+name) — la más confiable.
2. `Schemes/listacodigos/DIAN_UBL21-listacodigos_v1.6.sch` — el Schematron real que la DIAN
   usa para validar (la lista de códigos válidos en sí es de fiar; los *nombres* no siempre
   están, y el archivo está fechado 2019 mientras el Anexo es de 2023 — se encontró al menos
   una discrepancia real (`CustomizationID` aceptaría solo "1"/"2"/"3" según el `.sch`, pero
   "10"/"20"/"30" ya están confirmados reales contra la DIAN) — así que se usó como referencia
   fuerte, nunca como verdad absoluta sin verificar.

**Catálogos completados con datos 100% confiables** (`internal/database/seed/*.csv`):

| Catálogo | Antes | Después | Fuente |
|---|---|---|---|
| `departments` | 24/33 | **33/33** | `Departamentos-2.1.gc` — faltaban Amazonas/Arauca/Casanare/Guainía/Guaviare/Putumayo/San Andrés/Vaupés/Vichada |
| `municipalities` | 10/1.122 | **1.122/1.122** | `Municipio-2.1.gc` — `department_code` derivado de los primeros 2 dígitos del código DIVIPOLA de 5 |
| `tax_types` | 5/16 | **16/16** | `TipoImpuesto-2.1.gc` |
| `payment_methods` (Medios de Pago) | 6/75 | **75/75** | `MediosPago-2.1.gc` — **hallazgo crítico**: los 6 códigos que ya teníamos (10/20/30/42/47/48) tenían nombres INCORRECTOS (ej. "47" decía "Cheque", el oficial es "Transferencia Débito Bancaria") — no era solo incompleto, estaba mal |

**Catálogos nuevos** (se integraron directamente en `000002_catalogs`, la misma migración que
ya tenía el resto de catálogos — no en una migración separada: como todavía no hay datos
reales en juego, se corrige de cero en vez de acumular migraciones nuevas para algo que
lógicamente pertenece junto a los demás catálogos):

- **`payment_terms`** (cbc:PaymentMeans/ID — 2 valores: "1" Contado, "2" Crédito; nombre en
  inglés a propósito, igual que el resto del esquema — la versión inicial se llamó
  `formas_pago` por error, sin que ningún dato real dependiera de eso todavía, así que se
  corrigió en sitio). No confundir con `payment_methods` (Medios de Pago,
  cbc:PaymentMeansCode) — son catálogos DIAN distintos que antes se confundían.
  `domain.PaymentMean.Code`/`PaymentMethodCode` en `cofacture` ya distinguían los dos campos;
  faltaba el catálogo de uno de ellos.
- **`tax_regimes`** (Tipo de Régimen, listName de `cbc:TaxLevelCode`) — 5 códigos
  (`00,02,03,04,05`) confirmados válidos solo por el `.sch` (sin nombre oficial disponible en
  la Caja de Herramientas — el seed lo dice explícitamente, no se inventan nombres). Nuevo
  campo `TaxRegimeCode` en `cofacture/domain.Party` que **ya existía pero `apidian` nunca lo
  poblaba** — ahora `issuers.Issuer`/`customers.Customer` lo tienen (FK opcional,
  `*string`/`nullableString`) y `documents.Service` lo mapea al `domain.Party` del emisor.

**Corrección de nombres de `tax_regimes`** (2026-06-25, durante la construcción del formulario
de empresa del frontend): el usuario notó en su propio RUT real (`docs/reference/
141227953056.pdf`, casilla 53 "Responsabilidades, Calidades y Atributos") el código
`"05- Impto. renta y compl. régimen ordinario"` y preguntó si los nombres genéricos
("Régimen 00".."05") que se habían puesto decían algo. Investigación: `@listName` de
`cbc:TaxLevelCode` no tiene tabla propia en la Caja de Herramientas de facturación
electrónica (por eso no se le encontró nombre ahí), pero **sí es un código real de OTRA tabla
DIAN** — la tabla de "Responsabilidades Tributarias" del RUT (Formulario 001, casilla 53), que
es una tabla mucho más larga (no de la Caja de Herramientas de facturación, sino del RUT) de la
que la Anexo Técnico solo reutiliza un subconjunto pequeño para este atributo específico.
Confirmado contra dos fuentes externas (Gerencie.com, Actualícese) más el propio RUT del
usuario:

| Código | Nombre real (tabla RUT casilla 53) |
|---|---|
| `00` | No aplica — no es código RUT real (la tabla oficial empieza en `01`); es el valor que el Anexo Técnico instruye usar cuando `@listName` se informa sin régimen específico (FAJ27/CAJ27/DAJ27 etc.: "Opcional, si informado indicar 'No aplica'") |
| `02` | Gravamen a los movimientos financieros (GMF / 4×1000) |
| `03` | Impuesto al patrimonio |
| `04` | Renta y complementario - régimen especial |
| `05` | Renta y complementario - régimen ordinario — **confirmado literalmente en el RUT real del usuario** |

(`01`, "Aporte especial para la administración de justicia", existe en la tabla del RUT pero
queda fuera del set válido para `@listName` según el `.sch` — consistente con que no es una
clasificación de "régimen", es un aporte aparte). `internal/database/seed/tax_regimes.csv`
actualizado con estos nombres reales (antes: "Régimen 00".."05" genéricos); reseed verificado
contra Postgres real vía `GET /catalogs/tax-regimes` y visualmente en el selector del
formulario de empresa del frontend.
- **`liability_codes`** (Responsabilidades fiscales) — antes solo el default `"R-99-PN"` en
  código, sin catálogo real. Se cargaron los 5 códigos con nombre oficial confirmado
  (`responsabilidad-2.1.gc`: Gran Contribuyente, Autorretenedor, Agente de Retención IVA,
  Régimen Simple, y el catch-all `R-99-PJ`) más `R-99-PN` explícitamente, porque el `.sch` de
  2019 NO lo lista (solo `R-99-PJ`) pero el código real lo usa como default desde la sección
  9.29. **Resuelto con evidencia real** (ver más abajo): `R-99-PN` es válido.

**CIIU** (actividad económica, catálogo de la DANE — no de la DIAN, por eso NO tiene tabla):
nuevo campo `IndustryClassificationCodes []string` en `cofacture/domain.Party` (emite
`cbc:IndustryClassificationCode`, confirmado contra
`XSD/common/UBL-CommonAggregateComponents-2.1.xsd` que va ANTES de `PartyIdentification`/
`PartyName` en la secuencia de `cac:Party`, no después) y en `issuers.Issuer` — array libre,
máximo 4 códigos (`ErrTooManyIndustryClassificationCodes`), límite basado en la estructura
real del RUT del usuario (1 actividad principal + 1 secundaria + 2 "otras actividades" = 4),
no en una regla documentada de la DIAN. Solo aplica al emisor (nunca al receptor). El usuario
aportó además un catálogo CIIU completo y real propio
(`https://diegofxm.github.io/ciiu-classification/ciiu.json`, 21 secciones, 88 divisiones, 110
grupos, **419 códigos hoja**) — queda anotado como candidato natural para una futura tabla
`ciiu_codes` en `internal/catalogs` (convertiría el campo de "array libre" a "array validado
contra catálogo real"), no construido todavía.

**`internal/catalogs`** (paquete nuevo, solo lectura, sin `Service` — el `Repository` es el
contrato completo): expone los 11 catálogos vía
`GET /api/v1/catalogs/{departments,municipalities,identification-types,tax-types,
payment-methods,payment-terms,unit-measures,tax-regimes,liability-codes,
dian-document-types,currencies}` (`municipalities` acepta `?department_code=`), protegidos
solo con `Auth` (no `RequireTenant` — son iguales para cualquier usuario). `Repository` es una
interfaz (`PostgresRepository`/`MemoryRepository`, mismo patrón que el resto de dominios) que
también satisface `documents.CatalogPort` estructuralmente.

**`documents.CatalogPort`** — hallazgo de la auditoría: `payment_methods`/`payment_terms` viven
dentro del `payment_means` JSONB, nunca tuvieron FK posible. `documents.Service.validateBase`
ahora valida cada `payment_means[].code`/`payment_method_code` contra estos catálogos antes de
persistir el borrador (`ErrInvalidPaymentTerm`/`ErrInvalidPaymentMethod`) — antes, un código
inválido ahí solo se detectaba al confirmar, con la DIAN rechazando el documento ya con un
número real reclamado. `unit_measures` (lines[].UnitCode) deliberadamente NO se valida así:
ese catálogo sigue incompleto (11 códigos de muestra frente al estándar UN/ECE Rec. 20
completo) — validar contra un catálogo incompleto rechazaría códigos legítimos que la DIAN sí
aceptaría.

**Verificado real, de punta a punta, contra Postgres y la DIAN real**: re-migrado limpio
(conteos exactos: 33 departamentos, 1.122 municipios, 16 tributos, 75 medios de pago, 2 formas
de pago, 5 regímenes, 6 responsabilidades, sin tocar identification_types/dian_document_types/
unit_measures/currencies a propósito). Los 11 endpoints de catálogo confirmados vía curl. Una
factura real (NIT 6382356, persona natural, mismo emisor de la Fase 1.7) con `tax_regime_code:
"00"`, `industry_classification_codes: ["4711"]` (código CIIU real, "Comercio al por menor...")
y `liability_codes: ["R-99-PN"]` (el default sin cambios) fue **autorizada por la DIAN real**:
`StatusCode "00"`, `"Procesado Correctamente."`, `"La Factura electrónica SETP990300000, ha
sido autorizada."` — resuelve definitivamente la duda `R-99-PN` vs `R-99-PJ` del `.sch` de
2019: el código que ya usábamos es válido, el Schematron estaba desactualizado en ese punto.
Datos de prueba limpiados de la base real después.

Cobertura de tests nuevos: `issuers.TestRegisterIssuer_Validations` (caso "más de 4 códigos
CIIU"), `issuers.TestRegisterIssuer_FourIndustryClassificationCodes_OK` (límite exacto),
`documents.TestCreateInvoiceDraft_InvalidPaymentMeans` (`fakeCatalogPort` rechazando).

**Corrección posterior, antes del commit** (mismo día): el usuario pidió revisar dos cosas.
(1) La migración nueva se había dejado como `000003_catalogs_extra` separada de
`000002_catalogs` — como todavía no hay datos reales en juego, se unificó en sitio dentro de
`000002_catalogs` en vez de mantener una migración "extra" (mismo criterio ya aplicado toda
la sesión: sin `ALTER`, se corrige de cero mientras se pueda). (2) El nombre de tabla
`formas_pago` había quedado en español por error — se corrigió a `payment_terms` en todo el
código (`internal/catalogs`, `documents.CatalogPort.IsValidPaymentTerm`,
`ErrInvalidPaymentTerm`, ruta `GET /catalogs/payment-terms`, seed CSV, Postman) — confirmado
que es el único identificador de base de datos en español en todo el esquema, revisando los
nombres de columna de las 18 migraciones completas.

**Cierre del hueco de `liability_codes` sin validar** (mismo día, detectado al revisar
visualmente el diagrama de relaciones de la base de datos: `payment_terms`, `unit_measures`,
`liability_codes` y `payment_methods` no tienen línea de FK porque ninguno puede tenerla —
`payment_terms`/`payment_methods` viven dentro de JSONB (`documents.payment_means`),
`liability_codes` es un `TEXT[]` (Postgres no soporta FK contra cada elemento de un array), y
`unit_measures` queda deliberadamente sin validar porque el catálogo sigue incompleto). De los
cuatro, `liability_codes` era el único de los tres "huérfanos reales" que NUNCA se validó en
código — la migración ya tenía el comentario de que se validaría en `issuers.Service`, pero
nunca se implementó. Cerrado así:

- `catalogs.Repository` (+ `PostgresRepository`/`MemoryRepository`) ganó
  `IsValidLiabilityCode(ctx, code)`, mismo patrón que `IsValidPaymentTerm`/
  `IsValidPaymentMethod`.
- `issuers.CatalogPort` (nuevo, `internal/issuers/ports.go`) y `customers.CatalogPort` (nuevo,
  `internal/customers/ports.go`): interfaces angostas con ese único método, cumplidas
  estructuralmente por `*catalogs.PostgresRepository`/`*catalogs.MemoryRepository`.
  `issuers.Service`/`customers.Service` ganaron un campo `catalogs CatalogPort` (constructores
  `New(...)` con un parámetro más); `validateIssuer`/`validateParty` pasaron de función libre a
  método de `*Service` para poder consultar el catálogo (mismo motivo y mismo patrón que
  `documents.validateBase` cuando ganó la validación de `payment_means`).
- `documents.CatalogPort` también ganó `IsValidLiabilityCode` — `documents.Service.validateBase`
  ahora valida `customer.LiabilityCodes` (el pass-through del request, el único de los tres
  puntos de entrada — emisor/cliente guardado/cliente del documento — que seguía sin chequeo
  alguno antes de esto; el emisor ya queda cubierto porque se valida al registrarse, y un
  cliente guardado queda cubierto en `customers.Service`).
- Nuevos errores `issuers.ErrInvalidLiabilityCode`/`customers.ErrInvalidLiabilityCode`/
  `documents.ErrInvalidLiabilityCode`, los tres mapeados a 400 en `response.go` desde el
  principio (para no repetir el olvido que sí pasó con `ErrInvalidPaymentTerm`/
  `ErrInvalidPaymentMethod` en la corrección anterior).
- `internal/api/api.go`: `catalogsRepo` ahora se construye primero (antes se construía después
  de `issuerSvc`/`customersSvc`), para poder inyectarlo en los tres constructores.
- Verificado con tests unitarios nuevos en los tres paquetes y, además, real contra Postgres +
  servidor real: un emisor con un código inventado se rechaza con 400 y el mensaje esperado; un
  emisor con `O-15` (código real del catálogo) se acepta con 201. Datos de prueba limpiados de
  la base real después.
