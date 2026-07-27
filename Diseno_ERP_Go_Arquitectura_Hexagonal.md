# Diseño de un ERP en Go con Arquitectura Hexagonal

## Filosofía

Si fuera a diseñar un ERP moderno en Go pensando en que dure 15 o 20 años, **no haría un monolito tradicional**, pero tampoco empezaría con microservicios. Haría un **Monolito Modular (Modular Monolith)** usando **Arquitectura Hexagonal (Ports & Adapters)** y **DDD (Domain Driven Design)**.

¿Por qué?

Porque un ERP tiene cientos de relaciones entre módulos.

- Un empleado pertenece a una compañía.
- La nómina genera asientos contables.
- La factura electrónica afecta cartera.
- Inventario afecta contabilidad.
- Compras afectan proveedores.
- Ventas afectan clientes.

Separar todo en APIs desde el principio termina siendo un infierno de sincronización.

Yo haría un solo backend en Go y módulos completamente desacoplados.

---

## Un solo módulo Go

**Una sola `go.mod` en la raíz del proyecto.** Nada de go.work, nada de módulos separados por dominio. El monolito modular es un único binario y su código fuente vive en `internal/`. La separación es conceptual (packages, carpetas), no física (módulos Go independientes).

Las librerías que sí son reutilizables por terceros (como `cofacture` para la generación de documentos DIAN) viven en su propio repositorio con su propio `go.mod`. Todo lo demás: interno.

---

## Arquitectura General

```text
                 Frontend
          (Vue / React / Flutter)

                    │

             HTTP / REST

                    │

             API Gateway net/http

                    │
────────────────────────────────────────
                Application
────────────────────────────────────────

      Company Module
      Customer Module
      Supplier Module
      Product Module
      Inventory Module
      Purchase Module
      Sales Module
      Accounting Module
      Payroll Module
      HR Module
      Electronic Documents
      Catalog Module
      Security Module
      Shared Utilities

────────────────────────────────────────
                 Domain
────────────────────────────────────────

Entities
Aggregates
Value Objects
Domain Events
Repositories (Ports)

────────────────────────────────────────
            Infrastructure
────────────────────────────────────────

PostgreSQL
SMTP / Resend / SES
S3
DIAN (vía cofacture)
PDF
XML / SOAP
```

> **Nota v1:** Redis y RabbitMQ **no son parte del arranque**. Redis entra cuando haya un caso real de caché de alto costo o sesiones distribuidas. RabbitMQ/NATS entra cuando haya un caso real de comunicación asíncrona con sistemas externos. Hasta entonces, los eventos de dominio son llamadas Go en memoria dentro del mismo proceso — eso es suficiente y mucho más simple de depurar.

---

## Un solo Go module, organización por dominio

No se organiza el proyecto por capas (`handlers`, `models`, `repositories`, `services`), sino por dominio.

```text
erp/
    cmd/
        server/
            main.go         ← único punto de arranque y wire-up de todo el sistema
    internal/
        company/
        customer/
        supplier/
        product/
        inventory/
        purchase/
        sales/
        accounting/
        payroll/
        hr/
        electronic/
        security/
        catalog/
        shared/             ← utilidades transversales: dinero, fechas, paginación, tenant, eventos
    go.mod
```

Cada módulo es completamente independiente y se comunica con los demás solo a través de eventos de dominio o interfaces (ports) explícitas.

---

## Multi-tenancy como ciudadano de primera clase

Cada registro en la base de datos pertenece a una compañía (`company_id`). Esto no es una responsabilidad de cada módulo individual — es una preocupación transversal que vive en `shared/tenant/`.

```text
shared/
    tenant/
        context.go      ← GetCompanyID(ctx) uuid.UUID
        middleware.go   ← extrae el company_id del JWT y lo inyecta al ctx de cada request
```

El middleware corre antes que cualquier handler. Cada repositorio recibe el `ctx` y extrae el `company_id` de ahí, nunca de un parámetro suelto que alguien pueda olvidar. Si el `company_id` no está en el contexto, la request no llega al dominio.

```go
// shared/tenant/context.go
type contextKey struct{}

func WithCompanyID(ctx context.Context, id uuid.UUID) context.Context {
    return context.WithValue(ctx, contextKey{}, id)
}

func GetCompanyID(ctx context.Context) uuid.UUID {
    id, _ := ctx.Value(contextKey{}).(uuid.UUID)
    return id
}
```

---

## Estructura de un módulo

