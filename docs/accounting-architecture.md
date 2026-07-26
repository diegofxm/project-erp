# Arquitectura del Módulo Contable — `accounting`

> Estado auditado: **julio 2026**.
> Módulo Go independiente en `accounting/` con su propio `go.mod` (`github.com/diegofxm/accounting`).
> Se conecta a la misma base de datos que `apidian` pero usa el schema propio `accounting.*`
> y una tabla de migraciones separada (`accounting_schema_migrations`).

---

## 1. Principios de diseño

| Principio | Implementación actual |
|---|---|
| Partida doble obligatoria | `journals.Service.Post` rechaza si `SUM(debit) ≠ SUM(credit)` (tolerancia 1 centavo) |
| Naturaleza de cuentas | `Account.Nature()` — Activo/Gasto/Costo → Debit; Pasivo/Patrimonio/Ingreso → Credit |
| Libro inmutable | Los asientos no se eliminan; se anulan (`StatusVoid`) |
| PUC colombiano real | 2 502 cuentas cargadas via seed CSV |
| Schema aislado | Todas las tablas en `accounting.*`; no colisiona con `public.*` de apidian |
| Migración embebida | `//go:embed database/migrations/*.sql` — el binario se auto-migra al arrancar |

---

## 2. Estructura de paquetes

```
accounting/
├── accounting.go          ← Core: punto de entrada, Migrate(), Seed()
├── accounts/              ← PUC: Account, Nature(), GetPostable()
├── journals/              ← Motor de asientos: Post, Void, CloseYear, OpenYear
├── periods/               ← Períodos mensuales: GetOrCreate, Close, CloseYear
├── reports/               ← Reportes calculados: TrialBalance, Ledger, IncomeStatement, BalanceSheet, CostCenterBalance
├── banking/               ← Conciliación bancaria: BankAccount, StatementLine, Reconcile, GetReport
└── database/
    ├── migrations/        ← SQL embebido (000001–000004)
    └── seed/              ← PUC completo en CSV
```

---

## 3. Tablas en el schema `accounting`

| Tabla | Descripción |
|---|---|
| `accounts` | PUC colombiano — 2 502 cuentas con código, nombre, categoría, nivel, is_posting |
| `accounting_periods` | Períodos mensuales por empresa (OPEN / CLOSED) |
| `journal_entries` | Cabecera de asiento: fecha, descripción, tipo, estado, empresa |
| `journal_lines` | Líneas: cuenta, débito, crédito, centro de costo |
| `bank_accounts` | Cuentas bancarias vinculadas a su cuenta PUC (ej. 1110) |
| `bank_statement_lines` | Líneas del extracto bancario, marcables como conciliadas |

---

## 4. Lo que YA está implementado ✅

### Motor contable
- Registro de asientos con validación estricta de partida doble
- Naturaleza de las 6 clases de cuentas del PUC (Activo, Pasivo, Patrimonio, Ingresos, Gastos, Costos)
- Mínimo 2 líneas por asiento; exactamente uno de débito o crédito por línea
- Períodos contables mensuales con bloqueo de escritura al cerrar
- Tipos de asiento: Manual, Automático, Ajuste, Cierre, Apertura

### Cierre y apertura de año
- `CloseYear` — calcula saldos de P&G del año, genera asiento de cierre que lleva
  Ingresos/Gastos/Costos a cero y registra la diferencia en **3605 Utilidad del ejercicio**
  o **3610 Pérdida del ejercicio**; cierra todos los períodos del año automáticamente
- `OpenYear` — toma el balance general al 31-dic del año anterior y genera el asiento
  de apertura al 01-ene del año nuevo

### Reportes
- Balance de comprobación (`TrialBalance`) — suma débitos y créditos por cuenta en un rango
- Libro mayor (`GeneralLedger`) — movimientos de una cuenta con saldo acumulado
- Estado de resultados (`IncomeStatement`) — Ingresos − Gastos/Costos = Utilidad neta
- Balance general (`BalanceSheet`) — Activos = Pasivos + Patrimonio a fecha de corte
- Reporte por centro de costo (`CostCenterBalance`) — débito/crédito/saldo por CC

### Conciliación bancaria
- Registro de cuentas bancarias vinculadas a su cuenta PUC
- Importación de extractos en lote (`ImportStatementLines`)
- Cruce línea a línea extracto ↔ asiento (`Reconcile` / `Unreconcile`)
- Informe de conciliación: saldo extracto, partidas sin cruzar, saldo libros, diferencia

---

## 5. Lo que FALTA para uso profesional en Colombia 🔴🟡

### 🔴 Crítico — sin esto no funciona en producción

#### 5.1 Tercero (NIT) en cada línea contable
En Colombia todo movimiento contable debe estar asociado al NIT del tercero involucrado
(proveedor, cliente, empleado, banco). Es obligatorio para:
- **Medios Magnéticos / Información Exógena** (reporte anual a la DIAN por NIT)
- Conciliación de saldos con clientes y proveedores
- Libro auxiliar por tercero

**Lo que se necesita:** campo `tercero_nit VARCHAR(20)` en `journal_lines` +
`GetAuxiliaryByThird(ctx, companyID, accountCode, nit, from, to)` en Reports.

