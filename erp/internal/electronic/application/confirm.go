package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/diegofxm/cofacture/cude"
	"github.com/diegofxm/cofacture/cuds"
	"github.com/diegofxm/cofacture/cufe"
	cofdom "github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/qr"
	"github.com/diegofxm/cofacture/securitycode"
	"github.com/diegofxm/cofacture/zip"
	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/electronic/domain"
	"github.com/diegofxm/erp/internal/shared/nit"
	"github.com/diegofxm/erp/internal/shared/timeutil"
)

// Constantes verificadas contra documentos DIAN reales (legacy/apidian service.go).
const (
	invoiceProfileID         = "DIAN 2.1: Factura Electrónica de Venta"
	invoiceOperationTypeCode = "10"
	invoiceHashType          = "CUFE-SHA384"

	creditNoteProfileID         = "DIAN 2.1: Nota Crédito de Factura Electrónica de Venta"
	creditNoteOperationTypeCode = "20"
	creditNoteHashType          = "CUDE-SHA384"

	debitNoteProfileID         = "DIAN 2.1: Nota Débito de Factura Electrónica de Venta"
	debitNoteOperationTypeCode = "30"
	debitNoteHashType          = "CUDE-SHA384"

	supportDocumentProfileID = "DIAN 2.1: documento soporte en adquisiciones efectuadas a no obligados a facturar."
	supportDocumentHashType  = "CUDS-SHA384"

	adjustmentNoteProfileID = "DIAN 2.1: Nota de ajuste al documento soporte en adquisiciones efectuadas a sujetos no obligados a expedir factura o documento equivalente"
	adjustmentNoteHashType  = "CUDS-SHA384"
)

// ConfirmUseCase reclama el consecutivo, construye y firma el XML, y lo envía a la DIAN.
// Es el ÚNICO punto donde se "gasta" un número real — los borradores se pueden editar/eliminar.
type ConfirmUseCase struct {
	documents domain.DocumentRepository
	numbering domain.NumberingRepository
	companies domain.CompanyPort
	builder   domain.BuilderSignerPort
	zipper    domain.ZipperPort
	sender    domain.SenderPort
}

func NewConfirmUseCase(
	documents domain.DocumentRepository,
	numbering domain.NumberingRepository,
	companies domain.CompanyPort,
	builder domain.BuilderSignerPort,
	zipper domain.ZipperPort,
	sender domain.SenderPort,
) *ConfirmUseCase {
	return &ConfirmUseCase{
		documents: documents,
		numbering: numbering,
		companies: companies,
		builder:   builder,
		zipper:    zipper,
		sender:    sender,
	}
}

// Confirm confirma un borrador por ID. Solo el CompanyID que lo creó puede confirmarlo.
func (uc *ConfirmUseCase) Confirm(ctx context.Context, companyID, id uuid.UUID) (*domain.Document, error) {
	d, err := uc.documents.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d.CompanyID != companyID {
		return nil, domain.ErrDocumentNotFound
	}
	if d.Status != domain.StatusDraft {
		return nil, domain.ErrDocumentNotDraft
	}

	switch d.DianDocumentTypeCode {
	case dianFE:
		return uc.confirmInvoice(ctx, d)
	case dianNC:
		return uc.confirmCreditNote(ctx, d)
	case dianND:
		return uc.confirmDebitNote(ctx, d)
	case dianDS:
		return uc.confirmSupportDocument(ctx, d)
	case dianNA:
		return uc.confirmAdjustmentNote(ctx, d)
	default:
		return nil, fmt.Errorf("electronic: tipo de documento desconocido %q", d.DianDocumentTypeCode)
	}
}

// ── confirmaciones por tipo ───────────────────────────────────────────────────────────────

