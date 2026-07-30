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

> Refleja el estado actual (Fase 2 completa hasta 2.15+), no solo el plan original.
> Mismos patrones que `core-bank` (config/logger/database/server), sin `cache` ni
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
│   ├── documents/                      # orquesta cofacture (FE/NC/ND/DS) — el único paquete
│   │                                    # que importa cofacture directamente; usa pdf+email+nit
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
│   ├── suppliers/                         # catálogo de proveedores no obligados (SNO para DS) —
│   │                                      # mismo patrón que customers; ver 9.53/9.56
│   ├── pdf/                             # generación PDF en memoria con maroto v2 (ver 9.39);
│   │                                      # recibe domain.Invoice + IssuerForDocument, devuelve []byte
│   ├── email/                           # cliente SMTP con go-mail (ver 9.40); SMTPSender
│   │                                      # satisface documents.EmailPort sin que email importe documents
│   ├── nit/                             # ComputeCheckDigit — módulo 11, dígito verificador NIT
│   └── api/                             # capa HTTP — 66 rutas (ver sección 6)
│       ├── api.go                        # New()/NewFromServices(), Handler(), registerRoutes()
│       ├── dto.go                         # contrato JSON, independiente de domain.* de cofacture
│       ├── handler_auth.go                 # register/login/PUT me
│       ├── handler_public.go               # rutas públicas sin auth: GET /public/issuers/*, POST /public/issuers/{id}/customers
│       ├── handler_issuers.go              # issuers/me + numbering-ranges + PATCH profile + logo + software
│       ├── handler_documents.go             # invoices/credit-notes/debit-notes/documents + pdf/xml/email
│       ├── handler_support_documents.go     # POST/PUT /support-documents (DS — CUDS-SHA384, ver 9.55)
│       ├── handler_customers.go              # CRUD de customers
│       ├── handler_products.go                # CRUD de products
│       ├── handler_suppliers.go                 # CRUD de suppliers (para DS, ver 9.56)
│       ├── middleware/                       # RequestID/Logging/Recovery/Auth/LoginRateLimit
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
        ├──────────────┬──────────────┬──────────────┐
        ↓              ↓              ↓              ↓
       auth        customers       products       suppliers    (todos usan solo database;
        ↓              ↓              ↓              ↓        issuer_id es FK de tabla,
        │              │              │              │        no dependencia Go a issuers)
        │         nit  pdf  email     │              │
        │              ↓              │              │
        │          documents ←────────┴──────────────┘       (documents usa issuers+numbering+
        │              ↓                                       cofacture+customers+suppliers+
        └──────┬───────┘                                       pdf+email+nit; ver 9.23/9.55)
               ↓
              api          (usa todos los servicios; expone HTTP — middleware.Auth
                ↓           protege todo excepto /auth/*, /public/*, /health)
             server          (conecta todo + /health)
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
# ── Públicas (sin autenticación) ────────────────────────────────────────────────────────────
POST /api/v1/auth/register                        # crea SOLO el usuario, sin empresa (ver 9.32)
POST /api/v1/auth/login                           # inicia sesión; autoselecciona empresa si hay exactamente una
GET  /api/v1/public/issuers/{id}                  # datos públicos del emisor (nombre, logo, NIT) — para widget externo
GET  /api/v1/public/issuers/{id}/logo             # logo del emisor en binario
POST /api/v1/public/issuers/{id}/customers        # crear cliente desde portal público (sin auth)

# ── Autenticadas sin empresa activa (solo JWT válido) ───────────────────────────────────────
PUT  /api/v1/auth/me                              # actualizar nombre/email/contraseña del usuario autenticado
POST /api/v1/issuers                              # crear empresa nueva y vincularla (rol owner)
GET  /api/v1/issuers                              # listar las empresas a las que el usuario tiene acceso
POST /api/v1/issuers/{id}/select                  # reemitir el token con esa empresa como activa

GET  /api/v1/catalogs/departments                 # catálogos DIAN/DANE — iguales para cualquier usuario
GET  /api/v1/catalogs/municipalities              # acepta ?department_code= para filtrar
GET  /api/v1/catalogs/identification-types
GET  /api/v1/catalogs/tax-types
GET  /api/v1/catalogs/payment-methods
GET  /api/v1/catalogs/payment-terms
GET  /api/v1/catalogs/unit-measures
GET  /api/v1/catalogs/tax-regimes
GET  /api/v1/catalogs/liability-codes
GET  /api/v1/catalogs/dian-document-types
GET  /api/v1/catalogs/currencies
GET  /api/v1/catalogs/item-standards
GET  /api/v1/catalogs/ciiu-codes

# ── Autenticadas con empresa activa (JWT + RequireTenant) ───────────────────────────────────
GET    /api/v1/issuers/me                         # datos del emisor activo (nunca expone secretos)
PUT    /api/v1/issuers/me                         # actualizar credenciales DIAN, logo (base64), software
PATCH  /api/v1/issuers/me/profile                 # actualizar perfil público: razón social, dirección, CIIU, etc.
GET    /api/v1/issuers/me/logo                    # logo del emisor autenticado en binario (Content-Type real)
DELETE /api/v1/issuers/me/logo                    # quitar el logo del emisor
DELETE /api/v1/issuers/me/software                # quitar credenciales de software DIAN
DELETE /api/v1/issuers/me/ne-software             # quitar credenciales de software de Nómina Electrónica
DELETE /api/v1/issuers/me/certificate             # quitar certificado digital

POST   /api/v1/numbering-ranges                   # registrar resolución de numeración del emisor activo
GET    /api/v1/numbering-ranges                   # listar rangos del emisor (?dian_document_type_code=)
GET    /api/v1/numbering-ranges/{id}              # consultar rango (404 si no es del emisor)
DELETE /api/v1/numbering-ranges/{id}              # desactivar rango (no destruye consecutivos ya emitidos)
PUT    /api/v1/numbering-ranges/{id}/activate     # reactivar rango desactivado

GET    /api/v1/dian/verify-acquirer               # consultar NIT en RUES/DIAN (adquiriente antes de facturar)

POST   /api/v1/invoices                           # crear borrador FE (sin reclamar número, sin firmar)
PUT    /api/v1/invoices/{id}                      # reemplazar borrador FE
POST   /api/v1/credit-notes                       # crear borrador NC
PUT    /api/v1/credit-notes/{id}                  # reemplazar borrador NC
POST   /api/v1/debit-notes                        # crear borrador ND
PUT    /api/v1/debit-notes/{id}                   # reemplazar borrador ND
POST   /api/v1/support-documents                  # crear borrador DS (CUDS-SHA384, ver 9.55)
PUT    /api/v1/support-documents/{id}             # reemplazar borrador DS

POST   /api/v1/documents/{id}/confirm             # reclamar número + firmar + enviar (SendTestSetAsync o SendBillSync)
DELETE /api/v1/documents/{id}                     # eliminar borrador (409 si ya fue confirmado)
GET    /api/v1/documents                          # listar documentos del emisor (filtros: tipo, estado, fecha, ?limit=&offset=)
GET    /api/v1/documents/{id}                     # consultar documento (404 si no es del emisor)
GET    /api/v1/documents/{id}/pdf                 # PDF en memoria — nunca escribe a disco (ver 9.39)
GET    /api/v1/documents/{id}/xml                 # XML firmado original del documento
POST   /api/v1/documents/{id}/send-email          # reenviar por correo al adquiriente (ver 9.40)

POST   /api/v1/customers                          # catálogo de adquirientes (conveniencia, ver 9.21)
GET    /api/v1/customers
GET    /api/v1/customers/{id}
PUT    /api/v1/customers/{id}
DELETE /api/v1/customers/{id}

POST   /api/v1/products                           # catálogo de ítems/servicios (conveniencia, ver 9.21)
GET    /api/v1/products
GET    /api/v1/products/{id}
PUT    /api/v1/products/{id}
DELETE /api/v1/products/{id}

POST   /api/v1/suppliers                            # catálogo de proveedores no obligados para DS (ver 9.56)
GET    /api/v1/s
GET    /api/v1/suppliers/{id}
PUT    /api/v1/suppliers/{id}
DELETE /api/v1/suppliers/{id}

GET  /api/v1/stats/billing                         # métricas de facturación del emisor activo (ver 9.57)

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
| ~~PDF / representación gráfica~~ | **Ya no aplica — construido en `internal/pdf` con maroto v2 (sección 9.39)**. Generación en memoria, sin disco, servida por `GET /documents/{id}/pdf` |
| ~~Notificaciones (email al receptor)~~ | **Ya no aplica — construido en `internal/email` con go-mail (sección 9.40)**. SMTP propio; `POST /documents/{id}/send-email` reenvía con AttachedDocument UBL |
| ~~Listados de documentos/rangos~~ | **Ya no aplica — `GET /numbering-ranges` y `GET /documents` construidos (Fase 2.9, sección 9.19)**. Esto es CRUD básico del propio orquestador, no analítica — no era delegable a otro servicio |
| ~~Reportes / Dashboard / Analítica (agregaciones, gráficas)~~ | **Ya no aplica — construido en `GET /api/v1/stats/billing` (sección 9.57)**. Tres SQL (`FILTER WHERE` para períodos, `GROUP BY` tipo y mes) con timezone `America/Bogota`. El dashboard en React usa Recharts (series de área 12 meses + barras por tipo). |
| ~~Documento Soporte (CUDS)~~ | **Ya no aplica — construido en `internal/documents` y `cofacture` (sección 9.55)**. DS (tipo 05, CUDS-SHA384) autorizado por la DIAN en habilitación real (StatusCode 00) |
| Eventos RADIAN (Acuse de recibo, Reclamo, ApplicationResponse) | Solo obligatorio si la factura se negocia como título valor — fase futura explícita |
| Nómina Electrónica (CUNE) | Esquema XML distinto al UBL, webservice distinto — proyecto separado, no este |

---

## 9. Estado actual y hoja de ruta

### 9.1 Fase 1 (motor `cofacture`) — completa y validada contra la DIAN real

| Documento DIAN | Construir | Firmar (XAdES) | CUFE/CUDE | Enviado y validado en habilitación real |
|---|---|---|---|---|
| Factura electrónica de venta (01) | ✅ | ✅ | ✅ | ✅ Autorizada (`SETP-990068706`, StatusCode 00) |
| Nota Crédito (91) | ✅ | ✅ | ✅ | ✅ Procesada (referenciando la factura real anterior) |
| Nota Débito (92) | ✅ | ✅ | ✅ | ✅ Aceptada en habilitación real (2026-07-11) |
| AttachedDocument (contenedor para el adquiriente) | ✅ solo Invoice | ✅ (placeholder genérico) | — | ✅ Probado con el `ApplicationResponse` real de la factura autorizada |

El pipeline completo (build → CUFE/CUDE → `SoftwareSecurityCode` → QR → firma XAdES → ZIP → envío SOAP con WS-Security → lectura de respuesta) está probado de punta a punta, no solo contra los ejemplos del anexo.

### 9.2 Pendiente dentro de "Facturación Electrónica" (mismo Anexo 1.9)

| Pendiente | Por qué importa | Prioridad sugerida |
|---|---|---|
| ~~`AttachedDocument` completo (UBL con `ApplicationResponse` embebido) para los tres tipos~~ | **✅ Implementado** — ver sección 9.51. `application_response_xml` se persiste desde la DIAN; `email.go:buildAttachedDocumentXML` construye el sobre UBL firmado cuando el campo está presente, con fallback a XML crudo para documentos anteriores a la migración. | — |
| ~~Documento Soporte (05, CUDS)~~ | **✅ Implementado** — ver sección 9.55. CUDS-SHA384, roles invertidos (SNO=proveedor/ABS=emisor), autorizado por la DIAN en habilitación real (StatusCode 00). | — |
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
| 2.12 | `internal/pdf` — generación PDF en memoria con maroto v2 | ✅ Verificado, ver 9.39 |
| 2.13 | `internal/email` — SMTP con go-mail, AttachedDocument UBL al receptor | ✅ Verificado, ver 9.40 |
| 2.14 | `internal/nit` — dígito verificador módulo 11 | ✅ Implementado, ver 9.42 |
| 2.15 | Documento Soporte (DS tipo 05, CUDS-SHA384) de punta a punta | ✅ Autorizado por la DIAN en habilitación real (StatusCode 00), ver 9.55 |
| 2.16 | `internal/suppliers` — catálogo de proveedores no obligados para DS | ✅ Implementado, ver 9.56 |
| 2.17 | `GET /stats/billing` — métricas de facturación del emisor (KPIs, serie 12m, por tipo) | ✅ Implementado, ver 9.57 |

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
El catálogo CIIU (21 secciones, 88 divisiones, 110 grupos, **419 códigos hoja**) vive en la
tabla `ciiu_codes` de `internal/catalogs` — array validado contra catálogo real, expuesto en
`GET /api/v1/catalogs/ciiu-codes` y poblado via `internal/database/seed/ciiu_codes.csv`.

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

### 9.35 Hoja de ruta de roles y permisos (diseño, sin implementar todavía)

El usuario pidió dejar planeado desde ya un sistema de roles de tres niveles, para no tener
que deshacer nada cuando llegue el momento de construirlo. Decisión explícita: por ahora solo
se documenta la taxonomía y el mecanismo — no se toca código (ni JWT, ni middleware, ni
`user_issuers.role`) hasta que exista un caso real (un contador real que invitar, una
funcionalidad real de planes que vender).

**Nivel 1 — `platform_admin`** (el usuario, dueño de la plataforma apidian misma): control
total — planes, venta de documentos electrónicos, facturación de la plataforma, visibilidad
cruzada de todas las empresas que existan en el sistema. Es un concepto **distinto** de
`RoleOwner` (que es la relación de un usuario con UNA empresa puntual vía `user_issuers`):
`platform_admin` no está ligado a ningún `tenant_id`, opera por encima de todas las empresas.

- Backend: nuevo valor de `auth.User.Role` (junto al actual `RoleAdmin`, que en ese punto pasa
  a significar simplemente "usuario normal registrado", no "administrador" — el nombre quedó
  mal elegido desde el principio porque era el único rol que existía). Asignación manual/
  seed, nunca seleccionable desde el registro público.
- Middleware nuevo `RequirePlatformAdmin`, mismo patrón que `RequireTenant`
  (`internal/api/middleware/tenant.go`) — un helper más en el `handle`/`handleNoTenant` de
  `internal/api/api.go` (ej. `handlePlatform`).
- Rutas nuevas bajo `/api/v1/admin/...`, sin pasar por `RequireTenant` (un platform_admin no
  necesita una empresa activa para administrar la plataforma).
- Frontend: árbol de rutas separado `/admin/*` con su propio layout (NO `DashboardLayout`, que
  asume y exige una empresa activa — un panel de superadmin no opera "dentro" de una empresa).
- Diferido hasta que exista una funcionalidad real de planes/facturación de la plataforma que
  construir — no se reserva esquema de "planes"/"créditos" todavía, ese diseño se hace cuando
  se aborde explícitamente.

**Nivel 2 — `owner`** (dueño/administrador de una empresa): ya existe hoy
(`auth.RoleOwner = "owner"`, asignado en `user_issuers.role` por `CreateIssuerForUser` —
`internal/auth/service.go:120`). Ya tiene de facto acceso completo a su empresa, porque hoy
ningún middleware restringe nada ("cualquier vínculo da acceso completo",
`internal/auth/model.go:37`). A futuro sin cambios de modelo: configuración de la empresa,
software/certificado, rangos de numeración, y la capacidad de invitar/quitar usuarios de su
propia empresa (ver mecanismo de invitación abajo). Puede crear y administrar varias empresas
propias — esto YA funciona (Fase 9.32, multi-empresa vía `user_issuers` N:M).

**Nivel 3 — `accountant`** (u otro staff de una empresa): no existe todavía. Cuando se
construya: nuevo valor de `user_issuers.role` junto a `"owner"`; acceso al día a día
(documentos, clientes, productos) sin acceso a configuración/certificado/secretos de la
empresa ni a invitar/quitar otros usuarios. La taxonomía queda abierta a un nivel intermedio
adicional más adelante (ej. "viewer" de solo lectura) sin requerir un cambio de esquema —
agregar un rol nuevo es agregar un valor de enum + los chequeos donde haga falta, no una
migración.

**Mecanismo de enforcement, boceto para cuando se construya** (no implementado):
- `internal/auth/token.go`: el struct `claims{UserID, TenantID, jwt.RegisteredClaims}` ganaría
  un campo `Role` (el rol del usuario en `TenantID`) — mismo patrón exacto que `TenantID`, se
  reemitiría en `Login`/`SelectIssuer`/`CreateIssuerForUser`. Implica que un cambio de rol solo
  se refleja tras un nuevo login/selección de empresa, igual que un cambio de empresa activa
  hoy — compromiso aceptable, no se construye revocación inmediata.
- Middleware nuevo `RequireIssuerRole(min)`, mismo patrón que `RequireTenant`.
- `POST /api/v1/issuers/me/members` (solo owner): vincula a un usuario **ya registrado** por
  correo con un rol. `GET /api/v1/issuers/me/members` (listar), `DELETE .../members/{userID}`
  (quitar). Deliberadamente NO se soporta invitar a alguien sin cuenta todavía — no hay
  infraestructura de envío de correo en el proyecto; ese flujo se construye junto con esa
  infraestructura, no antes.

**Sucursales — aclaración de alcance** (el usuario confirmó que se refiere a una misma empresa
facturando desde varias ciudades/sedes, no a empresas separadas):
- Empresas legalmente separadas, cada una con su propio NIT (ej. una franquicia) — esto **ya
  está completamente soportado** hoy vía multi-empresa (`user_issuers` N:M, Fase 9.32). No hace
  falta construir nada nuevo para este caso.
- Una misma empresa (un solo NIT) facturando desde varias sedes/ciudades — **ya parcialmente
  soportado** hoy: cada sede puede tener su propio rango de numeración con su propio `prefix`
  (`numbering_ranges.prefix`, ej. "BOG"/"MED"), que es lo único que la DIAN exige distinguir
  por sede en la factura misma. Una entidad "sede" propia (con su propia dirección/contacto,
  más allá del prefijo) es una mejora posible a futuro, **solo si** el enfoque de
  solo-prefijo resulta insuficiente en la práctica — no se modela ahora.

### 9.36 `issuerResponse` gana `has_software_credentials`/`has_certificate`

Cambio chico, motivado por el frontend (Configuración → Empresa, ver
`docs/frontend-architecture.md` Fase 1.7): `issuerResponse` (`internal/api/handler_issuers.go`)
nunca expuso, ni expondrá, los secretos (`software_pin`/`certificate`/`certificate_password`)
de vuelta — pero antes tampoco había forma de saber si ya estaban configurados o no. Se
agregaron dos booleanos calculados (`iss.SoftwareID != "" && iss.SoftwarePIN != ""`,
`len(iss.Certificate) > 0 && iss.CertificatePassword != ""`), sin tocar el dominio ni el
esquema — propagan automáticamente a todas las respuestas que ya pasaban por
`issuerToResponse` (`GET/PUT /issuers/me`, y el campo `issuer` de `authResponse` en
login/register/crear empresa/seleccionar empresa).

### 9.37 El backend calcula, no confía — aritmética de líneas y nombres derivados del catálogo

Motivado por construir el formulario de Factura Electrónica: al revisar cómo
`documents.Service` iba a recibir las líneas, se encontró que `computeTotals`/`aggregateTaxes`
eran **pass-through puro** — solo sumaban `line_extension_cents`/`taxes[].tax_amount_cents` que
ya vinieran calculados en la petición, sin verificar que de verdad fueran cantidad×precio o
impuesto×porcentaje. Cualquier consumidor de la API (este frontend u otro) tenía que
reimplementar esa aritmética correctamente y sin ningún tipo de red de seguridad del lado del
servidor — para documentos legales firmados y enviados a la DIAN, ese es un riesgo real, no
cosmético.

Buscando el mismo patrón en el resto de la API se encontró una anomalía hermana, presente desde
antes: `tax_scheme_code`/`tax_scheme_name` (issuers, customers) y `tax_type_code`/
`tax_type_name` (products) eran pares que el cliente mandaba por separado, sin que el backend
verificara que el nombre correspondiera de verdad al código — la única corrección que existía
era del lado del frontend (un `<select>` que derivaba el nombre al elegir el código), no en el
backend. Mismo principio en los dos casos: el backend debe derivar/calcular valores que
dependen de otros campos, nunca confiar en que el cliente los mande coherentes.

**Nombres derivados del catálogo, no del cliente:**
- `catalogs.Repository` ganó `GetTaxTypeName(ctx, code) (name string, found bool, err error)`
  — mismo patrón que `IsValidLiabilityCode` (un booleano de "se encontró", el llamador decide
  qué error de dominio lanzar).
- `issuers`/`customers`: `CatalogPort` ganó ese método; `createIssuerRequest` y `partyDTO`
  dejaron de ser fuente de verdad para `tax_scheme_name` — `issuers.Service.RegisterIssuer`/
  `customers.Service.CreateCustomer`/`UpdateCustomer` lo derivan de `tax_scheme_code` justo
  después de aplicar los defaults de siempre. `createIssuerRequest` perdió el campo por
  completo (issuers nunca lo expuso en la respuesta); `partyDTO` lo conserva para la respuesta
  de customers (sigue siendo información real, solo que ya no se confía en lo que llega).
- `products`: no tenía ningún `CatalogPort` — se agregó (`internal/products/ports.go`, nuevo),
  igual que `New(repo, catalogPort)`. `productRequest` perdió `tax_type_name`.
- `documents`: `applyCustomerDefaults` (ya derivaba `EntityTypeCode`/`TaxSchemeCode`/
  `LiabilityCodes`) ganó la derivación de `TaxSchemeName` en el mismo lugar, así que cubre
  tanto un cliente guardado (ya viene correcto desde `customers`) como uno capturado a mano al
  facturar.
- Nuevos errores `ErrInvalidTaxSchemeCode`/`ErrInvalidTaxTypeCode` por paquete, mapeados a 400.

**Aritmética de líneas, no pass-through:** `internal/documents` ganó un tipo nuevo,
`LineInput` (`description/quantity/unit_code/unit_price_cents/item_code/item_type_*/
tax_type_code/tax_percent` — sin `line_extension_cents` ni `taxes[]`), que reemplaza a
`domain.Line` como tipo de **entrada** en `IssueInvoiceRequest.Lines`/`IssueNoteRequest.Lines`.
`domain.Line` no cambió — sigue siendo la forma ya calculada que cofacture necesita para
construir el XML; este cambio vive enteramente en `internal/documents`, sin tocar cofacture.
`(s *Service) linesFromInput(ctx, []LineInput) ([]domain.Line, error)` (nuevo,
`internal/documents/lines.go`) calcula `LineExtensionCents = round(Quantity × UnitPriceCents)`
y, si `TaxTypeCode != ""`, resuelve el nombre vía `GetTaxTypeName` y calcula
`TaxAmountCents = round(LineExtensionCents × TaxPercent / 100)` — soporta 0 o 1 impuesto por
línea (el caso común; la DIAN permite más, pero es el caso avanzado que se agrega cuando haga
falta de verdad). `CreateInvoiceDraft`/`UpdateInvoiceDraft`/`noteDraftFromRequest` (ahora método,
ya que necesita `ctx`+catálogo) llaman esto antes de `computeTotals` — que sigue exactamente
igual, ya hacía bien su parte (sumar), solo agregaba lo que ya estaba mal calculado más abajo.

En `apidian/internal/api/dto.go`, `lineDTO` (forma de salida, ya calculada — la sigue usando
`documentResponse`) se acompañó de un `lineInputDTO` nuevo (forma de entrada, sin aritmética);
`linesToDomain` se reemplazó por `linesToInput`. El contrato JSON de una línea al crear/editar
un borrador quedó simétrico con `products.Product` (precio + impuesto por defecto, sin
aritmética) — no por casualidad, es la misma idea aplicada en los dos lugares.

Verificado: `go test ./...` limpio en `apidian` (incluye los fixtures de
`internal/documents/service_test.go`, que pasaron de líneas pre-calculadas a `LineInput`); y
con un navegador real — crear empresa/cliente/producto eligiendo un impuesto distinto al
default ("01" IVA) y confirmar contra Postgres que el nombre persistido es el real del
catálogo, no algo que el frontend mandara (porque ya no manda nada: ver
`docs/frontend-architecture.md`).

### 9.38 Factura Electrónica end-to-end contra la DIAN real — `test_set_id` huérfano en el rango ya registrado

Al verificar el nuevo botón "Confirmar y enviar" (frontend, ver `docs/frontend-architecture.md`)
contra el emisor real del usuario (NIT 6382356, rango `SETP` 990000000–995000000 ya
registrado en sesiones anteriores), el primer intento devolvió `StatusCode "2"`/`IsValid
false` ("Set de prueba ... se encuentra Aceptado") — el mismo síntoma ya documentado en la
sección 9.10. La causa: ese rango **todavía tenía `test_set_id` cargado** desde antes de que
existiera `SendBillSync` (sección 9.14/9.15) — `finalizeAndSend` sigue enrutando por
`TestSetID presente → SendTestSetAsync` aunque ese camino ya no sirva para nada (el set de
pruebas oficial está cerrado desde hace varias fases), en vez de `SendBillSync` (el camino que
la 9.14 ya probó que SÍ sigue funcionando contra habilitación con el set cerrado).

No hay `PUT /numbering-ranges/{id}` (nunca hizo falta uno hasta ahora) — se corrigió
directamente en la base de datos, mismo criterio que el fix del emisor real en la sección
9.29. Al hacerlo apareció un bug real de paso: poner `test_set_id = NULL` (en vez de `''`)
rompió `GET /numbering-ranges` con un 500 — `numbering.NumberingRange.TestSetID` es un
`string` plano, sin envoltorio nullable, y `internal/numbering/postgres.go` lo escanea
directo (`&nr.TestSetID`) asumiendo que la columna nunca es `NULL` a nivel de fila (solo
`''`/vacío) aunque la columna en sí no tiene `NOT NULL`. Corregido a `''`. **No se tocó
código** — es un recordatorio de que esa columna admite `NULL` en el esquema pero el código
nunca lo espera; si alguna vez se agrega una migración para esa tabla, vale la pena agregarle
`NOT NULL DEFAULT ''` para que esto no se pueda repetir.

Con el rango ya corregido, la factura de prueba se confirmó de nuevo: la DIAN la **autorizó**
de verdad — `StatusCode "00"`, `"Procesado Correctamente."`, `"La Factura electrónica
SETP990000001, ha sido autorizada."` — construida/firmada/enviada/consultada/mostrada de
punta a punta a través de la UI nueva. El número 990000001 había sido reclamado y liberado una
vez antes (rechazo real de la sección 9.30, ya corregido); quedó consumido de verdad esta vez.
Verificación hecha con un usuario de prueba vinculado por `user_issuers` al emisor real (sin
tocar la cuenta ni el borrador propios del usuario, que seguían en uso en paralelo) — limpiado
todo lo propio de la verificación al terminar.

### 9.39 Representación gráfica (PDF) — `internal/pdf`, generada en memoria, nunca a disco

Siguiente pieza del ciclo de Factura (después de construcción + transmisión): el usuario pidió
la representación gráfica como módulo propio del backend, sin escribir nada a disco — se
genera al vuelo en cada petición y se sirve directo. Entregó un mini-paquete de referencia
(`docs/reference/maroto/`, no incorporado a este proyecto) con dos prototipos: uno a mano con
`gofpdf` (posicionamiento absoluto, `SetXY` por todas partes) y uno empezado con **maroto v2**
(grid de filas/columnas). Se evaluaron ambos más `chromedp` (HTML/CSS vía Chrome headless) y se
eligió **maroto v2**: mismo resultado visual profesional que `gofpdf` con una fracción del
código (el prototipo de referencia necesitaba ~700 líneas para una factura completa; el
encabezado equivalente en maroto tomaba ~180), sin la fragilidad de reajustar coordenadas en
cascada cada vez que cambia un espaciado, y sin agregar una dependencia de binario externo
(Chromium) a un proyecto que hasta ahora es 100% Go puro en tiempo de ejecución (`cofacture`
firma con la librería estándar, no un binario externo — mismo criterio aquí).

**`internal/pdf` (nuevo paquete)** — deliberadamente no conoce `documents`/`issuers`/
`cofacture`, mismo principio de puertos angostos que el resto de `apidian`:
- `InvoiceInput`: struct plano con exactamente lo que la plantilla necesita (emisor incluido
  logo opcional, cliente, líneas ya con montos calculados — no se recalcula nada acá, eso ya
  lo hizo `documents.Service` —, totales, impuestos agregados, forma/medio de pago ya
  resueltos a nombre legible, datos de la resolución/rango, y `IsDraft bool` que decide si se
  muestra CUFE/QR/número reales o "BORRADOR").
- `BuildInvoicePDF(InvoiceInput) ([]byte, error)`: arma el documento con maroto (header con
  logo+QR, barra CUFE o "BORRADOR — PENDIENTE DE CONFIRMACIÓN", bloque cliente/factura a dos
  columnas, tabla de items vía `list.Build` — pagina sola si hay muchas líneas, algo que la
  versión a mano con `gofpdf` no tenía —, totales, desglose de impuestos, total en letras, pie
  con el rango autorizado) y devuelve `document.GetBytes()` — nunca `Save()`.
- `amountwords.go`: conversión número→letras en español para el "Son: ... PESOS" — no existía
  en el proyecto. Verificado contra el mismo monto que ya había producido el mini-paquete de
  referencia (195001 → "CIENTO NOVENTA Y CINCO MIL UNO PESOS") como ancla de regresión.

**Logo del emisor** — campo nuevo en `issuers` (`Logo []byte`/`LogoContentType string`,
ambos opcionales, no son secretos, no se cifran). La migración `000003_issuers` se **editó
directamente** (no se agregó una nueva con `ALTER TABLE`) porque el usuario vació la base de
datos justo antes de esta fase para poder hacerlo así — mismo criterio que la sección 9.31
cuando no hay datos reales en juego. `PUT /issuers/me` gana `logo_base64`/`logo_content_type`
(mismo patrón *nil-no-toca* que `certificate_base64`); `issuerResponse` gana `has_logo` (no
el logo en sí, igual criterio que `has_certificate`); nuevo `GET /issuers/me/logo` sirve la
imagen en crudo. `ErrInvalidLogoContentType` solo acepta `"png"/"jpg"/"jpeg"` (los que maroto
soporta).

**Nombres legibles para el PDF** — `catalogs.Repository` ganó `GetPaymentTermName`/
`GetPaymentMethodName`/`GetIdentificationTypeName` (mismo patrón que `GetTaxTypeName` de la
Fase 0; se extrajo un helper `getName(table, code)` compartido por los cuatro, segundo caso
real de uso del mismo patrón). `documents.CatalogPort` los reutiliza para que la representación
gráfica muestre "Contado"/"Transferencia Débito Bancaria"/"Cédula de Ciudadanía" en vez de los
códigos numéricos crudos.

**`documents.Service.RenderInvoicePDF(ctx, issuerID, id) ([]byte, error)`** (nuevo,
`internal/documents/pdf.go`) — mismo rol orquestador que ya cumple este Service para el XML vía
`cofacture`: carga el documento (mismo chequeo de pertenencia al emisor que el resto de
`/documents`), el emisor (`IssuerPort.GetIssuer`, ya existía) y el rango
(`NumberingPort.GetRange`, ya existía) — **cero puertos nuevos** para esos tres, reusa
`aggregateTaxes(d.Lines)` (ya existía, para el desglose de impuestos del pie) — y le delega la
construcción real a `internal/pdf`. Solo Invoice por ahora
(`ErrPDFNotSupportedForDocumentType` en cualquier otro tipo) — Nota Crédito/Nota Débito se
agregan cuando el ciclo de Factura esté probado de punta a punta (PDF + correo).
`GET /api/v1/documents/{id}/pdf` (`internal/api`) sirve los bytes con `Content-Type:
application/pdf`/`Content-Disposition: inline` — nada se guarda a disco en ningún punto.

**Bug real encontrado verificando con un navegador real**: cuando un rango de numeración no
tiene `range_to` (nil — válido, `numbering.NumberingRange.RangeTo` es `*int64` justamente
porque CreditNote/DebitNote no tienen tope impuesto por la DIAN, ver sección del modelo), el
pie del PDF mostraba literalmente *"Rango autorizado desde QAPDF1 hasta **QAPDF0**"* — un dato
inventado, porque `InvoiceInput.RangeTo` era `int64` (no `*int64`), perdiendo la distinción
"sin tope" vs "tope es 0". Corregido: `RangeTo` pasó a `*int64` en `InvoiceInput`, y el texto
del pie omite la cláusula "hasta..." por completo cuando es `nil`, en vez de inventar un
número. Se extrajo `resolutionDisclaimer(in) string` (antes inline en `buildFooter`)
específicamente para poder probar este caso sin inspeccionar bytes de PDF.

**Verificado con un navegador real, de punta a punta**: empresa nueva → subir logo (PNG real,
se ve en la vista previa) → subir software/certificado real (mismo `.p12` de
`docs/reference/`, sin tocar el emisor real del usuario — ver el hallazgo de la sección 9.38
sobre por qué `LoadPKCS12` no necesita que el NIT coincida con el certificado) → rango de
numeración en habilitación con el set de pruebas real (ya cerrado, mismo resultado esperado de
la sección 9.10) → crear borrador de factura → "Ver PDF" muestra el logo + "BORRADOR" + QR
pendiente, sin CUFE → confirmar (construye/firma/persiste con CUFE/QR/número reales antes de
intentar enviar, independientemente de si la DIAN later acepta) → "Ver PDF" de nuevo muestra el
mismo logo + CUFE real + QR real + número real. 0 errores de consola en las dos corridas (la
primera reveló el bug del `RangeTo`, la segunda ya no lo mostraba). Los bytes del PDF se
bajaron directo por `fetch` con el token real (mismos bytes que vería el navegador) para
inspección visual, en vez de confiar en una captura de pantalla del visor nativo de PDF de
Chromium (poco fiable en modo headless). Datos de prueba eliminados al terminar.

### 9.40 Correo (`internal/email`) — solo el módulo SMTP, sin envío al cliente todavía

Siguiente pieza del ciclo (después de PDF, sección 9.39): el "enviar la factura al cliente" se
secuenció en dos partes por decisión explícita del usuario — primero el módulo de correo en sí
(genérico, reusable), después (fase aparte) conectarlo a un flujo real que adjunte PDF+XML y lo
dispare desde la UI. Esta sección documenta solo la primera parte.

**Decisión de proveedor** (discutida en conversación antes de tocar código): nada de servicios
transaccionales externos (SendGrid/SES/Mailgun/etc.) ni de costos recurrentes adicionales — el
usuario quiere SMTP propio con credenciales por `.env`, igual que cualquier otro secreto de este
proyecto. Se evaluó también auto-alojar el propio servidor de correo (Mailu, con TLS propio) por
"más control", pero la reputación de envío (que determina si los correos caen en spam) depende
del historial del dominio/IP emisor, no del software que corre el servidor — un VPS nuevo con
Mailu empieza en cero exactamente igual que un dominio nuevo en cualquier otro lado. Para
empezar, un proveedor de hosting de correo con reputación de salida ya establecida (ej.
dondominio, evaluado por el usuario: 5 cuentas / 100MB cada una) es la opción más segura;
usando la cuenta solo para enviar (SMTP directo) sin leerla por webmail, 100MB no es una
limitación real para este caso de uso. Esto puede cambiar de proveedor más adelante sin tocar
código: `internal/email` no conoce el proveedor, solo recibe host/usuario/contraseña por config.

**`internal/email` (nuevo paquete)** — sin interfaz propia todavía: solo hay una implementación
(SMTP) y ningún consumidor real aún. Mismo principio de puertos angostos del resto del proyecto:
el consumidor define su propia interfaz angosta el día que la necesite (ej. un futuro
`documents.EmailPort`), satisfecha estructuralmente por `*SMTPSender` sin que este paquete tenga
que anticipar esa forma — igual que `issuers.PostgresRepository` satisface `documents.IssuerPort`
hoy sin importarlo.

- `message.go`: `Message` (To, Subject, BodyText, BodyHTML, Attachments) y `Attachment`
  (Filename, ContentType, Content []byte) — `ContentType` explícito, mismo criterio que
  `issuers.LogoContentType` (no se adivina por extensión).
- `smtp.go`: `Config` (Host, Port, Username, Password, FromAddress, FromName);
  `NewSMTPSender(cfg)`; `(*SMTPSender).Send(ctx, Message) error`. Validaciones tempranas y
  explícitas dentro de `Send` (`Host` vacío, `To`/`Subject`/cuerpo vacíos) — un correo mal
  formado o sin configurar falla con un mensaje claro en español, no con un error críptico de
  red o de la librería SMTP.

**Librería**: `github.com/wneessen/go-mail` (sin dependencias externas más allá de la librería
estándar, mantenimiento activo, soporta STARTTLS/TLS implícito, adjuntos y multipart texto+HTML)
en vez de construir MIME a mano sobre `net/smtp` (tedioso y propenso a errores de encoding en
adjuntos) o usar `gomail.v2` (más antiguo, menos activo). Mismo criterio que maroto v2 para el
PDF: no reinventar un protocolo ya bien resuelto por una librería madura cuando no hay una razón
real para hacerlo (a diferencia de UBL/SOAP en `cofacture`, donde sí hizo falta construirlo a
mano porque no existía nada usable en Go). La API exacta se verificó con `go doc` contra el
módulo ya instalado antes de escribir el código final (mismo método ya usado con maroto):
`mail.NewMsg()` + `From/FromFormat/To/Subject/SetBodyString/AddAlternativeString/AttachReader`
para construir el mensaje, `mail.NewClient(host, ...opts) ` +
`client.DialAndSendWithContext(ctx, msg)` para enviarlo. Autenticación con
`mail.SMTPAuthAutoDiscover` (negocia el mecanismo que soporte el servidor — LOGIN/PLAIN/CRAM-MD5/
etc. — en vez de asumir uno fijo, ya que todavía no se sabe qué exigirá el proveedor real) y
`mail.TLSMandatory` (falla claro si el servidor no soporta STARTTLS, en vez de degradar
silenciosamente a texto plano).

**Config** (`internal/config`): `SMTPHost/SMTPPort/SMTPUsername/SMTPPassword/SMTPFromAddress/
SMTPFromName`, todas opcionales y sin validar en `validate()` — a diferencia de
`ISSUER_SECRETS_KEY`/`AUTH_JWT_SECRET`, esto no es crítico para que el servidor arranque, y
**deliberadamente no se conecta a `cmd/server/main.go` en esta fase**: construir
`email.NewSMTPSender(cfg)` y guardarlo donde se vaya a usar es trabajo de la fase "enviar al
cliente", cuando exista un consumidor real. `SMTP_PORT` default `587` (STARTTLS, lo normal en
hosting compartido) — documentado en `.env.example` junto con la alternativa `465` (TLS
implícito) por si el proveedor lo exige así.