```text
customer/
    domain/
        customer.go         ← entidad principal
        address.go
        contact.go
        repository.go       ← puerto (interface): Save, Update, Delete, GetByID, List
        errors.go
        events.go           ← CustomerCreated, CustomerUpdated
        notifier.go         ← puerto para notificaciones (email de bienvenida, etc.)
    application/
        create_customer.go  ← caso de uso: validar, guardar, emitir evento
        update_customer.go
        delete_customer.go
        get_customer.go
        search_customer.go
    infrastructure/
        persistence/
            postgres/
                repository.go       ← implementación concreta del puerto
                migrations/
                    000001_create_customers.up.sql
                    000001_create_customers.down.sql
    interfaces/
        http/
            handler.go      ← HTTP → DTO → UseCase → Respuesta. Solo transforma, no decide.
```

### Domain

No conoce HTTP, PostgreSQL, Fiber, ni JSON. Solo conoce el lenguaje del negocio.

### Application (casos de uso)

Flujo típico de un caso de uso:

```text
CreateCustomer
    ↓
Validar campos (dominio)
    ↓
Verificar que la empresa existe (via CompanyPort)
    ↓
Guardar (CustomerRepository)
    ↓
Emitir CustomerCreated (EventBus)
```

### Interfaces

El handler HTTP solo transforma — nunca decide:

```text
HTTP Request
    ↓
Parsear y validar DTO
    ↓
Llamar al caso de uso
    ↓
Serializar respuesta HTTP
```

### Infrastructure

Implementaciones concretas de los puertos. El dominio nunca importa un package de `infrastructure/`.

---

## Migraciones con el módulo que les corresponde

Las migraciones viven dentro del módulo que las necesita, no en una carpeta global:

```text
internal/
    customer/
        infrastructure/
            persistence/
                postgres/
                    migrations/
                        000001_create_customers.up.sql
    sales/
        infrastructure/
            persistence/
                postgres/
                    migrations/
                        000001_create_sales.up.sql
```

Cada módulo usa su propia tabla de tracking:

```
x-migrations-table=customer_schema_migrations
x-migrations-table=sales_schema_migrations
```

Ventaja: borrar el módulo `customer/` también borra su historia de migraciones. Sin artefactos huérfanos.

---

## Eventos de dominio (bus en memoria)

Para la comunicación entre módulos dentro del monolito, los eventos de dominio son llamadas Go en el mismo proceso. Simple, trazable, sin dependencias externas.

```text
shared/
    events/
        bus.go          ← Register(eventType, handler) / Publish(event)
        events.go       ← InvoiceConfirmed, PayrollGenerated, StockMoved, etc.
```

Ejemplo de flujo:

```text
Sales confirma una factura
    ↓
Publica InvoiceConfirmed{...}
    ↓
Accounting.OnInvoiceConfirmed → crea asiento contable
Inventory.OnInvoiceConfirmed  → descuenta stock
Electronic.OnInvoiceConfirmed → genera XML y envía a la DIAN
```

Ninguno de los suscriptores conoce a Sales. Sales no conoce a ninguno de ellos. Solo conoce el evento.

Cuando en el futuro se necesite garantía de entrega o comunicación con sistemas externos, el bus en memoria se reemplaza por uno sobre RabbitMQ/NATS/Kafka — sin tocar el dominio.

---

## Repositorios

```go
// customer/domain/repository.go
type Repository interface {
    Save(ctx context.Context, c Customer) (*Customer, error)
    Update(ctx context.Context, c Customer) (*Customer, error)
    Delete(ctx context.Context, id uuid.UUID) error
    GetByID(ctx context.Context, id uuid.UUID) (*Customer, error)
    List(ctx context.Context, filter Filter) ([]*Customer, error)
}
```

Implementación en infrastructure:

```go
// customer/infrastructure/persistence/postgres/repository.go
type PostgresRepository struct {
    pool *pgxpool.Pool
}

func (r *PostgresRepository) Save(ctx context.Context, c domain.Customer) (*domain.Customer, error) {
    companyID := tenant.GetCompanyID(ctx) // siempre del contexto, nunca de un parámetro suelto
    // ...
}
```

---

## Wire-up: cmd/server/main.go

Todo el cableado de dependencias vive en `main.go`. Sin framework de DI — Go no lo necesita. Un `main.go` largo y explícito es una virtud: se puede leer de arriba abajo y entender exactamente qué está conectado a qué.

