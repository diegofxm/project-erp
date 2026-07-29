// Package cofacture implementa los puertos BuilderSignerPort, ZipperPort y SenderPort
// usando la librería cofacture (github.com/diegofxm/cofacture).
package cofacture

import (
	"fmt"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/diegofxm/cofacture/builder"
	"github.com/diegofxm/cofacture/dian"
	cofdom "github.com/diegofxm/cofacture/domain"
	"github.com/diegofxm/cofacture/signer"
	"github.com/diegofxm/cofacture/soap"
	"github.com/diegofxm/cofacture/zip"

	"github.com/diegofxm/erp/internal/electronic/domain"
)

// Adapter implementa BuilderSignerPort, ZipperPort y SenderPort.
type Adapter struct{}

// New crea el adaptador de cofacture.
func New() *Adapter { return &Adapter{} }

var (
	_ domain.BuilderSignerPort    = (*Adapter)(nil)
	_ domain.ZipperPort           = (*Adapter)(nil)
	_ domain.SenderPort           = (*Adapter)(nil)
	_ domain.DianRangesFetcherPort = (*Adapter)(nil)
)

// ── BuilderSignerPort ─────────────────────────────────────────────────────────────────────

func (a *Adapter) SignedInvoiceXML(inv cofdom.Invoice, cert []byte, password string, signingTime time.Time) ([]byte, error) {
	doc, err := builder.BuildInvoice(inv)
	if err != nil {
		return nil, fmt.Errorf("build invoice: %w", err)
	}
	return signDoc(doc, cert, password, "supplier", signingTime)
}

func (a *Adapter) SignedCreditNoteXML(cn cofdom.CreditNote, cert []byte, password string, signingTime time.Time) ([]byte, error) {
	doc, err := builder.BuildCreditNote(cn)
	if err != nil {
		return nil, fmt.Errorf("build credit note: %w", err)
	}
	return signDoc(doc, cert, password, "supplier", signingTime)
}

func (a *Adapter) SignedDebitNoteXML(dn cofdom.DebitNote, cert []byte, password string, signingTime time.Time) ([]byte, error) {
	doc, err := builder.BuildDebitNote(dn)
	if err != nil {
		return nil, fmt.Errorf("build debit note: %w", err)
	}
	return signDoc(doc, cert, password, "supplier", signingTime)
}

func (a *Adapter) SignedSupportDocumentXML(inv cofdom.Invoice, cert []byte, password string, signingTime time.Time) ([]byte, error) {
	doc, err := builder.BuildSupportDocument(inv)
	if err != nil {
		return nil, fmt.Errorf("build support document: %w", err)
	}
	return signDoc(doc, cert, password, "supplier", signingTime)
}

func (a *Adapter) SignedAdjustmentNoteXML(an cofdom.AdjustmentNote, cert []byte, password string, signingTime time.Time) ([]byte, error) {
	doc, err := builder.BuildAdjustmentNote(an)
	if err != nil {
		return nil, fmt.Errorf("build adjustment note: %w", err)
	}
	return signDoc(doc, cert, password, "supplier", signingTime)
}

// signDoc firma el árbol etree y lo serializa a bytes.
// Nunca llamar Indent() después de firmar — invalida la firma.
func signDoc(doc *etree.Document, certBytes []byte, password, role string, signingTime time.Time) ([]byte, error) {
	x509cert, key, err := signer.LoadPKCS12(certBytes, password)
	if err != nil {
		return nil, fmt.Errorf("cargar certificado: %w", err)
	}
	placeholder, err := builder.SignaturePlaceholder(doc)
	if err != nil {
		return nil, fmt.Errorf("crear placeholder de firma: %w", err)
	}
	if err := signer.New(x509cert, key).Sign(doc.Root(), placeholder, role, signingTime); err != nil {
		return nil, fmt.Errorf("firmar documento: %w", err)
	}
	xmlBytes, err := doc.WriteToBytes()
	if err != nil {
		return nil, fmt.Errorf("serializar XML firmado: %w", err)
	}
	return xmlBytes, nil
}

// ── ZipperPort ────────────────────────────────────────────────────────────────────────────

func (a *Adapter) Zip(kind, nit string, year int, number int64, xmlBytes []byte) (string, []byte, error) {
	k := zip.DocumentKind(kind)
	xmlName := zip.DocumentFileName(k, nit, zip.SoftwarePropioCode, year, uint32(number))
	zipName := zip.PackageFileName(nit, zip.SoftwarePropioCode, year, uint32(number))
	zipBytes, err := zip.Build([]zip.File{{Name: xmlName, Content: xmlBytes}})
	if err != nil {
		return "", nil, fmt.Errorf("comprimir documento: %w", err)
	}
	return zipName, zipBytes, nil
}

// ── SenderPort ────────────────────────────────────────────────────────────────────────────

