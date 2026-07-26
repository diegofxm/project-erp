# Arquitectura General del ERP

> Última actualización: **julio 2026**.
> Este documento refleja el estado real del proyecto, no un plan futuro.
> Las secciones marcadas ✅ están en producción o listas para producción.
> Las marcadas 🔲 son trabajo pendiente con diseño ya definido.
> Las marcadas 🔵 son futuras sin diseño cerrado aún.

---

## Estado actual del ecosistema

```
project-ubl/
├── go.work                   ← workspace que une todos los módulos Go
│
├── cofacture/          ✅    ← librería DIAN: UBL, firma XML, SOAP, CUFE/CUDE
├── apidian/            ✅    ← facturación electrónica DIAN (FE/NC/ND/DS/NA)
├── accounting/         🔲    ← motor contable — ~60 % completo (ver § 3)
├── payroll/            🔵    ← nómina colombiana
├── inventory/          🔵    ← inventarios y costeo
├── purchasing/         🔵    ← compras y proveedores
├── hr/                 🔵    ← recursos humanos
└── frontend/           ✅    ← React/TypeScript SPA
```

---

## Módulos existentes

### `cofacture/` ✅ — completo

Librería de infraestructura DIAN. Construye y firma documentos UBL 2.1 para Colombia:
facturas, notas crédito/débito, documentos soporte. Sin lógica de negocio propia — es la
capa técnica que todos los módulos que necesiten hablar con la DIAN consumen.
Cuando `payroll` emita nómina electrónica, también usará `cofacture`.

### `apidian/` ✅ — ~95 % completo

Módulo de facturación electrónica DIAN. Maneja el ciclo de vida completo de documentos
electrónicos: creación, firma, envío a la DIAN, polling de resultado, PDF, correo al
adquiriente. Pendiente menor: mejoras de UX y edge cases de la DIAN.

**Integración contable**: `apidian/internal/integrations/accounting/` — adaptador directo al
`accounting.Core` (mismo proceso, vía `go.work`). Al confirmar una FE o DS llama
`core.Journals.Post()` con las líneas correctas.

### `frontend/` ✅ — facturación completa, contabilidad pendiente

SPA en React/TypeScript. El módulo de facturación (FE, NC, DS, clientes, productos,
configuración de empresa) está completo. Los módulos de contabilidad, nómina e inventario
se agregarán como rutas nuevas sin tocar lo existente.

---

## `accounting/` — motor contable

### Principio de diseño

`accounting` es una **librería Go pura**, no un servicio HTTP. No tiene `cmd/api/`.
Expone un `Core` que otros módulos importan directamente:

```go
core := accounting.New(pool)   // un pool PostgreSQL compartido
core.Journals.Post(ctx, req)   // registrar un asiento
core.Reports.IncomeStatement(ctx, ...)
```

Esto elimina latencia de red en el camino crítico (confirmar una FE) y simplifica el
despliegue: un solo binario de `apidian` ya incluye el motor contable.
Cuando haya 3+ módulos y el bus de eventos (NATS) entre en juego, `accounting` puede
exponerse opcionalmente como microservicio — sin cambiar su código interno.

### Estructura real de paquetes

```
accounting/
├── accounting.go          ← Core{}, New(), Migrate(), Seed()
├── accounts/              ← PUC: Account, Nature(), GetPostable()
├── journals/              ← Motor: Post, Void, CloseYear, OpenYear,
│                             comprobantes consecutivos, GetBySourceDocument,
│                             constantes SourceFE/DS/NC/NOM/INV…
│                             tipo Book: BOTH / PCGA / NIIF
├── periods/               ← Períodos mensuales: GetOrCreate, Close, CloseYear
├── reports/               ← 8 reportes calculados (sin balances guardados)
├── banking/               ← Conciliación bancaria
├── withholdings/          ← Retefuente, Reteiva, Reteica + UVT 2020-2025
├── assets/                ← PPE: depreciación línea recta, baja con ganancia/pérdida
├── iva/                   ← F300: generado/descontable, ciclo DRAFT→PAID
├── cartera/               ← Aging FIFO, 6 cubetas colombianas, provisiones Art.145 ET
├── budget/                ← Presupuesto anual vs. real (BvR)
└── database/
    ├── migrations/        ← 000001–000015 embebidas en el binario
    └── seed/              ← PUC (2502 cuentas) + retenciones + UVT en CSV
```

### Migraciones y tablas (schema `accounting.*`)

