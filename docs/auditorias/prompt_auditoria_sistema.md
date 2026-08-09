# PROMPT — Auditoría Técnica Multi-Agente del Sistema ERP/CRM + Cofacture

Copia y pega este prompt completo en el chat de Claude Code (VSCode) dentro de la raíz de tu proyecto.

---

## CONTEXTO DEL PROYECTO

Estoy desarrollando un sistema empresarial compuesto por **tres componentes** que ya existen (parcial o totalmente) en este entorno de VSCode:

1. **`cofacture/`** — Motor de facturación electrónica DIAN (Colombia). Responsabilidades: generación de XML UBL, firma digital (XAdES), envío al servicio SOAP de la DIAN, manejo de respuestas, CUFE, notas crédito/débito, eventos, y todo el ciclo de vida del documento electrónico.
2. **`erp/`** — Backend tipo ERP (similar a Odoo / SIESA) para medianas y grandes empresas. Está construido en **Go**, con **arquitectura hexagonal (ports & adapters)**, y **PostgreSQL basado en esquemas** (schema-per-tenant o schema-per-módulo, según se implementó). Este backend **consume a `cofacture`** como motor de facturación.
3. **`frontend/`** — Interfaz que consume la API de `erp`.

Relación de dependencia: `frontend → erp → cofacture`

El objetivo final es dejar el sistema **listo para producción**, con módulo ERP completo o en roadmap claro, y con la facturación electrónica 100% conforme a los requisitos legales de la DIAN.

---

## OBJETIVO DE ESTA TAREA

Actúa como un **equipo de auditores expertos, cada uno especializado en un dominio**, y realiza una auditoría exhaustiva y honesta de todo el código, arquitectura, configuración e infraestructura presentes en este repositorio/entorno. No asumas nada: **verifica leyendo el código real**, no generes suposiciones ni "relleno".

Quiero que actúes secuencialmente como los siguientes agentes/roles (uno a la vez, cada uno entregando su propia sección del informe):

### 1. Agente de Arquitectura (Hexagonal / DDD)
- Verifica si la arquitectura hexagonal está correctamente implementada en `erp/` (dominio, puertos, adaptadores, casos de uso, infraestructura separados de la lógica de negocio).
- Detecta violaciones de las fronteras de capas (ej. lógica de negocio en handlers HTTP, acceso directo a DB desde el dominio, etc.).
- Evalúa el desacoplamiento real entre `erp` y `cofacture` (¿se usa una interfaz/puerto o un acoplamiento directo?).
- Evalúa organización de carpetas, nomenclatura, y consistencia entre módulos.

### 2. Agente de Base de Datos (PostgreSQL con esquemas)
- Revisa el diseño de esquemas (schemas) de PostgreSQL: ¿multi-tenant por schema? ¿separación correcta por módulo/dominio?
- Verifica migraciones (herramienta usada, versionado, reversibilidad).
- Revisa índices, llaves foráneas, constraints, normalización, y posibles cuellos de botella.
- Verifica manejo de transacciones distribuidas o consistencia entre `erp` y `cofacture` si comparten o no la misma base de datos.

### 3. Agente Backend Go (erp)
- Revisa calidad de código Go: manejo de errores, uso de contextos, concurrencia (goroutines/channels), linters (golangci-lint), cobertura de tests.
- Revisa manejo de configuración/secretos (variables de entorno, vaults, .env expuestos por error).
- Revisa diseño de la API (REST/gRPC), versionado, documentación (OpenAPI/Swagger).
- Evalúa manejo de autenticación/autorización (JWT, RBAC, multi-tenant por usuario/empresa).
- Revisa logging, trazabilidad y manejo de errores centralizado.

### 4. Agente Cofacture (Motor de Facturación Electrónica DIAN)
- Verifica cumplimiento técnico con la Resolución DIAN vigente de facturación electrónica (estructura XML UBL 2.1, anexo técnico).
- Revisa el proceso completo: generación XML → firma digital XAdES → envío SOAP a DIAN → manejo de respuesta (aceptado/rechazado) → almacenamiento del CUFE/CUDE.
- Verifica manejo de contingencia (cuando el servicio DIAN no responde).
- Verifica soporte de notas crédito, notas débito, documento soporte, nómina electrónica (si aplica) y eventos RADIAN.
- Revisa manejo seguro de certificados digitales (.p12/.pfx) y llaves privadas.
- Revisa idempotencia y reintentos ante fallos de red con la DIAN.
- Señala si falta ambiente de habilitación/pruebas DIAN vs producción.

### 5. Agente Frontend
- Revisa consumo de la API del `erp`, manejo de errores, estados de carga.
- Revisa arquitectura del frontend (framework usado, gestión de estado, componentes reutilizables).
- Revisa manejo de sesión/autenticación en el cliente.
- Evalúa accesibilidad, responsividad y consistencia de UI.
- Identifica módulos de CRM/ERP que existen en backend pero no tienen vista en frontend (o viceversa).

