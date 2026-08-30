# Auditoría pre-publicación de `cofacture`

**Fecha:** 2026-08-30
**Alcance:** todo `cofacture/` (librería DIAN de facturación electrónica), de cara a su publicación pública en GitHub y pkg.go.dev.
**Método:** 4 agentes en paralelo, cada uno con un ángulo distinto, más verificación manual de los hallazgos críticos por parte de Claude en la sesión principal. Auditoría de solo lectura — ningún cambio de código se hizo durante esta fase.

Ángulos cubiertos:
1. Código muerto, comentarios en español fuera de lugar, idiomatismo Go.
2. Manejo de centavos y redondeo (consistencia XML ↔ hash).
3. Verificación adversarial del `README.md` (cada afirmación comprobada contra el código real).
4. Arquitectura, filosofía de diseño y búsqueda sistemática de patrones "todo lo demás es X" (la clase de bug corregido el mismo día en `line_items.go`).

---

## 🔴 Crítico — arreglar antes de publicar

### `builder/notes.go:19-26` (`appendBillingReference`) hardcodea `schemeName="CUFE-SHA384"`

```go
// appendBillingReference adds cac:BillingReference — the reference to the invoice this note
// corrects. node is "InvoiceDocumentReference" for both Credit Note and Debit Note.
func appendBillingReference(parent *etree.Element, node string, br domain.BillingReference) {
	ref := parent.CreateElement("cac:BillingReference").CreateElement("cac:" + node)
	ref.CreateElement("cbc:ID").SetText(br.Prefix + br.Number)
	uuid := ref.CreateElement("cbc:UUID")
	uuid.CreateAttr("schemeName", "CUFE-SHA384")   // <-- literal fijo, sin importar el documento referenciado
	uuid.SetText(br.CUFE)
	ref.CreateElement("cbc:IssueDate").SetText(br.IssueDate)
}
```

Esta función la usan tanto `BuildCreditNote` (`builder/credit_note.go:62`) como `BuildDebitNote` (`builder/debit_note.go:61`), **sin distinguir** si el documento referenciado es una Factura normal (CUFE-SHA384, correcto) o un Documento Equivalente Electrónico tipo POS (CUDE-SHA384, para las Notas de Ajuste 93/94).

El propio paquete `cude` documenta explícitamente que el Documento Equivalente Electrónico usa CUDE-SHA384, no CUFE-SHA384 (`cude/cude.go:8-17`).

**Verificado directamente en el golden file** `builder/testdata/pos_credit_adjustment_golden.xml`:
- Línea 31: el propio documento (la Nota de Ajuste al DE-POS) tiene `<cbc:UUID schemeID="2" schemeName="CUDE-SHA384"/>` — correcto.
- Línea 45: su `BillingReference` al DE-POS original sale como `<cbc:UUID schemeName="CUFE-SHA384">db07502c...` — **incorrecto**, debería ser `CUDE-SHA384` porque referencia el mismo tipo de documento.

Mismo problema en `pos_debit_adjustment_golden.xml`.

**Por qué es grave:** es el mismo patrón exacto que el bug corregido hoy en `line_items.go` — una suposición implícita ("todo lo que se referencia en un BillingReference es una Factura con CUFE") que era cierta cuando solo existían Notas de Crédito/Débito 91/92, y se rompió silenciosamente al agregar las Notas de Ajuste al Documento Equivalente Electrónico (93/94). Sin ningún error de compilación ni de build.

**Contraste correcto:** `appendNABillingReference` (`builder/adjustment_note.go:94-101`) también hardcodea un literal (`"CUDS-SHA384"`), pero ahí sí es válido porque `AdjustmentNote` (tipo 95, ajuste al Documento Soporte) siempre referencia un Support Document — no hay otra familia que la use.

**Solución recomendada:** agregar un campo `HashType` a `domain.BillingReference` (`domain/notes.go:5-10`) siguiendo el mismo patrón que ya usan `domain.EventDocumentReference` (`domain/event.go:19`) y `domain.ValidationResult` (`domain/attached_document.go:25`), y pasarlo a `appendBillingReference` en vez del literal fijo.

**Impacto real:** cualquier consumidor que construya una Nota de Crédito/Débito de tipo 93/94 (ajuste a un DE-POS) obtiene un `cbc:UUID/@schemeName="CUFE-SHA384"` estructuralmente incorrecto dentro de su `BillingReference`.

