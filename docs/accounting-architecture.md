# Arquitectura del Módulo Contable — `accounting`

> Última actualización: **julio 2026**.
> Módulo Go independiente en `accounting/` con su propio `go.mod` (`github.com/diegofxm/accounting`).
> Conecta a la misma base de datos que `apidian` pero usa el schema `accounting.*`
> y una tabla de migraciones separada (`accounting_schema_migrations`).

---

## 1. Principios de diseño

| Principio | Implementación |
|---|---|
| Partida doble obligatoria | `Service.Post` rechaza si `SUM(debit) ≠ SUM(credit)` (exacto, sin tolerancia) |
| Aritmética sin punto flotante | Todos los montos en `int64` centavos — nunca `float64` |
| Libro inmutable | Los asientos no se eliminan; se anulan (`StatusVoid`) |
| PUC colombiano real | 2 502 cuentas cargadas vía seed CSV; idempotente |
| Schema aislado | Todas las tablas en `accounting.*`; no colisiona con `public.*` de apidian |
| Migración embebida | `//go:embed database/migrations/*.sql` — el binario se auto-migra al arrancar |
| Sin FK a otros módulos | `source_document_id UUID` sin FK referencial — cross-module sin acoplamiento |
| Doble libro opcional | Campo `book` en asientos: `BOTH` (defecto), `PCGA`, `NIIF` |

---

## 2. Estructura de paquetes

```
accounting/
├── accounting.go              ← Core: punto de entrada, New(), Migrate(), Seed()
├── accounts/                  ← PUC: Account, Nature(), GetPostable(), List()
├── journals/                  ← Motor de asientos: Post, Void, CloseYear, OpenYear,
│                                 GetBySourceDocument, vouchers, source constants
├── periods/                   ← Períodos mensuales: GetOrCreate, Close, CloseYear
├── reports/                   ← Reportes calculados (ver § 4.3)
├── banking/                   ← Conciliación bancaria: BankAccount, StatementLine
├── withholdings/              ← Retenciones: Calculate, CalculateMany, UVT values
├── assets/                    ← Activos fijos PPE: Run/Depreciation, Dispose
├── iva/                       ← IVA declarado: GenerateForm300, CreatePaymentEntry
├── cartera/                   ← Cartera: AgeByThird (FIFO), Reconcile, ProvisionsEstimate
├── budget/                    ← Presupuesto vs. real: Create, SetLine, Approve, BvR
└── database/
    ├── migrations/            ← SQL embebido (000001–000015)
    └── seed/                  ← PUC + conceptos de retención + valores UVT en CSV
```

---

## 3. Tablas en el schema `accounting`

| Migración | Tabla(s) | Descripción |
|---|---|---|
| 000001 | `accounts`, `accounting_periods` | PUC y períodos |
| 000002 | `journal_entries`, `journal_lines` | Motor de asientos; `third_party_nit` en líneas |
| 000003 | `bank_accounts`, `bank_statement_lines` | Conciliación bancaria |
| 000004 | `medios_magneticos_*, auxiliary_by_third_*` | Índices de reportes |
| 000005 | — | Migración de tipos monetarios a `int64` centavos |
| 000006 | `journal_entries`: auxiliar por tercero | Reporte auxiliar |
| 000007 | `withholding_concepts`, `uvt_values` | Catálogo de retenciones y UVT |
| 000008 | `journal_lines`: renombre `tercero_nit` → `third_party_nit` | Naming en inglés |
| 000009 | `voucher_types`, `voucher_counters`; columnas en `journal_entries` | Comprobantes consecutivos |
| 000010 | `fixed_assets`, `depreciation_runs`, `depreciation_entries` | Activos fijos PPE |
| 000011 | `iva_declarations` | Declaraciones de IVA |
| 000012 | `reconciliation_marks` | Conciliación de cartera |
| 000013 | `journal_entries`, `fixed_assets`: `source_document_id / _type` | Trazabilidad |
| 000014 | `budgets`, `budget_lines` | Presupuesto anual |
| 000015 | `journal_entries`: columna `book` | Doble libro PCGA/NIIF |