func (uc *ConfirmUseCase) confirmInvoice(ctx context.Context, d *domain.Document) (*domain.Document, error) {
	p, err := uc.prepare(ctx, d)
	if err != nil {
		return nil, err
	}

	headerTaxes := aggregateTaxes(d.Lines)
	inv := cofdom.Invoice{
		ProfileID:         invoiceProfileID,
		EnvironmentCode:   string(p.company.Environment),
		OperationTypeCode: invoiceOperationTypeCode,
		DocumentTypeCode:  dianFE,
		HashType:          invoiceHashType,
		Prefix:            p.nr.Prefix,
		Number:            strconv.FormatInt(p.number, 10),
		IssueDate:         p.now.Format("2006-01-02"),
		IssueTime:         p.now.Format("15:04:05-07:00"),
		CurrencyCode:      d.CurrencyCode,
		Note:              d.Note,
		Supplier:          partyFromCompany(p.company),
		Customer:          d.Customer,
		PaymentMeans:      d.PaymentMeans,
		HeaderTaxes:       headerTaxes,
		Totals:            d.Totals,
		Lines:             d.Lines,
		NumberingRange:    numberingRangeFrom(p.nr),
		SoftwareProvider:  softwareProviderFrom(p.company),
	}
	inv.CUFE = cufe.Compute(inv, p.nr.TechnicalKey)
	inv.SoftwareSecurityCode = securitycode.Compute(p.company.SoftwareID, p.company.SoftwarePIN, inv.Prefix+inv.Number)
	inv.QRURL = qr.URL(inv.EnvironmentCode, inv.CUFE)

	d.Prefix, d.Number, d.DocumentKey = inv.Prefix, p.number, inv.CUFE
	d.IssueDate, d.IssueTime, d.QRURL = p.now, inv.IssueTime, inv.QRURL

	xmlBytes, err := uc.builder.SignedInvoiceXML(inv, p.company.Certificate, p.company.CertificatePassword, p.now)
	if err != nil {
		_ = uc.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, p.number)
		return nil, fmt.Errorf("construir/firmar invoice: %w", err)
	}
	return uc.finalizeAndSend(ctx, xmlBytes, p, d, zip.KindInvoice)
}

func (uc *ConfirmUseCase) confirmCreditNote(ctx context.Context, d *domain.Document) (*domain.Document, error) {
	if d.BillingReference == nil {
		return nil, domain.ErrMissingBillingReference
	}
	p, err := uc.prepare(ctx, d)
	if err != nil {
		return nil, err
	}

	base := uc.noteBase(d, p, creditNoteProfileID, creditNoteOperationTypeCode, creditNoteHashType, dianNC)
	cn := cofdom.CreditNote{
		Invoice:             base,
		CreditNoteTypeCode:  dianNC,
		BillingReference:    billingRefFrom(*d.BillingReference),
		DiscrepancyResponse: discrepancyFrom(d.DiscrepancyResponse),
	}
	cn.CUFE = cude.Compute(cn.Invoice, p.company.SoftwarePIN)
	cn.SoftwareSecurityCode = securitycode.Compute(p.company.SoftwareID, p.company.SoftwarePIN, cn.Prefix+cn.Number)
	cn.QRURL = qr.URL(cn.EnvironmentCode, cn.CUFE)

	d.Prefix, d.Number, d.DocumentKey = cn.Prefix, p.number, cn.CUFE
	d.IssueDate, d.IssueTime, d.QRURL = p.now, cn.IssueTime, cn.QRURL

	xmlBytes, err := uc.builder.SignedCreditNoteXML(cn, p.company.Certificate, p.company.CertificatePassword, p.now)
	if err != nil {
		_ = uc.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, p.number)
		return nil, fmt.Errorf("construir/firmar credit note: %w", err)
	}
	return uc.finalizeAndSend(ctx, xmlBytes, p, d, zip.KindCreditNote)
}

func (uc *ConfirmUseCase) confirmDebitNote(ctx context.Context, d *domain.Document) (*domain.Document, error) {
	if d.BillingReference == nil {
		return nil, domain.ErrMissingBillingReference
	}
	p, err := uc.prepare(ctx, d)
	if err != nil {
		return nil, err
	}

	base := uc.noteBase(d, p, debitNoteProfileID, debitNoteOperationTypeCode, debitNoteHashType, dianND)
	dn := cofdom.DebitNote{
		Invoice:             base,
		BillingReference:    billingRefFrom(*d.BillingReference),
		DiscrepancyResponse: discrepancyFrom(d.DiscrepancyResponse),
	}
	dn.CUFE = cude.Compute(dn.Invoice, p.company.SoftwarePIN)
	dn.SoftwareSecurityCode = securitycode.Compute(p.company.SoftwareID, p.company.SoftwarePIN, dn.Prefix+dn.Number)
	dn.QRURL = qr.URL(dn.EnvironmentCode, dn.CUFE)

	d.Prefix, d.Number, d.DocumentKey = dn.Prefix, p.number, dn.CUFE
	d.IssueDate, d.IssueTime, d.QRURL = p.now, dn.IssueTime, dn.QRURL

	xmlBytes, err := uc.builder.SignedDebitNoteXML(dn, p.company.Certificate, p.company.CertificatePassword, p.now)
	if err != nil {
		_ = uc.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, p.number)
		return nil, fmt.Errorf("construir/firmar debit note: %w", err)
	}
	return uc.finalizeAndSend(ctx, xmlBytes, p, d, zip.KindDebitNote)
}

