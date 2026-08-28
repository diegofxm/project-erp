package soap

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// canned simulates a complete SOAP response from the DIAN — the real client
// (TestBuildEnvelope_SignatureVerifies in envelope_test.go) already proves that the REQUEST is
// signed correctly; these tests cover the other side: that each response is parsed correctly.

const getNumberingRangeResponseXML = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
  <s:Body>
    <GetNumberingRangeResponse xmlns="http://wcf.dian.colombia">
      <GetNumberingRangeResult xmlns:b="http://schemas.datacontract.org/2004/07/NumberRangeResponseList"
                               xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
        <b:OperationCode>0</b:OperationCode>
        <b:OperationDescription>Proceso Exitoso</b:OperationDescription>
        <b:ResponseList xmlns:c="http://schemas.datacontract.org/2004/07/NumberRangeResponse">
          <c:NumberRangeResponse>
            <c:ResolutionNumber>18760000001</c:ResolutionNumber>
            <c:ResolutionDate>2019-01-19T00:00:00</c:ResolutionDate>
            <c:Prefix>SETP</c:Prefix>
            <c:FromNumber>990000000</c:FromNumber>
            <c:ToNumber>995000000</c:ToNumber>
            <c:ValidDateFrom>2019-01-19T00:00:00</c:ValidDateFrom>
            <c:ValidDateTo>2030-01-19T00:00:00</c:ValidDateTo>
            <c:TechnicalKey>fc8eac422eba16e22ffd8c6f94b3f40a6e38162c</c:TechnicalKey>
          </c:NumberRangeResponse>
        </b:ResponseList>
      </GetNumberingRangeResult>
    </GetNumberingRangeResponse>
  </s:Body>
</s:Envelope>`

// TestGetNumberingRange_ParsesResponse confirms that GetNumberingRange correctly parses a
// response with one range — including the fields the CUFE needs (TechnicalKey) and the fields
// needed to record the range in the DB (Prefix, From/To, dates, resolution).
func TestGetNumberingRange_ParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
		_, _ = w.Write([]byte(getNumberingRangeResponseXML))
	}))
	defer server.Close()

	cert, key := generateTestCert(t)
	c := &Client{BaseURL: server.URL, Cert: cert, Key: key, HTTPClient: server.Client()}

	got, err := c.GetNumberingRange("6382356", "6382356", "12345678-1234-1234-1234-123456789012")
	if err != nil {
		t.Fatalf("GetNumberingRange: %v", err)
	}
	if got.OperationCode != "0" {
		t.Errorf("OperationCode = %q, want %q", got.OperationCode, "0")
	}
	if len(got.ResponseList) != 1 {
		t.Fatalf("len(ResponseList) = %d, want 1", len(got.ResponseList))
	}
	r := got.ResponseList[0]
	if r.ResolutionNumber != "18760000001" {
		t.Errorf("ResolutionNumber = %q, want %q", r.ResolutionNumber, "18760000001")
	}
	if r.Prefix != "SETP" {
		t.Errorf("Prefix = %q, want %q", r.Prefix, "SETP")
	}
	if r.FromNumber != 990000000 {
		t.Errorf("FromNumber = %d, want %d", r.FromNumber, 990000000)
	}
	if r.ToNumber != 995000000 {
		t.Errorf("ToNumber = %d, want %d", r.ToNumber, 995000000)
	}
	if r.TechnicalKey != "fc8eac422eba16e22ffd8c6f94b3f40a6e38162c" {
		t.Errorf("TechnicalKey = %q, want %q", r.TechnicalKey, "fc8eac422eba16e22ffd8c6f94b3f40a6e38162c")
	}
}

// TestGetNumberingRange_EmptyList confirms that a response with an empty list doesn't fail —
// this can happen when the software has no authorized ranges yet or the credentials are new.
func TestGetNumberingRange_EmptyList(t *testing.T) {
	emptyXML := `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
  <s:Body>
    <GetNumberingRangeResponse xmlns="http://wcf.dian.colombia">
      <GetNumberingRangeResult xmlns:b="http://schemas.datacontract.org/2004/07/NumberRangeResponseList">
        <b:OperationCode>0</b:OperationCode>
        <b:OperationDescription>Proceso Exitoso</b:OperationDescription>
        <b:ResponseList/>
      </GetNumberingRangeResult>
    </GetNumberingRangeResponse>
  </s:Body>
</s:Envelope>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(emptyXML))
	}))
	defer server.Close()

	cert, key := generateTestCert(t)
	c := &Client{BaseURL: server.URL, Cert: cert, Key: key, HTTPClient: server.Client()}

	got, err := c.GetNumberingRange("6382356", "6382356", "any-software-id")
	if err != nil {
		t.Fatalf("GetNumberingRange lista vacía: %v", err)
	}
	if len(got.ResponseList) != 0 {
		t.Errorf("len(ResponseList) = %d, want 0", len(got.ResponseList))
	}
}