```go
// cmd/server/main.go
func main() {
    pool := mustOpenDB(cfg.DatabaseURL)

    // Migraciones
    customer.Migrate(cfg.DatabaseURL)
    sales.Migrate(cfg.DatabaseURL)
    // ...

    // Repos
    customerRepo := customerPostgres.NewRepository(pool)
    salesRepo    := salesPostgres.NewRepository(pool)

    // Bus de eventos
    bus := events.NewBus()

    // Casos de uso
    createCustomer := customerApp.NewCreateCustomer(customerRepo, bus)
    // ...

    // Handlers HTTP
    mux := http.NewServeMux()
    customerHTTP.Register(mux, createCustomer, ...)
    salesHTTP.Register(mux, ...)

    http.ListenAndServe(cfg.Addr, mux)
}
```

---

## Procesos de negocio

Ejemplo del ciclo completo de ventas:

```text
Cotización
    ↓
Pedido
    ↓
Despacho
    ↓
Factura
    ↓ (evento InvoiceConfirmed)
DIAN (Electronic)
    ↓ (evento InvoiceConfirmed)
Contabilidad (asiento)
    ↓ (evento InvoiceConfirmed)
Inventario (descuento de stock)
    ↓ (evento InvoiceConfirmed)
Cartera (cuenta por cobrar)
```

---

## Organización de módulos

### Company

```text
company/
    company.go          ← entidad raíz: nombre, NIT, configuración fiscal
    branch.go           ← sucursales
    warehouse.go        ← bodegas (usa Inventory)
    configuration.go    ← parámetros ERP por empresa
```

### Customer

```text
customer/
    customer.go
    contact.go
    address.go
    tax.go              ← datos tributarios del cliente
    credit.go           ← cupo de crédito, condiciones de pago
```

### Supplier

```text
supplier/
    supplier.go
    payment_terms.go
    contact.go
```

### Product

```text
product/
    product.go
    category.go
    unit.go
    barcode.go
    price.go
    tax.go
```

### Inventory

```text
inventory/
    warehouse.go
    movement.go         ← entrada, salida, traslado
    stock.go            ← stock actual por producto/bodega
    lot.go              ← lotes (vencimientos, trazabilidad)
    serial.go           ← seriales
```

### Sales

```text
sales/
    quotation/
    order/
    shipment/
    invoice/
    payment/
    receivable/         ← cartera
```

### Accounting

Es el corazón del ERP. Todo lo que genera dinero o lo mueve termina aquí.

```text
accounting/
    chart/              ← plan de cuentas
    journal/            ← libro diario
    ledger/             ← libro mayor
    fiscal/             ← períodos fiscales, cierre
    period/
    posting/            ← motor de posteo: PostDocument(event) → asiento
```

El motor de posteo recibe eventos de dominio y genera asientos contables:

- Factura de Venta → Débito Cartera / Crédito Ingresos / Crédito IVA por pagar
- Factura de Compra → Débito Inventario o Gasto / Crédito Cuentas por Pagar
- Nómina → Débito Gasto de Personal / Crédito Provisiones / Crédito Bancos
- Pago → Débito Bancos / Crédito Cartera

### Payroll

```text
payroll/
    employee/
    contract/
    payslip/            ← liquidación mensual
    deduction/
    earning/
    social_security/    ← salud, pensión, ARL, caja, SENA, ICBF
    vacation/
    severance/
    liquidation/        ← liquidación definitiva
```

### HR

```text
hr/
    employee/           ← ficha del empleado (datos personales, historia)
    attendance/         ← control de asistencia
    absence/
    leave/              ← vacaciones solicitadas vs. ejecutadas
    recruitment/
```

### Electronic Documents

Motor documental que orquesta el ciclo DIAN. Delega la construcción del XML a `cofacture` (librería externa).

```text
electronic/
    domain/
        document.go     ← Document: borrador → confirmado → enviado → aceptado/rechazado
        signer.go       ← puerto: Sign(xml) (implementado por cofacture)
        sender.go       ← puerto: Send(zip) a la DIAN (implementado por cofacture)
        validator.go
```

Reutilizado por:

- Factura Electrónica (tipo 01)
- Documento Soporte (tipo 05)
- Nota Crédito (tipo 91)
- Nota Débito (tipo 92)
- Nota de Ajuste (tipo 95)
- Nómina Electrónica (tipo 103/102)

### Catalog

```text
catalog/
    countries/
    cities/
    currencies/
    taxes/
    units/
    banks/
    eps/
    arl/
    pension/
    occupations/
    document_types/
```

### Security

