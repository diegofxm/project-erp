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

// NumberRangeResponse es un rango de numeración individual devuelto por GetNumberingRange
// (namespace http://schemas.datacontract.org/2004/07/NumberRangeResponse en el WSDL real).
// Fechas en formato string tal como las devuelve la DIAN ("2019-01-19T00:00:00") — el
// llamador convierte según necesite.
type NumberRangeResponse struct {
	ResolutionNumber string `xml:"ResolutionNumber"`
	ResolutionDate   string `xml:"ResolutionDate"`
	Prefix           string `xml:"Prefix"`
	FromNumber       int64  `xml:"FromNumber"`
	ToNumber         int64  `xml:"ToNumber"`
	ValidDateFrom    string `xml:"ValidDateFrom"`
	ValidDateTo      string `xml:"ValidDateTo"`
	TechnicalKey     string `xml:"TechnicalKey"`
}

// NumberRangeResponseList es el resultado completo de GetNumberingRange — una lista de
// rangos autorizados para el emisor + software dados (namespace
// http://schemas.datacontract.org/2004/07/NumberRangeResponseList).
// OperationCode "0" indica éxito; cualquier otro código indica error de la DIAN.
type NumberRangeResponseList struct {
	OperationCode        string                `xml:"OperationCode"`
	OperationDescription string                `xml:"OperationDescription"`
	ResponseList         []NumberRangeResponse `xml:"ResponseList>NumberRangeResponse"`
}

// AcquirerResponse es el resultado de GetAcquirer (namespace
// Gosocket.Dian.Services.Utils.Common en el WSDL real) — consulta el registro de
// intercambio/notificación de la DIAN para un tercero dado, no un RUT completo: solo trae
// ReceiverName/ReceiverEmail si ese identificationNumber ya tiene un nombre/correo registrados
// para recibir documentos electrónicos. StatusCode/Message vacíos (o un StatusCode de error) es
// el resultado normal y esperado para la mayoría de cédulas — no significa que la consulta
// falló, ver docs/apidian-architecture.md sección 9.41.
type AcquirerResponse struct {
	Message       string `xml:"Message"`
	StatusCode    string `xml:"StatusCode"`
	ReceiverName  string `xml:"ReceiverName"`
	ReceiverEmail string `xml:"ReceiverEmail"`
}