const sendBillAttachmentAsyncResponseXML = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
  <s:Body>
    <SendBillAttachmentAsyncResponse xmlns="http://wcf.dian.colombia">
      <SendBillAttachmentAsyncResult xmlns:b="http://schemas.datacontract.org/2004/07/"
                                      xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
        <b:ZipKey>a1b2c3d4-e5f6-7890-abcd-ef1234567890</b:ZipKey>
      </SendBillAttachmentAsyncResult>
    </SendBillAttachmentAsyncResponse>
  </s:Body>
</s:Envelope>`

// TestSendBillAttachmentAsync_ParsesResponse confirms SendBillAttachmentAsync sends the same
// fileName/contentFile request shape as SendBillAsync and correctly parses the ZipKey out of
// its own distinct <SendBillAttachmentAsyncResult> response element.
func TestSendBillAttachmentAsync_ParsesResponse(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
		_, _ = w.Write([]byte(sendBillAttachmentAsyncResponseXML))
	}))
	defer server.Close()

	cert, key := generateTestCert(t)
	c := &Client{BaseURL: server.URL, Cert: cert, Key: key, HTTPClient: server.Client()}

	got, err := c.SendBillAttachmentAsync("ad900123456000123456789.zip", []byte("<AttachedDocument/>"))
	if err != nil {
		t.Fatalf("SendBillAttachmentAsync: %v", err)
	}
	if got.ZipKey != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Errorf("ZipKey = %q, want %q", got.ZipKey, "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	}
	if !strings.Contains(string(gotBody), "<wcf:SendBillAttachmentAsync>") {
		t.Errorf("request body missing wcf:SendBillAttachmentAsync element: %s", gotBody)
	}
	if !strings.Contains(string(gotBody), "ad900123456000123456789.zip") {
		t.Errorf("request body missing fileName: %s", gotBody)
	}
}

const sendEventUpdateStatusResponseXML = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
  <s:Body>
    <SendEventUpdateStatusResponse xmlns="http://wcf.dian.colombia">
      <SendEventUpdateStatusResult xmlns:b="http://schemas.datacontract.org/2004/07/"
                                    xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
        <b:IsValid>true</b:IsValid>
        <b:StatusCode>00</b:StatusCode>
        <b:StatusDescription>La Notificación ha sido autorizada</b:StatusDescription>
      </SendEventUpdateStatusResult>
    </SendEventUpdateStatusResponse>
  </s:Body>
</s:Envelope>`

// TestSendEventUpdateStatus_ParsesResponse confirms SendEventUpdateStatus sends the single
// contentFile parameter (no fileName, same shape as SendNominaSync) and correctly parses its
// own distinct <SendEventUpdateStatusResult> response element.
func TestSendEventUpdateStatus_ParsesResponse(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
		_, _ = w.Write([]byte(sendEventUpdateStatusResponseXML))
	}))
	defer server.Close()

	cert, key := generateTestCert(t)
	c := &Client{BaseURL: server.URL, Cert: cert, Key: key, HTTPClient: server.Client()}

	got, err := c.SendEventUpdateStatus([]byte("<ApplicationResponse/>"))
	if err != nil {
		t.Fatalf("SendEventUpdateStatus: %v", err)
	}
	if !got.IsValid {
		t.Errorf("IsValid = %v, want true", got.IsValid)
	}
	if got.StatusCode != "00" {
		t.Errorf("StatusCode = %q, want %q", got.StatusCode, "00")
	}
	if !strings.Contains(string(gotBody), "<wcf:SendEventUpdateStatus>") {
		t.Errorf("request body missing wcf:SendEventUpdateStatus element: %s", gotBody)
	}
}

const getStatusEventResponseXML = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
  <s:Body>
    <GetStatusEventResponse xmlns="http://wcf.dian.colombia">
      <GetStatusEventResult xmlns:b="http://schemas.datacontract.org/2004/07/"
                             xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
        <b:IsValid>true</b:IsValid>
        <b:StatusCode>00</b:StatusCode>
      </GetStatusEventResult>
    </GetStatusEventResponse>
  </s:Body>
</s:Envelope>`

// TestGetStatusEvent_ParsesResponse confirms GetStatusEvent sends trackId (same shape as
// GetStatus) and parses its own <GetStatusEventResult> element.
func TestGetStatusEvent_ParsesResponse(t *testing.T) {
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
		_, _ = w.Write([]byte(getStatusEventResponseXML))
	}))
	defer server.Close()

	cert, key := generateTestCert(t)
	c := &Client{BaseURL: server.URL, Cert: cert, Key: key, HTTPClient: server.Client()}

	got, err := c.GetStatusEvent("some-track-id")
	if err != nil {
		t.Fatalf("GetStatusEvent: %v", err)
	}
	if !got.IsValid {
		t.Errorf("IsValid = %v, want true", got.IsValid)
	}
	if !strings.Contains(string(gotBody), "<wcf:trackId>some-track-id</wcf:trackId>") {
		t.Errorf("request body missing trackId: %s", gotBody)
	}
}

const getAcquirerResponseXML = `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
  <s:Body>
    <GetAcquirerResponse xmlns="http://wcf.dian.colombia">
      <GetAcquirerResult>
        <Message>OK</Message>
        <StatusCode>0</StatusCode>
        <ReceiverName>Cliente De Prueba</ReceiverName>
        <ReceiverEmail>cliente@prueba.test</ReceiverEmail>
      </GetAcquirerResult>
    </GetAcquirerResponse>
  </s:Body>