```text
security/
    user/
    role/
    permission/
    session/            ← JWT + refreshtoken
    audit/              ← log de acciones sensibles
```

---

## Reportería: capa de lectura (CQRS ligero)

Los reportes no son casos de uso del dominio — son consultas que cruzan múltiples schemas y necesitan joins específicos. Intentar implementarlos con los repositorios de dominio llena esos repositorios de métodos que no pertenecen ahí.

La solución: una capa de lectura separada en `shared/queries/`.

```text
shared/
    queries/
        sales_summary.go        ← SELECT directo multi-schema, sin pasar por repos de dominio
        payroll_report.go
        balance_sheet.go
        income_statement.go
        inventory_valuation.go
```

Las queries leen directamente de la base de datos con SQL optimizado para el caso específico. El dominio escribe a través de sus repositorios. Las queries solo leen. Eso es CQRS en su forma más simple y práctica.

---

## Shared: solo utilidades transversales

```text
shared/
    money/          ← tipo Money (cents int64, currency string) con operaciones seguras
    date/           ← helpers de fecha (colombiana, fiscal, etc.)
    pagination/     ← Cursor, Offset, Page
    uuid/           ← helpers
    validation/     ← validadores reutilizables
    tenant/         ← context.go, middleware.go (ver sección Multi-tenancy)
    events/         ← bus.go, events.go
    queries/        ← capa de lectura cross-module
    email/          ← shared porque TODOS los módulos lo usan
        domain/
            notifier.go     ← puerto (interface)
        infrastructure/
            smtp/
            resend/
            ses/
        templates/
```

**Regla de oro de `shared/`:** solo entra lo que es reutilizado por 3 o más módulos independientes y no contiene lógica de negocio propia. Las entidades de dominio nunca van aquí.

---

## Comunicación entre módulos

Los módulos se comunican exclusivamente a través de eventos de dominio. Ningún módulo importa directamente el package de dominio de otro.

```text
InvoiceConfirmed (publicado por Sales)
    ↓
Inventory.OnInvoiceConfirmed → DecreaseStock
    ↓
Accounting.OnInvoiceConfirmed → CreateJournalEntry
    ↓
Electronic.OnInvoiceConfirmed → GenerateXML → SendToDIAN
```

Eventos principales:

- `OrderConfirmed`
- `InvoiceConfirmed`
- `InvoiceCancelled`
- `PaymentReceived`
- `EmployeeCreated`
- `PayrollGenerated`
- `StockReserved`
- `StockMoved`
- `PurchaseReceived`

---

## Base de datos

Una sola base PostgreSQL. Un schema por módulo de dominio.

```text
company.*
customer.*
supplier.*
sales.*
inventory.*
payroll.*
hr.*
accounting.*
catalog.*
security.*
electronic.*
```

Cada módulo es responsable de sus propias migraciones y de no hacer queries directas a schemas de otros módulos. Si necesita datos de otro módulo, los obtiene a través de un evento o de un puerto explícito.

---

## Pila tecnológica v1

| Necesidad | Decisión v1 | Cuándo cambiar |
|---|---|---|
| Base de datos | PostgreSQL (schemas) | No cambiar |
| HTTP | `net/http` estándar | Si se necesita routing avanzado: chi/gorilla |
| Eventos | Bus en memoria (goroutines) | Cuando haya caso real de durabilidad o sistemas externos |
| Email | SMTP directo | Cuando el volumen exija un servicio transaccional (Resend/SES) |
| Caché | Sin caché | Cuando haya consultas medibles y costosas |
| Cola | Sin cola | Cuando haya integración con sistemas externos que requieran async |
| ORM | Sin ORM — pgx directo | No cambiar |
| DI | Manual en main.go | No cambiar |
| gRPC | No en v1 | Si se expone API para terceros o se extrae algún módulo |

---

## Recomendación final

- Un único backend en Go, un único módulo Go (`go.mod`).
- Monolito Modular — nunca microservicios prematuros.
- Arquitectura Hexagonal (Ports & Adapters).
- DDD: organizado por dominio, no por capa técnica.
- CQRS ligero: repos de dominio para escritura, queries directas para lectura/reportería.
- Eventos de dominio en memoria para comunicación inter-módulo.
- PostgreSQL con schemas — uno por módulo.
- Multi-tenancy por contexto Go, no por parámetro.
- Motor contable centralizado que recibe eventos del resto de módulos.
- `cofacture` como librería externa (propio repo Go) — el módulo `electronic/` la consume.
- `cmd/server/main.go` como único punto de wire-up, explícito y legible.