---

## 🟡 Huecos ya documentados con honestidad, pendientes de decisión de diseño

### 1. `payroll` no sigue la convención de centavos del resto de la librería

- Todo el resto de `cofacture` usa `int64` centavos + `domain.FormatCents` (`domain/format.go:8-18`) como única función compartida de formateo, usada tanto para el XML (`builder/format.go`, `line_items.go`, `monetary_total.go`, `tax.go`) como para el hash (`internal/dianhash.Seed`, `cuds.Compute`, `qr`). Esto garantiza estructuralmente que el monto mostrado en el XML y el monto hasheado nunca pueden divergir — siempre que el caller haya truncado (no redondeado) antes de convertir a centavos, responsabilidad que la librería no advierte explícitamente en ningún lado.
- `payroll` rompe esta garantía por completo:
  - Usa `float64` pesos, no `int64` centavos (`payroll/domain.go:77-80, 116, 138-144, 157-168`).
  - Formatea con `fmt.Sprintf("%.2f", v)` (`payroll/builder.go:223-224`, funciones `money()`/`pct()`), que **redondea** según semántica IEEE754/Go, no trunca como exige el anexo.
  - `payroll.Cune(...)` (`payroll/cune.go:16-20`) recibe los montos ya formateados como **strings** en vez de calcularlos desde la misma fuente de datos que el XML — no hay garantía estructural de que lo impreso en el XML coincida con lo hasheado en el CUNE.
- `domain.FormatCents` no tiene ningún test unitario directo (`domain/format_test.go` no existe), pese a ser la pieza más crítica de consistencia de toda la librería.

**Decisión pendiente:** ¿vale la pena refactorizar `payroll` a `int64` centavos + truncado (cambio disruptivo a los structs públicos de `payroll/domain.go`), o documentar el riesgo y dejarlo así por ahora?

### 2. Falta soporte para `PayableRoundingAmount`

El Anexo Técnico permite que todos los valores monetarios sean positivos **excepto** `PayableRoundingAmount`, que puede ser negativo (redondeo del total a pagar al múltiplo de 50/100 más cercano, común en POS). `domain.Totals` (`domain/types.go:~99-105`) no tiene ningún campo equivalente; no existe `AllowanceCharge`/`RoundingAmount`/`Discount` en todo el repo.

**Decisión pendiente:** ¿agregar el campo ahora (antes de la primera publicación) o dejarlo para una versión menor futura, documentado como hueco conocido?

### 3. Ningún builder valida consistencia línea-vs-totales del header

`builder/tax.go`, `line_items.go`, `monetary_total.go` son renderizadores puros de lo que el caller ya armó en `domain.Line[]`/`domain.Totals` — no hay ningún chequeo cruzado de que la suma de líneas coincida con los totales del header. Si el caller se equivoca sumando, el XML sale mal sin ningún aviso de la librería.

**Decisión pendiente:** ¿agregar una validación (error duro) antes de construir el XML, o mantenerlo como responsabilidad documentada del caller (filosofía actual de "no validar catálogos", pero esto no es un catálogo sino aritmética interna)?

### 4. `payroll/cune_test.go` — discrepancia sin resolver con el ejemplo oficial

`TestCune_AnexoTecnicoExample` (`payroll/cune_test.go:27-91`) nunca falla intencionalmente porque no logra reproducir el hash de ejemplo publicado en la Cartilla Técnica de Nómina Electrónica. A diferencia del caso análogo en Nota Débito (que sí se resolvió y se documentó como error de transcripción del propio anexo DIAN), aquí quedó como investigación abierta. Es transparente y está bien comunicado en el propio test, pero es objetivamente el punto más débil de correctitud verificable del repo. `TestCune` (el test principal) sí aserta duro contra un hash de regresión propio.

---

## 🟢 Pulido para "repo estrella" (no son bugs, pero un revisor externo de la comunidad Go los señalaría)

