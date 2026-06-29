package soap

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// canned simula una respuesta SOAP completa de la DIAN — el cliente real (TestBuildEnvelope_
// SignatureVerifies en envelope_test.go) ya prueba que el REQUEST se firma correctamente; este
// test cubre el otro lado, que la respuesta se interprete bien.
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

// TestGetAcquirer_ParsesResponse confirma que GetAcquirer interpreta una respuesta real de la
// DIAN — sin llamar al servidor real, vía un httptest.Server que devuelve el sobre tal cual.
func TestGetAcquirer_ParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body) // drena el request firmado, no se inspecciona aquí
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

// TestGetAcquirer_NotFoundIsNotAnError confirma el caso normal/esperado para la mayoría de
// cédulas: la DIAN responde sin ErrorMessage/Fault pero con los campos vacíos — no debe
// tratarse como un error de Go, solo como "sin registro" (ver docs/apidian-architecture.md
// sección 9.41, "no bloqueante").
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

// TestGetAcquirer_HTTP404WithValidBodyIsNotAnError reproduce EXACTAMENTE una respuesta real
// confirmada contra la DIAN de habilitación (2026-06-29): cuando el adquiriente no existe, la
// DIAN responde con status HTTP 404 pero un cuerpo SOAP perfectamente válido — StatusCode
// "404" DENTRO del body, Message "El adquirente No existe en la base de datos", sin
// soap:Fault. Antes de este test, client.call() rechazaba cualquier status != 200 sin
// inspeccionar el body, así que este caso normal/esperado se reportaba como un error de Go.
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
