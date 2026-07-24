# Arquitectura General del ERP — Plan de Evolución

> Basado en el análisis del chat con ChatGPT (ver referencia completa en
> `docs/reference/chat-general-architecture.md`) y validado contra el estado real del proyecto
> a julio 2026.

---

## Veredicto

La arquitectura propuesta es viable y bien fundamentada. La separación actual
(`apidian` / `cofacture` / `frontend`) es el punto de partida correcto.
El siguiente bloque natural es `accounting-core/` — sin tocar nada de lo que ya funciona.

---

## Mapa del ecosistema objetivo

```
project-ubl/
├── go.work
│
├── apidian/            ← dominio fiscal electrónico DIAN (FE/NC/ND/DS/NA) — ya existe
├── cofacture/          ← librería de infraestructura DIAN (UBL + firma + SOAP) — ya existe
├── accounting-core/    ← motor contable puro — SIGUIENTE FASE
├── inventory/          ← inventario y almacén — futuro cercano
├── payroll/            ← nómina electrónica — futuro
├── hr/                 ← RRHH — futuro
└── frontend/           ← React SPA, evoluciona en módulos — ya existe
```

### Módulos proyectados (orden de prioridad)

| Módulo | Prioridad | Descripción |
|---|---|---|
| `accounting-core` | **Siguiente** | Motor contable: PUC, journal ledger, partida doble, reportes |
| `inventory` | **Futuro cercano** | Productos, bodegas, movimientos de stock, costo de ventas |
| `payroll` | Futuro | Liquidación de nómina, nómina electrónica DIAN |
| `hr` | Futuro | Empleados, contratos, cargos |
| `purchasing` | Futuro | Órdenes de compra (el DS ya cubre el lado fiscal) |
| `crm` | Futuro | Cotizaciones, órdenes de venta → generan FE |
| `fixed-assets` | Futuro | Activos fijos, depreciación → asientos automáticos |
| `treasury` | Futuro | Conciliación bancaria → conecta con core-bank |
| `logistics` | Futuro lejano | Despachos, rutas, flota |

---

## Comunicación entre servicios

### Por qué NATS y no HTTP siempre

Con HTTP sincrónico, los servicios están acoplados en tiempo: si `accounting-core` está caído
cuando `apidian` confirma una FE, el asiento nunca se crea y la contabilidad queda incompleta
(la FE ya fue a la DIAN — no se puede revertir).

Con un bus de eventos, `apidian` publica `"invoice.confirmed"` y olvida. Si `accounting-core`
estaba caído, levanta después y lee el evento de la cola. Los servicios no necesitan estar
vivos al mismo tiempo.

### Por qué NATS específicamente

- **Escrito en Go** (`github.com/nats-io/nats-server`) — compila a un solo binario sin
  dependencias externas, igual que `apidian`.
- **Sin Docker obligatorio** — se sube el binario al VPS y se corre como cualquier proceso Go.
  Inicio en milisegundos, ~10-15 MB de RAM en idle.
- **Sin costo de licencia** — open source. Nube administrada tiene tier gratuito.
- **Cliente Go nativo** (`github.com/nats-io/nats.go`) — un `go get` y está disponible en
  cualquier módulo del ecosistema.
- **Configuración mínima** — `./nats-server` sin ningún archivo de config es suficiente para
  desarrollo. Puerto 4222 por defecto.
- Mucho más simple que RabbitMQ o Kafka para este caso de uso.

### Estrategia de transición

**Fase 1-2 (MVP):** HTTP directo entre servicios. Suficiente mientras hay pocos módulos.

**Fase 3+ (cuando haya 3+ módulos):** agregar NATS. Cada módulo publica eventos; cada
adaptador contable los consume. El binario se despliega en el VPS igual que los demás.

```
Módulo publica →  nats-server (VPS, binario Go)  → consumidor procesa
                       puerto 4222
```

---

## Principio fundamental

> El `accounting-core` no sabe que existen facturas DIAN, empleados, inventarios ni bancos.
> Solo recibe instrucciones contables válidas y mantiene la verdad financiera de la empresa.
>
> Igual que el core bancario no sabe si una transferencia viene de una app móvil o de una
> sucursal — recibe una orden válida y mueve el ledger.

---

## 1. accounting-core — estructura interna

```
accounting-core/
├── cmd/
│   └── api/                    ← punto de entrada HTTP
└── internal/
    ├── domain/
    │   ├── account/            ← PUC
    │   ├── journal/            ← journal_entries + journal_lines
    │   ├── period/             ← periodos contables
    │   └── currency/           ← monedas
    ├── application/
    │   ├── posting/            ← motor de partida doble
    │   ├── reports/            ← libro diario, mayor, estados financieros
    │   └── closing/            ← cierres de periodo
    ├── infrastructure/
    │   └── postgres/           ← acceso a BD
    └── interfaces/
        └── http/               ← handlers REST
```