---

#### 5.2 Retenciones (Retefuente, Reteiva, Reteica)
Cualquier pago o abono a un proveedor en Colombia genera retención automática según:
- **Concepto de retención** (tabla de conceptos con base mínima y tarifa)
- **Tipo de proveedor** (persona natural vs. jurídica, declarante vs. no declarante)
- **Monto base** del pago

Sin esto es imposible registrar una compra o pago a proveedor correctamente.

Cuentas involucradas del PUC:
- `2365` Retefuente por pagar
- `2367` Reteiva por pagar
- `2368` Reteica por pagar
- `1355` Anticipo de impuestos y contribuciones (retención que nos practican)

**Lo que se necesita:** tabla `withholding_concepts` (concepto, base mínima, tarifa,
tipo de tercero) + lógica `CalculateWithholdings(paymentAmount, conceptCode, vendorType)`.

---

#### 5.3 Tipo de dato monetario — `int64` en lugar de `float64`
El módulo usa `float64` para débitos y créditos. En contabilidad esto es un bug latente:
`0.1 + 0.2 = 0.30000000000000004` en punto flotante. El parche actual
(`math.Abs(d-c) > 0.01`) enmascara el problema sin resolverlo.

**Lo que se necesita:** migrar todos los campos monetarios a `int64` (centavos),
igual que ya lo hace `apidian` con `PayableCents int64`.

---

#### 5.4 Número de comprobante consecutivo
La regulación colombiana exige numeración consecutiva de comprobantes de diario
(Decreto 2649). Actualmente los asientos solo tienen UUID.

**Lo que se necesita:** secuencia por empresa+año (`comprobante_seq`) +
campo `consecutive int NOT NULL` en `journal_entries`.

---

### 🟡 Importante — necesario para uso real

#### 5.5 Activos fijos y depreciación
No hay registro de activos fijos ni cálculo automático de depreciación mensual.
Cuentas: `15xx` (propiedades y equipo), `159505` (depreciación acumulada).

#### 5.6 IVA — descontable vs. generado
No hay distinción entre IVA descontable (compras) e IVA generado (ventas), ni apoyo
para generar la declaración bimestral (Formulario 300).
Cuentas: `2408` IVA por pagar, `2409` IVA régimen común.

#### 5.7 Cartera — aging por tercero
No hay reporte de cartera vencida (aging) por cliente ni conciliación de saldos
por proveedor.

#### 5.8 Trazabilidad al documento fuente
El campo `source string` es solo texto. Debería poder apuntar al UUID del documento
que originó el asiento (factura, DS, nómina, orden de compra).

#### 5.9 Presupuesto (Budget)
No hay módulo de presupuesto ni comparación real vs. presupuestado por cuenta/CC.

#### 5.10 NIIF vs. PCGA local
Colombia migró a NIIF desde 2015–2016. Muchas empresas mantienen dos libros
(local PCGA + NIIF). El módulo no distingue el marco normativo del asiento.

---

## 6. Orden de implementación propuesto

| # | Feature | Prioridad | Impacto |
|---|---|---|---|
| 1 | `int64` centavos (migración de tipos) | 🔴 | Base para todo lo demás |
| 2 | Tercero NIT en líneas + libro auxiliar | 🔴 | Medios magnéticos |
| 3 | Conceptos de retención + cálculo automático | 🔴 | Compras/pagos reales |
| 4 | Número de comprobante consecutivo | 🔴 | Cumplimiento Dec. 2649 |
| 5 | Activos fijos + depreciación automática | 🟡 | Patrimonio correcto |
| 6 | IVA descontable/generado + Form. 300 | 🟡 | Declaraciones tributarias |
| 7 | Cartera aging + conciliación por tercero | 🟡 | Gestión de cobro |
| 8 | Trazabilidad a documento fuente (UUID) | 🟡 | Auditoría completa |
| 9 | Presupuesto vs. real | 🟢 | Gestión gerencial |
| 10 | NIIF vs. PCGA (doble libro) | 🟢 | Empresas grandes |

---

## 7. Integración con otros módulos del ERP

Una vez que el motor contable esté completo, cada módulo genera asientos automáticos:

```
Facturación (apidian)  →  Ingresos + Cartera + IVA generado
Compras / DS           →  Gastos + Proveedores + Retefuente + IVA descontable
Nómina (payroll)       →  Gastos laborales + Provisiones + Aportes + Retefuente
Inventario             →  Costo de mercancía + Inventario + Variaciones
Tesorería              →  Bancos + Caja + Anticipos
```

El campo `source` de `journal_entries` + `entry_type = AUTOMATIC` es el puente
entre cada módulo y el motor contable.

---

## 8. Referencias

- `accounting/` — código fuente del módulo
- `docs/contabilidad-colombia/` — material de investigación base (transcripciones de videos)
- Decreto 2649 de 1993 — principios de contabilidad generalmente aceptados en Colombia
- Decreto 2420 de 2015 — adopción de NIIF en Colombia
- Resolución DIAN — Información Exógena (Medios Magnéticos, actualización anual)
