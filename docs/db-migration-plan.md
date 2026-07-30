# Plan de Migración de Arquitectura de Base de Datos

> Creado: julio 2026  
> Rama objetivo: `v2/db-architecture`  
> Estado: **pendiente de ejecución**

---

## Contexto y motivación

### El problema que origina este plan

Durante el desarrollo del módulo `accounting/forex/`, se detectó que la arquitectura de BD
actual mezcla responsabilidades de forma inconsistente:

- `public.currencies` debería ser un catálogo compartido, pero `accounting.exchange_rates`
  guarda `from_currency VARCHAR(3)` sin FK porque se evitaron las FKs cruzadas de esquema.
- `public.tax_types` tiene un nombre ambiguo — se confunde con el módulo `accounting/tax/`
  que estaba siendo construido, aunque son cosas completamente distintas.
- El dominio de documentos electrónicos (FE, NC, ND, DS, NA) vive en `apidian/internal/`,
  mezclado con la capa HTTP/SaaS del API, impidiendo su reutilización como librería.

### Por qué hacerlo ahora

- No hay clientes ni usuarios reales en producción — costo de migración de datos: cero.
- Ya se tienen `cofacture/` y `accounting/` como ejemplos funcionales del patrón correcto
  (librerías Go puras con schema propio).
- Cada módulo nuevo que se agregue (payroll, inventory, purchasing) heredará el problema
  si no se corrige la base primero.

### El error de diseño original

El módulo `accounting` se diseñó con una regla tomada de **microservicios**: _"no FKs entre
esquemas — cada servicio dueño de su propia base de datos"_. Pero aquí no hay microservicios.
Hay un **monolito modular** con una sola base de datos. El resultado fue el peor de ambos mundos:
acoplamiento de monolito sin integridad referencial de monolito.

La regla correcta para un monolito modular con PostgreSQL es:

> **FKs cruzadas entre schemas: sí** (integridad de datos, responsabilidad de la BD)  
> **Imports cruzados entre módulos Go: no** (acoplamiento de código, responsabilidad de Go)

---

## Decisión de naming de módulos

| Módulo | Nombre | Razón |
|--------|--------|-------|
| `cofacture/` | sin cambio | librería XML DIAN — nombre es marca, no dominio |
| `catalogs/` | nuevo | datos de referencia DIAN/DANE compartidos |
| `edocuments/` | nuevo (extraído de `apidian/`) | documentos electrónicos fiscales |
| `accounting/` | sin cambio | motor contable |
| `apidian/` | reestructurado | API + orquestador SaaS |
| `payroll/` | futuro | nómina |
| `inventory/` | futuro | inventarios |
| `purchasing/` | futuro | compras |
| `hr/` | futuro | recursos humanos |

**¿Por qué `edocuments/` y no `dian/`?**  
`dian/` sería ambiguo: `payroll` también habla con la DIAN (nómina electrónica). El módulo
`edocuments` es el dominio específico de **documentos electrónicos fiscales** (FE, NC, ND,
DS, NA) — no toda la DIAN. Un futuro desarrollador que lea el nombre entiende el dominio
sin saber el contexto histórico del proyecto.

---

## Arquitectura de schemas propuesta

### Principio de capas

```
catalogs        datos de referencia — nadie les tiene FK cruzada entre sí
    ↑
public          capa SaaS de apidian (usuarios, empresas, planes)
edocuments      documentos electrónicos fiscales
accounting      motor contable
payroll         nómina                                        [futuro]
inventory       inventarios                                   [futuro]
purchasing      compras                                       [futuro]
```

**Reglas de FK entre schemas:**
1. Cualquier schema puede tener FK hacia `catalogs.*` ✅
2. Ningún módulo de dominio tiene FK hacia `public.*` — `issuer_id`/`company_id` viajan
   como `UUID NOT NULL` sin FK; el API valida existencia antes de llamar al módulo ✅
3. Los módulos de dominio no tienen FK entre sí — el vínculo es vía
   `source_document_id UUID` en los asientos contables ✅

---

## Schema `catalogs` — datos de referencia

Todos vienen de `public` (migración `000002_catalogs.up.sql` de apidian actual).
Un único rename: `tax_types` → `dian_tax_types` para eliminar la ambigüedad con el módulo
`accounting/tax/`.