### Catálogos que viven en el core (y solo en el core)

| Catálogo | Estado |
|---|---|
| PUC (`accounts`) | CSV ya disponible en `docs/reference/accounts.csv` — importar como seed |
| Naturaleza contable | derivable del `category` del PUC (ver tabla abajo) |
| Monedas | tabla simple, iniciar solo con COP |
| Periodos contables | control de apertura/cierre de meses |
| Tipos de asiento | MANUAL / AUTOMATIC / ADJUSTMENT / CLOSING / OPENING |

### Naturaleza contable por categoría PUC

| Categoría | Naturaleza | Regla |
|---|---|---|
| Activo, Gastos, Costos | Débito | aumenta con débito, disminuye con crédito |
| Pasivo, Patrimonio, Ingresos | Crédito | aumenta con crédito, disminuye con débito |

### Tablas centrales del motor

```sql
-- Cabecera del asiento
journal_entries (
  id, company_id, date, description,
  status,     -- DRAFT | POSTED | VOID
  source,     -- "apidian", "inventory", "payroll", "manual", etc.
  entry_type, -- MANUAL | AUTOMATIC | ADJUSTMENT | CLOSING | OPENING
  created_at
)

-- Partida doble — el verdadero ledger
journal_lines (
  id, journal_id, account_id,
  debit, credit,      -- exactamente uno debe ser > 0, el otro 0
  cost_center,        -- opcional, el core guarda el código pero no lo administra
  created_at
)
```

**Invariante del motor:** `SUM(debit) = SUM(credit)` por cada `journal_id`. Si no se cumple, el asiento no se guarda.

### API pública del core

```
POST   /api/v1/journals               ← crear asiento (valida PUC, doble partida, periodo)
GET    /api/v1/journals/:id           ← consultar asiento
GET    /api/v1/accounts               ← listar PUC
GET    /api/v1/accounts/:id/ledger    ← libro mayor de una cuenta
GET    /api/v1/reports/trial-balance  ← balance de comprobación
GET    /api/v1/reports/income-statement
GET    /api/v1/reports/balance-sheet
GET    /api/v1/reports/general-ledger
```

Todos los reportes son **queries calculadas** sobre `journal_lines` — no se guardan balances.

---

## 2. inventory — estructura y relación contable

El módulo de inventario cierra el ciclo compra-venta: sin él el sistema no sabe cuántas
unidades hay ni qué costo tienen, y la contabilidad queda sin el asiento de costo de ventas.

```
inventory/
├── cmd/api/
└── internal/
    ├── domain/
    │   ├── product/        ← productos con costo unitario
    │   ├── warehouse/      ← bodegas / ubicaciones
    │   └── movement/       ← entradas, salidas, traslados
    ├── application/
    │   ├── stock/          ← control de existencias
    │   └── costing/        ← PEPS, promedio ponderado
    ├── infrastructure/postgres/
    └── integrations/
        └── accounting/     ← adaptador → accounting-core
```

### Eventos que genera inventory hacia accounting-core

| Evento | Asiento generado |
|---|---|
| Venta (salida por FE) | `610505 Costo de ventas` Débito / `143505 Mercancía` Crédito |
| Compra (entrada por DS) | `143505 Mercancía` Débito / `220505 Proveedores` Crédito |
| Ajuste de inventario | `143505 Mercancía` Débito o Crédito según dirección |
| Traslado entre bodegas | Asiento interno (no afecta P&G) |

### Relación con apidian

- Al confirmar una **FE** → `apidian` notifica a `inventory` (salida de stock) e `inventory`
  notifica a `accounting-core` (asiento de costo).
- Al confirmar un **DS** → `apidian` notifica a `inventory` (entrada de stock) e `inventory`
  notifica a `accounting-core` (asiento de compra).

---

## 3. Adaptador contable en apidian

No se toca el dominio de documentos. Se agrega una capa de integración aislada:

```
apidian/internal/integrations/accounting/
    client.go       ← llama al accounting-core via HTTP (o publica evento NATS en Fase 3)
    mapper.go       ← traduce Document → JournalEntry
    dto.go          ← structs del request/response
```

### Flujo cuando se confirma una FE

```
Usuario confirma FE
        ↓
InvoiceService (apidian)
        ↓
Guarda documento, actualiza estado DIAN
        ↓
accounting.Mapper.FromInvoice(doc) → JournalEntryRequest
        ↓
accounting.Client.PostJournal(entry)          ← HTTP en Fase 1-2
        ↓                                     ← NATS en Fase 3+
accounting-core API
        ↓
Valida PUC + partida doble + periodo
        ↓
Guarda journal_entry + journal_lines
```

