# Arquitectura Profesional DIAN en Go (COFACTURE + API-DIAN)

> Verificado contra el Anexo Técnico de Factura Electrónica de Venta v1.9 (Resolución DIAN 000165/2023).

## 0. Principio rector

API-DIAN existe **únicamente** para emitir documentos electrónicos válidos ante la DIAN: construir, firmar, numerar, transmitir y rastrear el estado de Invoice / CreditNote / DebitNote. No es un CRM, no es un ERP, no es un sistema de reportes.

Todo lo que no sea estrictamente necesario para que un documento sea legalmente válido ante la DIAN se delega a otros servicios (ver sección 8). Mismo principio que el Ledger de `core-bank`: el núcleo no sabe nada de KYC ni de redes de pago externas — aquí, API-DIAN no sabe nada de CRM ni de catálogos de producto.

El proyecto hermano `api-dian` (Fiber + GORM, con CRUD de Company/Customer/Product) queda **retirado**. Todo se reconstruye aquí, delgado desde el día uno.

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

### 1.2 API-DIAN (Aplicación / Orquestador)
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

## 4. Estructura del Proyecto API-DIAN

> Igual que en la sección 2: esto refleja lo que ya existe (Fase 2.1-2.2), no solo el plan
> original. Mismos patrones que `core-bank` (config/logger/database/server), sin `cache` ni
> `telemetry` — no hacen falta todavía aquí.