```
catalogs.currencies              ← era public.currencies
catalogs.departments             ← era public.departments
catalogs.municipalities          ← era public.municipalities        FK→departments
catalogs.identification_types    ← era public.identification_types
catalogs.payment_methods         ← era public.payment_methods
catalogs.payment_terms           ← era public.payment_terms
catalogs.dian_tax_types          ← era public.tax_types (RENOMBRADA)
catalogs.tax_regimes             ← era public.tax_regimes
catalogs.liability_codes         ← era public.liability_codes
catalogs.unit_measures           ← era public.unit_measures
catalogs.dian_document_types     ← era public.dian_document_types
catalogs.ciiu_codes              ← era public.ciiu_codes
```

### Tablas de seed que se mueven

| Seed actual | Nuevo destino |
|-------------|---------------|
| `apidian/internal/database/seed/currencies.csv` | `catalogs/database/seed/` |
| `apidian/internal/database/seed/municipalities.csv` | `catalogs/database/seed/` |
| `apidian/internal/database/seed/departments.csv` | `catalogs/database/seed/` |
| `apidian/internal/database/seed/identification_types.csv` | `catalogs/database/seed/` |
| `apidian/internal/database/seed/payment_methods.csv` | `catalogs/database/seed/` |
| `apidian/internal/database/seed/payment_terms.csv` | `catalogs/database/seed/` |
| `apidian/internal/database/seed/tax_types.csv` | `catalogs/database/seed/dian_tax_types.csv` |
| `apidian/internal/database/seed/tax_regimes.csv` | `catalogs/database/seed/` |
| `apidian/internal/database/seed/liability_codes.csv` | `catalogs/database/seed/` |
| `apidian/internal/database/seed/unit_measures.csv` | `catalogs/database/seed/` |
| `apidian/internal/database/seed/dian_document_types.csv` | `catalogs/database/seed/` |
| `apidian/internal/database/seed/ciiu_codes.csv` | `catalogs/database/seed/` |

---

## Schema `public` — capa SaaS de apidian

Solo lo que es del API como plataforma multi-tenant. Todos los campos que hoy referencian
tablas de `public` (ej. `tax_types`) pasan a referenciar `catalogs.*`.

```sql
public.users
    id, email, password_hash, name, role, is_superadmin, is_active

public.issuers                                    -- FKs → catalogs.*
    id, nit, check_digit, business_name, trade_name
    identification_type_code → catalogs.identification_types
    department_code          → catalogs.departments
    municipality_code        → catalogs.municipalities
    tax_scheme_code          → catalogs.dian_tax_types   -- antes: public.tax_types
    tax_regime_code          → catalogs.tax_regimes
    liability_codes          TEXT[]    -- sin FK, Postgres no soporta FK sobre arrays
    industry_classification_codes TEXT[]
    software_id, software_pin(enc), certificate(enc), certificate_password(enc)
    logo, logo_content_type, email_body_template
    environment, entity_type_code, tax_scheme_name
    merchant_registration_number, is_active

public.user_issuers
    user_id → public.users, issuer_id → public.issuers

public.plans
    id, name, description, max_documents_per_month, max_issuers, price_cop, is_active

public.subscriptions
    issuer_id → public.issuers, plan_id → public.plans, status, started_at, ends_at

public.issuer_settings
    issuer_id → public.issuers, brand_color

public.invitations
public.payments
public.prospects
public.audit_events
```

---

## Schema `edocuments` — documentos electrónicos fiscales

Hoy vive en `apidian/internal/{documents,customers,suppliers,products,numbering}/`.
FKs solo hacia `catalogs.*`. `issuer_id UUID NOT NULL` sin FK a `public.issuers`
(el API valida existencia; mismo patrón que `accounting` usa para `company_id`).