func (uc *ConfirmUseCase) confirmSupportDocument(ctx context.Context, d *domain.Document) (*domain.Document, error) {
	if d.Supplier == nil {
		return nil, domain.ErrMissingSupplier
	}
	p, err := uc.prepare(ctx, d)
	if err != nil {
		return nil, err
	}

	headerTaxes := aggregateTaxes(d.Lines)
	inv := cofdom.Invoice{
		ProfileID:         supportDocumentProfileID,
		EnvironmentCode:   string(p.company.Environment),
		OperationTypeCode: d.OperationTypeCode,
		DocumentTypeCode:  dianDS,
		HashType:          supportDocumentHashType,
		Prefix:            p.nr.Prefix,
		Number:            strconv.FormatInt(p.number, 10),
		IssueDate:         p.now.Format("2006-01-02"),
		IssueTime:         p.now.Format("15:04:05-07:00"),
		CurrencyCode:      d.CurrencyCode,
		Note:              d.Note,
		// Roles invertidos: Supplier = tercero no obligado, Customer = empresa compradora.
		Supplier:         supplierAsNIT(*d.Supplier),
		Customer:         partyFromCompanyAsNIT(p.company),
		PaymentMeans:     d.PaymentMeans,
		HeaderTaxes:      headerTaxes,
		WithholdingTaxes: d.WithholdingTaxes,
		Totals:           d.Totals,
		Lines:            d.Lines,
		NumberingRange:   numberingRangeFrom(p.nr),
		SoftwareProvider: softwareProviderFrom(p.company),
	}
	inv.CUFE = cuds.Compute(inv, p.company.SoftwarePIN)
	inv.SoftwareSecurityCode = securitycode.Compute(p.company.SoftwareID, p.company.SoftwarePIN, inv.Prefix+inv.Number)
	inv.QRURL = qr.SupportDocumentContent(inv, inv.CUFE, p.company.SoftwarePIN)

	d.Prefix, d.Number, d.DocumentKey = inv.Prefix, p.number, inv.CUFE
	d.IssueDate, d.IssueTime = p.now, inv.IssueTime
	d.QRURL = qr.SupportDocumentURL(inv.EnvironmentCode, inv.CUFE)

	xmlBytes, err := uc.builder.SignedSupportDocumentXML(inv, p.company.Certificate, p.company.CertificatePassword, p.now)
	if err != nil {
		_ = uc.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, p.number)
		return nil, fmt.Errorf("construir/firmar support document: %w", err)
	}
	return uc.finalizeAndSend(ctx, xmlBytes, p, d, zip.KindSupportDocument)
}

func (uc *ConfirmUseCase) confirmAdjustmentNote(ctx context.Context, d *domain.Document) (*domain.Document, error) {
	if d.Supplier == nil {
		return nil, domain.ErrMissingSupplier
	}
	if d.BillingReference == nil {
		return nil, domain.ErrMissingBillingReference
	}
	p, err := uc.prepare(ctx, d)
	if err != nil {
		return nil, err
	}

	headerTaxes := aggregateTaxes(d.Lines)
	inv := cofdom.Invoice{
		ProfileID:         adjustmentNoteProfileID,
		EnvironmentCode:   string(p.company.Environment),
		OperationTypeCode: d.OperationTypeCode,
		DocumentTypeCode:  dianNA,
		HashType:          adjustmentNoteHashType,
		Prefix:            p.nr.Prefix,
		Number:            strconv.FormatInt(p.number, 10),
		IssueDate:         p.now.Format("2006-01-02"),
		IssueTime:         p.now.Format("15:04:05-07:00"),
		CurrencyCode:      d.CurrencyCode,
		Note:              d.Note,
		Supplier:          supplierAsNIT(*d.Supplier),
		Customer:          partyFromCompanyAsNIT(p.company),
		PaymentMeans:      d.PaymentMeans,
		HeaderTaxes:       headerTaxes,
		WithholdingTaxes:  d.WithholdingTaxes,
		Totals:            d.Totals,
		Lines:             d.Lines,
		NumberingRange:    numberingRangeFrom(p.nr),
		SoftwareProvider:  softwareProviderFrom(p.company),
	}
	an := cofdom.AdjustmentNote{
		Invoice:             inv,
		BillingReference:    billingRefFrom(*d.BillingReference),
		DiscrepancyResponse: discrepancyFrom(d.DiscrepancyResponse),
	}
	an.CUFE = cuds.Compute(an.Invoice, p.company.SoftwarePIN)
	an.SoftwareSecurityCode = securitycode.Compute(p.company.SoftwareID, p.company.SoftwarePIN, an.Prefix+an.Number)
	an.QRURL = qr.AdjustmentNoteContent(an.Invoice, an.CUFE, p.company.SoftwarePIN)

	d.Prefix, d.Number, d.DocumentKey = an.Prefix, p.number, an.CUFE
	d.IssueDate, d.IssueTime = p.now, an.IssueTime
	d.QRURL = qr.SupportDocumentURL(an.EnvironmentCode, an.CUFE)

	xmlBytes, err := uc.builder.SignedAdjustmentNoteXML(an, p.company.Certificate, p.company.CertificatePassword, p.now)
	if err != nil {
		_ = uc.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, p.number)
		return nil, fmt.Errorf("construir/firmar adjustment note: %w", err)
	}
	return uc.finalizeAndSend(ctx, xmlBytes, p, d, zip.KindAdjustmentNote)
}

