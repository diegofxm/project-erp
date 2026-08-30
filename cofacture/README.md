# cofacture

[![Go Reference](https://pkg.go.dev/badge/github.com/diegofxm/cofacture.svg)](https://pkg.go.dev/github.com/diegofxm/cofacture)
[![Go Report Card](https://goreportcard.com/badge/github.com/diegofxm/cofacture)](https://goreportcard.com/report/github.com/diegofxm/cofacture)
[![Go Version](https://img.shields.io/github/go-mod/go-version/diegofxm/cofacture)](go.mod)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

A Go toolkit for Colombian DIAN electronic invoicing: UBL 2.1 document generation, XAdES-EPES signing, CUFE/CUDE/CUDS/CUNE computation, and a SOAP client for DIAN's `WcfDianCustomerServices` web services.

`cofacture` is a **library, not a platform** — no database, no HTTP server, no opinion about how you store your data. You feed it plain Go structs; it hands back signed XML, ready to send. You wire the pipeline (build → hash → sign → zip → send) yourself: nothing here retries on your behalf, persists a consecutive number, or validates a catalog code before putting it where the annex says it goes.

---

## What's implemented

Every row below reflects the current state of this repository, checked by running its test suite — not aspiration.

| Document | Type code(s) | Golden/unit test in this repo | Confirmed against real, DIAN-accepted documents |
|---|---|---|---|
| Electronic Sales Invoice | `01` | Yes | Yes |
| Credit Note | `91` | Yes | CUDE formula matches DIAN's official worked example; no dedicated real-submission test in this repo |
| Debit Note | `92` | Yes | CUDE formula matches the *corrected* official worked example — DIAN's own published example has a transcription error (see [`cude`](./cude)); no dedicated real-submission test in this repo |
| Support Document | `05` | Yes | Yes |
| Adjustment Note to the Support Document | `95` | Yes | Yes |
| Attached Document (container for Invoice/Credit Note/Debit Note) | — | Yes | Yes — never submitted to DIAN itself, see note below |
| Individual Electronic Payroll | `102` | Regression test only — does not reproduce DIAN's own published CUNE worked example (documented, unresolved; see [`payroll`](./payroll)) | Not currently confirmed in this environment |
| Payroll Adjustment | `103`/`104` | Yes (Novedad/CUNENovedad logic) | Same caveat as above |
| RADIAN events — Acuse de Recibo (`030`), Reclamo (`031`), Recibo del Bien (`032`), Aceptación Expresa (`033`), Aceptación Tácita (`034`) | — | Yes, one per type | No — Aceptación Tácita was built, CUDE-computed and XAdES-signed against a real certificate and a real accepted invoice, but not submitted (see note below); the other four were never built against real data |
| Documento Equivalente Electrónico — POS ticket | `20` | Yes | Yes, submitted through a downstream application built on this library — not through this repo's own tests |
| Documento Equivalente Electrónico — other 9 sub-types (cinema, public shows, gaming, ground transport, tolls, financial extracts, air tickets, exchange/commodities settlements, public utility bills) | `25`/`27`/`30`/`35`/`40`/`45`/`50`/`55`/`60` | No | No |
| Documento Equivalente Electrónico Adjustment Note | `93` (débito) / `94` (crédito) | Yes | No |

Notes:

- **Attached Document** is never submitted to DIAN by SOAP — it's built *after* DIAN has already validated the document it wraps (it embeds that validation response) and is delivered directly to the acquirer instead. There is no `Send*` operation that takes one.
- **Aceptación Tácita** is a sworn statement that a fixed number of business days passed since a prior Recibo del Bien event with no response. Submitting one for a document minutes old, referencing an event that doesn't exist, would assert something false into DIAN's system — even in the certification environment. The test that exercises it (in the sibling `integration-tests` module) builds, hashes and signs it against real material and stops there, deliberately.
- **Documento Equivalente Electrónico** reuses this library's Invoice/Credit Note/Debit Note builders and `cude`'s CUDE formula field-for-field — confirmed against the Documento Equivalente Electrónico Technical Annex V1.0 (Resolution 000165/2023). Only the POS ticket and its two adjustment note types have a dedicated test; the other 9 sub-types share the same mechanism but have not been exercised individually.
- The DIAN web service (`WcfDianCustomerServices`) also exposes `GetXmlByDocumentKey`, `GetReferenceNotes`, `GetDocumentInfo`, and `GetExchangeEmails`, which this library does not implement.

### SOAP operations implemented (package `soap`)

`SendBillSync` · `SendBillAsync` · `SendBillAttachmentAsync` · `SendTestSetAsync` · `GetStatus` · `GetStatusZip` · `GetNumberingRange` · `SendNominaSync` · `SendNominaSyncTestSet` · `SendEventUpdateStatus` · `GetStatusEvent` · `GetAcquirer`

### Design boundaries (not gaps)

- **No catalog/reference-data validation** — tax types, unit codes, city/DANE codes, payment methods, DIAN's RADIAN rejection-reason catalog, etc. Documented in `domain/types.go`: the library trusts the caller and only knows where a code goes in the XML, not whether it's valid.
- **No graphic representation (RIDE/PDF) generator.** DIAN doesn't validate this over SOAP; it's outside what this library does.
- **No orchestration layer.** No single "send an invoice" call — you own numbering, idempotency, retrying `SendBillAsync`/`GetStatusZip` polling, and persisting consecutive numbers. See [Quick start](#quick-start) for the shape of what you'd wire up.

---

## Package map

| Package | Responsibility |
|---|---|
| [`domain`](./domain) | Plain Go structs for every document (`Invoice`, `CreditNote`, `DebitNote`, `AdjustmentNote`, `AttachedDocument`, `Event`, `Reclamo`, `Party`, `Tax`, `Line`, ...). No validation, no persistence. |
| [`builder`](./builder) | Assembles the UBL 2.1 + DIAN-extension XML tree from a domain model (`etree.Document`) — every document type plus the RADIAN `ApplicationResponse` events. Does not sign, hash, or send anything. |
| [`cufe`](./cufe) | Computes the CUFE (Invoice) per Technical Annex 1.9. |
| [`cude`](./cude) | Computes the CUDE for Credit Notes, Debit Notes, **and Documento Equivalente Electrónico** — same formula, see the package's doc comment for why one function covers all three. |
| [`cuds`](./cuds) | Computes the CUDS (Support Document, Adjustment Note to the Support Document). |
| [`event`](./event) | Computes the CUDE for RADIAN events (a different field composition from `cude`'s) and holds their `ResponseCode` catalog and the Aceptación Tácita note template. |
| [`payroll`](./payroll) | Builds `NominaIndividual` XML (not UBL) and computes the CUNE. |
| [`securitycode`](./securitycode) | Computes `sts:SoftwareSecurityCode`. |
| [`qr`](./qr) | Builds the QR URL/content required in every document type's graphic representation. |
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

Requires Go 1.26.4 or newer (see `go.mod`).

---

## Quick start

The full pipeline for a single invoice, end to end. Error handling is abbreviated for readability — check every error in real code.

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

Building a Support Document, Credit Note, Debit Note, Adjustment Note, or Payroll document follows the same shape — swap `builder.BuildInvoice` for `builder.BuildSupportDocument` / `builder.BuildCreditNote` / `builder.BuildDebitNote` / `builder.BuildAdjustmentNote` / `payroll.Build`, and `cufe.Compute` for `cuds.Compute` / `cude.Compute` / `payroll.Cune` as appropriate.

RADIAN events (`builder.BuildAcuseRecibo` and its four siblings) follow a related but not identical shape: `event.Compute` instead of `cufe.Compute`, `Sender`/`Receiver` instead of `Supplier`/`Customer`, and a `DocumentReference` pointing at the document the event applies to instead of a `BillingReference`. The event's QR is still built from the *referenced* document's CUFE (`qr.URL(ev.EnvironmentCode, ev.DocumentReference.CUFE)`), not from the event's own CUDE.

A Documento Equivalente Electrónico (e.g. a POS ticket) is `builder.BuildInvoice` and `cude.Compute` again — same `domain.Invoice`, same pipeline as above — with five fields set differently: `ProfileID: "DIAN 2.1: Documento Equivalente POS"`, `OperationTypeCode: "10"` (this is what ends up as the `cbc:CustomizationID` XML element — the struct field is named after the DIAN concept it represents, not the element it renders to), `DocumentTypeCode: "20"` (the Technical Annex Documento Equivalente Electrónico V1.0, section 16.3, lists the codes for the other 9 sub-types), `HashType: "CUDE-SHA384"` (instead of `"CUFE-SHA384"`), and `softwarePIN` passed to `cude.Compute` where the invoice above passes `technicalKey` to `cufe.Compute`. Its Adjustment Notes are `builder.BuildCreditNote`/`BuildDebitNote` with `DocumentTypeCode`/`CreditNoteTypeCode` `"94"`/`"93"`, `OperationTypeCode` set to the *referenced* document's own type code (e.g. `"20"` for one adjusting a POS ticket), and `BillingReference.HashType` set to that referenced document's own hash scheme (`"CUDE-SHA384"` for a POS ticket, not `"CUFE-SHA384"`).

---

## Security notes

- **Never commit certificates, `.p12`/`.pfx` files, PINs, or software IDs.** `.gitignore` already excludes common patterns (`*.p12`, `*.pfx`, `*_cert.pem`, `*_key.pem`, `.env*`), but review `git status` before every commit regardless.
- `signer.LoadPEM` / `signer.LoadPKCS12` load key material into memory only — the library never persists it anywhere.
- `securitycode.Compute` takes your DIAN-assigned `softwareID`/`pin` directly; treat both as secrets with the same care as a private key.
- Tests that talk to DIAN's real certification server live entirely outside this repository — see [Testing](#testing). `go test ./...` inside this module never touches the network or requires credentials.

---

## Testing

```bash
go test ./...
```

Runs the full unit test suite (XML golden-file comparisons, hash/signature vector checks, response-parsing tests) with no network access and no credentials required.

Tests that submit real documents to DIAN's certification (habilitación) environment are **not part of this module**. They live in a separate, unpublished sibling project (`integration-tests/`, a standalone Go module that depends on `cofacture` through its public API only) so that a real certificate, DIAN credentials, and test data never need to touch this repository. If you're working from the same workspace this library was developed in, point `COFACTURE_TEST_FIXTURES_DIR` at your local fixtures directory and run `go test ./...` from within that sibling module.

---

## Disclaimer

This project is an independent, community-built toolkit. It is **not affiliated with, endorsed by, or officially certified by DIAN** (*Dirección de Impuestos y Aduanas Nacionales*). Achieving DIAN's *habilitación* (certification) for a specific NIT/software combination is a separate process the taxpayer/technology provider must complete directly with DIAN; this library helps you build the documents involved, but using it correctly does not by itself grant certification.

---

## License

[MIT](LICENSE) — © 2026 Diego Montoya.