- **8 archivos de test con mensajes de aserción en español** sin traducir — patrón copiado y pegado, más varios sueltos en `securitycode`, `qr`, `soap`, `dian`, `signer`. Ningún comentario en español quedó en código de producción (ya se tradujo todo en una fase previa de esta sesión) — esto es exclusivamente texto de mensajes de test.
- **3 símbolos exportados sin ningún uso ni test dedicado**: `payroll.AdjustXMLFileName` (`payroll/filename.go:18`), `soap.GetStatus` (`soap/operations.go:105`), `soap.SendNominaSync` (`soap/operations.go:144`). Probablemente API pública legítima (operaciones reales del WSDL de DIAN), pero sin cobertura de test — a diferencia de, por ejemplo, `zip/filename.go`, que sí tiene su `filename_test.go`.
- **`payroll.Build(n Nomina, cune, softwareSC, codigoQR string)` rompe el patrón de forma de API** del resto de funciones `Build*`. La familia UBL (`BuildInvoice`, etc.) recibe esos valores ya inyectados como campos del struct de dominio (`inv.CUFE`, `inv.SoftwareSecurityCode`, `inv.QRURL`) y toma un solo argumento; `payroll.Build` los recibe como tres parámetros posicionales adicionales porque `payroll.Nomina` no tiene esos campos.
- **Firmas `Compute` inconsistentes entre familias**, sin documentar como grupo: `cufe.Compute`, `cude.Compute`, `cuds.Compute` siguen `(struct de dominio, secreto) string`; `event.Compute` (9 strings sueltos) y `payroll.Cune` (11 strings sueltos) no. Ambas asimetrías están técnicamente justificadas (`ResponseCode` y `ValTolNE` no viven en ningún struct de dominio existente), pero no hay ninguna nota en el README o en el doc de paquete que lo explique.
- **Inconsistencia de prefijo de error** en `signer/certificate.go`: líneas 19, 39 y 58 devuelven errores crudos de librerías externas (`x509.ParseCertificate`, `pkcs12.Decode`, `x509.ParsePKCS1PrivateKey`) sin el prefijo `"signer: "` que sí llevan las demás líneas del mismo archivo (17, 26, 45, 54).
- **Los paquetes `xml` (`cofacture/xml`) y `zip` (`cofacture/zip`) sombrean `encoding/xml` y `archive/zip`** de la librería estándar. El propio código interno y el README ya aliasean (`ubl`, `cfzip`) para mitigarlo, pero herramientas como `goreportcard` y revisores de Go suelen marcar nombres de paquete genéricos que sombrean la stdlib como nit de idiomatismo en librerías públicas.
- **Duplicación**: `qr.AdjustmentNoteContent` y `qr.SupportDocumentContent` (`qr/qr.go:39-82` y `93-136`) son casi idénticas línea por línea (mismo bucle de búsqueda de IVA, mismo fallback, mismo `ambLabel`).
- **Asimetría cosmética**: `domain.CreditNote.CreditNoteTypeCode` (`domain/notes.go:32`) es un campo dedicado que en la práctica siempre coincide con `DocumentTypeCode`; `AdjustmentNote` no tiene ese campo y reutiliza `an.DocumentTypeCode` directamente (`builder/adjustment_note.go:48`). Severidad cosmética/baja.
- **`payroll.Nomina` vive en el paquete `payroll`, no en `domain/`** como todas las demás familias de documento. Justificado en el comentario de paquete (`payroll/domain.go:1-11`: no comparte helpers de `builder/`, no es UBL 2.1), pero rompe la regla implícita "los structs de documento viven en `domain/`" y el README no lo aclara explícitamente hoy.
- **No hay sentinel errors exportados** en ningún paquete (`errors.Is`/`errors.As` no aparece en código de producción) — decisión de minimalismo válida, pero significa que un caller no puede distinguir programáticamente, por ejemplo, "falta ReceiverPerson" de cualquier otro error de `BuildAcuseRecibo` sin hacer string matching sobre el mensaje.
- **No hay CI** (`.github/workflows` vacío). Fuera del alcance de esta auditoría de código, pero es lo primero que un visitante de GitHub busca junto al badge de Go Report Card que ya está en el README.

### Patrones revisados y descartados (dirección segura — no son el bug de tipo "todo lo demás es X")