### 6. Agente de Seguridad
- Busca vulnerabilidades comunes: inyección SQL, XSS, CSRF, secretos hardcodeados, dependencias desactualizadas con CVEs.
- Revisa manejo de datos sensibles (información fiscal, certificados DIAN, datos de clientes) y cumplimiento con protección de datos (Habeas Data / Ley 1581 Colombia).
- Revisa configuración de CORS, rate limiting, y protección contra fuerza bruta.
- Revisa gestión de secretos en CI/CD y en el repositorio (busca archivos `.env` commiteados, llaves privadas, tokens).

### 7. Agente de Testing y Calidad
- Evalúa cobertura de pruebas unitarias, de integración y end-to-end en los tres componentes.
- Revisa si existen pruebas específicas del flujo crítico: generación y envío de factura electrónica.
- Evalúa la existencia de un pipeline de CI que ejecute estas pruebas automáticamente.

### 8. Agente DevOps / Preparación para Producción
- Revisa Dockerfiles, docker-compose, o manifiestos de Kubernetes si existen.
- Evalúa estrategia de despliegue, variables de entorno por ambiente (dev/staging/prod).
- Revisa backups de base de datos, estrategia de recuperación ante desastres.
- Revisa monitoreo, alertas, health checks y logging centralizado (observabilidad).
- Evalúa escalabilidad (¿el sistema soporta múltiples empresas/tenants con carga concurrente?).

### 9. Agente de Producto / Roadmap ERP
- Compara lo que existe hoy contra un ERP/CRM de referencia tipo Odoo o SIESA (módulos típicos: ventas, inventario, compras, contabilidad, RRHH, CRM comercial, reportes/BI).
- Identifica qué módulos ya existen, cuáles están a medias, y cuáles faltan completamente.
- Ya que mencionas que a futuro quieres agregar un módulo de **CRM comercial** (gestión de leads, oportunidades, pipeline de ventas, seguimiento a clientes): evalúa cómo encajaría dentro de la arquitectura hexagonal actual y qué tan fácil/difícil sería integrarlo sin romper lo existente.

---

## FORMATO DE ENTREGA (MUY IMPORTANTE)

Para cada agente, entrega su análisis con esta estructura exacta:

```
## [Nombre del Agente]

### ✅ Lo que está bien implementado
- ...

### ⚠️ Lo que existe pero tiene problemas / riesgos
- ... (explica el riesgo y la severidad: alta / media / baja)

### ❌ Lo que falta por completo
- ...

### 🔧 Recomendaciones concretas y accionables
- ...
```

Al final, agrega una sección de **Resumen Ejecutivo** con:
- Estado general del proyecto (%) por componente (cofacture, erp, frontend).
- Los 5 riesgos más críticos que bloquean producción.
- Una checklist priorizada (orden sugerido) para llegar a producción.
- Estimado cualitativo de esfuerzo (bajo/medio/alto) por cada pendiente.

---

## DÓNDE GUARDAR EL RESULTADO (INSTRUCCIÓN OBLIGATORIA)

1. Crea (si no existe) la carpeta `docs/auditorias/`.
2. Dentro de ella, crea una subcarpeta con la fecha de hoy en formato `AAAA-MM-DD`, ejemplo: `docs/auditorias/2026-08-09/`.
3. Dentro de esa carpeta, guarda **un archivo Markdown por cada agente**, nombrado así:
   - `01-arquitectura.md`
   - `02-base-de-datos.md`
   - `03-backend-go.md`
   - `04-cofacture-dian.md`
   - `05-frontend.md`
   - `06-seguridad.md`
   - `07-testing-calidad.md`
   - `08-devops-produccion.md`
   - `09-roadmap-erp.md`
4. Además, crea un archivo `00-resumen-ejecutivo.md` con el resumen ejecutivo y checklist priorizada general.
5. Si en el futuro se ejecuta otra auditoría, debe crearse una **nueva carpeta con la nueva fecha**, sin sobrescribir auditorías anteriores, de modo que quede un historial cronológico en `docs/auditorias/`.

---

## REGLAS PARA EL AGENTE QUE EJECUTA ESTA AUDITORÍA

- No inventes hallazgos: si no puedes verificar algo porque el archivo/módulo no existe o no es accesible, dilo explícitamente como "❌ No implementado / no encontrado".
- Lee el código real (archivos, configuración, migraciones, tests) antes de emitir juicios.
- Sé crítico y honesto, incluso si algo "casi funciona" pero tiene deuda técnica importante.
- Prioriza los hallazgos relacionados con el cumplimiento legal DIAN, seguridad de datos financieros y de certificados digitales, ya que son bloqueantes legales para producción.
- Al finalizar todos los agentes, pregúntame si quiero que generes un plan de acción con tareas específicas (issues) para ir cerrando cada pendiente.

---

Empieza por el **Agente de Arquitectura** y ve avanzando uno por uno hasta completar los nueve, guardando cada archivo en la carpeta correspondiente a medida que terminas cada sección.