```sql
edocuments.customers                              -- FKs → catalogs.*
    id, issuer_id UUID NOT NULL                   -- sin FK a public.issuers
    entity_type_code
    identification_number
    identification_type_code → catalogs.identification_types
    identification_verification_code
    name, address_line
    address_city_code, address_city_name          -- sin FK (puede ser extranjero)
    address_state_code, address_state_name
    address_country_code, address_country_name
    tax_scheme_code  → catalogs.dian_tax_types
    tax_scheme_name
    tax_regime_code  → catalogs.tax_regimes
    liability_codes  TEXT[]
    phone, email, merchant_registration_number

edocuments.suppliers                                -- misma estructura que customers
    id, issuer_id UUID NOT NULL
    ... (ídem customers)

edocuments.products                               -- FKs → catalogs.*
    id, issuer_id UUID NOT NULL
    description
    unit_measure_code → catalogs.unit_measures    -- HOY sin FK, ahora correcto
    unit_price_cents BIGINT
    item_code, item_type_code, item_type_name, item_type_agency_id
    tax_type_code → catalogs.dian_tax_types
    tax_type_name, tax_percent

edocuments.numbering_ranges                       -- FKs → catalogs.*
    id, issuer_id UUID NOT NULL
    dian_document_type_code → catalogs.dian_document_types
    prefix, resolution_number, resolution_date
    range_from, range_to, current_number
    valid_from, valid_to, environment
    technical_key BYTEA (cifrada AES-256-GCM)
    test_set_id, is_active

edocuments.documents                              -- FKs → catalogs.* + edocuments.*
    id UUID PRIMARY KEY
    issuer_id UUID NOT NULL                       -- sin FK a public.issuers
    numbering_range_id    → edocuments.numbering_ranges
    dian_document_type_code → catalogs.dian_document_types
    currency_code         → catalogs.currencies   -- HOY → public.currencies, ahora correcto
    prefix, number BIGINT, document_key TEXT
    issue_date DATE, issue_time TEXT
    customer JSONB NOT NULL                       -- snapshot firmado, inmutable
    customer_id → edocuments.customers ON DELETE SET NULL
    supplier JSONB                                  -- snapshot firmado, inmutable (DS/NA)
    supplier_id → edocuments.suppliers ON DELETE SET NULL
    lines JSONB NOT NULL
    payment_means JSONB NOT NULL DEFAULT '[]'
    withholding_taxes JSONB                       -- DS/NA
    totals_line_extension_cents BIGINT
    totals_tax_exclusive_cents  BIGINT
    totals_tax_inclusive_cents  BIGINT
    totals_prepaid_cents        BIGINT DEFAULT 0
    totals_payable_cents        BIGINT
    billing_reference JSONB                       -- solo NC/ND
    discrepancy_response JSONB                    -- solo NC/ND
    note_type_code VARCHAR(2)                     -- solo NC
    note TEXT
    qr_url TEXT, signed_xml TEXT                  -- retención legal, inmutable post-firma
    status VARCHAR(20) NOT NULL
    dian_track_id TEXT
    dian_status_code TEXT, dian_status_description TEXT
    dian_status_message TEXT
    application_response_xml TEXT
```

---

## Schema `accounting` — ajustes de FKs únicamente

La estructura interna no cambia. Solo se agregan las FKs que hoy "vuelan" hacia `catalogs.*`,
como parte de las nuevas migraciones del módulo `tax/`.

```
accounting.exchange_rates
    from_currency → catalogs.currencies   ← HOY VARCHAR sin FK
    to_currency   → catalogs.currencies   ← HOY VARCHAR sin FK

accounting.journal_lines
    foreign_currency → catalogs.currencies (FK nullable) ← HOY VARCHAR sin FK

accounting.ica_tariffs    [nuevo, aún no escrito]
    municipality_code → catalogs.municipalities
    ciiu_code         → catalogs.ciiu_codes

accounting.ica_declarations   [nuevo, aún no escrito]
    municipality_code → catalogs.municipalities
```

---

## Mapa completo de FKs cruzadas (estado final)

```
catalogs.currencies
    ← edocuments.documents.currency_code
    ← accounting.exchange_rates.from_currency / to_currency
    ← accounting.journal_lines.foreign_currency

catalogs.departments
    ← catalogs.municipalities.department_code
    ← public.issuers.department_code

catalogs.municipalities
    ← public.issuers.municipality_code
    ← accounting.ica_tariffs.municipality_code
    ← accounting.ica_declarations.municipality_code

catalogs.dian_document_types
    ← edocuments.documents.dian_document_type_code
    ← edocuments.numbering_ranges.dian_document_type_code

catalogs.dian_tax_types          (antes: public.tax_types — ambigüedad eliminada)
    ← public.issuers.tax_scheme_code
    ← edocuments.customers.tax_scheme_code
    ← edocuments.suppliers.tax_scheme_code
    ← edocuments.products.tax_type_code

catalogs.identification_types
    ← public.issuers.identification_type_code
    ← edocuments.customers.identification_type_code
    ← edocuments.suppliers.identification_type_code

catalogs.tax_regimes
    ← public.issuers.tax_regime_code
    ← edocuments.customers.tax_regime_code
    ← edocuments.suppliers.tax_regime_code

catalogs.unit_measures
    ← edocuments.products.unit_measure_code   (HOY sin FK — ahora correcto)

catalogs.ciiu_codes
    ← accounting.ica_tariffs.ciiu_code
    ← accounting.ica_declarations.ciiu_code
```

