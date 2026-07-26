# Arquitectura de Procesos Empresariales para un ERP (Tipo SAP / Siigo)

## Principio fundamental

> La contabilidad es la consecuencia de las operaciones del negocio.

``` text
Compra
   ↓
Recepción de mercancía
   ↓
Inventario
   ↓
Producción
   ↓
Costo
   ↓
Venta
   ↓
Cartera
   ↓
Tesorería
   ↓
Contabilidad
```

------------------------------------------------------------------------

# Orden recomendado de estudio

1.  Contabilidad

    -   Contabilidad Universitaria -- Gerardo Guajardo Cantú
    -   Contabilidad Financiera -- Jerry Weygandt
    -   Contabilidad General -- Horngren

2.  NIIF (IFRS para PYMES)

3.  Contabilidad de Costos

    -   Horngren
    -   Hansen & Mowen

4.  Inventarios

    -   Inventory Management Explained
    -   Production and Inventory Control (APICS)

5.  Producción

    -   Manufacturing Planning and Control (Jacobs)

6.  Centros de Costos

7.  Presupuestos

8.  Tesorería

9.  Auditoría

10. ERP

-   ERP Demystified

11. SAP Business One

-   FI
-   MM
-   PP
-   SD

12. Odoo

13. Microsoft Dynamics

14. Oracle NetSuite

------------------------------------------------------------------------

# Normativa colombiana

-   DIAN (Factura Electrónica, Documento Soporte, Nómina Electrónica,
    RADIAN)
-   Estatuto Tributario
-   PUC
-   NIIF para Colombia

------------------------------------------------------------------------

# Macroprocesos empresariales

1.  Strategy-to-Portfolio (S2P)
2.  Idea-to-Market (I2M)
3.  Source-to-Contract (S2C)
4.  Procure-to-Pay (P2P)
5.  Forecast-to-Plan (F2P)
6.  Plan-to-Produce (PTP)
7.  Warehouse-to-Deliver (W2D)
8.  Order-to-Cash (O2C)
9.  Service-to-Cash (S2C)
10. Project-to-Profit (P2P)
11. Asset-to-Retire (A2R)
12. Hire-to-Retire (H2R)
13. Time-to-Pay (T2P)
14. Expense-to-Reimburse (E2R)
15. Treasury-to-Cash (T2C)
16. Record-to-Report (R2R)
17. Tax-to-Compliance (T2C)
18. Risk-to-Control (R2C)

Todos estos procesos desembocan en **Record-to-Report**, donde se
generan los asientos y estados financieros.

------------------------------------------------------------------------

# Dominios del ERP

  Dominio            Prioridad
  ------------------ -----------
  Core / Seguridad   Alta
  Terceros           Alta
  Inventario         Alta
  Compras            Alta
  Ventas             Alta
  Tesorería          Alta
  Contabilidad       Crítica
  Impuestos          Crítica
  DIAN Electrónica   Crítica
  Producción         Media
  Activos Fijos      Media
  Nómina             Media
  Proyectos          Baja
  WMS Avanzado       Baja

------------------------------------------------------------------------

# Cinco procesos que deben quedar perfectos

1.  Procure-to-Pay
2.  Inventory-to-Cost
3.  Order-to-Cash
4.  Treasury-to-Cash
5.  Record-to-Report

------------------------------------------------------------------------

# Arquitectura recomendada

-   Ningún módulo debe escribir directamente en la contabilidad.
-   Todos los módulos generan eventos de negocio.
-   Un motor contable central interpreta reglas configurables y genera
    los comprobantes.

Ejemplo:

``` text
Venta realizada
      ↓
Evento: SALE_POSTED
      ↓
Motor Contable
      ├─ Débito: Clientes
      ├─ Crédito: Ingresos
      ├─ Crédito: IVA generado
      └─ Débito: Costo de ventas
                Crédito: Inventarios
```

Beneficios:

-   Sin lógica contable duplicada.
-   Cambios del PUC sin modificar código.
-   Soporte multiempresa.
-   Trazabilidad completa.
-   Fácil auditoría.

------------------------------------------------------------------------

# Módulos principales

-   Core
-   Terceros
-   Compras
-   Inventario
-   Producción
-   Ventas
-   Tesorería
-   Activos Fijos
-   Nómina
-   Contabilidad
-   Impuestos
-   Reportes

------------------------------------------------------------------------

# Recomendación final

Diseña primero los procesos de negocio y después las pantallas. Un ERP
robusto se construye sobre procesos bien definidos, eventos de negocio
claros y un motor contable desacoplado de los módulos operativos.