**Pruebas** (`internal/email/smtp_test.go`) — sin red real: se prueba `buildMsg` (la traducción
de `Message` a `*gomail.Msg`, separada de `Send` justamente para poder probarla así) volcando el
mensaje generado a bytes vía `Msg.WriteTo` e inspeccionando que contenga To/From/Subject/cuerpo/
adjunto esperados — cubre texto+HTML, solo texto, con adjunto, y remitente con/sin nombre. Más
las 4 validaciones tempranas de `Send` (sin host, sin destinatario, sin asunto, sin cuerpo).

**Envío real verificado (2026-06-27)**: el usuario consiguió una cuenta sandbox de Mailtrap
(`sandbox.smtp.mailtrap.io`, pensada exactamente para esto — captura los correos sin entregarlos
a una bandeja real, ideal para probar sin arriesgar nada) y la configuró en `apidian/.env`
(`SMTP_*`, password puesta directamente por el usuario, nunca pegada en el chat). Verificado con
un script descartable (`cmd/sendtest`, borrado después de usarlo — mismo patrón ya usado para
PDF/DIAN, no se commitea): `email.NewSMTPSender` con la config real + `Send` de un mensaje con
texto, HTML y un adjunto simulado → la llamada devolvió éxito, confirmando que `internal/email`
funciona de punta a punta contra un servidor SMTP real (dial, STARTTLS, auth, envío), no solo en
las pruebas unitarias sin red. La verificación visual del correo capturado en el inbox de
Mailtrap queda del lado del usuario (su cuenta).

