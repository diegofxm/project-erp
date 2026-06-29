package soap

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/beevik/etree"
)

// URLs reales del servicio (verificadas contra el WSDL descargado directamente, no
// inferidas del anexo técnico).
const (
	HabilitacionURL = "https://vpfe-hab.dian.gov.co/WcfDianCustomerServices.svc"
	ProduccionURL   = "https://vpfe.dian.gov.co/WcfDianCustomerServices.svc"
)

// Client es el cliente SOAP de los webservices de recepción de la DIAN. No requiere TLS
// mutuo (la política real del WSDL trae RequireClientCertificate="false") — Cert/Key se
// usan únicamente para firmar el header WS-Security wsa:To.
type Client struct {
	BaseURL    string
	Cert       *x509.Certificate
	Key        *rsa.PrivateKey
	HTTPClient *http.Client
}

// New crea un Client. baseURL normalmente es HabilitacionURL o ProduccionURL.
func New(baseURL string, cert *x509.Certificate, key *rsa.PrivateKey) *Client {
	return &Client{
		BaseURL:    baseURL,
		Cert:       cert,
		Key:        key,
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// Fault es un soap:Fault devuelto por la DIAN.
type Fault struct {
	Code   string
	Reason string
}

func (f *Fault) Error() string {
	return fmt.Sprintf("soap fault [%s]: %s", f.Code, f.Reason)
}

type soapFaultXML struct {
	Code struct {
		Value string `xml:"Value"`
	} `xml:"Code"`
	Reason struct {
		Text string `xml:"Text"`
	} `xml:"Reason"`
}

// call envía action con el cuerpo que construya bodyBuilder y deja la respuesta cruda
// (el contenido de soap:Body, ya verificado que no es un Fault) en out vía xml.Unmarshal.
// out debe ser un puntero a un struct con el tag xml correspondiente al elemento
// "{action}Response" esperado dentro de soap:Body.
func (c *Client) call(action string, bodyBuilder func(body *etree.Element), out any) error {
	doc, err := c.buildEnvelope(action, bodyBuilder)
	if err != nil {
		return err
	}
	reqXML, err := doc.WriteToBytes()
	if err != nil {
		return fmt.Errorf("soap: serializar request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.BaseURL, bytes.NewReader(reqXML))
	if err != nil {
		return fmt.Errorf("soap: construir request HTTP: %w", err)
	}
	httpReq.Header.Set("Content-Type", fmt.Sprintf(`application/soap+xml; charset=utf-8; action="%s%s"`, actionBase, action))

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("soap: enviar request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("soap: leer respuesta: %w", err)
	}

	var envelope struct {
		Body struct {
			Fault   *soapFaultXML `xml:"http://www.w3.org/2003/05/soap-envelope Fault"`
			Content []byte        `xml:",innerxml"`
		} `xml:"http://www.w3.org/2003/05/soap-envelope Body"`
	}
	if err := xml.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("soap: parsear respuesta (HTTP %d, headers %v, %d bytes): %w\n%s", resp.StatusCode, resp.Header, len(respBody), err, respBody)
	}
	if envelope.Body.Fault != nil {
		return &Fault{Code: envelope.Body.Fault.Code.Value, Reason: envelope.Body.Fault.Reason.Text}
	}

	// El HTTP status NO es la señal confiable de éxito/error en este servicio — confirmado
	// contra la DIAN real: GetAcquirer responde HTTP 404 con un cuerpo SOAP perfectamente
	// válido cuando el adquiriente no existe (StatusCode "404" DENTRO del body, Message "El
	// adquirente No existe en la base de datos", sin soap:Fault) — ese es el resultado normal
	// y esperado para la mayoría de identificaciones, no un error de transporte. Por eso se
	// intenta parsear el body en out PRIMERO, sin importar el status; solo si el body no es la
	// respuesta esperada se usa el status para dar un mensaje de error más claro.
	if err := xml.Unmarshal(envelope.Body.Content, out); err != nil {
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("soap: HTTP %d sin soap:Fault explícito:\n%s", resp.StatusCode, respBody)
		}
		return fmt.Errorf("soap: parsear %sResponse: %w", action, err)
	}
	return nil
}