// ── pipeline común ────────────────────────────────────────────────────────────────────────

type prepared struct {
	company *domain.CompanyInfo
	nr      *domain.NumberingRange
	number  int64
	now     time.Time
}

// prepare carga la empresa, el rango, valida credenciales y reclama el consecutivo.
func (uc *ConfirmUseCase) prepare(ctx context.Context, d *domain.Document) (*prepared, error) {
	company, err := uc.companies.GetCompany(ctx, d.CompanyID)
	if err != nil {
		return nil, err
	}
	if company.SoftwareID == "" || company.SoftwarePIN == "" || len(company.Certificate) == 0 {
		return nil, domain.ErrCompanyNotReadyToIssue
	}
	nr, err := uc.numbering.GetByID(ctx, d.NumberingRangeID)
	if err != nil {
		return nil, err
	}
	if nr.CompanyID != d.CompanyID {
		return nil, domain.ErrRangeCompanyMismatch
	}
	number, err := uc.numbering.ClaimNext(ctx, d.NumberingRangeID)
	if err != nil {
		return nil, err
	}
	return &prepared{
		company: company,
		nr:      nr,
		number:  number,
		now:     timeutil.Now(),
	}, nil
}

// finalizeAndSend persiste el estado "built", empaqueta y envía a la DIAN.
func (uc *ConfirmUseCase) finalizeAndSend(ctx context.Context, xmlBytes []byte, p *prepared, d *domain.Document, kind zip.DocumentKind) (*domain.Document, error) {
	d.SignedXML = string(xmlBytes)
	d.Status = domain.StatusBuilt

	confirmed, err := uc.documents.Confirm(ctx, *d)
	if err != nil {
		_ = uc.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, p.number)
		return nil, err
	}

	// No enviar si el ambiente del rango no coincide con el de la empresa -- error de
	// configuración, no de comunicación con la DIAN. Antes quedaba en StatusBuilt para siempre
	// sin ningún error visible; ahora queda en un estado dedicado y libera el consecutivo (nunca
	// se transmitió, no hay riesgo de doble facturación al liberarlo).
	if string(p.company.Environment) != string(p.nr.Environment) {
		return uc.finish(ctx, confirmed, domain.StatusEnvironmentMismatch, "", "", "",
			fmt.Sprintf("el ambiente del rango de numeración (%s) no coincide con el de la empresa (%s) — corrija la configuración del rango o de la empresa antes de reintentar", p.nr.Environment, p.company.Environment),
			"")
	}

	zipName, zipBytes, err := uc.zipper.Zip(string(kind), p.company.NIT, p.now.Year(), p.number, xmlBytes)
	if err != nil {
		// Falla local (empaquetado), antes de cualquier comunicación con la DIAN -- sin
		// ambigüedad posible, nunca llegó a enviarse. Va directo a StatusSendError (libera el
		// consecutivo), NO a través de markError (que asume un intento de envío real).
		return uc.finish(ctx, confirmed, domain.StatusSendError, "", "", "", fmt.Sprintf("empaquetar ZIP: %v", err), "")
	}

	// Producción o habilitación sin TestSetID → SendBillSync (síncrono).
	if p.company.Environment == domain.EnvProduccion || p.nr.TestSetID == "" {
		return uc.sendSync(ctx, confirmed, zipName, zipBytes, p)
	}
	return uc.sendTestSet(ctx, confirmed, zipName, zipBytes, p)
}

func (uc *ConfirmUseCase) sendSync(ctx context.Context, d *domain.Document, zipName string, zipBytes []byte, p *prepared) (*domain.Document, error) {
	result, err := uc.sender.SendBillSync(zipName, zipBytes, p.company.Certificate, p.company.CertificatePassword, string(p.company.Environment))
	if err != nil {
		if errors.Is(err, domain.ErrDianRejectedSync) {
			return uc.markError(ctx, d, err)
		}
		// Error de transporte (timeout/conexión): en vez de rendirse, reintentar el MISMO
		// documento ya firmado (mismo CUFE, no un documento nuevo) por la vía asíncrona antes
		// de marcarlo como ambiguo. A diferencia de SendBillSync, SendBillAsync deja un zipKey
		// consultable después, así que si este reintento también resulta ambiguo, el documento
		// queda recuperable (StatusSent) en vez de perdido (StatusSendUnknown).
		return uc.retryAsyncAfterSyncFailure(ctx, d, zipName, zipBytes, p, err)
	}
	status := domain.StatusAccepted
	statusMessage := result.StatusMessage
	if result.HasRejections || !result.IsValid {
		status = domain.StatusRejected
	}
	if warn := consumedNumberWarning(d, result.RespondedDocumentKey); warn != "" {
		statusMessage = warn
	}
	return uc.finish(ctx, d, status, "", result.StatusCode, result.StatusDescription, statusMessage, result.ApplicationResponseXML)
}