**Alcance deliberadamente fuera de esta fase**: sin interfaz/puerto dentro de `internal/email`,
sin wiring a `cmd/server`/HTTP, sin CC/BCC/múltiples destinatarios, sin plantilla de cuerpo real
de correo (eso se diseña junto con el flujo real de "enviar al cliente"). Eso es la fase
siguiente.

### 9.41 Verificación de adquiriente vía DIAN + autorregistro de clientes por QR — implementado (2026-06-29)

Dos preguntas del usuario (2026-06-27) sobre el flujo de captura de clientes, registradas aquí
como notas de diseño. **Todo esto se implementó el 2026-06-29** — ver la sección de implementación
al final de esta sección.

**1. `GetAcquirer` — confirmado real en el WSDL** (`docs/reference/wsdl/xsd0.xsd`/`xsd10.xsd`),
parte de la misma interfaz `IWcfDianCustomerServices` de la que `cofacture/soap` ya implementa
`GetStatus`/`GetStatusZip` (mismo endpoint de habilitación, mismo sobre WS-Security):

```
GetAcquirer(identificationType, identificationNumber) -> AdquirienteResponse {
  Message, StatusCode, ReceiverName, ReceiverEmail
}
```

El namespace de la respuesta (`Gosocket.Dian.Services.Utils.Common`) y los campos
(`ReceiverName`/`ReceiverEmail`, no datos completos de RUT como régimen/responsabilidades/
dirección) sugieren que esto consulta el **registro de intercambio/notificación** de la DIAN
(si ese NIT/cédula ya tiene un nombre y correo registrados para recibir documentos
electrónicos) — no es una consulta de RUT completa. Es información parcial: útil para validar
que el dato coincide con lo que la DIAN tiene registrado para ese adquiriente, no para traer
régimen tributario/responsabilidades/dirección completos.