---

## Estructura Go final

```
project-ubl/
├── go.work                       ← lista todos los módulos del workspace
│
├── cofacture/                ✅  sin cambios — librería XML DIAN
│   └── go.mod: github.com/diegofxm/cofacture
│
├── catalogs/                 🔲  NUEVO
│   ├── go.mod: github.com/diegofxm/catalogs
│   ├── catalogs.go               Migrate(), Seed()
│   └── database/
│       ├── migrations/           000001_catalogs.up.sql / .down.sql
│       └── seed/                 *.csv (movidos de apidian/internal/database/seed/)
│
├── edocuments/               🔲  NUEVO (extraído de apidian/internal/)
│   ├── go.mod: github.com/diegofxm/edocuments
│   ├── edocuments.go             Migrate()
│   ├── documents/                model, service, postgres, pdf
│   ├── customers/
│   ├── suppliers/
│   ├── products/
│   ├── numbering/
│   └── database/
│       └── migrations/           000001_edocuments.up.sql / .down.sql
│
├── accounting/               🔲  ajuste menor de FKs en nuevas migrations
│   └── go.mod: github.com/diegofxm/accounting
│
├── apidian/                  🔲  reestructurado — solo API + SaaS
│   ├── go.mod: github.com/diegofxm/apidian
│   ├── cmd/server/
│   ├── internal/
│   │   ├── api/              ← handlers HTTP (importan edocuments + accounting)
│   │   ├── auth/
│   │   ├── config/
│   │   ├── server/
│   │   ├── issuers/          ← gestión de emisores/tenants (SaaS, queda aquí)
│   │   ├── users/
│   │   ├── plans/
│   │   ├── subscriptions/
│   │   ├── payments/
│   │   ├── prospects/
│   │   ├── audit/
│   │   └── integrations/
│   │       └── accounting/   ← adaptador edocuments → accounting (queda aquí)
│   └── database/
│       └── migrations/       public schema: users, issuers, plans, subscriptions...
│
├── payroll/                  🔵  futuro — mismo patrón que edocuments
├── inventory/                🔵  futuro
├── purchasing/               🔵  futuro
├── hr/                       🔵  futuro
└── frontend/                 ✅  sin cambios
```

---

## Orden de migración en startup

```go
// apidian/cmd/server/main.go
catalogs.Migrate(dbURL)     // primero — todos dependen de estos schemas
accounting.Migrate(dbURL)   // segundo — independiente de edocuments y public
edocuments.Migrate(dbURL)   // tercero — depende de catalogs en BD, no en Go
apidian.Migrate(dbURL)      // último — public schema; issuers FK→catalogs.municipalities etc.
```

---

## Plan de ejecución paso a paso

```
Paso 1  git checkout -b v2/db-architecture
        Crear módulo catalogs/ con go.mod, Migrate(), migrations + seed

Paso 2  Crear módulo edocuments/ con go.mod, Migrate(), migrations
        Mover tablas de public → edocuments schema

Paso 3  Reescribir migrations de apidian/ (public schema)
        Actualizar FKs que antes apuntaban a public.* → ahora a catalogs.*

Paso 4  Ajustar accounting/ migrations
        Agregar FKs de exchange_rates + journal_lines → catalogs.currencies
        Agregar FKs de ica_tariffs + ica_declarations → catalogs.municipalities / ciiu_codes

Paso 5  Actualizar go.work
        Agregar catalogs/ y edocuments/ como módulos del workspace

Paso 6  go build ./... — debe compilar limpio

Paso 7  Mover código Go de apidian/internal/{documents,customers,suppliers,products,numbering}/
        hacia edocuments/ con sus nuevos package paths

Paso 8  Actualizar imports en apidian/internal/api/ e integrations/
        Apuntar a github.com/diegofxm/edocuments en vez de paquetes internos

Paso 9  go test ./... — todos los tests deben pasar verdes

Paso 10 Merge a main cuando esté verde y documentado
```

---

## Lo que NO cambia

- `cofacture/` — sin cambios, ya es correcto
- Los endpoints HTTP de `apidian/api/` — mismo contrato REST, el frontend no se toca
- La lógica de dominio existente — se mueve, no se reescribe
- Los tests existentes — se re-apuntan con nuevos imports, no se reescriben
- El módulo `accounting/` internamente — solo se agregan FKs en migrations nuevas

---

*Documento creado como antecedente del rediseño de arquitectura de BD — julio 2026.*  
*Actualizar este documento si cambian decisiones durante la ejecución.*
