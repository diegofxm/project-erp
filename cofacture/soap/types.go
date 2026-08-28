package soap

// Types that mirror the real schemas downloaded from the certification-environment WSDL
// (https://vpfe-hab.dian.gov.co/WcfDianCustomerServices.svc?xsd=xsd3, xsd5, xsd6) — they are
// not a translation of the technical annex, they are the service's actual contract as exposed.
//
// The xml tags do not qualify a namespace: encoding/xml matches by local name when the tag
// specifies none, and there is no possible name collision across namespaces here.

type stringArray struct {
	Items []string `xml:"string"`
}

// DianResponse is the result of validating a document (SendBillSyncResult,
// GetStatusResult, each element of GetStatusZipResult).
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

// XMLParamsResponseTrackId is an initial ZIP validation error (it never made it into the
// validation queue).
type XMLParamsResponseTrackId struct {
	DocumentKey      string `xml:"DocumentKey"`
	ProcessedMessage string `xml:"ProcessedMessage"`
	SenderCode       string `xml:"SenderCode"`
	Success          bool   `xml:"Success"`
	XmlFileName      string `xml:"XmlFileName"`
}

// UploadDocumentResponse is the result of SendBillAsync/SendTestSetAsync/
// SendBillAttachmentAsync: either there are initial errors (ErrorMessageList) or there is a
// ZipKey to query later with GetStatusZip.
type UploadDocumentResponse struct {
	ErrorMessageList *struct {
		Items []XMLParamsResponseTrackId `xml:"XmlParamsResponseTrackId"`
	} `xml:"ErrorMessageList"`
	ZipKey string `xml:"ZipKey"`
}

// NumberRangeResponse is a single numbering range returned by GetNumberingRange (namespace
// http://schemas.datacontract.org/2004/07/NumberRangeResponse in the real WSDL). Dates are
// plain strings exactly as DIAN returns them ("2019-01-19T00:00:00") — the caller converts as
// needed.
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

// NumberRangeResponseList is the full result of GetNumberingRange — a list of ranges
// authorized for the given issuer + software pair (namespace
// http://schemas.datacontract.org/2004/07/NumberRangeResponseList). OperationCode "0" means
// success; any other code indicates a DIAN-side error.
type NumberRangeResponseList struct {
	OperationCode        string                `xml:"OperationCode"`
	OperationDescription string                `xml:"OperationDescription"`
	ResponseList         []NumberRangeResponse `xml:"ResponseList>NumberRangeResponse"`
}

// AcquirerResponse is the result of GetAcquirer (namespace
// Gosocket.Dian.Services.Utils.Common in the real WSDL) — it queries DIAN's exchange/
// notification registry for a given third party, not a full RUT lookup: it only returns
// ReceiverName/ReceiverEmail if that identificationNumber already has a name/email registered
// to receive electronic documents. Empty StatusCode/Message (or an error StatusCode) is the
// normal, expected result for most national IDs — it does not mean the query failed.
type AcquirerResponse struct {
	Message       string `xml:"Message"`
	StatusCode    string `xml:"StatusCode"`
	ReceiverName  string `xml:"ReceiverName"`
	ReceiverEmail string `xml:"ReceiverEmail"`
}