| # | Tablas | Contenido |
|---|---|---|
| 001 | `accounts`, `accounting_periods` | PUC y períodos |
| 002 | `journal_entries`, `journal_lines` | Motor de asientos con `third_party_nit` |
| 003 | `bank_accounts`, `bank_statement_lines` | Conciliación bancaria |
| 004–006 | índices auxiliares | Auxiliar por tercero, Medios Magnéticos |
| 007 | `withholding_concepts`, `uvt_values` | Catálogo de retenciones |
| 008 | renombre `tercero_nit → third_party_nit` | Naming en inglés |
| 009 | `voucher_types`, `voucher_counters` | Comprobantes consecutivos |
| 010 | `fixed_assets`, `depreciation_runs`, `depreciation_entries` | PPE |
| 011 | `iva_declarations` | Formulario 300 |
| 012 | `reconciliation_marks` | Conciliación de cartera |
| 013 | columnas `source_document_id / _type` | Trazabilidad a documento fuente |
| 014 | `budgets`, `budget_lines` | Presupuesto vs. real |
| 015 | columna `book` en `journal_entries` | Doble libro PCGA/NIIF |

### Reportes disponibles

| Reporte | Filtro de libro |
|---|---|
| Balance de comprobación | ✅ PCGA / NIIF |
| Libro mayor por cuenta | — |
| Estado de resultados | ✅ PCGA / NIIF |
| Balance general | ✅ PCGA / NIIF |
| Auxiliar por tercero (NIT) | — |
| Saldo por tercero | — |
| Medios Magnéticos (base Información Exógena) | — |
| Centro de costo | — |

### Invariante central

`SUM(debit) == SUM(credit)` exacto por asiento, en centavos `int64`.
Sin tolerancia. Sin `float64`. El motor rechaza si no cuadra.

---

## Estado real de `accounting/` — julio 2026

### Avance estimado: ~60 %

Lo implementado es correcto y profesional. Lo que falta es ortogonal — se agrega sin
romper nada de lo existente.

### ✅ Funcional hoy para

- Microempresas en régimen ordinario sin empleados formales ni inventario.
- Registro automático de asientos de FE y DS desde apidian.
- Reportes gerenciales: P&G, balance general, libro mayor, medios magnéticos base.
- Cierres y aperturas de año.
- Presupuesto vs. real.
- Doble libro PCGA/NIIF con filtro en reportes.

### 🔲 Pendiente para la PYME colombiana típica

| Área | Por qué es necesaria |
|---|---|
| **Nómina** | Parafiscales, UGPP, aportes seguridad social, cesantías, vacaciones, prima. Sin esto no se pueden registrar los asientos de nómina. |
| **Inventarios** | PEPS/promedio ponderado, kardex, Costo de Mercancía Vendida automático. Sin esto el costo de ventas queda sin registrar. |
| **Declaraciones de impuestos** | Solo existe F300 (IVA). Falta F210 (Renta jurídicas), F220 (RetFuente anual), F490 (ICA por municipio). |
| **Moneda extranjera** | Diferencial cambiario, cuentas en USD/EUR. Necesario para importadores/exportadores. |
| **Posting rules configurables** | Hoy las reglas de contabilización están hardcodeadas en el mapper de apidian. Deben vivir en BD para ser configurables por tipo de documento y categoría de producto/proveedor. |

---

## ¿Qué tan grande es el trabajo restante?

### El 40% que falta vs. apidian

`apidian` es complejo por razones *externas*: protocolo DIAN, UBL/XML, firmas digitales,
polling asíncrono. La mayor parte de su código existe para hablarle a un tercero de una
forma muy específica.

El 40% restante de `accounting` es complejo por razones *internas*: reglas de negocio
colombianas puras. No hay protocolo externo que dominar.

| Módulo pendiente | Esfuerzo relativo vs. apidian |
|---|---|
| Nómina completa (UGPP, parafiscales, liquidaciones, provisiones) | ~40 % |
| Inventarios (PEPS/promedio, kardex, CMV) | ~25 % |
| Declaraciones de impuestos (F210, F220, F490 ICA) | ~15 % |
| Posting rules configurables | ~10 % |
| Moneda extranjera + diferencial cambiario | ~10 % |

**En total: el 40% faltante equivale aproximadamente a construir un apidian completo**,
pero sin la parte más difícil de apidian (los protocolos externos de la DIAN).

### ¿Los demás módulos son pequeños una vez el core esté listo?

El patrón de integración es siempre el mismo y ya está definido:

```go
// Cualquier módulo, siempre igual:
core.Journals.Post(ctx, journals.PostRequest{
    SourceDocumentID:   docUUID,
    SourceDocumentType: journals.SourceNOM,  // o SourceINV, SourceOC...
    Lines:              lineasCalculadas,
})
```

Lo que varía es cuánta lógica de dominio tiene cada módulo *antes* de llegar a ese `Post`:

| Módulo | Lógica de dominio | Integración contable |
|---|---|---|
| **HR** (contratos, cargos, novedades) | Pequeño | Ninguna directa — alimenta nómina |
| **Tesorería** (flujo de caja, anticipos) | Pequeño-medio | Simple: 1-2 líneas por movimiento |
| **Compras** (OC, recepción, 3-way match) | Medio | Medio: CxP + retenciones |
| **Inventarios** (kardex, valoración, ajustes) | Medio-grande | Medio: CMV automático |
| **Nómina** | **Grande** | Complejo: 15-20 líneas por período |

La nómina es la excepción: tiene tanta lógica de dominio propia que es comparable en
tamaño a lo que ya lleva `accounting/` completo.

---

## Estructura objetivo para una empresa grande

Para una empresa grande (manufactura, 200+ empleados, múltiples sucursales, operaciones
en USD, vigilada por Supersociedades):

```
project-ubl/
│
├── cofacture/              ✅  ← UBL XML + firmas (completo)
│
├── apidian/                ✅  ← Facturación DIAN — FE, DS, NC, ND (~95 %)
│
├── accounting/             🔲  ← Motor contable core (~60 % listo)
│   ├── journals/               Partida doble, comprobantes, doble libro ✅
│   ├── accounts/               PUC 2502 cuentas ✅
│   ├── periods/                Períodos + cierre de año ✅
│   ├── reports/                8 reportes con filtro PCGA/NIIF ✅
│   ├── banking/                Conciliación bancaria ✅
│   ├── withholdings/           Retefuente/Reteiva/Reteica ✅
│   ├── assets/                 PPE + depreciación + baja ✅
│   ├── iva/                    F300 + ciclo de pago ✅
│   ├── cartera/                Aging FIFO + conciliación ✅
│   ├── budget/                 Presupuesto vs. real ✅
│   ├── forex/                  Moneda extranjera + diferencial cambiario 🔲
│   ├── consolidation/          Consolidación multi-empresa 🔵
│   └── tax/                    F210 Renta, F220 RetFuente, F490 ICA 🔲
│
├── payroll/                🔵  ← Nómina colombiana (módulo propio)
│   ├── concepts/               Devengados y deducciones configurables
│   ├── settlements/            Liquidación mensual
│   ├── provisions/             Cesantías, vacaciones, prima (provisión mensual)
│   ├── social/                 Parafiscales + aportes seguridad social
│   └── → accounting            Post de ~15-20 líneas por período
│
├── inventory/              🔵  ← Inventarios (módulo propio)
│   ├── items/                  Referencia de productos con costo
│   ├── movements/              Entradas, salidas, traslados, ajustes
│   ├── valuation/              PEPS o Promedio ponderado
│   └── → accounting            CMV automático al vender
│
├── purchasing/             🔵  ← Compras y proveedores (módulo propio)
│   ├── orders/                 Órdenes de compra
│   ├── receipts/               Recepción de mercancía
│   ├── matching/               3-way match (OC + recepción + factura)
│   └── → accounting            CxP + retenciones al registrar factura proveedor
│
├── hr/                     🔵  ← RRHH (alimenta payroll, no tiene contabilidad propia)
│   ├── employees/              Ficha personal, contratos, cargos
│   ├── attendance/             Novedades: incapacidades, vacaciones, horas extra
│   └── → payroll               Novedades del período
│
└── frontend/               🔲  ← UI web
    ├── facturación         ✅  Completo
    ├── contabilidad        🔲  Por construir
    ├── nómina              🔵  Por construir
    └── inventario          🔵  Por construir
```

---

## Flujo de integración entre módulos

El contrato entre cualquier módulo y el motor contable siempre es el mismo.
Lo que cambia es quién llama y con qué líneas:

```
apidian (FE/DS)   →  SourceFE / SourceDS   →  Clientes, Ventas, IVA, Retenciones
payroll           →  SourceNOM             →  Gastos laborales, Parafiscales, Provisiones
inventory         →  SourceINV             →  CMV, Inventario, Variaciones de costo
purchasing        →  SourceOC              →  Proveedores, Gastos, IVA descontable
assets            →  SourceAF              →  PPE, Depreciación acumulada
```