---

## 4. Lo que está implementado ✅

### 4.1 Motor de asientos (journals/)

- **Partida doble estricta**: mínimo 2 líneas; exactamente uno de débito o crédito por línea; `SUM(debit) == SUM(credit)` sin tolerancia.
- **Tipos de asiento**: `MANUAL`, `AUTOMATIC`, `ADJUSTMENT`, `CLOSING`, `OPENING`.
- **Estados**: `POSTED` → `VOID`; nunca eliminación.
- **Tercero (`ThirdPartyNIT`)**: NIT del cliente/proveedor/empleado en cada línea; requerido por Información Exógena DIAN.
- **Centro de costo** por línea.
- **Comprobantes consecutivos**: tipos CE/CI/NC/NI/CJ/AP; formato `CE-2025-00001`; contador atómico por empresa+tipo+año (upsert PostgreSQL, seguro bajo concurrencia).
- **Doble libro**: campo `book` (`BOTH`/`PCGA`/`NIIF`) por asiento.
- **Trazabilidad**: `source_document_id UUID` + `source_document_type VARCHAR(30)` sin FK referencial; `GetBySourceDocument()` para auditoría inversa.
- **Constantes de origen**: `SourceFE`, `SourceDS`, `SourceNA`, `SourceNC`, `SourceND`, `SourceNOM`, `SourceINV`, `SourceOC`, `SourceLC`, `SourceAF`.
- **Cierre de año** (`CloseYear`): saldos P&G → cero; diferencia → 3605 Utilidad o 3610 Pérdida; cierra todos los períodos del año.
- **Apertura de año** (`OpenYear`): balance general al 31-dic → asiento 01-ene del año nuevo.

### 4.2 Retenciones (withholdings/)

- Catálogo de **16 conceptos** de Retefuente, Reteiva y Reteica con tarifa en puntos básicos y base mínima en UVT.
- **Valores UVT** 2020–2025 en centavos (seed idempotente).
- `Calculate(code, type, base, supplierType, year)`: aplica tarifa solo si base ≥ mínimo UVT; prioriza concepto exacto para NATURAL/JURIDICA sobre BOTH.
- `CalculateMany(items, year)`: lote de retenciones.

### 4.3 Reportes (reports/)

| Reporte | Método | Filtro de libro |
|---|---|---|
| Balance de comprobación | `TrialBalance(ctx, companyID, from, to, book?)` | ✅ variádico |
| Libro mayor | `GeneralLedger(ctx, companyID, accountCode, from, to)` | — |
| Estado de resultados | `IncomeStatement(ctx, companyID, from, to, book?)` | ✅ variádico |
| Balance general | `BalanceSheet(ctx, companyID, asOf, book?)` | ✅ variádico |
| Auxiliar por tercero | `AuxiliaryByThird(ctx, companyID, accountCode, nit, from, to)` | — |
| Saldo por tercero | `TerceroBalance(ctx, companyID, accountCode, from, to)` | — |
| Medios magnéticos | `MediosMagneticos(ctx, companyID, year)` | — |
| Centro de costo | `CostCenterBalance(ctx, companyID, from, to)` | — |

El parámetro `book ...string` es variádico — los llamadores actuales sin argumento funcionan igual (retrocompatible).

### 4.4 Activos fijos (assets/)

- Registro de PPE con cuentas de activo, gasto depreciación y depreciación acumulada.
- **Depreciación línea recta**: `RunDepreciation(companyID, date)` — calcula el mínimo entre la cuota mensual y el valor en libros restante; genera asiento automático con comprobante tipo `"DA"`.
- Protección contra doble-corrida: `UNIQUE` parcial en `depreciation_runs WHERE status='COMPLETED'`.
- **Baja de activo** (`Dispose`): maneja escritura total (proceeds=0), venta con ganancia (3605) y venta con pérdida (5290).
- Trazabilidad al DS/FE de compra que originó el activo (`SourceDocumentID / Type`).

### 4.5 IVA (iva/)