```
api-dian/
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
│   ├── auth/                            # usuarios + login + JWT — "un usuario = un emisor"
│   │   ├── service.go                     # Register (crea emisor+usuario juntos) / Login
│   │   ├── token.go                        # TokenIssuer: firma/valida JWT (HS256)
│   │   └── password.go                      # hash/verify con bcrypt
│   └── api/                             # capa HTTP — primera vez que esto se expone por red
│       ├── api.go                        # New()/NewFromServices(), Handler(), rutas
│       ├── dto.go                         # contrato JSON, independiente de domain.* de cofacture
│       ├── handler_auth.go                 # register/login (únicas rutas públicas)
│       ├── handler_issuers.go              # issuers/me + numbering-ranges (protegidas)
│       ├── handler_documents.go             # invoices/credit-notes/debit-notes/documents
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
        ├──────────────┐
        ↓              ↓
    documents         auth      (documents usa issuers+numbering+cofacture; auth usa SOLO
        ↓              ↓         issuers — RegisterIssuer al crear el primer usuario admin,
        └──────┬───────┘         mismo patrón de "ports" angostos que documents)
               ↓
              api                 (usa documents + issuers + auth; expone HTTP — middleware.Auth
                ↓                  protege todo excepto /auth/register y /auth/login)
             server                 (conecta todo + /health — ya existe desde la Fase 2.1)
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
**otro sistema/servicio**, no dentro de `api-dian` — el mismo patrón que ya usa `core-bank`
(el ledger no tiene KYC completo de sus clientes, asume que vive afuera). `issuers` y un
eventual `companies` externo describirían la misma empresa real desde dos ángulos distintos
("bounded context"): no es duplicación accidental, es separación de responsabilidades a
propósito. `api-dian` no debe crecer para absorber lo que le corresponde a un CRM.

`Customer` y los `Items` de una factura **no son entidades propias**: llegan embebidos en el payload de creación de cada documento y se persisten como snapshot dentro del documento emitido (porque eso es lo que la ley exige conservar), sin CRUD ni reglas de negocio propias sobre ellos.

---

## 5. Flujo API-DIAN

```
HTTP → Handler → Service → Repository → COFACTURE → DIAN
```

---

## 6. Endpoints

> Esto ya existe (Fase 2.7-2.9), no es solo el plan. `POST /invoices`/`credit-notes`/
> `debit-notes` construyen, firman y — si el ambiente lo permite — envían en la misma llamada
> (no hay un `/send` separado: separar "crear" de "enviar" no tenía un caso de uso real una
> vez que `cofacture` ya hace las dos cosas en un solo pipeline).
>
> Todo excepto `/auth/*` y `/health` exige `Authorization: Bearer <token>`. "Un usuario = un
> emisor" (sección 9.17): ningún endpoint recibe `issuer_id` del cliente — siempre se toma del
> token, nunca de algo que el cliente pueda elegir. Por eso ya no hay `POST /issuers` público
> ni `{id}` en el path de `/issuers` o de `numbering-ranges` al crear: el emisor del usuario
> autenticado es implícito.

```
POST /api/v1/auth/register                        # crea el emisor Y su primer usuario admin (público)
POST /api/v1/auth/login                           # inicia sesión, devuelve el token (público)

GET  /api/v1/issuers/me                           # consultar el emisor propio (nunca expone secretos)
POST /api/v1/numbering-ranges                     # registrar rango del emisor propio
GET  /api/v1/numbering-ranges                     # listar mis rangos (?dian_document_type_code=, sin paginar)
GET  /api/v1/numbering-ranges/{id}                # consultar rango (debe ser del emisor propio; si no, 404)
POST /api/v1/invoices                             # construir + firmar (+ enviar si aplica)
POST /api/v1/credit-notes
POST /api/v1/debit-notes
GET  /api/v1/documents                            # listar mis documentos (filtros + ?limit=&offset=)
GET  /api/v1/documents/{id}                       # documento emitido (debe ser del emisor propio; si no, 404)
GET  /health
```

---

## 7. Regla de oro

- COFACTURE no conoce HTTP ni DB.
- API-DIAN no conoce firma ni XML.
- API-DIAN no es un CRM ni un ERP: no gestiona clientes ni productos como catálogos propios
  (llegan como snapshot en el payload de cada documento). Usuarios/auth sí viven aquí —
  replanteo explícito de 2026-06-21 (sección 9.17): API-DIAN es el backend completo, no una
  pieza dentro de un ecosistema con un servicio de identidad aparte.

---

## 8. Fuera de alcance (delegado a otros servicios)

Decisión consciente, no descuido — si se necesitan, se integran como servicios externos consumidos vía API, nunca como módulos internos:

| Función | Por qué no vive aquí |
|---|---|
| CRM de Companies/Customers (contactos, direcciones, KYC) | No lo exige la DIAN; el XML solo necesita un snapshot al momento de emitir. **Decisión 2026-06-21: diferido, no descartado** — se construye cuando el frontend real necesite autocompletar, no antes (sección 9.18); el orquestador ya funciona sin él |
| Catálogo de Productos / Inventario (precios, stock) | Mismo motivo que Customers; los items llegan en el payload de la factura. Misma decisión: diferido hasta que haya un consumidor real |
| ~~Usuarios, roles, multi-tenant auth~~ | **Ya no aplica — construido en `internal/auth` (Fase 2.9)**. Replanteo de 2026-06-21: API-DIAN es el backend completo, no hay servicio de identidad externo en esta topología |
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

### 9.5 `cofacture` y `api-dian` siguen siendo dos módulos Go independientes

Cada uno con su propio `go.mod` y su propio repo git — `api-dian` consume a `cofacture` como
dependencia (`import "github.com/diegofxm/cofacture/..."`), nunca al revés. Es el mismo patrón
que usar cualquier paquete externo de Go, salvo que por ahora no está publicado en GitHub.

Mientras los dos se desarrollan en paralelo, `project-ubl/go.work` (no es un módulo Go en sí,
solo el archivo de workspace) le dice al compilador que resuelva `github.com/diegofxm/cofacture`
contra la carpeta local `./cofacture` en vez de ir a buscarlo a un remoto — así `api-dian` ve
los cambios de `cofacture` al instante, sin `git push`, sin tags, sin que `cofacture` necesite
siquiera tener un remoto configurado. El día que se quiera congelar una versión estable para
desplegar de verdad, se publica `cofacture` en un repo real con un tag (`v0.1.0`...), se
agrega `require github.com/diegofxm/cofacture v0.1.0` al `go.mod` de `api-dian`, y se quita (o
se deja de usar) el `go.work` — ahí es donde Go vuelve a resolver la dependencia "de verdad".

### 9.6 Fase 2 (`api-dian`) — en marcha

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

El primer paquete de `api-dian` que importa `cofacture` directamente. `Service.IssueInvoice`
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

### 9.17 Replanteo de alcance: API-DIAN como backend completo + `internal/auth` propio

El usuario corrigió (2026-06-21) la aplicación estricta de la filosofía de `core-bank` a este
proyecto: `core-bank` es minimalista (sin "listar todo", sin auth, sin CRM completo) porque
asume que existen otros servicios alrededor para esas responsabilidades. `api-dian` no tiene
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
  "quién firmó el token" (api-dian) con "qué emisor DIAN administra este usuario".
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

### 9.20 Próximo paso

Sin tareas pendientes explícitas en este momento — Fase 2 (`api-dian`) cubre hoy: bootstrap,
catálogos, emisores, numeración, documentos (Invoice/CreditNote/DebitNote, construir+firmar+
enviar), auth con aislamiento entre tenants, y listados. Lo diferido (Customers/Products,
Documento Soporte/RADIAN/Nómina, PDF/Notificaciones) tiene su trigger de retomado explícito en
las secciones 9.18 y 8 — no son items "olvidados", son decisiones de alcance con criterio
escrito de cuándo reabrirlas.
