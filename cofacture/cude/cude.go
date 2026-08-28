// Package cude computes the Unique Electronic Document Code (Código Único de Documento
// Electrónico, CUDE) for Credit Notes and Debit Notes, per DIAN's Technical Annex 1.9
// (sections 11.4.3-11.4.6).
//
// The CUFE (Invoice) lives in package cufe, not here — the formulas are nearly identical, but
// they are distinct identifiers for distinct documents.
package cude

import (
	"crypto/sha512"
	"encoding/hex"

	"github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/internal/dianhash"
)

// Compute calculates the CUDE of a Credit Note or Debit Note.
//
// The formula is identical to CUFE's in structure and field order (validated against the two
// official annex examples, one per note type) — the only real difference is that instead of
// the numbering range's technical key, the software's PIN is used ("Software-PIN", the same
// one used for securitycode.Compute). noteBase is the Invoice embedded in
// domain.CreditNote/domain.DebitNote (the promoted field, e.g. creditNote.Invoice).
func Compute(noteBase domain.Invoice, softwarePIN string) string {
	sum := sha512.Sum384([]byte(dianhash.Seed(noteBase, softwarePIN)))
	return hex.EncodeToString(sum[:])
}