- Lectura de movimientos de cuentas `2408*` (IVA generado/descontable) y `1365*` (Reteiva a favor) excluendo asientos propios de pago de IVA.
- `GenerateForm300(companyID, from, to)`: estructura `F300` con saldo a pagar o a favor.
- Ciclo `DRAFT → FILED → PAID → CORRECTED` en `iva_declarations`.
- `CreatePaymentEntry`: asiento de pago IVA con `source = "iva_payment"` para no contar doble en el siguiente F300.

### 4.6 Cartera (cartera/)

- **Aging FIFO** (`AgeByThird`): créditos absorben los débitos más antiguos primero; saldo restante clasificado en 6 cubetas colombianas estándar (corriente, 1-30, 31-60, 61-90, 91-180, >180 días).
- **Estimación de provisiones** según tasas del Art. 145 ET: 0 % / 5 % / 10 % / 15 % / 25 % / 50 %.
- **Conciliación bidireccional** (`Reconcile`): marca las dos líneas como conciliadas en una transacción; si la segunda falla revierte la primera.

### 4.7 Conciliación bancaria (banking/)

- Registro de cuentas bancarias con su cuenta PUC asociada.
- Importación de extracto en lote.
- Cruce extracto ↔ asiento contable; informe de partidas sin cruzar y diferencia.

### 4.8 Presupuesto vs. real (budget/)

- **Encabezado** (`budgets`): por empresa, año y nombre; estados `DRAFT → APPROVED → CLOSED`.
- **Líneas mensuales** (`budget_lines`): columnas `jan`–`dec` en centavos; `UNIQUE(budget_id, account_id)` para upsert limpio.
- `SetLine`: rechaza modificación si el presupuesto ya está `APPROVED`.
- **`BvR(companyID, budgetID, fromMonth, toMonth)`**: merge en memoria de líneas presupuestadas y actuals de `journal_entries POSTED`; incluye cuentas con ejecución real sin línea presupuestada (varianza = 100 % no planificada).

### 4.9 Doble libro PCGA / NIIF (000015)

- Campo `book VARCHAR(10) NOT NULL DEFAULT 'BOTH'` en `journal_entries`.
- `CHECK (book IN ('PCGA', 'NIIF', 'BOTH'))` + índice parcial `WHERE book != 'BOTH'`.
- Constantes `BookBoth`, `BookPCGA`, `BookNIIF` en `journals/journal.go`.
- `PostRequest.Book` propagado por `Service.Post`; vacío normaliza a `BOTH`.
- `IncomeStatement`, `BalanceSheet`, `TrialBalance` aceptan `book ...string` variádico.

---

## 5. Evaluación de madurez — julio 2026

**Avance estimado: ~60 % para uso en producción con empresas colombianas.**

Lo implementado es correcto y profesional en todo lo que toca: aritmética de centavos, partida doble exacta, PUC real, migraciones idempotentes, separación de paquetes por dominio. Lo que falta es ortogonal — no rompe lo existente, se agrega encima.

### ✅ Funcional hoy para

- Microempresas en **régimen ordinario simple** sin empleados formales ni inventarios.
- **Registro automático** de asientos de FE y DS desde apidian.
- **Reportes gerenciales** básicos (P&G, balance general, libro mayor) con filtro de libro PCGA/NIIF.
- Cierres de año y aperturas contables.

### ❌ No funcional aún para la PYME colombiana típica

| Área faltante | Por qué es crítico |
|---|---|
| **Nómina** | Parafiscales (SENA 2 %, ICBF 3 %, Caja 4 %), aportes salud (8.5 %/4 %) y pensión (12 %/4 %), cesantías, vacaciones. Sin esto no se pueden registrar asientos de nómina. |
| **Inventarios** | Métodos PEPS/Promedio ponderado, kardex, costeo automático de ventas (Costo de Mercancía Vendida). Obligatorio para empresas comerciales e industriales. |
| **Declaraciones de impuestos completas** | Solo hay F300 (IVA). Falta F210 (Renta personas jurídicas), F220 (Retención en la fuente anual), F260 (Renta naturales), F490 (ICA por municipio). |
| **Moneda extranjera** | Sin diferencial cambiario, sin cuentas en USD/EUR. Indispensable para importadores/exportadores. |
| **Posting rules configurables** | Las reglas de contabilización están hardcodeadas en el mapper de apidian. Deben vivir en base de datos (por tipo de documento, categoría de producto/proveedor). |
| **Cuentas de orden** | El PUC las incluye (clase 8 y 9) pero el motor las trata igual que cualquier otra. Requieren tratamiento especial (no afectan el balance). |
| **Sucursales / establecimientos** | El ICA varía por municipio; sin soporte multi-establecimiento no se puede calcular correctamente. |