**2. Autorregistro de clientes por QR (patrón D1 y similares) — sí es un flujo real y normal**
en retail colombiano de alto volumen/bajo valor: pedir al cajero que digite los datos de cada
cliente en el mostrador no escala, y la DIAN no exige revisión manual de un RUT físico para
facturación de consumidor final — esa exigencia (KYC documental) es típica de relaciones de
crédito/financieras, no de un punto de venta. La responsabilidad de que el dato sea correcto
sigue siendo del emisor, pero la mitigación estándar es validación automática, no revisión
manual:

- Validación de formato (longitud/patrón por tipo de identificación; dígito de verificación
  módulo 11 para NIT) — derivado en `customers.Service`/`issuers.Service` via `internal/nit.ComputeCheckDigit`
  cuando `identification_type_code == "31"`. Nunca se acepta del cliente.
- Cruce opcional contra `GetAcquirer` cuando el tipo de identificación sugiere una empresa
  (NIT) — útil para detectar un NIT mal escrito antes de que quede en un documento legal difícil
  de corregir (una factura aceptada solo se "deshace" con nota crédito). **No es universal**:
  la mayoría de personas naturales (cédula, el caso típico de un cliente de mostrador) no tienen
  registro de intercambio en la DIAN, así que un `GetAcquirer` vacío/"no encontrado" es el
  resultado normal y esperado, no un error — el flujo debe poder continuar igual, usando el
  nombre que la persona escribió.

**Implementación (2026-06-29)**:

- **Dígito de verificación (módulo 11)**: nuevo paquete `apidian/internal/nit`. Se verificó
  contra NIT real del usuario (`6382356` → `7`) antes de implementar. Se deriva en
  `customers.Service`/`issuers.Service` cuando `identification_type_code == "31"`.

- **Autorregistro por QR**: rutas sin `middleware.Auth` — `GET /public/issuers/{id}` (solo
  nombre del emisor) y `POST /public/issuers/{id}/customers` (nombre + identificación + correo/
  teléfono opcionales; dígito de verificación calculado server-side). Frontend: página pública
  `/r/:issuerId` sin Navbar/Sidebar + panel QR en Configuración → Empresa (genera el QR en el
  navegador sin backend adicional).

- **GetAcquirer**: agregado a `cofacture/soap/operations.go` (mismo patrón que `GetStatus`).
  `documents.Service.VerifyAcquirer` expuesto como `GET /api/v1/dian/verify-acquirer`. Solo
  visible en frontend para NIT; nunca bloqueante; no conectado al flujo público del QR
  (requeriría el certificado real del emisor, que puede no estar configurado en ese momento).
  **Bug real encontrado contra la DIAN**: HTTP 404 con body SOAP válido cuando el adquiriente
  no existe — el cliente SOAP rechazaba cualquier non-200 antes de parsear. Fix en
  `cofacture/soap/client.go`: intenta parsear el body PRIMERO, usa el HTTP status solo si
  la unmarshal falla. Fix general para cualquier operación futura en el mismo escenario.

### 9.42 Enviar la Factura al cliente por correo — cierre del ciclo

Última pieza del ciclo completo con un solo documento (Factura): conectar `internal/email`
(sección 9.40, hasta ahora desconectado de cualquier flujo real a propósito) a un envío real que
adjunte PDF+XML y se dispare desde la UI.

**`documents.EmailPort`** (nuevo puerto angosto, `ports.go`) — `Send(ctx, email.Message) error`,
satisfecho estructuralmente por `*email.SMTPSender` sin que `internal/email` lo importe, mismo
patrón que `IssuerPort`/`CatalogPort`/`CustomerPort`. Existe para poder fakear el envío en tests
(`fakeEmailPort` en `service_test.go`, guarda los `email.Message` enviados o devuelve un error
fijo configurable), no porque haya más de una implementación real. `documents.Service` gana el
campo `email EmailPort` y `New(...)` un sexto parámetro posicional — mismo estilo que los otros
4 puertos, sin introducir un patrón de options struct solo para este. Esto tocó los ~30 call
sites de `documents.New` en `service_test.go` (dos `Edit` con `replace_all` cubrieron casi
todos, 2 más a mano por tener una forma ligeramente distinta) y el único call site de
`api.go`/`api_test.go`.

**`loadInvoiceAndIssuer`** (refactor pequeño, antes inline en `RenderInvoicePDF`) — extraído
porque `SendInvoiceEmail` necesita la MISMA secuencia (cargar documento, validar pertenencia al
emisor, validar que es Factura, cargar el emisor) que `RenderInvoicePDF` ya hacía — segunda
necesidad real del mismo bloque, justo el criterio de extracción que se usa en todo el proyecto.
`notSupportedErr` es un parámetro porque cada llamador tiene su propio mensaje
(`ErrPDFNotSupportedForDocumentType` vs `ErrEmailNotSupportedForDocumentType`) para el mismo
chequeo.

**`documents.Service.SendInvoiceEmail(ctx, issuerID, id) error`** (nuevo, `email.go`) — solo
Factura (mismo alcance que PDF) y **solo `StatusAccepted`**: nunca un borrador, nunca
`StatusRejected`/`StatusSendError` (no es una factura válida que mandarle a nadie), nunca
`StatusSent` (resultado de la DIAN todavía desconocido en ese momento, podría acabar
rechazado más tarde — mandar el correo antes de saberlo sería prometerle al cliente un
documento que después podría no ser válido). `ErrCustomerEmailMissing` si el snapshot del
cliente no trae correo. Reusa `RenderInvoicePDF` tal cual para el adjunto PDF (duplica un par
de `SELECT` por UUID dentro de la misma petición — aceptable, no se justificó una variante
interna solo para evitarlo) y `d.SignedXML` tal cual para el adjunto XML. Asunto/cuerpo fijos en
español (`invoiceEmailText`/`invoiceEmailHTML`, funciones puras) — `invoiceEmailHTML` escapa con
`html.EscapeString` los campos interpolados (`Customer.Name`/`iss.BusinessName` son datos del
cliente/snapshot, no se confía en que no traigan caracteres HTML especiales).

**Endpoint**: `POST /api/v1/documents/{id}/send-email` → `204 No Content`, mismo estilo que
`handleDeleteDocument`. Sin test dedicado en `api_test.go` — mismo criterio que
`GET .../pdf`, que tampoco lo tiene: la cobertura real está en `service_test.go` (6 casos:
éxito, no aceptado, sin correo, otro emisor, tipo no soportado, error de `Send` propagado) más
la verificación con navegador real de abajo.

**Wiring de config**: `api.New` gana un parámetro `smtpCfg email.Config`; `internal/server`
lo construye desde `cfg.SMTPHost/Port/Username/Password/FromAddress/FromName` (ya existían
desde la sección 9.40) y lo pasa. Dentro de `api.New`: `email.NewSMTPSender(smtpCfg)` →
`documents.New(..., emailSender)` — primer consumidor real del módulo de correo.

**Alcance deliberadamente excluido**: sin columna de "enviado al cliente"/historial — el botón
siempre está disponible mientras el documento esté `accepted`, se puede reenviar las veces que
haga falta, sin marca persistida de cuándo se envió la última vez (si hace falta auditoría de
envíos más adelante, se agrega con su propia migración). Sin CC al emisor, sin plantilla
configurable por el usuario.

**Frontend**: `lib/documents.ts` gana `sendInvoiceEmail(id)`; `InvoiceEditorPage` gana el botón
"Enviar al cliente" (icono `Mail`) junto a "Ver PDF" — visible solo cuando
`doc?.status === "accepted"`, con `window.confirm` antes de enviar (mismo criterio que eliminar
borrador/confirmar factura: es una acción visible para un tercero real) y un
`<Banner tone="success">` al terminar.

**Verificado de punta a punta contra Mailtrap real** (el usuario ya tenía una cuenta sandbox
configurada en `apidian/.env`, ver sección 9.40): en vez de gastar un consecutivo real de la
DIAN solo para llegar a `StatusAccepted` (ese camino ya está probado de sobra en secciones
anteriores), se creó una empresa/rango/factura de prueba vía API real contra Postgres real, y se
marcó el documento como `accepted` directamente en la base (mismo criterio que `seedInvoice` en
los tests unitarios, pero contra Postgres real en vez de memoria) — la parte que esta
verificación necesitaba probar es el envío, no repetir la aceptación de la DIAN. Primero
`POST /documents/{id}/send-email` por `curl` directo → `204`. Después, en el navegador real:
login → factura → botón "Enviar al cliente" visible solo por estar `accepted` → diálogo de
confirmación con el correo correcto del cliente → banner de éxito. 0 errores de consola. Dos
correos reales llegaron al sandbox de Mailtrap del usuario en esta verificación (uno por curl,
uno por el botón) — confirmación visual de contenido queda del lado del usuario revisando su
inbox, fuera de mi alcance. Datos de prueba eliminados al terminar.

Con esto queda cerrado el ciclo completo de un solo documento: Factura Electrónica →
representación gráfica en PDF → envío al cliente por correo, los tres verificados contra
servicios reales (DIAN real, Postgres real, SMTP real). El patrón se extendió a Nota Crédito y
Nota Débito en la sección 9.49 (`SendDocumentEmail` generalizado para los 3 tipos).

### 9.43 Auto-curar "set de pruebas cerrado" — sin intervención manual nunca más

Probando el ciclo completo en la UI real (2026-06-28), una factura salió `rejected` con
`dian_status_code: 2`, `dian_status_description: "Set de prueba con identificador
653bf9d9-b2b1-44ae-a66d-3b9cdc4271c3 se encuentra Aceptado."`. Causa: el rango de numeración
real del usuario todavía tenía `test_set_id` cargado — mientras lo tenga, `finalizeAndSend`
siempre enruta por `SendTestSetAsync` (sección 9.14), que la DIAN rechaza para siempre una vez
ese set queda "Aceptado" de su lado. **Esto ya había pasado antes en este proyecto** (sección
9.38) y se corrigió a mano, una vez, con un `UPDATE` directo en la base de datos — pero eso no
escala: cualquier empresa real futura que use `apidian` va a pasar por "registrar software → la
DIAN le da un `test_set_id` → certificar con `SendTestSetAsync` → la DIAN marca el set
Aceptado" y se va a topar con el mismo rechazo, sin que haya nadie del lado de desarrollo para
arreglarlo a mano en su base de datos. El usuario pidió explícitamente que el sistema se cure
solo — alcance acotado a Habilitación (producción real sigue diferida aparte, sección 9.11, por
las implicaciones de enviar documentos fiscales reales).

**`cofacture/dian.Result.IsTestSetClosed() bool`** (nuevo, `parser.go`) — detecta esta
respuesta específica: `StatusDescription` contiene tanto "Set de prueba" como "se encuentra
Aceptado", **sin** `Messages` (los rechazos de contenido real llegan vía
`ErrorMessage.Items`/`Messages`, nunca por aquí) — distingue un detalle de certificación de un
rechazo genuino del documento, ambos con el mismo `StatusCode` genérico "2". Unit test con la
respuesta real exacta que produjo el rechazo, más un caso negativo (rechazo de contenido normal
con `Messages`, que no debe activar esto).

**`numbering.ClearTestSetID`** (nuevo, mismo patrón en las 4 capas que `ReleaseIfCurrent`:
`Repository`/`PostgresRepository`/`MemoryRepository`/`Service`) — vacía `test_set_id` del
rango. `PostgresRepository`: `UPDATE ... WHERE id = $1`, sin error si no afecta ninguna fila
(mismo criterio que `ReleaseIfCurrent` — "ya no existe" no es un caso que el llamador necesite
distinguir de "no había nada que limpiar"). `documents.NumberingPort` gana este método.