// retryAsyncAfterSyncFailure reenvía el mismo ZIP ya firmado por la ruta asíncrona
// (SendBillAsync + sondeo de GetStatusZip) tras un error ambiguo de SendBillSync. Es un
// reintento de ENTREGA del mismo documento (mismo CUFE), no la creación de uno nuevo -- por
// eso es seguro repetirlo: si la DIAN ya había recibido el primer intento, este segundo
// registro no cambia el CUFE ni duplica el consecutivo.
func (uc *ConfirmUseCase) retryAsyncAfterSyncFailure(ctx context.Context, d *domain.Document, zipName string, zipBytes []byte, p *prepared, originalErr error) (*domain.Document, error) {
	zipKey, err := uc.sender.SendBillAsync(zipName, zipBytes, p.company.Certificate, p.company.CertificatePassword, string(p.company.Environment))
	if err != nil {
		if errors.Is(err, domain.ErrDianRejectedSync) {
			return uc.markError(ctx, d, err)
		}
		// Ambos intentos (síncrono y asíncrono de contingencia) fallaron de forma ambigua --
		// se conservan los dos mensajes para que la revisión manual tenga todo el contexto.
		return uc.markError(ctx, d, fmt.Errorf("envío síncrono y reintento asíncrono ambos ambiguos -- síncrono: %v, asíncrono: %w", originalErr, err))
	}
	if zipKey == "" {
		return uc.finish(ctx, d, domain.StatusSendError, "", "", "", fmt.Sprintf("envío síncrono ambiguo (%v); reintento asíncrono de contingencia rechazado por la DIAN sin ZipKey", originalErr), "")
	}
	last := uc.pollZipKey(zipKey, p)
	return uc.finishFromPoll(ctx, d, zipKey, last, fmt.Sprintf("reintento de contingencia tras envío síncrono ambiguo (%v): respuesta de la DIAN no disponible todavía (sondeo agotado)", originalErr))
}

func (uc *ConfirmUseCase) sendTestSet(ctx context.Context, d *domain.Document, zipName string, zipBytes []byte, p *prepared) (*domain.Document, error) {
	zipKey, err := uc.sender.SendTestSetAsync(zipName, zipBytes, p.nr.TestSetID, p.company.Certificate, p.company.CertificatePassword, string(p.company.Environment))
	if err != nil {
		return uc.markError(ctx, d, err)
	}
	if zipKey == "" {
		return uc.finish(ctx, d, domain.StatusSendError, "", "", "", "la DIAN rechazó el envío sin ZipKey", "")
	}

	last := uc.pollZipKey(zipKey, p)
	if last != nil && last.IsTestSetClosed {
		_ = uc.numbering.ClearTestSetID(ctx, d.NumberingRangeID)
		return uc.sendSync(ctx, d, zipName, zipBytes, p)
	}
	return uc.finishFromPoll(ctx, d, zipKey, last, "respuesta de la DIAN no disponible todavía (sondeo agotado)")
}

// PollInterval es la espera entre sondeos de PollStatusZip. Variable (no const) para que los
// tests de reconciliación puedan acortarla en vez de esperar hasta 25s reales por caso.
var PollInterval = 5 * time.Second

// pollZipKey sondea el estado de un envío asíncrono (6 intentos, PollInterval entre cada uno —
// mismo patrón que el legacy) y devuelve el último resultado disponible, o nil si se agota el
// sondeo sin respuesta útil.
func (uc *ConfirmUseCase) pollZipKey(zipKey string, p *prepared) *domain.SendResult {
	var last *domain.SendResult
	for attempt := 0; attempt < 6; attempt++ {
		if attempt > 0 {
			time.Sleep(PollInterval)
		}
		res, err := uc.sender.PollStatusZip(zipKey, p.company.Certificate, p.company.CertificatePassword, string(p.company.Environment))
		if err != nil || res == nil {
			continue
		}
		last = res
		if last.StatusCode != "" {
			break
		}
	}
	return last
}

