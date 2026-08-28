# cofacture

[![Go Reference](https://pkg.go.dev/badge/github.com/diegofxm/cofacture.svg)](https://pkg.go.dev/github.com/diegofxm/cofacture)
[![Go Report Card](https://goreportcard.com/badge/github.com/diegofxm/cofacture)](https://goreportcard.com/report/github.com/diegofxm/cofacture)
[![Go Version](https://img.shields.io/github/go-mod/go-version/diegofxm/cofacture)](go.mod)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

**A Go toolkit for Colombian DIAN electronic invoicing** — UBL 2.1 document generation, XAdES-EPES signing, CUFE/CUDE/CUDS/CUNE computation, and a SOAP client for DIAN's `WcfDianCustomerServices` web services.

`cofacture` builds, signs, packages and submits the electronic documents required by Colombia's tax authority (DIAN — *Dirección de Impuestos y Aduanas Nacionales*): Electronic Sales Invoices, Credit Notes, Debit Notes, Support Documents (and their Adjustment Notes), Attached Documents, Individual Electronic Payroll, RADIAN acknowledgment events (Acuse de Recibo, Reclamo, Recibo del Bien, Aceptación Expresa/Tácita), and Documento Equivalente Electrónico (POS tickets and its 9 other sub-types, plus their Adjustment Notes). It is a **library, not a platform**: it has no database, no HTTP server, and no opinion about how you store your data — you feed it plain Go structs and it hands back signed XML, ready to send.

> Built and verified against DIAN's real certification (*habilitación*) environment — not just inferred from the technical annexes. Several structural decisions in this codebase (WS-Security signing quirks, XAdES reference ordering, Support Document party structure, etc.) were confirmed byte-for-byte against real, DIAN-accepted documents from multiple technology providers. See the package-level doc comments for the specific evidence behind each one.

**Contents:** [Why this library](#why-this-library) · [What it supports today](#what-it-supports-today) · [Package map](#package-map) · [Installation](#installation) · [Quick start](#quick-start) · [DIAN Technical Annex coverage](#dian-technical-annex-coverage) · [Security notes](#security-notes) · [Testing](#testing) · [Documentation](#documentation) · [Disclaimer](#disclaimer) · [License](#license)

---

## Why this library

Most "facturación electrónica" integrations for Colombia end up being a pile of XML string templates glued to a SOAP call. `cofacture` instead gives you:

- **A typed domain model** (`domain` package) instead of raw XML — build a Go struct, not a template.
- **Composable, single-purpose packages** — you wire the pipeline (build → hash → sign → zip → send) yourself, which means you can stop, resume, retry, persist, or swap any step without fighting a framework.
- **No hidden catalog validation.** The library does not silently "fix" or reject your tax/unit/city codes — it trusts the caller and puts the value where the annex says it goes. Catalog validation belongs in your orchestration layer, not baked into the wire format.
- **Real-world-verified encoding decisions**, documented inline, for the parts of the annexes that are ambiguous, inconsistently illustrated, or don't match what the production server actually accepts.

---

## What it supports today

| Document (DIAN name) | Type code | Status |
|---|---|---|
| Electronic Sales Invoice (*Factura Electrónica de Venta*) | `01` | ✅ Build, sign, CUFE, QR, send |
| Credit Note (*Nota Crédito*) | — | ✅ Build, sign, CUDE, QR, send |
| Debit Note (*Nota Débito*) | — | ✅ Build, sign, CUDE, QR, send |
| Support Document (*Documento Soporte*) | `05` | ✅ Build, sign, CUDS, QR, send |
| Adjustment Note to the Support Document (*Nota de Ajuste*) | `95` | ✅ Build, sign, CUDS, QR, send |
| Attached Document (*AttachedDocument* container) | — | ✅ Build, sign (Invoice/Credit Note/Debit Note) — not sent to DIAN, see note below |
| Individual Electronic Payroll (*Nómina Individual*) | `102` | ✅ Build, sign, CUNE, QR, send |
| Payroll Adjustment (*Nómina de Ajuste*) | `103`/`104` | ✅ Build, sign, CUNE, QR, send |
| RADIAN events — Acuse de Recibo (`030`), Reclamo (`031`), Recibo del Bien (`032`), Aceptación Expresa (`033`), Aceptación Tácita (`034`) | — | ✅ Build, sign, CUDE, send — see the note below |
| Documento Equivalente Electrónico — POS ticket (`20`); cinema, public shows, gaming, ground transport, tolls, financial extracts, air tickets, exchange/commodities settlements, and public utility bills (`25`/`27`/`30`/`35`/`40`/`45`/`50`/`55`/`60`) | `20`/`25`/`27`/`30`/`35`/`40`/`45`/`50`/`55`/`60` | ⚠️ Build, CUDE — see the note below, POS is the only one exercised so far |
| Documento Equivalente Electrónico Adjustment Note (débito `93` / crédito `94`) | `93`/`94` | ⚠️ Build, CUDE — same caveat as the row above |

An AttachedDocument is never submitted to DIAN by SOAP — it's built *after* DIAN has already validated the document it wraps (it embeds that validation response) and is delivered directly to the acquirer instead, so there's no `Send*` operation that takes one.

**RADIAN events are less battle-tested than the rest of this library.** Every other document type here was verified byte-for-byte against real, DIAN-accepted documents; the event builders (`builder.BuildAcuseRecibo` and friends, package `event`) were instead assembled directly from the Technical Annex's field tables (section 6.5.4/6.5.5) and a worked CUDE example from section 11.5 that the `event` package's test reproduces exactly — but no real event has been submitted to DIAN's certification environment yet to confirm the XML structure end-to-end. Treat it as "should be correct per the annex" until that happens; see the doc comments in `builder/event.go` for the specifics. One event (Aceptación Tácita) was built, CUDE-computed, and XAdES-signed against a real certificate and a real, DIAN-accepted invoice — but deliberately not submitted, because that event is a sworn statement about elapsed time and a prior event that were both simulated for the test; see `integration-tests/tacit_acceptance_test.go`.

**Documento Equivalente Electrónico (POS and friends) reuses this library's existing Invoice/CreditNote/DebitNote machinery almost entirely** — confirmed against the Documento Equivalente Electrónico Technical Annex V1.0 (Resolution 000165/2023): same UBL subset, same `WcfDianCustomerServices` SOAP operations, and a CUDE formula that turned out to be field-for-field identical to `cude.Compute`'s (see that function's doc comment) — swap the numbering range's technical key for the software PIN, same as a Credit/Debit Note. `builder.BuildInvoice` needed one real fix along the way: a closed allowlist in `line_items.go` assumed any `DocumentTypeCode` outside `{01, 05, 95}` meant a Credit/Debit Note's "Mandante" item shape, which silently mis-rendered the new type codes (`20` and friends) until `builder/pos_test.go` caught it structurally. Only the POS ticket (`20`) and its adjustment notes (`93`/`94`) have a dedicated golden test so far — the other 9 Documento Equivalente Electrónico types (`25`/`27`/`30`/`35`/`40`/`45`/`50`/`55`/`60`) should work the same way (same annex, same mechanism) but haven't been exercised individually, and none of this family has been submitted to DIAN's certification environment yet.

DIAN web service (`WcfDianCustomerServices`) operations implemented in the `soap` package:

`SendBillSync` · `SendBillAsync` · `SendBillAttachmentAsync` · `SendTestSetAsync` · `GetStatus` · `GetStatusZip` · `GetNumberingRange` · `SendNominaSync` · `SendNominaSyncTestSet` · `SendEventUpdateStatus` · `GetStatusEvent` · `GetAcquirer`

---

## Package map

The pipeline is intentionally unbundled — each package does exactly one job, and nothing calls the next step for you.

| Package | Responsibility |
|---|---|
| [`domain`](./domain) | Plain Go structs for every document (`Invoice`, `CreditNote`, `DebitNote`, `AdjustmentNote`, `AttachedDocument`, `Event`, `Reclamo`, `Party`, `Tax`, `Line`, ...). No validation, no persistence. |
| [`builder`](./builder) | Assembles the UBL 2.1 + DIAN-extension XML tree from a domain model (`etree.Document`) — every document type plus the RADIAN `ApplicationResponse` events. Does not sign, hash, or send anything. |
| [`cufe`](./cufe) / [`cude`](./cude) / [`cuds`](./cuds) | Compute the unique document codes (CUFE for invoices, CUDE for Credit/Debit Notes **and Documento Equivalente Electrónico** — same formula, see `cude.Compute`'s doc comment —, CUDS for Support Documents) per Technical Annex 1.9 and the Documento Equivalente Electrónico Technical Annex V1.0. |
| [`event`](./event) | Computes the CUDE for RADIAN events (a different formula from `cude`'s) and holds their `ResponseCode` catalog and the Aceptación Tácita note template. |
| [`payroll`](./payroll) | Builds `NominaIndividual` XML (not UBL) and computes the CUNE, per the Payroll Technical Annex. |
| [`securitycode`](./securitycode) | Computes `sts:SoftwareSecurityCode`. |
| [`qr`](./qr) | Builds the QR URL/content required in the graphic representation of every document type. |
| [`signer`](./signer) | XAdES-EPES signing (C14N 1.0, RSA-SHA256) plus certificate/key loading (PEM and PKCS#12). |
| [`zip`](./zip) | Packages signed XML into the ZIP format and file-naming convention DIAN's receiving service requires. |
| [`soap`](./soap) | SOAP 1.2 + WS-Security client for `WcfDianCustomerServices` (habilitación and producción). |
| [`dian`](./dian) | Interprets DIAN's validation responses into a structured `Result` (rejections vs. notices, embedded `ApplicationResponse`, etc.). |
| [`xml`](./xml) | Shared UBL/DIAN namespace constants. |

---

## Installation

```bash
go get github.com/diegofxm/cofacture
```

Requires Go 1.26+ (see `go.mod`).

---

## Quick start

This is the full pipeline for a single invoice, end to end. Error handling is abbreviated for readability — check every error in real code.

```go
package main

import (
	"log"
	"os"
	"time"

	"github.com/diegofxm/cofacture/builder"
	"github.com/diegofxm/cofacture/cufe"
	"github.com/diegofxm/cofacture/dian"
	"github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/qr"
	"github.com/diegofxm/cofacture/securitycode"
	"github.com/diegofxm/cofacture/signer"
	"github.com/diegofxm/cofacture/soap"
	cfzip "github.com/diegofxm/cofacture/zip"
)

func main() {
	// 1. Load your DIAN-issued certificate (.p12) once, reuse it across documents.
	certBytes, _ := os.ReadFile("software.p12")
	cert, key, err := signer.LoadPKCS12(certBytes, "your-p12-password")
	if err != nil {
		log.Fatal(err)
	}

	// 2. Build the domain model. Everything here comes from your own system
	//    (ERP, database, order service, etc.) — cofacture never reaches out to fetch it.
	//    supplierParty, customerParty, headerTaxes, totals, lines, numberingRange and
	//    softwareProvider below are placeholders for values you supply; numberingRange in
	//    particular comes from a prior call to soap.Client.GetNumberingRange.
	inv := domain.Invoice{
		ProfileID:         "DIAN 2.1",
		EnvironmentCode:   "2", // "1" production, "2" certification (habilitación)
		OperationTypeCode: "10",
		DocumentTypeCode:  "01",
		HashType:          "CUFE-SHA384",
		Prefix:            "SETP",
		Number:            "990000001",
		IssueDate:         "2026-08-27",
		IssueTime:         "10:15:00-05:00",
		CurrencyCode:      "COP",
		Supplier:          supplierParty,
		Customer:          customerParty,
		HeaderTaxes:       headerTaxes,
		Totals:            totals,
		Lines:             lines,
		NumberingRange:    numberingRange,
		SoftwareProvider:  softwareProvider,
	}

	// 3. Compute the identifiers DIAN requires before the document can be signed.
	//    technicalKey comes from GetNumberingRange; softwareID/pin are the credentials
	//    DIAN assigned when you activated your software.
	inv.CUFE = cufe.Compute(inv, technicalKey)
	inv.SoftwareSecurityCode = securitycode.Compute(softwareID, pin, inv.Prefix+inv.Number)
	inv.QRURL = qr.URL(inv.EnvironmentCode, inv.CUFE)

	// 4. Build the UBL XML tree.
	doc, err := builder.BuildInvoice(inv)
	if err != nil {
		log.Fatal(err)
	}

	// 5. Sign it (XAdES-EPES).
	placeholder, err := builder.SignaturePlaceholder(doc)
	if err != nil {
		log.Fatal(err)
	}
	if err := signer.New(cert, key).Sign(doc.Root(), placeholder, "supplier", time.Now().In(domain.Bogota)); err != nil {
		log.Fatal(err)
	}
	xmlBytes, err := doc.WriteToBytes()
	if err != nil {
		log.Fatal(err)
	}

	// 6. Name and package the file the way DIAN's receiving service expects.
	fileName := cfzip.DocumentFileName(cfzip.KindInvoice, "900123456", cfzip.SoftwarePropioCode, 2026, 1)
	zipBytes, err := cfzip.Build([]cfzip.File{{Name: fileName, Content: xmlBytes}})
	if err != nil {
		log.Fatal(err)
	}

	// 7. Send it to DIAN and interpret the response.
	client := soap.New(soap.HabilitacionURL, cert, key)
	resp, err := client.SendBillSync(fileName, zipBytes)
	if err != nil {
		log.Fatal(err)
	}
	result, err := dian.Interpret(*resp)
	if err != nil {
		log.Fatal(err)
	}
	if !result.IsValid {
		log.Fatalf("DIAN rejected the invoice: %+v", result.Messages)
	}
	log.Println("Accepted:", result.StatusDescription)
}
```

Building a Support Document, Credit Note, Debit Note, Adjustment Note or Payroll document follows the same shape — swap `builder.BuildInvoice` for `builder.BuildSupportDocument` / `builder.BuildCreditNote` / `builder.BuildDebitNote` / `builder.BuildAdjustmentNote` / `payroll.Build`, and `cufe.Compute` for `cuds.Compute` / `cude.Compute` / `payroll.Cune` as appropriate.

RADIAN events (`builder.BuildAcuseRecibo` and its four siblings) follow a related but not identical shape: `event.Compute` instead of `cufe.Compute`, `Sender`/`Receiver` instead of `Supplier`/`Customer`, and a `DocumentReference` pointing at the document the event applies to instead of a `BillingReference`. One detail worth calling out because it's easy to get backwards: the event's QR is still built from the *referenced* document's CUFE (`qr.URL(ev.EnvironmentCode, ev.DocumentReference.CUFE)`), not from the event's own CUDE.

A Documento Equivalente Electrónico (e.g. a POS ticket) is `builder.BuildInvoice` and `cude.Compute` again — same `domain.Invoice`, same pipeline as the Quick start above — with four fields set differently: `ProfileID: "DIAN 2.1: Documento Equivalente POS"`, `CustomizationID: "10"`, `DocumentTypeCode: "20"` (see the Technical Annex Documento Equivalente Electrónico V1.0, section 16.3, for the other 9 sub-types' codes), and pass `softwarePIN` to `cude.Compute` where the Quick start passes `technicalKey` to `cufe.Compute`. Its Adjustment Notes are `builder.BuildCreditNote`/`BuildDebitNote` with `DocumentTypeCode`/`CreditNoteTypeCode` `"94"`/`"93"` and `CustomizationID` set to the *referenced* document's own type code (e.g. `"20"` for one adjusting a POS ticket) instead of a generic "references some invoice" code.

---

## DIAN Technical Annex coverage

An honest assessment of where this library stands against DIAN's Technical Annex 1.9 (Electronic Invoicing, Resolution 000165/2023) and the Electronic Payroll Technical Annex, as of this writing.

**~95% complete as an issuance library**, and now also covering **RADIAN events** (the receiver-acknowledgment side of the protocol) and **Documento Equivalente Electrónico** (POS tickets and its 9 sibling sub-types) — every document family and event type DIAN's Technical Annex 1.9, the RADIAN annex, and the Documento Equivalente Electrónico Technical Annex V1.0 define is implemented and covers the SOAP operations needed to submit and query them (Documento Equivalente Electrónico reuses the same `WcfDianCustomerServices` operations as Invoice — no new client code was needed). The core document types (Invoice, Credit/Debit Note, Support Document, Adjustment Note, Payroll) were verified line-by-line against the annex sections and, where the annex was ambiguous, against real accepted documents. The RADIAN event builders and the Documento Equivalente Electrónico support are newer and verified against the annex text only so far — see the caveats in the [feature table](#what-it-supports-today) above before relying on either in production.

The real certification WSDL also exposes a few more read-oriented operations this library doesn't wrap yet — `GetXmlByDocumentKey`, `GetReferenceNotes`, `GetDocumentInfo`, `GetExchangeEmails` — reasonable candidates for a future addition.

### Known gaps and roadmap

1. **RADIAN events need a real-DIAN pass.** Structurally complete (all 5 event types, correct CUDE formula verified against the annex's own worked example), but not yet confirmed end-to-end against DIAN's certification environment the way the rest of this library was. Do that before depending on it for anything that matters.
2. **Documento Equivalente Electrónico needs a real-DIAN pass, and only POS has a dedicated test.** The mechanism is confirmed on paper (same UBL, same webservice, byte-identical CUDE formula to `cude.Compute`), and `builder/pos_test.go` / `builder/pos_adjustment_note_test.go` prove the POS ticket (`20`) and its adjustment notes (`93`/`94`) build correctly against the annex's field tables — including a real bug this caught in `builder`'s line-item logic (a closed allowlist that assumed any non-Invoice/Support-Document type code meant a note's "Mandante" shape, silently mis-rendering the new codes; fixed). The other 9 sub-types (`25`/`27`/`30`/`35`/`40`/`45`/`50`/`55`/`60`) should work identically but have no dedicated test yet, and none of this family has been submitted to DIAN's certification environment.
3. **No graphic representation (RIDE / PDF) generator.** This isn't part of what DIAN validates over SOAP, so it's out of scope for this "core" library — but it's something almost every real integration needs downstream, and it's worth deciding explicitly whether it belongs in a companion module.
4. **No catalog/reference-data validation** (tax types, unit codes, city/DANE codes, payment methods, DIAN's RADIAN rejection-reason catalog for `Reclamo`, etc.). This is a **deliberate design choice**, documented in `domain/types.go` — the library trusts the caller and only knows where a code goes in the XML, not whether it's valid. Not a gap; a boundary.
5. **No orchestration layer.** There's no single "send an invoice" call — you own the pipeline (numbering, idempotency, retry on `SendBillAsync`/`GetStatusZip` polling, persistence of consecutive numbers, etc.). Also a deliberate boundary, not a bug — the [Quick start](#quick-start) example shows the shape of what you'd wire up.

None of the above blocks issuing real, DIAN-accepted Invoices, Credit/Debit Notes, Support Documents, Adjustment Notes, Attached Documents, or Payroll documents (standard or adjustment) today.

---

## Security notes

- **Never commit certificates, `.p12`/`.pfx` files, PINs, or software IDs.** `.gitignore` already excludes common patterns (`*.p12`, `*.pfx`, `*_cert.pem`, `*_key.pem`, `.env*`), but review `git status` before every commit regardless.
- `signer.LoadPEM` / `signer.LoadPKCS12` load key material into memory only — the library never persists it anywhere.
- `securitycode.Compute` takes your DIAN-assigned `softwareID`/`pin` directly; treat both as secrets with the same care as a private key.
- Tests that talk to DIAN's real certification server live entirely outside this repository — see [Testing](#testing) below. `go test ./...` inside this module never touches the network or requires credentials.

---

## Testing

```bash
go test ./...
```

Runs the full unit test suite (XML golden-file comparisons, hash/signature vector checks, response-parsing tests) with no network access and no credentials required.

Tests that submit real documents to DIAN's certification (habilitación) environment are **not part of this module**. They live in a separate, unpublished sibling project (`integration-tests/`, a standalone Go module that depends on `cofacture` through its public API only) so that a real certificate, DIAN credentials, and test data never need to touch this repository. If you're working from the same workspace this library was developed in, point `COFACTURE_TEST_FIXTURES_DIR` at your local fixtures directory and run `go test ./...` from within that sibling module.

---

## Documentation

A section-by-section reference scaffold (architecture, package-by-package API reference, DIAN annex cross-reference, SOAP operation catalog) is being drafted outside this repository for now and isn't tracked here yet.

---

## Disclaimer

This project is an independent, community-built toolkit. It is **not affiliated with, endorsed by, or officially certified by DIAN** (*Dirección de Impuestos y Aduanas Nacionales*). Achieving DIAN's *habilitación* (certification) for a specific NIT/software combination is a separate process the taxpayer/technology provider must complete directly with DIAN; this library helps you build the documents involved, but using it correctly does not by itself grant certification.

---

## License

[MIT](LICENSE) — © 2026 Diego Montoya.