**El self-healing real** vive en `documents.Service.sendAndUpdate` (la rama de
`SendTestSetAsync`), justo después de interpretar la respuesta de `GetStatusZip` y antes de
decidir `StatusAccepted`/`StatusRejected`: si `interpreted.IsTestSetClosed()`, se limpia
`test_set_id` (best-effort, mismo criterio que `ReleaseIfCurrent`) y se reintenta **el mismo
envío, en la misma petición de confirmar**, vía `sendSyncAndUpdate` (`SendBillSync`, que sigue
funcionando con el set cerrado, sección 9.14) — en vez de devolver `StatusRejected` por un
detalle de certificación. El número ya reclamado no se libera en este camino (no hace falta:
`sendSyncAndUpdate` reusa el mismo documento/número, no reclama uno nuevo). De cara al usuario,
esto es completamente invisible: un confirm que antes habría salido `rejected` por esta causa
ahora simplemente sale `accepted` (o lo que la DIAN responda de verdad vía `SendBillSync`), y
ningún confirm futuro sobre ese rango vuelve a intentar `SendTestSetAsync`.

**Sin test automatizado para la rama de `sendAndUpdate` en sí** — a propósito, no por omisión:
`documents.Service` no tiene (ni este fix agrega) un punto de inyección para el cliente SOAP;
`soap.New(soap.HabilitacionURL, ...)` está fijo. Todos los tests existentes de
`finalizeAndSend`/confirm evitan esta rama a propósito usando `issuers.EnvironmentProduccion`
en `testIssuer()` (ver el comentario ahí: "es el único ambiente que finalizeAndSend nunca envía
por red"). Agregar una interfaz inyectable solo para simular esta respuesta sería un cambio de
arquitectura mayor no justificado por un solo caso — mismo criterio ya aplicado en todo este
proyecto con la DIAN (se verifica contra el servicio real, no se construye un doble elaborado
para simular su SOAP). La detección en sí (`IsTestSetClosed`, la parte realmente delicada de
acertar sin falsos positivos) sí tiene cobertura unitaria completa; el comportamiento de extremo
a extremo se verificó contra la DIAN real: parche puntual aplicado al rango real del usuario
(limpiar `test_set_id` manualmente esta vez, para no bloquearlo mientras se construía el fix
genérico), y queda pendiente que el usuario confirme una factura nueva sobre ese mismo rango
para validar que el camino normal (ya sin `test_set_id`) sigue funcionando — y, si quiere
probar el self-healing de verdad, recargarle un `test_set_id` cerrado a mano una vez más.

**El self-healing funcionó de verdad** (confirmado el mismo día): el siguiente intento del
usuario ya no mostró "Set de prueba... se encuentra Aceptado" — `dian_track_id` quedó vacío
(huella de `sendSyncAndUpdate`, que no usa `ZipKey`, a diferencia de `sendAndUpdate`/
`SendTestSetAsync`), confirmando que el sistema cambió de camino solo, sin ninguna intervención
manual. Pero salió un rechazo **real y distinto**: `StatusCode 99`, "Validación contiene
errores en campos mandatorios" — ver sección 9.44.

### 9.44 `cac:Country` faltante en la dirección del cliente — primer cliente real con dirección completa

El rechazo de arriba llevó a inspeccionar el XML firmado real (`documents.SignedXML`, columna
persistida incluso para documentos rechazados — nunca se recalcula después, ver model.go).
Comparando `cac:AccountingCustomerParty/.../cac:RegistrationAddress` contra
`cac:AccountingSupplierParty/.../cac:RegistrationAddress` del mismo XML: la del emisor sí
tenía `<cac:Country><cbc:IdentificationCode>CO</cbc:IdentificationCode>...`, la del cliente
**no tenía ningún `cac:Country`**. Confirmado contra los XMLs de ejemplo reales de la DIAN
(`docs/reference/Caja de herramientas FE_V19_(v2026)/Ejemplificaciones/XMLs de ejemplo/*.xml`,
ej. "Excluido de IVA.xml"/"Combustible.xml"): **todo** ejemplo oficial con dirección de
registro incluye `cac:Country` — confirma que es mandatorio, no algo que la DIAN tolere omitir
(a diferencia de `cac:TaxTotal` por línea, que sí se confirmó opcional contra el mismo ejemplo
"Excluido de IVA.xml", una línea sin impuesto y sin `cac:TaxTotal`, aceptada por la DIAN — esa
parte de `cofacture/builder/tax.go` ya estaba bien, no se tocó).

**Causa real**: `cofacture/builder/party.go.appendAddressFields` solo agrega `cac:Country` si
`Address.CountryCode != ""` — correcto, comparte la misma función para emisor y cliente. Pero
`documents/service.go.partyFromIssuer` **hardcodea** `CountryCode: "CO"`/`CountryName:
"Colombia"` para el emisor, mientras que `applyCustomerDefaults` (la función equivalente para
el cliente) nunca tuvo ese default — un default que existe de un lado y nunca se replicó del
otro, exactamente el mismo patrón de bug que `EntityTypeCode`/`LiabilityCodes` en la sección
9.29 (issuer) y 9.38 (software provider). **Invisible hasta ahora** porque `cac:Country` solo
se renderiza dentro de `cac:RegistrationAddress`, que a su vez solo se construye si el cliente
trae dirección (`Address.Line != ""`) — y ningún cliente de prueba en toda la historia de este
proyecto había tenido dirección hasta este intento real del usuario (todos los fixtures/tests
usan `Customer: domain.Party{Identification: ..., Name: "Consumidor Final"}`, sin `Address`).

**Fix**: `applyCustomerDefaults` (`documents/service.go`) ganó el mismo default condicional que
el resto de la función (`if Address.CountryCode == "" { ... = "CO"/"Colombia" }`) — respeta un
país explícito distinto si alguna vez se factura a un cliente extranjero, igual criterio que
los demás defaults de esa función. Test nuevo
(`TestCreateInvoiceDraft_DefaultsCustomerCountry`) cubre el caso que expuso el bug: cliente con
dirección completa pero sin país explícito.

### 9.45 Catálogo real de `@schemeID`/`@schemeName`/`@schemeAgencyID` (clasificación de ítems)

Tercer rechazo real (mismo `StatusCode 99`) en la misma factura de prueba, ya con el bug del
país del cliente corregido (9.44). Inspeccionando el XML firmado real otra vez:
`cac:StandardItemIdentification/cbc:ID` traía `schemeID="43211500"` — un código UNSPSC real
puesto en el atributo equivocado.

**El hallazgo**: `@schemeID`/`@schemeName`/`@schemeAgencyID` no son campos libres — el Anexo
Técnico 1.9 los define en la tabla 13.3.5, que vive aparte de el texto principal
(`docs/reference/Caja de herramientas FE_V19_(v2026)/Anexo Tecnico/Tablas Referenciadas/13.3.5
Productos @schemeID, @schemeName, @schemeAgencyID.xlsx`, extraída con
`System.IO.Compression.ZipFile` + lectura directa del XML de la hoja, sin Python/Excel
disponibles). Es una tripleta cerrada de exactamente 4 filas, y la DIAN **rechaza
explícitamente** si el valor no coincide ("Rechazo si el valor informado es diferente al de la
tabla 13.3.5"):

| schemeID | schemeName | schemeAgencyID |
|---|---|---|
| `001` | UNSPSC | `10` |
| `010` | GTIN | `9` |
| `020` | Partida Arancelaria | `195` |
| `999` | Estándar de adopción del contribuyente | *(no se usa)* |

El propio comentario en `cofacture/domain/types.go` (`ItemTypeCode string // catálogo de
estándares... (ej. "999")`) ya anticipaba esta idea — pero nunca se construyó el catálogo real
ni se validó/derivó, así que `ProductForm.tsx` dejaba un campo "Código de estándar" de texto
libre con placeholder "Ej. UNSPSC", y terminó guardando el código UNSPSC real del producto
("43211500") donde debía ir el selector ("001"). Confirmado contra los ejemplos oficiales de la
DIAN (`docs/reference/.../Ejemplificaciones/XMLs de ejemplo/`): "ExcluidosExentos.xml" usa
exactamente `schemeID="001" schemeName="UNSPSC" schemeAgencyID="10"` con el código real en el
texto del elemento — de paso confirmó que `cac:TaxTotal` por línea SÍ es opcional cuando no hay
impuesto ("Excluido de IVA.xml" no lo trae), así que esa parte de `cofacture/builder/tax.go` no
tenía ningún bug.

**Decisión del usuario**: quiso el sistema completo (selector real, pensando en facturación de
importaciones a futuro con Partida Arancelaria), no solo defaultear a 999 y listo. Aclaración
importante: UNSPSC/GTIN/Arancel **no son catálogos que se puedan cargar completos** (UNSPSC
tiene decenas de miles de códigos, Arancel miles, GTIN son códigos de barra externos) — lo que
sí es un catálogo chico y real, mismo criterio que `tax_types`/`dian_document_types`, es el
**selector de 4 filas** de la tabla de arriba. El código real dentro de cada estándar se acepta
como texto libre, sin validar contra una tabla completa — mismo tratamiento que ya recibe CIIU
en este proyecto.

**`catalogs.ItemStandard`** (nuevo, mismo patrón en las 4 capas que el resto de catálogos) —
migración `000010_item_standards` (de verdad nueva, no se edita una existente: hay datos reales
del usuario en juego) + `seed/item_standards.csv`. `agency_id` es `''` (no NULL) para la fila
999 — mismo criterio que el resto del proyecto, que modela "ausente" como string vacío en vez
de `*string`. `ListItemStandards`/`GetItemStandardName`/`GetItemStandardAgencyID`
(`AgencyID` separado de `Name` porque puede ser `""` con `found=true`, a diferencia del resto
de `Get*Name` donde `found=false` ya cubre "nada que mostrar"). Endpoint nuevo de solo lectura
`GET /api/v1/catalogs/item-standards`.

**`documents`/`products` derivan, no confían en el cliente** (mismo patrón de la sección 9.37):
`linesFromInput`/`resolveItemStandard` (nuevo en `products.Service`) defaultean
`item_type_code` vacío a `"999"`, validan contra el catálogo (`ErrInvalidItemStandardCode` si
no existe) y derivan `item_type_name`/`item_type_agency_id` — el cliente ya no los manda
(quitados de `lineInputDTO`/`productRequest`). A diferencia de `tax_type_code` (opcional, "sin
impuesto" es válido), `item_type_code` SIEMPRE tiene un valor tras el default — la DIAN exige
que la clasificación esté presente, nunca "ninguna".

**`cofacture/builder/line_items.go`**: `schemeAgencyID` ahora solo se agrega como atributo si
`line.ItemTypeAgencyID != ""` (mismo criterio que `appendAddressFields` con `cac:Country`,
sección 9.44) — antes se agregaba siempre, incluso vacío. Se corrigió también un golden test
preexistente (`invoice_builder_test.go`/`testdata/invoice_golden.xml`) que ya usaba
`schemeID="999"` correctamente pero con `ItemTypeAgencyID: "0"` — un valor inventado antes de
conocer la tabla real; ahora vacío, sin el atributo en el XML esperado.

**Frontend**: `ProductForm.tsx` reemplaza los 3 campos libres ("Código de estándar"/"Nombre del
estándar"/"ID de agencia") por un `<Select>` real (cargado de `listItemStandards()`,
`lib/catalogs.ts`) — "Sin clasificar (mi propio código)" (vacío → 999) o uno de los 3 estándares
externos. El campo "Código del ítem" existente se reutiliza para el código real dentro del
estándar elegido, con un placeholder contextual (`ITEM_CODE_PLACEHOLDERS`) — sin agregar un
campo nuevo. `LineItemsEditor.tsx` no necesitó UI propia (ya copiaba `item_type_code` del
producto sin inputs propios para los otros dos) — solo se le quitaron `itemTypeName`/
`itemTypeAgencyId` del estado interno, que habían quedado muertos (nunca tuvieron input, solo
se escribían).

**Hallazgo de proceso, no de producto**: durante esta fase se descubrió que `npx tsc --noEmit`
en `frontend/` no estaba revisando ningún archivo — `tsconfig.json` es "solution-style"
(`files: []` + `references`), que `tsc` sin `-b`/`-p` ignora silenciosamente (confirmado con
`--listFiles`, devolvió 0 archivos). El comando correcto es `npx tsc -b` (lo que de verdad usa
`npm run build`) — varias verificaciones "TSC_OK" de sesiones anteriores en este proyecto
pueden haber sido falsos positivos. Ver memoria `feedback-tsc-noemit-silently-checks-nothing`.

**Datos reales corregidos**: los 2 productos de prueba del usuario ("Producto de prueba 1"/"2")
tenían exactamente el bug descrito (`item_type_code` con códigos UNSPSC reales,
`item_type_agency_id` inconsistente entre los dos) — corregidos directamente en Postgres a
`item_type_code = '999'`/nombre derivado/agency_id vacío. Ningún documento ya existente se
tocó (son snapshots, nunca se recalculan).

**Pendiente para el usuario**: reintentar la factura real una vez más — ya con los bugs de país
del cliente (9.44) y clasificación de ítems (9.45) corregidos, junto con el self-healing del
set de pruebas (9.43), debería pasar la validación de campos mandatorios de punta a punta.

### 9.46 Detalle real de la DIAN vía `ApplicationResponseXML` + dos bugs más (unidad de medida, base imponible)

El cuarto rechazo (mismo `StatusCode 99` genérico) ya no cedió pistas inspeccionando solo el
XML propio — se necesitaba el detalle que la DIAN sí manda pero `documents.Service` nunca
persiste. `dian.Interpret` ya decodifica `ApplicationResponseXML` (`cofacture/dian/parser.go`,
existe desde antes) a partir de `resp.XmlBytes`/`XmlBase64Bytes`, pero `finish()` solo guarda
`StatusCode`/`StatusDescription`/`StatusMessage` — nunca ese XML. Para diagnosticar sin tocar
producto, se reenvió el `signed_xml` ya persistido del documento rechazado directo a
`SendBillSync` desde un script descartable dentro de `apidian/cmd/inspect` (necesita
`cryptutil`/`cofacture` directamente, por eso vive dentro del módulo en vez del scratchpad —
borrado al terminar, no se commiteó) — eso sí trajo el `ApplicationResponse` completo con
`cac:LineResponse` por línea:

```
Regla: FAV05, Rechazo: La unidad de la cantidad utilizada NO existe en la lista de unidades.
Regla: FAU04, Rechazo: Base Imponible es distinto a la suma de los valores de las bases imponibles de todas líneas de detalle.
```

**FAV05 — `M2`/`M3` no son códigos UN/ECE Rec. 20 reales**: el catálogo `unit_measures`
(sembrado con una muestra de 11 códigos a propósito, sección 9.21) tenía dos códigos
**inventados** (no solo "catálogo incompleto" — directamente equivocados): el real para metro
cuadrado es `MTK`, para metro cúbico `MTQ` (confirmado contra la tabla oficial extraída de
`docs/reference/.../Tablas Referenciadas/13.3.6 Unidades de Cantidad @unitCode.xlsx`, mismo
método de extracción que la 13.3.5 de la sección 9.45). Corregido en `seed/unit_measures.csv`;
las filas viejas `M2`/`M3` se borraron de la tabla real (un `UPDATE` de seed no las habría
tocado, solo inserta las nuevas) y el producto real del usuario que usaba `M2` se corrigió a
`MTK`.

**FAU04 — una línea sin impuesto rompe la base imponible del encabezado**: ya estaba resuelto
una vez, pero solo en `cofacture` — `cofacture/soap/realsend_test.go` documenta exactamente
este rechazo real (mismo código FAU04) y su fix: modelar "sin impuesto" como **IVA al 0%**, no
como ausencia total de `cac:TaxTotal`. Ese fix nunca se replicó en `apidian/internal/documents`,
que seguía dejando `Taxes` vacío cuando `tax_type_code` no se mandaba — invisible mientras
ningún test/factura real mezclara una línea con impuesto y una sin él (todos los fixtures de
este proyecto, incluyendo `testRequest()`, siempre usaban `"01"`/0% explícito, nunca vacío).

Fix en dos partes:
- `documents/lines.go.linesFromInput`: `tax_type_code` vacío defaultea a `"01"` con
  `tax_percent` forzado a `0` — siempre hay exactamente 1 impuesto por línea, nunca 0.
- `documents/service.go.aggregateTaxes`: agrupaba el `cac:TaxTotal` de cabecera solo por
  `TypeCode` — con el fix anterior, una factura con una línea al 19% y otra (defaulteada) al
  0% comparten `TypeCode "01"` pero deben quedar en `cac:TaxSubtotal` **separados** (fusionarlos
  habría reportado la base de la línea al 0% como si fuera al 19%, el mismo tipo de
  inconsistencia que dispara FAU04). Se cambió la clave de agrupación a `(TypeCode, Percent)`.

Tests nuevos: `TestCreateInvoiceDraft_DefaultsTaxToIVAZero`,
`TestCreateInvoiceDraft_AggregatesMixedTaxRatesSeparately` (este último necesitó exponer
`aggregateTaxes` vía `export_test.go` — no exportada en producción, solo visible para
`service_test.go`).

**Patrón a repetir si vuelve a pasar**: cuando la DIAN rechaza con un mensaje genérico
("errores en campos mandatorios") sin texto específico, vale la pena reenviar el XML ya
firmado directo por `SendBillSync` (fuera de la app, con un script descartable) e inspeccionar
`dian.Interpret(...).ApplicationResponseXML` — ahí sí vienen los códigos de regla exactos
(`cac:LineResponse` por línea), mucho más rápido que adivinar comparando contra ejemplos.

**`unit_measures` completado (1.081/1.081 códigos reales)**: el usuario preguntó directamente
si la tabla 13.3.6 completa estaba disponible — sí, en el mismo archivo de la Caja de
Herramientas usado para confirmar `M2`/`M3`. Mismo criterio que `municipalities`/`departments`
en la sección 9.34 (catálogo grande, cargarlo completo en vez de una muestra): se extrajo la
tabla completa (367 filas × 3 pares código/descripción por fila en el `.xlsx` real, no 1 por
fila) con el mismo método de `System.IO.Compression.ZipFile` + lectura directa de
`xl/worksheets/sheet1.xml` — la diferencia esta vez es que las columnas empiezan en `B` (no
`A`), hay que leer el atributo `r` de cada `<c>` en vez de asumir un orden fijo. Filtrado a
`^[A-Z0-9]{1,3}$` para descartar ruido de la extracción (notas al pie con `ª`/`º`, fragmentos
de texto en español que se filtraron por columnas vacías) — quedaron 1.081 códigos únicos de
1.093 pares brutos. **De paso confirmó que `MON`/`ANN` (mes/año), ya en el seed desde antes de
esta sesión, TAMPOCO existen en la tabla oficial** — los códigos reales son `LUN`/`ANA`. Mismo
patrón que `M2`/`M3`: alguien adivinó un código "lógico" en vez de verificarlo contra la tabla.
Seed reemplazado por completo (1.081 filas); `MON`/`ANN`/`M2`/`M3` eliminados de la tabla real
en Postgres (un `UPDATE` del seed no los habría tocado, solo inserta los nuevos). Los productos
reales del usuario solo usaban `94`/`MTK`, ambos confirmados correctos — no necesitaron más
ajustes. Algunos nombres tienen artefactos de codificación menores (ej. "Ã ¥ngström") que vienen
del archivo original de la DIAN, no de la extracción — no se intentó adivinar la corrección,
solo afecta el texto descriptivo, nunca el código que valida la DIAN.

### 9.47 Nota Crédito (NC) — frontend completo + bug `CreditNoteTypeCode` corregido

**Frontend NC**: `CreditNotesPage` (listado con columna "Factura referenciada"), `CreditNoteEditorPage`
(nuevo borrador desde factura aceptada vía `?from=<uuid>`, edición, confirmación), `CreditNoteForm`
(pre-llena cliente/líneas desde la factura origen, concepto de Lista 22, `DiscrepancyResponse`
auto-poblada al elegir el concepto). Entrada al flujo: botón "Emitir Nota Crédito" visible solo
en facturas `accepted` (`InvoiceEditorPage`). Sidebar habilitado.

**Bug CAD12a — `cbc:CreditNoteTypeCode` incorrecto**: al confirmar la primera NC real, la DIAN
rechazó con regla `CAD12a` ("Código de tipo de nota crédito inválido"). El bug: el campo
`CreditNoteTypeCode` de `domain.CreditNote` se estaba llenando con el código de **concepto** de
la Lista 22 (1–5, ej. "2 = Anulación"), que es correcto para `DiscrepancyResponse.ResponseCode`
pero no para `cbc:CreditNoteTypeCode`. Los XMLs de ejemplo oficiales de la DIAN (Caja de
Herramientas FE v1.9, `CreditNote.xml`) muestran `<cbc:CreditNoteTypeCode>91</cbc:CreditNoteTypeCode>`
— el **tipo de documento DIAN** (91 = NC), no el concepto. Mismo patrón que
`cbc:InvoiceTypeCode = "01"` en facturas. Fix en `service.go`: `CreditNoteTypeCode: creditNoteDianDocumentType`
(constante "91") en vez de `d.NoteTypeCode`. Corregido también en `realsend_creditnote_test.go`,
`credit_note_test.go` y `credit_note_golden.xml`. NC confirmada y aceptada por la DIAN después
del fix.

**Rango de numeración para NC en habilitación**: la DIAN no restringe la resolución por tipo de
documento — se registra un rango en apidian con `dian_document_type_code: "91"` usando los mismos
datos de la resolución real (prefijo, rango, test_set_id) que la factura de habilitación.

### 9.48 `GET /documents?source_document_id=<uuid>` — trazabilidad de NC/ND sobre una factura

**Motivación**: cualquier frontend (o desarrollador externo) que quiera saber si una factura tiene
NC/ND asociadas no debe conocer el detalle interno de `billing_reference` — la relación debe ser
descubrible desde el contrato público de la API.

**Implementación sin migración**: la tabla `documents` ya tiene `billing_reference JSONB` con el
`prefix`/`number` de la factura de origen. Se agregó `SourceDocumentID *uuid.UUID` a
`documents.ListFilter` y se filtra con un subquery de PostgreSQL:

```sql
AND (billing_reference->>'prefix', billing_reference->>'number') = (
    SELECT prefix, number::text FROM documents WHERE id = $N AND issuer_id = $1
)
```

El `issuer_id = $1` en el subquery garantiza que el usuario solo puede consultar notas sobre
facturas propias — nunca de otro emisor. No hay nueva columna, ni índice adicional, ni migración.

**En el handler**: `?source_document_id=<uuid>` se parsea con `parseUUID` (mismo helper que el
resto de UUIDs de la API) y se pasa a `filter.SourceDocumentID`.

**En el frontend**: `listDocuments({ source_document_id: id })` (param nuevo en
`ListDocumentsFilter`/`documents.ts`). `InvoiceEditorPage` carga las notas relacionadas en un
`useEffect` separado cuando `doc.status === "accepted"` y las muestra en una sección "Notas
emitidas sobre esta factura" con el tipo (NC/ND), el motivo (descripción del concepto), el status
con badge y un enlace directo al detalle de la nota. La lista de facturas (`InvoicesPage`) no
muestra esta señal — requeriría N+1 queries; el detalle es el lugar natural para verlo.

### 9.49 Nota Débito (ND) validada en habilitación real DIAN — ciclo completo cerrado (2026-07-11)

La Nota Débito (tipo 92) fue construida, firmada, enviada a la DIAN sandbox y **aceptada**
(StatusCode 00) en julio 2026, referenciando una factura real previa. Con esto el ciclo completo
de los tres tipos de documentos del Anexo 1.9 queda validado en habilitación real:

| Documento DIAN | Estado |
|---|---|
| Factura electrónica (01) | ✅ Autorizada en habilitación real |
| Nota Crédito (91) | ✅ Procesada y aceptada en habilitación real |
| Nota Débito (92) | ✅ Aceptada en habilitación real (2026-07-11) |

La tabla en 9.1 fue actualizada para reflejar esto.

**Hallazgo de conformidad — correo al adquiriente** *(documentado aquí; implementado después, ver sección 9.51)*: el Anexo Técnico 1.9 sección 9.1 especifica que el adjunto del correo al cliente debe ser un único `.zip` que contenga un `AttachedDocument` UBL firmado que embebe el XML firmado de la factura **y** el `ApplicationResponse` de la DIAN en CDATA. En el momento de esta sección se enviaba un ZIP con el XML crudo + el PDF. El problema se resolvió completamente en una sesión posterior sin que se documentara en su momento — ver sección 9.51 para el detalle.

### 9.50 Paridad NC/ND con Factura — "Descargar XML" + alertas visuales de vigencia (2026-07-14)

#### "Descargar XML" en Nota Crédito y Nota Débito

`InvoiceEditorPage` ya tenía el botón "Descargar XML" (visible cuando `status !== "draft"`)
desde antes, pero `CreditNoteEditorPage` y `DebitNoteEditorPage` no lo tenían. El hueco se
detectó comparando imports: `getDocumentXmlBlobUrl` y `FileCode` (lucide) presentes en la
factura, ausentes en NC/ND.

Fix idéntico en ambos editores:
- Import `getDocumentXmlBlobUrl` desde `lib/documents` y `FileCode` desde `lucide-react`.
- Estado `loadingXml` (mismo patrón que `loadingPdf`).
- Handler `handleDownloadXml` — crea un enlace `<a>` temporal, lo dispara, revoca la blob URL;
  nombre de archivo `{prefix}{number}.xml` (fallback `nota-credito.xml` / `nota-debito.xml`).
- Botón "Descargar XML" en el header del editor, visible cuando `!isNew && doc && doc.status !== "draft"`.

Con este cambio, los tres editores son funcionalmente idénticos en cuanto a botones:
Ver PDF · Descargar XML · Enviar al cliente · (Eliminar borrador / Confirmar y enviar).

#### Correo en NC/ND — ya estaba implementado desde 9.49

La memoria marcaba "correo NC/ND pendiente de verificar end-to-end". Revisando el código,
`sendDocumentEmail` (función genérica para los 3 tipos) ya estaba importada y cableada en
`CreditNoteEditorPage` y `DebitNoteEditorPage` desde la sección 9.49 — incluyendo estado
`sendingEmail`, confirmación vía `useConfirm()` y toast de resultado. No se requirió ningún
cambio.

#### Alerta visual "Por vencer" en rangos de numeración

`NumberingRangesPanel` ya mostraba la columna "Vence" y el badge "Vencido" (rojo) cuando
`valid_to` había pasado. Lo que faltaba era el estado intermedio de advertencia.

Se añadió la función auxiliar `isExpiringSoon(r: NumberingRange): boolean` en el frontend:

```tsx
function isExpiringSoon(r: NumberingRange): boolean {
  if (r.status !== "active") return false;
  const daysLeft = (new Date(r.valid_to).getTime() - Date.now()) / (1000 * 60 * 60 * 24);
  return daysLeft <= 30;
}
```

En la celda del badge, si `isExpiringSoon(r)` es `true` se muestra "Por vencer" con los tokens
`--color-warning-bg` / `--color-warning-text` (amarillo) en lugar de "Activo" verde. La lógica
vive exclusivamente en el frontend — el `status` que llega del backend sigue siendo `"active"`, y
el botón Desactivar sigue apareciendo igual.

El umbral de 30 días es idéntico al que ya usaba `CertStatusBadge` para el certificado digital
(en `SoftwareCertificateForm.tsx`), que además muestra el conteo exacto de días restantes
("Alerta — vence en X día(s)"). La coherencia entre certificado y rangos es intencional; si en
el futuro se quiere añadir el conteo de días al badge de rangos, el patrón ya está establecido.

### 9.51 `AttachedDocument` UBL conforme al Anexo 1.9 — verificado como ya implementado (2026-07-14)

Al revisar el código en detalle (motivado por el hallazgo de 9.49 que lo marcaba como pendiente),
se confirmó que el `AttachedDocument` UBL conforme estaba **completamente implementado** en una
sesión anterior sin que se documentara en su momento. El código cubre los tres documentos (01, 91,
92) y el fallback para documentos históricos. No se escribió código nuevo — solo se corrigió la
documentación.

#### Qué hay implementado y dónde

**Persistencia del `ApplicationResponse`**

`documents.Service.finish()` (`service.go`) recibe `applicationResponseXML string` desde
`sendAsyncAndUpdate` y `sendSyncAndUpdate`, que a su vez lo obtienen de
`dian.Interpret(resp).ApplicationResponseXML`. `dian.Interpret` (`cofacture/dian/parser.go`)
decodifica el base64 de `XmlBase64Bytes` que la DIAN devuelve en el cuerpo SOAP de su respuesta.
`PostgresRepository.UpdateDianStatus` (`postgres.go:130`) persiste el valor en la columna
`application_response_xml TEXT` (migración `000008_documents.up.sql`) vía `$6`.

**Construcción del sobre UBL y firma**

`buildAttachedDocumentXML` (`email.go:251`) — función interna llamada por `SendDocumentEmail`:

```
si d.ApplicationResponseXML == ""  →  devuelve d.SignedXML crudo + filename.xml
                                       (fallback para documentos sin el campo)
si d.ApplicationResponseXML != ""  →  construye domain.AttachedDocument con todos los campos
                                       del Anexo (Sender/Receiver/ValidationResult con el XML
                                       de la DIAN embebido), llama al builder correcto según
                                       tipo de documento, firma con el certificado del emisor
                                       vía signer.Sign, devuelve filenamead.xml
```

Los tres builders ya existían en `cofacture/builder/attached_document.go`:
- `BuildInvoiceAttachedDocument` — Factura (01)
- `BuildCreditNoteAttachedDocument` — Nota Crédito (91)
- `BuildDebitNoteAttachedDocument` — Nota Débito (92)

**Por qué unos documentos tienen el campo y otros no**

La columna existe desde la migración correspondiente. Los documentos confirmados *antes* de que
existiera tienen `application_response_xml = NULL` → el ZIP del correo incluye el XML crudo + PDF
(fallback, útil aunque no formalmente conforme). Los documentos confirmados *después* tienen el
campo → el ZIP incluye el `AttachedDocument` UBL firmado + PDF (conforme al Anexo 1.9 sección 9.1).
Este comportamiento dual es intencional y está documentado en el comentario de `SendDocumentEmail`
(`email.go:51-55`).

**Estado en 9.2**: la fila de `AttachedDocument` en la tabla de pendientes fue actualizada para
reflejar que está implementado.

### 9.52 Alertas proactivas por correo — rangos por vencer y certificado (pendiente)

> **Estado: diseñado, no implementado.** Este es el diseño acordado para cuando se construya.

#### Qué notificar

- **Rangos de numeración**: cualquier rango con `status = active` y `valid_to` dentro de los
  próximos 30 días, o ya vencido (`status = expired`). Se notifica al correo del emisor (`issuers.email`).
- **Certificado digital**: cualquier emisor con `certificate_expires_at` dentro de los próximos
  30 días, o ya vencido. Mismo correo destinatario.

El umbral de 30 días es coherente con el badge "Por vencer" del frontend (sección 9.50) y con
`CertStatusBadge` en `SoftwareCertificateForm.tsx`.

#### Mecanismo de disparo — goroutine interna (sin dependencias externas)

Una goroutine lanzada desde `cmd/server/main.go` al arrancar, usando el `context` de shutdown
que ya existe. Esquema:

```go
go func() {
    alertSvc.RunOnce(ctx)               // pasada inmediata al arrancar
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            alertSvc.RunOnce(ctx)
        case <-ctx.Done():
            return
        }
    }
}()
```

La pasada inmediata al arrancar garantiza que si el servidor estuvo caído la noche anterior,
el chequeo igual se ejecuta al volver — nunca se pierde un día entero.

#### Evitar envíos duplicados — dos columnas nuevas

Sin registro de "ya avisé", el chequeo mandaría el correo cada día mientras el rango/cert siga
en riesgo. Solución: dos columnas nullable en las tablas correspondientes:

```sql
-- en numbering_ranges (integrar en la migración existente 000009):
expiry_alert_sent_at  TIMESTAMPTZ  -- NULL = nunca enviada

-- en issuers (integrar en la migración existente 000003):
cert_alert_sent_at    TIMESTAMPTZ  -- NULL = nunca enviada
```

Criterio de envío: `expiry_alert_sent_at IS NULL` (o han pasado más de 7 días desde la última,
para recordatorios de seguimiento). Al renovar el rango o reemplazar el certificado, la columna
se limpia a NULL automáticamente.

#### Template HTML de alerta

Un nuevo `alert_template.html` embebido (`//go:embed`) en `internal/documents` o en un nuevo
`internal/alerting`. Más simple que `email_template.html` — sin DocumentNumber/IssueDate/Total
(esos son datos de documento, no aplican aquí). Contiene: logo del emisor, título de la alerta
("Tu resolución de numeración está por vencer" / "Tu certificado digital vence pronto"),
cuerpo con los rangos/cert afectados, y un enlace o instrucción para ir a Configuración.

#### Piezas de implementación

| Pieza | Dónde vive | Notas |
|---|---|---|
| Migración: `expiry_alert_sent_at` | `000009_numbering_ranges` editado | Integrar, no ALTER TABLE nuevo |
| Migración: `cert_alert_sent_at` | `000003_issuers` editado | Mismo criterio |
| `alert_template.html` | `internal/documents/` (o `internal/alerting/`) | `//go:embed`, más simple que el de facturas |
| `alerting.go` — lógica de chequeo | `internal/documents/` (o nuevo paquete) | Lee rangos/certs, filtra candidatos, envía, marca columna |
| Goroutine en `cmd/server/main.go` | `cmd/server/main.go` | `RunOnce` + `time.Ticker(24h)` + `ctx.Done()` |

### 9.53 Modelo de entidades — decisión de tablas separadas (2026-07-15)

El usuario evaluó si convenía unificar `issuers`, `customers`, y los futuros `employees`/
`suppliers` en una sola tabla de "actores" o "partes", dado que comparten campos de identificación,
nombre y dirección.

**Decisión: mantener tablas separadas por rol.** Razonamiento:

El solapamiento real entre tablas es de ~10 columnas (NIT/número de identificación, nombre,
dirección, teléfono, correo). Lo que difiere es mucho mayor:

- `issuers` es una tabla de **configuración de tenant**: contiene credenciales cifradas (`BYTEA`
  para `software_pin`, `certificate`, `certificate_password`), logo, plantilla de correo,
  `environment`, `is_active` — campos que no tienen análogo en ninguna otra entidad.
- `customers` usa campos de dirección como texto libre (acepta clientes extranjeros sin DIVIPOLA
  válido); `issuers` usa FK a `departments`/`municipalities` (siempre entidades colombianas).
- `employees` (nómina, futuro): tienen `TipoContrato`, `Sueldo`, `AltoRiesgoPension`,
  `SalarioIntegral`, `CodigoTrabajador`, `NumeroCuenta`/`TipoCuenta` — datos laborales que no
  tienen nada que ver con datos fiscales de una factura. Una tabla unificada generaría columnas
  NULL masivo para cada rol.
- La fuente de verdad de los datos del tercero en un documento ya emitido **no son estas tablas**
  — es el snapshot JSONB (`documents.customer`) capturado al momento de emitir. Las tablas de
  catálogo son solo directorios de conveniencia para no retipear en la UI.

Una tabla unificada implicaría discriminadores de rol en todos los queries, índices compuestos
`(issuer_id, role)`, NULLs masivos y pérdida de FK específicas por rol. El ahorro es ruido
frente a ese costo. La comparación con ERPNext/Odoo (`res.partner` unificado) no aplica:
esos son CRMs primero; aquí el core es el pipeline DIAN, no un CRM.

**Mapa de entidades definitivo:**

| Tabla | Rol | Estado |
|---|---|---|
| `issuers` | Tenant / emisor (la empresa) | Implementado |
| `customers` | Adquirientes (directorio por emisor) | Implementado |
| `suppliers` | Vendedores no obligados (para Documento Soporte) | ✅ Implementado — ver sección 9.56 |
| `employees` | Trabajadores (para Nómina Electrónica) | Pendiente — se agrega al implementar nómina |

---

### 9.54 Hoja de ruta hacia producción — flujo comercial completo (2026-07-15)

> **Estado: diseñado, no implementado.** El motor de emisión (FE + NC + ND) está listo para
> producción. Lo que falta es el flujo de adquisición de clientes y la infraestructura de
> despliegue.

#### Estado de producción del motor de emisión

| Documento | Estado DIAN real |
|---|---|
| Factura electrónica (01) | ✅ Autorizada en habilitación real |
| Nota Crédito (91) | ✅ Aceptada en habilitación real |
| Nota Débito (92) | ✅ Aceptada en habilitación real |
| Documento Soporte (05) | ✅ Autorizado en habilitación real (CUDS-SHA384, StatusCode 00), ver 9.55 |
| Nota de Ajuste (95) | ✅ Implementada (backend + frontend completo, 2026-07-19) — pendiente prueba contra DIAN real con rango tipo "95" |

#### Modelo de precios elegido

Tarificación de dos capas, sin tiers ni upgrades:

- **Suscripción anual** (tarifa de acceso): cubre plataforma, soporte y actualizaciones.
  Precio a definir por el usuario.
- **Cobro por documento emitido**: se registra cada `confirmDocument` exitoso en una tabla
  `billing_events`. Precio por documento a definir por el usuario.
- **Sin tiers**: un solo plan para todos los clientes. Eliminates complejidad de gestión de
  planes y es fácil de explicar ("$X/año + $Y por factura").

Justificación: el cobro por documento alinea el costo operativo (DIAN, servidor, firma) con
el ingreso; la tarifa anual cubre el costo fijo. La competencia colombiana (Alegra, Siigo)
usa suscripciones mensuales con tiers — diferenciarse con simplicidad es una ventaja real en
el segmento PYME.

#### Infraestructura de despliegue

- **Plataforma**: Railway (elegida por el usuario). Un servicio Go + un servicio Postgres
  gestionado. Railway maneja SSL y dominios custom nativamente.
- **Dominio**: `cofacture.co` (ya adquirido). El dashboard vive en `app.cofacture.co` (o
  subdirectorio); la landing en `cofacture.co` (sitio estático independiente).
- **Pasarela de pago**: ePayco (elegida por el usuario). SDK Go propio del usuario:
  `github.com/diegofxm/epayco-go` — sin dependencias externas, ya tiene soporte para cobros
  con tarjeta (`Charges`), PSE (`PSE`), efectivo (`Cash`), Daviplata, webhooks con
  verificación de firma, y **suscripciones/planes** (`Plans`/`Subscriptions` vía
  `api.secure.payco.co`). Permite implementar tanto el cobro único anual como el recurrente
  sin otro SDK.
- **Correo transaccional**: SMTP propio (ya configurado en `internal/email`, sección 9.40).
  Ya usado para envío de documentos al cliente; se reutiliza sin cambios para notificaciones
  de onboarding y activación de cuenta.
- **Backups**: Railway Postgres incluye snapshots automáticos. Suficiente para la fase inicial.

#### Flujo de adquisición de clientes (diseño)

```
cofacture.co (landing estática — Astro o HTML puro con Tailwind CDN)
    → Descripción del servicio + precios + CTA "Empezar"
    → Redirige a app.cofacture.co/registro

/registro (formulario en el frontend existente)
    → Datos básicos: nombre, correo, contraseña, NIT de la empresa
    → Documentos: cédula del representante legal (PDF) + RUT (PDF)
    → Pago: checkout ePayco para la suscripción anual
    → Al completar el pago, ePayco llama al webhook de apidian

apidian recibe webhook ePayco
    → Verifica firma (epayco.VerifySignature con p_cust_id_cliente + p_key del dashboard ePayco)
    → Si pago aprobado: cambia user.account_status a "pending_review"
    → Envía correo al platform_admin: "Nueva solicitud de cuenta — NIT XXXX"
    → Envía correo al usuario: "Recibimos tu solicitud, en revisión"

Panel de admin (/admin — protegido por RequirePlatformAdmin, sección 9.35)
    → Lista solicitudes pending_review con sus documentos adjuntos
    → Botón "Aprobar": crea el issuer a partir de los datos del RUT enviado,
      cambia account_status a "active", notifica al usuario
    → Botón "Rechazar" + campo de motivo: cambia a "rejected", notifica al usuario
      con el motivo y ofrece reembolso (proceso manual en esta fase)

Cliente recibe correo "Tu cuenta en Cofacture está lista"
    → Entra a app.cofacture.co → login
    → La empresa ya está creada (el admin la creó del RUT) pero sin software/certificado
    → El wizard de configuración existente lo guía: subir certificado → registrar software → crear rango
    → Listo para emitir
```

#### Piezas de implementación pendientes

| Componente | Dónde vive | Notas |
|---|---|---|
| `account_status` en `users` | Migración `000007_users` editada | `pending_review \| active \| suspended \| rejected`; default `active` para los usuarios existentes en dev |
| `onboarding_documents` (nueva tabla) | Nueva migración `000013` | `(id, user_id, document_type, content BYTEA, content_type, uploaded_at)`. Tipos: `cedula`, `rut` |
| Upload en `/registro` | `frontend/src/pages/RegisterPage.tsx` | Dos inputs `<input type="file" accept=".pdf">`, FileReader → base64 → `POST /auth/register` ampliado |
| Webhook ePayco | `apidian/internal/api/handler_epayco.go` | `POST /api/webhooks/epayco` — sin auth JWT (llamado por ePayco), con `epayco.VerifySignature`. Actualiza `account_status`. |
| Guard por `account_status` | `internal/api/middleware/tenant.go` | `RequireTenant` ya existe; agregar chequeo de `account_status == "active"` antes de inyectar `tenantID` en el contexto |
| Panel `/admin` | `frontend/src/pages/admin/` + layout propio | Lista solicitudes, descarga PDFs, botones Aprobar/Rechazar. Layout separado de `DashboardLayout` (sección 9.35) |
| `POST /admin/users/{id}/approve` | `apidian/internal/api/handler_admin.go` | Crea issuer desde datos recibidos en el registro, activa cuenta, envía email |
| `POST /admin/users/{id}/reject` | Mismo archivo | Cambia status a rejected, envía email con motivo |
| `billing_events` (nueva tabla) | Nueva migración `000014` | `(id, issuer_id, document_id, document_type, emitted_at, amount_cents)`. Se inserta en `documents.Service.finish()` cuando `StatusCode == "00"` |
| Landing page | Repositorio separado o `frontend/public/` | Estática. No comparte código con el dashboard. Incluye precios, características, CTA |
| ePayco SDK | `go.work` / `go get github.com/diegofxm/epayco-go` | Ya existe y tiene tests. Se importa en `apidian` como cualquier dependencia |

#### Secuencia de implementación recomendada

1. **Infraestructura Railway** — configurar el proyecto, el servicio Postgres, las variables de
   entorno de producción, el dominio `app.cofacture.co`, y hacer un primer despliegue del
   estado actual. Esto valida el pipeline de deploy antes de agregar más código.
2. **`account_status` + guard** — migración + middleware. Sin esto, cualquier usuario que se
   registre tiene acceso inmediato sin validación; esto cierra ese hueco antes de abrir el
   registro al público.
3. **Upload documentos en `/registro`** — ampliar el formulario de registro para recibir
   cédula + RUT como PDFs. Guardar en `onboarding_documents`.
4. **Webhook ePayco** — `POST /api/webhooks/epayco`. Con esto el pago cierra el loop de
   registro sin intervención manual.
5. **Panel de admin** — lista de solicitudes + aprobar/rechazar. Es el único punto de
   intervención humana en el flujo.
6. **`billing_events`** — tabla + inserción en `finish()`. Permite empezar a registrar uso
   desde el día uno, aunque la facturación al cliente sea manual al principio.
7. **Landing page** — puede construirse en paralelo con los pasos 2–6; no bloquea ningún
   paso técnico.

#### Lo que deliberadamente queda fuera de esta fase

- **Facturación automática al cliente** (generación automática de la factura mensual/anual
  de Cofacture al cliente): manual en la primera fase. Con `billing_events` ya poblada, se
  puede hacer el cálculo y cobro a mano hasta que el volumen justifique automatizarlo.
- **Reembolsos automáticos**: el flujo de rechazo notifica al usuario y el reembolso es manual
  (vía panel de ePayco). Aceptable para volúmenes iniciales.
- **Invitación de usuarios por empresa** (`POST /issuers/me/members`, sección 9.35): no es
  requisito para el lanzamiento — cada empresa empieza con un solo usuario (el owner).
- **Suscripciones recurrentes automáticas con ePayco** (`Plans`/`Subscriptions`): la primera
  versión cobra manualmente o vía checkout al vencer la suscripción. Se puede automatizar
  con `epayco-go`'s `Plans`/`Subscriptions` una vez se tenga el volumen que lo justifique.

---

### 9.55 Documento Soporte (DS tipo 05, CUDS-SHA384) — implementado (2026-07-18)

El DS es el mecanismo por el que una empresa registra ante la DIAN una compra hecha a un
proveedor **no obligado a facturar** (agricultor, persona natural sin RUT activo, etc.). A
diferencia de la FE, el emisor del documento es el **comprador** (ABS), y el proveedor actúa
como Supplier (SNO).

#### Diferencias técnicas clave respecto a FE/NC/ND

| Aspecto | FE/NC/ND | DS (tipo 05) |
|---|---|---|
| Hash del documento | CUFE/CUDE (SHA256) | CUDS (SHA384) — algoritmo diferente |
| Rol Supplier en UBL | Emisor (empresa) | SNO — proveedor no obligado |
| Rol Customer en UBL | Adquiriente | ABS — empresa compradora (emisor del DS) |
| `schemeName` en CompanyID | Según tipo real | Siempre `"31"` (NIT) para el SNO |
| `TaxLevelCode` | Con o sin `listName` | **Sin `listName`** — la DIAN rechaza si está presente |
| `PostalZone` en Address | Opcional | **Obligatorio** — `"000000"` si no se conoce el real |
| `ProfileID` | `"DIAN 2.1: Factura…"` | `"DIAN 2.1: documento soporte en adquisiciones…"` |

#### Fórmula CUDS

A diferencia del CUFE (que usa tres slots fijos IVA+INC+ICA acumulados), el CUDS usa un único
par tomado de `HeaderTaxes[0]`:

```
seed = Prefix + Number + IssueDate + IssueTime +
       FormatCents(LineExtension) +
       HeaderTaxes[0].TypeCode + FormatCents(HeaderTaxes[0].TaxAmountCents) +
       FormatCents(Payable) +
       SNO.Identification.Number + ABS.Identification.Number +
       softwarePIN + EnvironmentCode
CUDS = hex(SHA384(seed))
```

Implementado en `cofacture/cuds/cuds.go` → `Compute()`. El orden es SNO+ABS (igual que CUFE
tiene OFE+ADQ). Verificado contra el CUDS oficial del ejemplo del Anexo Técnico de DS.

#### Flujo en apidian

`confirmSupportDocument` (en `internal/documents/service.go`):

1. `supplierAsNIT()` — convierte el proveedor a estructura SNO: fuerza `schemeName="31"`,
   recalcula el DV si el tipo original no era NIT, y rellena `PostalZone="000000"` si vacío.
2. `partyFromIssuerAsNIT()` — convierte el emisor al ABS: mismo tratamiento.
3. `cuds.Compute(inv, pin)` — calcula el hash.
4. `builder.BuildSupportDocument(inv)` — construye el XML UBL.
5. `finalizeAndSend(zip.KindSupportDocument)` — envía con `SendTestSetAsync` si hay `TestSetID`
   en el rango (habilitación), o con `SendBillSync` si no (producción). Mismo flujo que FE.

El DS de habilitación requiere un `test_set_id` **distinto** al de FE — la DIAN asigna uno
por tipo de documento.

#### Validación real

`cofacture/soap/realsend_support_document_test.go` envía un DS real al ambiente de
habilitación de la DIAN. Resultado: `IsValid=true`, `StatusCode=00` (autorizado).

**Fuente canónica**: `docs/reference/DS-real.xml` — DS de producción aceptado por la DIAN.
La **Caja de Herramientas DS v1.1 está DESACTUALIZADA** en party structure — no usarla.

---

### 9.56 Catálogo de Proveedores (`internal/suppliers`) — implementado (2026-07-18)

`suppliers` es el directorio de proveedores no obligados a facturar que aparecen como SNO en
los Documentos Soporte. Mismo patrón arquitectónico que `internal/customers`:

- Tabla `suppliers` en Postgres (migración `000014_suppliers.up.sql`)
- JSONB `party` column — mismo snapshot pattern que customers
- 5 endpoints: `POST /suppliers`, `GET /suppliers`, `GET /suppliers/{id}`, `PUT /suppliers/{id}`,
  `DELETE /suppliers/{id}`
- `supplierSection` en `SupportDocumentForm` del frontend — selector con búsqueda, igual que
  `CustomerSection` en `InvoiceForm`

La relación suppliers–DS es solo de conveniencia: al confirmar un DS, `documents.Service`
extrae los datos del proveedor del snapshot del documento (no de la tabla `suppliers`) — mismo
principio que clientes en FE. La tabla sirve para no retipear los datos del proveedor en
cada DS.

---

### 9.57 Dashboard de métricas de facturación (`GET /api/v1/stats/billing`) — implementado (2026-07-19)

**Endpoint**: `GET /api/v1/stats/billing` — requiere auth + tenant activo. Devuelve `BillingStats`.

**Estructura de respuesta** (todo en centavos de COP):

```json
{
  "current_month":  { "revenue_cents": 0, "document_count": 0, "accepted_count": 0, "rejected_count": 0, "draft_count": 0 },
  "previous_month": { ... },
  "ytd":            { ... },
  "by_type": [{ "type_code": "01", "type_name": "Factura Electrónica", "count": 0, "revenue_cents": 0 }],
  "series":   [{ "month": "2025-08", "revenue_cents": 0, "count": 0, "accepted_count": 0 }]
}
```

**Implementación SQL** (`internal/documents/postgres.go:GetBillingStats`, 3 queries):

1. **Períodos** — una sola query con `FILTER WHERE` para mes actual, mes anterior y YTD:
   ```sql
   SELECT
     COUNT(*) FILTER (WHERE date_trunc('month', issue_date AT TIME ZONE 'America/Bogota') = date_trunc('month', NOW() AT TIME ZONE 'America/Bogota')) AS current_count,
     SUM(payable_amount_cents) FILTER (...) AS current_revenue,
     ...
   FROM documents WHERE issuer_id = $1 AND status != 'draft'
   ```
2. **Por tipo** — `GROUP BY dian_document_type_code` del mes actual.
3. **Serie mensual** — `GROUP BY date_trunc('month', issue_date)` últimos 12 meses.

Todas las queries usan `AT TIME ZONE 'America/Bogota'` para acotar correctamente el "mes
actual" en horario colombiano (UTC-5, sin DST).

**Archivos involucrados**:
- `internal/documents/stats.go` — structs `BillingStats`, `PeriodStats`, `TypeStats`, `MonthSeries`
- `internal/documents/repository.go` — interfaz `GetBillingStats`
- `internal/documents/postgres.go` — implementación SQL
- `internal/documents/memory_repository.go` — stub para tests (devuelve slices vacíos)
- `internal/api/handler_stats.go` — handler `handleGetBillingStats`
- `internal/api/api.go` — ruta `GET /api/v1/stats/billing`

**Frontend** (`DashboardPage.tsx`): usa Recharts (instalado vía `npm install recharts`).
- 4 KPI cards: documentos mes actual, ingresos mes actual, tasa de aceptación, ingresos YTD.
- Gráfica de área (12 meses) con gradiente `url(#revenueGradient)`.
- Gráfica de barras horizontales por tipo de documento (colores por tipo: FE=#3498db,
  NC=#f39c12, ND=#27ae60, DS=#9b59b6).
- Tabla de actividad reciente: últimos 6 documentos con `StatusBadge`.
- Peticiones en paralelo (`getBillingStats()` + `listDocuments({limit:6})`) en un `useEffect`
  que se re-ejecuta cuando cambia `activeIssuer?.id`.

### 9.58 `GET /api/v1/dian/numbering-ranges` — sincronizar rangos desde la DIAN (2026-07-22)

**Propósito**: consultar directamente a la DIAN las resoluciones de numeración asociadas al
software del emisor autenticado y pre-llenar los campos al registrar un rango nuevo.

**Endpoint**: `GET /api/v1/dian/numbering-ranges` — requiere auth + tenant activo + emisor con
certificado y `software_id` configurados (sino `ErrIssuerNotReadyToIssue` → HTTP 422).

**Respuesta**:
```json
{
  "ranges": [
    {
      "resolution_number": "18760000001",
      "resolution_date": "2019-01-19",
      "prefix": "SETP",
      "range_from": 990000000,
      "range_to": 999999999,
      "valid_from": "2019-01-19",
      "valid_to": "2030-01-19",
      "technical_key": "fc8eac422eba16e22ffd8c6f94b3f40a...",
      "suggested_doc_type_code": "01"
    }
  ]
}
```

**Implementación**:
- `apidian/internal/documents/dian_ranges.go` — `Service.GetDianNumberingRanges`:
  - Mismo patrón que `VerifyAcquirer` (vive en `documents`, no en `numbering`, porque necesita
    el certificado PKCS12 del emisor y el cliente SOAP cofacture)
  - `trimISODateTime`: convierte "2019-01-19T00:00:00" → "2019-01-19"
  - `inferDocTypeFromPrefix`: "SETP"→"01" (Factura), "SEDS"→"05" (DS), resto→""
  - Detecta HAB vs PRD de `issuer.Environment`, igual que el resto del pipeline
- `apidian/internal/api/handler_documents.go` — `handleGetDianNumberingRanges` + `dianRangeItem`
- `apidian/internal/api/api.go` — ruta `GET /api/v1/dian/numbering-ranges`

**De los 10 campos de un `numbering_range`**, 8 se auto-llenan del SOAP (resolution_number,
resolution_date, prefix, range_from, range_to, valid_from, valid_to, technical_key); el usuario
solo elige `dian_document_type_code` (pre-sugerido) y `test_set_id` (solo en HAB).

Ver `docs/frontend-architecture.md` para el UX del modal "Sincronizar con DIAN".