func (a *Adapter) SendBillSync(zipFileName string, zipBytes []byte, cert []byte, password string, environmentCode string) (*domain.SendResult, error) {
	x509cert, key, err := signer.LoadPKCS12(cert, password)
	if err != nil {
		return nil, fmt.Errorf("cargar certificado para SOAP: %w", err)
	}
	soapURL := soapURL(environmentCode)
	client := soap.New(soapURL, x509cert, key)
	resp, err := client.SendBillSync(zipFileName, zipBytes)
	if err != nil {
		return nil, fmt.Errorf("SendBillSync: %w", err)
	}
	interpreted, err := dian.Interpret(*resp)
	if err != nil {
		return &domain.SendResult{
			StatusCode:    resp.StatusCode,
			StatusMessage: fmt.Sprintf("interpretar respuesta: %v", err),
		}, nil
	}
	return &domain.SendResult{
		StatusCode:             interpreted.StatusCode,
		StatusDescription:      interpreted.StatusDescription,
		StatusMessage:          interpreted.StatusMessage,
		IsValid:                interpreted.IsValid,
		HasRejections:          interpreted.HasRejections(),
		ApplicationResponseXML: string(interpreted.ApplicationResponseXML),
	}, nil
}

func (a *Adapter) SendTestSetAsync(zipFileName string, zipBytes []byte, testSetID string, cert []byte, password string, environmentCode string) (string, error) {
	x509cert, key, err := signer.LoadPKCS12(cert, password)
	if err != nil {
		return "", fmt.Errorf("cargar certificado para SOAP: %w", err)
	}
	client := soap.New(soapURL(environmentCode), x509cert, key)
	resp, err := client.SendTestSetAsync(zipFileName, zipBytes, testSetID)
	if err != nil {
		return "", fmt.Errorf("SendTestSetAsync: %w", err)
	}
	return resp.ZipKey, nil
}

func (a *Adapter) PollStatusZip(zipKey string, cert []byte, password string, environmentCode string) (*domain.SendResult, error) {
	x509cert, key, err := signer.LoadPKCS12(cert, password)
	if err != nil {
		return nil, fmt.Errorf("cargar certificado para SOAP: %w", err)
	}
	client := soap.New(soapURL(environmentCode), x509cert, key)
	statuses, err := client.GetStatusZip(zipKey)
	if err != nil || len(statuses) == 0 {
		return nil, err
	}
	last := statuses[len(statuses)-1]
	interpreted, err := dian.Interpret(last)
	if err != nil {
		return &domain.SendResult{
			ZipKey:        zipKey,
			StatusCode:    last.StatusCode,
			StatusMessage: fmt.Sprintf("interpretar respuesta: %v", err),
		}, nil
	}
	return &domain.SendResult{
		ZipKey:                 zipKey,
		StatusCode:             interpreted.StatusCode,
		StatusDescription:      interpreted.StatusDescription,
		StatusMessage:          interpreted.StatusMessage,
		IsValid:                interpreted.IsValid,
		HasRejections:          interpreted.HasRejections(),
		IsTestSetClosed:        interpreted.IsTestSetClosed(),
		ApplicationResponseXML: string(interpreted.ApplicationResponseXML),
	}, nil
}

// ── DianRangesFetcherPort ─────────────────────────────────────────────────────────────────

func (a *Adapter) GetNumberingRanges(nit, softwareID string, cert []byte, password, environmentCode string) ([]domain.DianRange, error) {
	x509cert, key, err := signer.LoadPKCS12(cert, password)
	if err != nil {
		return nil, fmt.Errorf("cargar certificado: %w", err)
	}
	client := soap.New(soapURL(environmentCode), x509cert, key)
	result, err := client.GetNumberingRange(nit, nit, softwareID)
	if err != nil {
		return nil, fmt.Errorf("GetNumberingRange DIAN: %w", err)
	}
	ranges := make([]domain.DianRange, 0, len(result.ResponseList))
	for _, r := range result.ResponseList {
		ranges = append(ranges, domain.DianRange{
			ResolutionNumber:     r.ResolutionNumber,
			ResolutionDate:       trimDate(r.ResolutionDate),
			Prefix:               r.Prefix,
			RangeFrom:            r.FromNumber,
			RangeTo:              r.ToNumber,
			ValidFrom:            trimDate(r.ValidDateFrom),
			ValidTo:              trimDate(r.ValidDateTo),
			TechnicalKey:         r.TechnicalKey,
			SuggestedDocTypeCode: inferDocType(r.Prefix),
		})
	}
	return ranges, nil
}

func trimDate(s string) string {
	if i := strings.Index(s, "T"); i > 0 {
		return s[:i]
	}
	return s
}

func inferDocType(prefix string) string {
	switch strings.ToUpper(prefix) {
	case "SETP":
		return "01"
	case "SEDS":
		return "05"
	default:
		return ""
	}
}

func soapURL(environmentCode string) string {
	if environmentCode == "1" {
		return soap.ProduccionURL
	}
	return soap.HabilitacionURL
}