### Posting rules de ejemplo para FE

| Cuenta | Código | Movimiento |
|---|---|---|
| Clientes nacionales | 130505 | Débito (total con IVA) |
| Ventas — Comercio | 413505 | Crédito (subtotal) |
| IVA por pagar | 240805 | Crédito (IVA generado) |

Para DS (Documento Soporte):

| Cuenta | Código | Movimiento |
|---|---|---|
| Proveedores nacionales | 220505 | Crédito (total) |
| Gasto o costo correspondiente | según naturaleza | Débito |
| IVA descontable | 135530 | Débito (si aplica) |

Las reglas se implementan primero hardcodeadas en el mapper y se hacen configurables en Fase 2.

---

## 4. Lo que NO se toca ni reorganiza

- `cofacture/` — librería de infraestructura DIAN. Se mantiene exactamente como está.
  Cuando payroll y otros módulos necesiten UBL/firma/SOAP, consumirán cofacture.
- `apidian/` — dominio de facturación. Solo se agrega `internal/integrations/accounting/`.
- `frontend/` — evoluciona progresivamente; los módulos contables se agregan bajo `src/modules/accounting/`.
- No se crea un `shared/` genérico. Lo realmente transversal (UUIDs, errores, logging) vive en cofacture o se duplica con criterio.

---

## 5. Riesgos identificados y decisiones

### Consistencia transaccional (Fase 1-2 con HTTP)

Si `apidian` confirma una FE pero `accounting-core` está caído, el asiento no se crea.
La FE ya fue a la DIAN — no se puede revertir.
**Decisión para MVP:** loggear el error y continuar. En Fase 3, NATS resuelve esto
estructuralmente: el evento queda en cola hasta que el core levante.

### Base de datos

**Decisión para MVP:** PostgreSQL compartido con schema separado (`accounting.*`, `inventory.*`
vs `public.*`). Migrar a BD independiente por servicio cuando haya razón operacional real.

### Posting rules

**Decisión para MVP:** reglas hardcodeadas en `mapper.go` de cada módulo adaptador.
Las reglas configurables (tabla `posting_rules` en el core) van en Fase 2.

---

## 6. Orden de construcción

### Fase 1 — Núcleo contable (accounting-core)
- [ ] Crear `accounting-core/` con su `go.mod`, agregar al `go.work`
- [ ] Schema SQL: `accounts`, `accounting_periods`, `journal_entries`, `journal_lines`
- [ ] Seed del PUC desde `docs/reference/accounts.csv`
- [ ] Motor de validación de partida doble
- [ ] API REST: `POST /journals`, `GET /accounts`, `GET /reports/trial-balance`

### Fase 2 — Adaptadores (apidian → accounting-core)
- [ ] `apidian/internal/integrations/accounting/`
- [ ] Mapper para FE y DS (posting rules básicas hardcodeadas)
- [ ] Llamada HTTP al core al confirmar documento

### Fase 3 — Reportes completos + UI
- [ ] Libro diario, libro mayor, estado de resultados, balance general
- [ ] UI en frontend: módulo `src/modules/accounting/`

### Fase 4 — Inventario
- [ ] Crear `inventory/` con su `go.mod`, agregar al `go.work`
- [ ] Schema SQL: `products`, `warehouses`, `stock_movements`
- [ ] Adaptador contable: asientos de costo al confirmar FE/DS
- [ ] UI en frontend: módulo `src/modules/inventory/`

### Fase 5 — Bus de eventos (NATS)
- [ ] Desplegar binario `nats-server` en VPS (puerto 4222)
- [ ] Migrar adaptadores de HTTP directo a publicación de eventos NATS
- [ ] Consumidores en `accounting-core` e `inventory` suscritos a los eventos

### Fase 6 — Módulos futuros
- [ ] `payroll/` + nómina electrónica DIAN
- [ ] `hr/` (empleados, contratos)
- [ ] `purchasing/` (órdenes de compra)
- [ ] `crm/` (cotizaciones → FE)
- [ ] `fixed-assets/` (depreciación → asientos automáticos)

---

## 7. Relación entre módulos (visión final)

```
                              ERP

        |           |            |           |
     Apidian    Inventory     Payroll      CRM
        |           |            |           |
  Accounting  Accounting    Accounting  Accounting
   Adapter     Adapter       Adapter     Adapter
        |           |            |           |
        -------------------------------------------
                         |
                   nats-server          ← binario Go en VPS
                   (pub/sub)
                         |
                  accounting-core
                  (Journal Ledger)
                         |
                  Estados financieros


         CoFacture (infraestructura DIAN)
         consumida por Apidian y Payroll
```

---

*Documento actualizado: 2026-07-24. Actualizar cuando cambien decisiones de arquitectura.*