// finishFromPoll traduce el último SendResult de un sondeo en el estado final del documento.
// last == nil (sondeo agotado sin respuesta) deja el documento en StatusSent con el zipKey
// guardado -- recuperable después vía CheckPendingStatus, a diferencia de StatusSendUnknown que
// no tiene ningún identificador para volver a consultar.
func (uc *ConfirmUseCase) finishFromPoll(ctx context.Context, d *domain.Document, zipKey string, last *domain.SendResult, pendingMessage string) (*domain.Document, error) {
	if last == nil {
		return uc.finish(ctx, d, domain.StatusSent, zipKey, "", "", pendingMessage, "")
	}
	status := domain.StatusAccepted
	statusMessage := last.StatusMessage
	if last.HasRejections || !last.IsValid {
		status = domain.StatusRejected
	}
	if warn := consumedNumberWarning(d, last.RespondedDocumentKey); warn != "" {
		statusMessage = warn
	}
	return uc.finish(ctx, d, status, zipKey, last.StatusCode, last.StatusDescription, statusMessage, last.ApplicationResponseXML)
}

// CheckPendingStatus reintenta consultar el estado de un documento en StatusSent (un envío
// asíncrono cuyo sondeo se agotó sin respuesta) usando el zipKey ya guardado en DianTrackID. No
// genera ningún envío nuevo -- es una consulta pura contra la DIAN sobre un envío que ya se
// hizo, pensada para que un usuario resuelva manualmente un documento que quedó pendiente
// (ver limitación conocida del punto 02 del plan de acción: sin esto, StatusSent era un
// callejón sin salida permanente).
func (uc *ConfirmUseCase) CheckPendingStatus(ctx context.Context, companyID, id uuid.UUID) (*domain.Document, error) {
	d, err := uc.documents.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d.CompanyID != companyID {
		return nil, domain.ErrDocumentNotFound
	}
	if d.Status != domain.StatusSent || d.DianTrackID == "" {
		return nil, domain.ErrDocumentNotPending
	}
	company, err := uc.companies.GetCompany(ctx, d.CompanyID)
	if err != nil {
		return nil, err
	}
	last := uc.pollZipKey(d.DianTrackID, &prepared{company: company})
	return uc.finishFromPoll(ctx, d, d.DianTrackID, last, "respuesta de la DIAN todavía no disponible (nuevo sondeo agotado)")
}

// markError decide el estado final ante un fallo de envío. Distingue dos casos, porque no son
// igual de seguros para liberar el consecutivo:
//   - domain.ErrDianRejectedSync (la DIAN respondió con un soap:Fault explícito): sin ambigüedad,
//     la solicitud sí llegó y fue procesada/rechazada a nivel de protocolo -> StatusSendError,
//     seguro liberar el número.
//   - cualquier otro error (timeout, conexión, respuesta ilegible): no hay forma de saber si la
//     DIAN recibió y procesó el documento antes de que fallara la conexión -> StatusSendUnknown,
//     el número NO se libera (ver finish) para no arriesgar que un reintento (CloneDraft) lo
//     reutilice en un documento distinto -- doble facturación ante la DIAN.
func (uc *ConfirmUseCase) markError(ctx context.Context, d *domain.Document, sendErr error) (*domain.Document, error) {
	if errors.Is(sendErr, domain.ErrDianRejectedSync) {
		return uc.finish(ctx, d, domain.StatusSendError, "", "", "", sendErr.Error(), "")
	}
	return uc.finish(ctx, d, domain.StatusSendUnknown, "", "", "", sendErr.Error(), "")
}

// consumedNumberWarning detecta que la DIAN respondió con un CUFE/CUDE distinto al que este
// documento generó (RespondedDocumentKey, tomado de XmlDocumentKey en la respuesta SOAP) --
// señal inequívoca de que la DIAN no validó el contenido que acabamos de enviar, sino que
// devolvió el resultado ya registrado de un envío anterior para ese mismo consecutivo (ej. un
// número quemado manualmente en el portal de habilitación, fuera de este sistema). En ese caso
// el StatusDescription/StatusMessage originales describen un documento ajeno y serían engañosos
// -- se reemplazan por una alerta clara y accionable (bug real diagnosticado 2026-08-11: el
// mismo consecutivo devolvía siempre el rechazo de un documento de junio, sin importar el
// contenido enviado hoy).
func consumedNumberWarning(d *domain.Document, respondedDocumentKey string) string {
	if respondedDocumentKey == "" || respondedDocumentKey == d.DocumentKey {
		return ""
	}
	return fmt.Sprintf(
		"El consecutivo %s%d ya fue usado anteriormente ante la DIAN por fuera de este sistema (la DIAN respondió con un CUFE/CUDE distinto al que este documento generó, por lo que no llegó a validar su contenido). Verifica en el portal de habilitación de la DIAN cuál es el próximo consecutivo disponible y ajusta el rango de numeración antes de reintentar.",
		d.Prefix, d.Number,
	)
}