| Ubicación | Patrón | Veredicto |
|---|---|---|
| `builder/line_items.go:21,67` (`documentTypeCodesUsingMandante`) | Conjunto cerrado positivo de los 4 códigos raros (91/92/93/94) que sí usan Mandante | Seguro — es la corrección de hoy |
| `builder/line_items.go:44` (`documentTypeCode != "92"`) | Excluye solo el caso raro conocido; todo lo demás (incl. futuros) recibe `FreeOfChargeIndicator` | Seguro, verificado contra goldens |
| `builder/monetary_total.go:26` (`documentTypeCode == "01"`) | Positivo, solo agrega `PrepaidAmount` para Factura | Seguro |
| `builder/extensions.go:44` (`"01" \|\| "05"`) | Positivo, `sts:InvoiceControl` solo para Factura/Documento Soporte | Seguro, verificado contra golden POS |
| `internal/dianhash/dianhash.go:16-23` (`"01","04","03"`) | Exigido tal cual por la fórmula CUFE/CUDE del anexo, no es un default implícito | Correcto por especificación |
| `qr/qr.go:42-55,96-109` | Fallback sobre impuestos (busca IVA, si no hay usa el primero), no sobre tipo de documento | Seguro, con fallback documentado |
| `builder/party.go:34` (`EntityTypeCode == "2"`) | Catálogo DIAN binario cerrado por naturaleza (natural/jurídica) | Bajo riesgo |
| `builder/payment_means.go:14` (`pm.Code == "2"`) | Positivo, coincide con el comentario de dominio | Seguro |
| `payroll/builder.go:54` (`"103" \|\| "104"`) | Conjunto cerrado positivo de los 2 casos de ajuste conocidos | Aceptable |

### Consistencia de la promesa "no se validan catálogos"

No se encontró ninguna violación. Los únicos `return nil, error` de la librería son validaciones **estructurales** (campo requerido ausente), nunca de **catálogo** (rechazar un valor por no coincidir con uno "esperado"): `payroll/builder.go:56,166`, `builder/event.go:30`, `builder/extensions.go:17,21,26,29`. La filosofía documentada en `domain/types.go:1-8` se sostiene en todo el código revisado.

### `go.work` / estructura del módulo

- `go build ./...`, `go test ./...` y `go mod tidy -diff` corren limpios en todo `cofacture/`.
- No hay ciclos de import; única dependencia interna es `internal/dianhash`, usada solo por `cufe` y `cude`.
- `go.work`/`go.work.sum` viven fuera del repo `cofacture` (en `project-ubl/`), correctamente, sin filtrar esa estructura al repo público.

---

## 📄 README — correcciones necesarias

1. **El Quick Start de Documento Equivalente Electrónico referencia un campo inexistente.** El texto actual dice que hay que fijar `CustomizationID: "10"` en `domain.Invoice` — ese campo **no existe**; el campo real es `OperationTypeCode` (`CustomizationID` es solo el nombre del elemento XML que `builder/invoice_builder.go:38` termina escribiendo). El párrafo también omite que hace falta cambiar `HashType` de `"CUFE-SHA384"` a `"CUDE-SHA384"` (que sí hace `builder/pos_test.go:23`).
2. **La columna "Confirmado contra DIAN real" no distingue evidencia fuerte de débil.** `integration-tests/send_sync_test.go`, `send_testset_test.go`, `credit_note_test.go` y `support_document_test.go` solo hacen `t.Logf` del resultado sin `t.Fatalf`/`t.Errorf` sobre `result.IsValid` — quedarían en verde incluso ante un rechazo real de DIAN. `adjustment_note_test.go` y `nomina_test.go` sí aseveran duro. Vale la pena que la tabla lo refleje (por ejemplo, distinguiendo "confirmado con aserción dura" de "confirmado, verificado manualmente en el log").

---

## Resumen de prioridad

1. **Crítico, arreglar antes de publicar:** bug de `appendBillingReference` en `builder/notes.go` (schemeName hardcodeado).
2. **Decisión de diseño antes de publicar (o documentar explícitamente como hueco conocido en el README):** arquitectura de centavos de `payroll`, `PayableRoundingAmount` ausente, falta de validación línea-vs-totales, discrepancia sin resolver del CUNE contra el ejemplo oficial.
3. **Pulido recomendado para "repo estrella":** traducir mensajes de aserción de los 8 archivos de test en español; agregar tests a los 3 símbolos sin cobertura; prefijo `"signer: "` faltante; documentar en README las asimetrías de forma de API (`payroll.Build`, `Compute`); considerar CI.
4. **Sin acción necesaria:** todos los demás patrones de comparación de código de tipo de documento revisados usan la dirección segura; la promesa de "no validar catálogos" se sostiene en todo el código.