El campo `book` (BOTH / PCGA / NIIF) permite que cada módulo indique si el asiento aplica
a ambos libros o solo al local o al IFRS — crítico para la convergencia NIIF que exige la
Superintendencia de Sociedades.

---

## Comunicación entre módulos

### Fase actual (Fase 1-2): llamada directa

Los módulos comparten el mismo proceso Go vía `go.work`. `apidian` importa `accounting`
directamente y llama `core.Journals.Post()` en la misma transacción que confirma el documento.

**Ventaja**: sin latencia, sin coordinación de versiones, sin red.  
**Riesgo controlado**: si el `Post` falla, se loggea y la FE igual se confirma (decisión MVP
documentada — la FE ya fue a la DIAN, no se puede revertir).

### Fase 3+ (cuando haya 3+ módulos): NATS

```
Módulo publica "invoice.confirmed"
        ↓
nats-server (VPS, binario Go, puerto 4222)
        ↓
accounting consume el evento y registra el asiento
```

NATS resuelve el problema de consistencia: si `accounting` estaba caído cuando se confirmó
la FE, levanta después y lee el evento de la cola. La FE no queda sin asiento.

Solo se cambia `apidian/internal/integrations/accounting/client.go` — el mapper, el dominio
de documentos y el core contable no se tocan.

---

## Pendientes en `apidian/internal/integrations/accounting/`

Estado al julio 2026: los tres bugs críticos fueron corregidos. Pendiente de Fase 2:

| # | Pendiente | Estado |
|---|---|---|
| 6.1 | Bug float64→int64 en mapper (centavos directos) | ✅ corregido |
| 6.2 | ThirdPartyNIT en líneas de cliente/proveedor | ✅ corregido |
| 6.3 | SourceDocumentID/Type en PostRequest | ✅ corregido |
| 6.4 | Retenciones: calcular y contabilizar Retefuente/Reteiva cuando el doc las trae | 🔲 Fase 2 |
| 6.5 | VoucherType: asignar consecutivo "FE"/"DS" al asiento | 🔲 Fase 2 |
| 6.6 | Posting rules configurables en BD (hoy hardcodeadas en mapper) | 🔵 Fase 3 |

---

## Lo que no se toca ni reorganiza

- `cofacture/` — librería de infraestructura DIAN. Se consume, no se modifica.
- El dominio de `apidian/internal/documents/` — solo se agregan adaptadores en `integrations/`.
- No se crea un `shared/` genérico. Lo realmente transversal (UUIDs, errores) se duplica
  con criterio o vive en `cofacture`.

---

## Orden de construcción — próximos pasos

### Inmediato — completar `accounting/`
- [ ] `accounting/forex/` — moneda extranjera, diferencial cambiario (cuentas `4210`/`5306`)
- [ ] `accounting/tax/` — F210 Renta, F220 Retención fuente, F490 ICA por municipio
- [ ] Posting rules configurables en BD (tabla `posting_rules` en el core)

### Siguiente módulo — `payroll/`
La nómina es el módulo más complejo después del core contable. Incluye:
- Liquidación mensual (devengados, deducciones, neto a pagar)
- Parafiscales: SENA 2 %, ICBF 3 %, Caja de Compensación 4 %
- Aportes seguridad social: salud 12.5 % (8.5 empleador + 4 empleado), pensión 16 % (12 + 4)
- Provisiones mensuales: cesantías 8.33 %, vacaciones 4.17 %, prima 8.33 %
- Integración con UGPP
- Nómina electrónica DIAN (usa `cofacture`)
- Asiento contable: 15-20 líneas por período por empresa

### Luego — `inventory/`
- Kardex de entradas/salidas por referencia
- Valoración PEPS o Promedio ponderado
- CMV automático al confirmar una FE
- Ajustes de inventario con asiento contable

### Luego — `purchasing/` y `hr/`
Módulos más pequeños; `hr` alimenta a `payroll`, `purchasing` cierra el ciclo con el DS de `apidian`.

### Infraestructura — NATS (cuando haya 3+ módulos activos)
- Desplegar `nats-server` en VPS como binario Go
- Migrar `integrations/accounting/client.go` de llamada directa a publicación de evento
- Consumidores en `accounting`, `inventory`, `payroll` suscritos a los eventos relevantes

---

*Actualizado: julio 2026. Actualizar cuando cambien decisiones de arquitectura o avance el estado de un módulo.*