---

## 6. Integración `apidian/internal/integrations/accounting/`

El adaptador entre apidian y la librería contable está **completo para todos los tipos de documento** que maneja apidian. Estado actual: compila, tests pasan, wiring activo en `handleConfirmDocument`.

### 6.1–6.5 Completados ✅

| # | Ítem | Estado |
|---|---|---|
| 6.1 | Bug float64→int64 en mapper (centavos directos) | ✅ |
| 6.2 | ThirdPartyNIT en líneas de cliente/proveedor (130505 / 220505) | ✅ |
| 6.3 | SourceDocumentID/Type en PostRequest | ✅ |
| 6.4 | Retenciones DS: 220505 neto + CR 236505/236540/236560 por TypeCode DIAN | ✅ |
| 6.5 | VoucherType "FE"/"DS"/"NC"/"ND"/"NA" asignado en cada PostRequest | ✅ |

### Cobertura de documentos — julio 2026

Todos los documentos que confirma apidian generan asiento contable automático:

| Documento | DianDocumentTypeCode | Asiento | VoucherType |
|---|---|---|---|
| FE — Factura de venta | 01 | DR 130505(cliente) / CR 413505 + CR 240805(IVA) | "FE" |
| NC — Nota Crédito | 91 | DR 413505 + DR 240805 / CR 130505(cliente) | "NC" |
| ND — Nota Débito | 92 | DR 130505(cliente) / CR 413505 + CR 240805 | "ND" |
| DS — Documento Soporte | 05 | DR gasto + DR 135530(IVA) / CR 220505(neto) + CR 2365xx(retenciones) | "DS" |
| NA — Nota de Ajuste DS | 95 | Espejo exacto del DS (DR↔CR invertidos) | "NA" |

**Punto de entrada**: `handleConfirmDocument` llama `postAccountingEntry` después de confirmar.
`POST /api/v1/documents/{id}/confirm` acepta body opcional `{ "expense_account_code": "5135" }` para DS y NA (si se omite, el asiento no se registra y se emite un aviso en logs).

### 6.6 Pendiente — Fase 3

| # | Pendiente | Estado |
|---|---|---|
| 6.6 | Posting rules configurables en BD (hoy hardcodeadas en mapper: 130505, 413505, 220505…) | 🔵 Fase 3 |

---

## 7. Integración con los demás módulos del ERP

Cada módulo futuro genera asientos automáticos via `core.Journals.Post()` con el `SourceDocumentType` correspondiente:

```
apidian (FE/DS)  → SourceFE / SourceDS    → Ingresos, Cartera, IVA, Retenciones
payroll          → SourceNOM              → Gastos laborales, Parafiscales, Provisiones
inventory        → SourceINV              → CMV, Inventario, Variaciones de costo
purchasing       → SourceOC               → Proveedores, Gastos, IVA descontable
assets           → SourceAF               → PPE, Depreciación acumulada
```

El campo `book` distingue si el asiento es PCGA, NIIF o ambos — vital para la convergencia NIIF que exige la Superintendencia de Sociedades.

---

## 8. Referencias

- `accounting/` — código fuente del módulo contable
- `apidian/internal/integrations/accounting/` — adaptador apidian ↔ contabilidad
- Decreto 2649 de 1993 — principios de contabilidad colombianos (PCGA)
- Decreto 2420 de 2015 — adopción de NIIF en Colombia (Grupo 1, 2 y 3)
- Resolución DIAN anual — Información Exógena / Medios Magnéticos
- Art. 145 E.T. — provisión de cartera deducible