func (uc *ConfirmUseCase) finish(ctx context.Context, d *domain.Document, status domain.Status, trackID, statusCode, statusDescription, statusMessage, applicationResponseXML string) (*domain.Document, error) {
	if err := uc.documents.UpdateDianStatus(ctx, d.ID, status, trackID, statusCode, statusDescription, statusMessage, applicationResponseXML); err != nil {
		return nil, err
	}
	d.Status = status
	d.DianTrackID = trackID
	d.DianStatusCode = statusCode
	d.DianStatusDescription = statusDescription
	d.DianStatusMessage = statusMessage
	d.ApplicationResponseXML = applicationResponseXML
	// StatusSendUnknown queda deliberadamente fuera: liberar el consecutivo ahí es justo el
	// riesgo que este cambio corrige (ver domain.StatusSendUnknown). StatusEnvironmentMismatch
	// SÍ libera -- el documento nunca se transmitió, no hay ambigüedad ni riesgo de duplicado.
	if isRetryableFailure(status) {
		_ = uc.numbering.ReleaseIfCurrent(ctx, d.NumberingRangeID, d.Number)
	}
	return d, nil
}

// isRetryableFailure es el mismo criterio en los dos lugares donde importa si un documento
// fallido puede reintentarse sin riesgo: acá decide si se libera el consecutivo, y en
// from_sale.go/from_purchase.go decide si una venta/compra que ya generó un documento puede
// generar uno nuevo (antes esa venta quedaba bloqueada para siempre apuntando a la factura
// rechazada, sin forma de corregir y reenviar -- bug real reportado 2026-08-11). StatusSendUnknown
// queda deliberadamente fuera de ambos: es ambiguo si la DIAN llegó a recibir el documento,
// reintentar (con un número nuevo) arriesga doble facturación.
func isRetryableFailure(status domain.Status) bool {
	return status == domain.StatusRejected || status == domain.StatusSendError || status == domain.StatusEnvironmentMismatch
}

// ── helpers de construcción de cofacture.* ────────────────────────────────────────────────

func (uc *ConfirmUseCase) noteBase(d *domain.Document, p *prepared, profileID, operationTypeCode, hashType, docType string) cofdom.Invoice {
	return cofdom.Invoice{
		ProfileID:         profileID,
		EnvironmentCode:   string(p.company.Environment),
		OperationTypeCode: operationTypeCode,
		DocumentTypeCode:  docType,
		HashType:          hashType,
		Prefix:            p.nr.Prefix,
		Number:            strconv.FormatInt(p.number, 10),
		IssueDate:         p.now.Format("2006-01-02"),
		IssueTime:         p.now.Format("15:04:05-07:00"),
		CurrencyCode:      d.CurrencyCode,
		Note:              d.Note,
		Supplier:          partyFromCompany(p.company),
		Customer:          d.Customer,
		PaymentMeans:      d.PaymentMeans,
		HeaderTaxes:       aggregateTaxes(d.Lines),
		Totals:            d.Totals,
		Lines:             d.Lines,
		NumberingRange:    numberingRangeFrom(p.nr),
		SoftwareProvider:  softwareProviderFrom(p.company),
	}
}

func partyFromCompany(c *domain.CompanyInfo) cofdom.Party {
	var taxRegimeCode string
	if c.TaxRegimeCode != nil {
		taxRegimeCode = *c.TaxRegimeCode
	}
	return cofdom.Party{
		EntityTypeCode: c.EntityTypeCode,
		Identification: cofdom.Identification{
			Number:           c.NIT,
			TypeCode:         c.IdentificationTypeCode,
			VerificationCode: c.CheckDigit,
		},
		Name: c.BusinessName,
		Address: cofdom.Address{
			Line:        c.AddressLine,
			CityCode:    c.MunicipalityCode,
			CityName:    c.MunicipalityCode, // código como nombre (suficiente para la DIAN)
			StateCode:   c.DepartmentCode,
			StateName:   c.DepartmentCode,
			CountryCode: "CO",
			CountryName: "Colombia",
		},
		LiabilityCodes:              c.LiabilityCodes,
		TaxRegimeCode:               taxRegimeCode,
		IndustryClassificationCodes: c.IndustryClassificationCodes,
		TaxSchemeCode:               c.TaxSchemeCode,
		TaxSchemeName:               c.TaxSchemeName,
		Phone:                       c.Phone,
		Email:                       c.Email,
		MerchantRegistrationNumber:  c.MerchantRegistrationNumber,
	}
}

