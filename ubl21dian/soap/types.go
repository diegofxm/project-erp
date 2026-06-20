package soap

// Tipos que reflejan los esquemas reales descargados del WSDL de habilitación
// (https://vpfe-hab.dian.gov.co/WcfDianCustomerServices.svc?xsd=xsd3, xsd5, xsd6) — no son
// una traducción del anexo técnico, son el contrato real tal cual lo expone el servicio.
//
// Los tags xml no califican namespace: encoding/xml empareja por nombre local cuando el
// tag no especifica uno, y aquí no hay colisión posible de nombres entre namespaces.

type stringArray struct {
	Items []string `xml:"string"`
}

// DianResponse es el resultado de validar un documento (SendBillSyncResult,
// GetStatusResult, cada elemento de GetStatusZipResult).
type DianResponse struct {
	ErrorMessage      *stringArray `xml:"ErrorMessage"`
	IsValid           bool         `xml:"IsValid"`
	StatusCode        string       `xml:"StatusCode"`
	StatusDescription string       `xml:"StatusDescription"`
	StatusMessage     string       `xml:"StatusMessage"`
	XmlBase64Bytes    []byte       `xml:"XmlBase64Bytes"`
	XmlBytes          []byte       `xml:"XmlBytes"`
	XmlDocumentKey    string       `xml:"XmlDocumentKey"`
	XmlFileName       string       `xml:"XmlFileName"`
}

// XMLParamsResponseTrackId es un error de validación inicial del ZIP (no llegó a ponerse
// en cola de validación).
type XMLParamsResponseTrackId struct {
	DocumentKey      string `xml:"DocumentKey"`
	ProcessedMessage string `xml:"ProcessedMessage"`
	SenderCode       string `xml:"SenderCode"`
	Success          bool   `xml:"Success"`
	XmlFileName      string `xml:"XmlFileName"`
}

// UploadDocumentResponse es el resultado de SendBillAsync/SendTestSetAsync/
// SendBillAttachmentAsync: o hay errores iniciales (ErrorMessageList) o hay un ZipKey para
// consultar después con GetStatusZip.
type UploadDocumentResponse struct {
	ErrorMessageList *struct {
		Items []XMLParamsResponseTrackId `xml:"XmlParamsResponseTrackId"`
	} `xml:"ErrorMessageList"`
	ZipKey string `xml:"ZipKey"`
}