</s:Envelope>`

// TestGetAcquirer_ParsesResponse confirms that GetAcquirer parses a real DIAN response — without
// calling the real server, via an httptest.Server that returns the envelope verbatim.
func TestGetAcquirer_ParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body) // drain the signed request, not inspected here
		w.Header().Set("Content-Type", "application/soap+xml; charset=utf-8")
		_, _ = w.Write([]byte(getAcquirerResponseXML))
	}))
	defer server.Close()

	cert, key := generateTestCert(t)
	c := &Client{BaseURL: server.URL, Cert: cert, Key: key, HTTPClient: server.Client()}

	got, err := c.GetAcquirer("31", "900373076")
	if err != nil {
		t.Fatalf("GetAcquirer: %v", err)
	}
	if got.ReceiverName != "Cliente De Prueba" {
		t.Errorf("ReceiverName = %q, want %q", got.ReceiverName, "Cliente De Prueba")
	}
	if got.ReceiverEmail != "cliente@prueba.test" {
		t.Errorf("ReceiverEmail = %q, want %q", got.ReceiverEmail, "cliente@prueba.test")
	}
	if got.StatusCode != "0" {
		t.Errorf("StatusCode = %q, want %q", got.StatusCode, "0")
	}
}

// TestGetAcquirer_NotFoundIsNotAnError confirms the normal/expected case for most ID numbers:
// the DIAN responds without an ErrorMessage/Fault but with empty fields — this must not be
// treated as a Go error, only as "no record found" (non-blocking, see the doc comment on
// Client.GetAcquirer).
func TestGetAcquirer_NotFoundIsNotAnError(t *testing.T) {
	emptyXML := `<?xml version="1.0" encoding="UTF-8"?>
<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">
  <s:Body>
    <GetAcquirerResponse xmlns="http://wcf.dian.colombia">
      <GetAcquirerResult></GetAcquirerResult>
    </GetAcquirerResponse>
  </s:Body>
</s:Envelope>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(emptyXML))
	}))
	defer server.Close()

	cert, key := generateTestCert(t)
	c := &Client{BaseURL: server.URL, Cert: cert, Key: key, HTTPClient: server.Client()}

	got, err := c.GetAcquirer("13", "1122334455")
	if err != nil {
		t.Fatalf("GetAcquirer no debería fallar cuando no hay registro: %v", err)
	}
	if got.ReceiverName != "" || got.ReceiverEmail != "" {
		t.Errorf("se esperaban campos vacíos, got %+v", got)
	}
}

// TestGetAcquirer_HTTP404WithValidBodyIsNotAnError reproduces EXACTLY a real response confirmed
// against the DIAN certification environment (2026-06-29): when the acquirer does not exist,
// the DIAN responds with HTTP status 404 but a perfectly valid SOAP body — StatusCode "404"
// INSIDE the body, Message "El adquirente No existe en la base de datos" ("The acquirer does
// not exist in the database"), with no soap:Fault. Before this test, client.call() rejected
// any status != 200 without inspecting the body, so this normal/expected case was being
// reported as a Go error.
func TestGetAcquirer_HTTP404WithValidBodyIsNotAnError(t *testing.T) {
	realDianNotFoundXML := `<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://www.w3.org/2005/08/addressing">
  <s:Body>
    <GetAcquirerResponse xmlns="http://wcf.dian.colombia">
      <GetAcquirerResult xmlns:b="http://schemas.datacontract.org/2004/07/Gosocket.Dian.Services.Utils.Common" xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
        <b:Message>El adquirente No existe en la base de datos</b:Message>
        <b:ReceiverEmail i:nil="true"/>
        <b:ReceiverName i:nil="true"/>
        <b:StatusCode>404</b:StatusCode>
      </GetAcquirerResult>
    </GetAcquirerResponse>
  </s:Body>
</s:Envelope>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(realDianNotFoundXML))
	}))
	defer server.Close()

	cert, key := generateTestCert(t)
	c := &Client{BaseURL: server.URL, Cert: cert, Key: key, HTTPClient: server.Client()}

	got, err := c.GetAcquirer("13", "6382356")
	if err != nil {
		t.Fatalf("GetAcquirer no debería fallar con HTTP 404 si el body es válido: %v", err)
	}
	if got.StatusCode != "404" {
		t.Errorf("StatusCode = %q, want %q", got.StatusCode, "404")
	}
	if got.Message != "El adquirente No existe en la base de datos" {
		t.Errorf("Message = %q, no coincide con la respuesta real", got.Message)
	}
	if got.ReceiverName != "" || got.ReceiverEmail != "" {
		t.Errorf("se esperaban campos vacíos, got %+v", got)
	}
}