// partyFromCompanyAsNIT fuerza TypeCode="31" y VerificationCode para DS/NA.
func partyFromCompanyAsNIT(c *domain.CompanyInfo) cofdom.Party {
	p := partyFromCompany(c)
	p.Identification.TypeCode = "31"
	p.Identification.VerificationCode = c.CheckDigit
	if p.Address.PostalZone == "" {
		// El código de municipio (DANE) es un dato real de la empresa, no un relleno -- mismo
		// criterio que ya usan from_sale.go/from_purchase.go para poblar PostalZone en FE
		// normal. "000000" queda solo como último recurso si ni el municipio está registrado.
		p.Address.PostalZone = firstNonEmpty(c.MunicipalityCode, "000000")
	}
	return p
}

// supplierAsNIT normaliza el proveedor del DS: fuerza schemeName="31" (DSAJ25a).
func supplierAsNIT(p cofdom.Party) cofdom.Party {
	// Siempre recomputar el DV: el valor almacenado en BD puede ser incorrecto.
	if dv, err := nit.ComputeCheckDigit(p.Identification.Number); err == nil {
		p.Identification.VerificationCode = dv
	}
	p.Identification.TypeCode = "31"
	if p.Address.PostalZone == "" {
		// p.Address.CityCode es el código de municipio (DANE) real del tercero -- mismo criterio
		// que partyFromCompanyAsNIT. "000000" queda solo como último recurso.
		p.Address.PostalZone = firstNonEmpty(p.Address.CityCode, "000000")
	}
	return p
}

// firstNonEmpty devuelve el primer valor no vacío, o el último si todos están vacíos.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	if len(values) == 0 {
		return ""
	}
	return values[len(values)-1]
}

func numberingRangeFrom(nr *domain.NumberingRange) cofdom.NumberingRange {
	end := ""
	if nr.RangeTo != nil {
		end = strconv.FormatInt(*nr.RangeTo, 10)
	}
	startDate := ""
	if !nr.ValidFrom.IsZero() {
		startDate = nr.ValidFrom.Format("2006-01-02")
	}
	endDate := ""
	if !nr.ValidTo.IsZero() {
		endDate = nr.ValidTo.Format("2006-01-02")
	}
	return cofdom.NumberingRange{
		AuthorizedCode: nr.ResolutionNumber,
		Prefix:         nr.Prefix,
		StartNumber:    strconv.FormatInt(nr.RangeFrom, 10),
		EndNumber:      end,
		StartDate:      startDate,
		EndDate:        endDate,
	}
}

func softwareProviderFrom(c *domain.CompanyInfo) cofdom.SoftwareProvider {
	return cofdom.SoftwareProvider{
		ProviderIdentification: cofdom.Identification{
			Number:           c.NIT,
			TypeCode:         "31",
			VerificationCode: c.CheckDigit,
		},
		SoftwareID: c.SoftwareID,
	}
}

func billingRefFrom(b domain.BillingReferenceInput) cofdom.BillingReference {
	return cofdom.BillingReference{
		Prefix:    b.Prefix,
		Number:    b.Number,
		CUFE:      b.CUFE,
		IssueDate: b.IssueDate,
	}
}

func discrepancyFrom(d *domain.DiscrepancyResponseInput) *cofdom.DiscrepancyResponse {
	if d == nil {
		return nil
	}
	return &cofdom.DiscrepancyResponse{
		ReferenceID:  d.ReferenceID,
		ResponseCode: d.ResponseCode,
		Description:  d.Description,
	}
}

// aggregateTaxes agrupa impuestos de todas las líneas por (TypeCode, Percent).
func aggregateTaxes(lines []cofdom.Line) []cofdom.Tax {
	type key struct {
		typeCode string
		percent  float64
	}
	type agg struct {
		typeName           string
		taxableAmountCents int64
		taxAmountCents     int64
	}
	byKey := make(map[key]*agg)
	var order []key
	for _, l := range lines {
		for _, t := range l.Taxes {
			k := key{typeCode: t.TypeCode, percent: t.Percent}
			a, ok := byKey[k]
			if !ok {
				a = &agg{typeName: t.TypeName}
				byKey[k] = a
				order = append(order, k)
			}
			a.taxableAmountCents += t.TaxableAmountCents
			a.taxAmountCents += t.TaxAmountCents
		}
	}
	taxes := make([]cofdom.Tax, 0, len(order))
	for _, k := range order {
		a := byKey[k]
		taxes = append(taxes, cofdom.Tax{
			TaxableAmountCents: a.taxableAmountCents,
			TaxAmountCents:     a.taxAmountCents,
			Percent:            k.percent,
			TypeCode:           k.typeCode,
			TypeName:           a.typeName,
		})
	}
	return taxes
}
