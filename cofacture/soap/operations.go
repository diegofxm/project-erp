package soap

import (
	"encoding/base64"

	"github.com/beevik/etree"
)

// SendTestSetAsync envía un ZIP del set de pruebas de habilitación. Devuelve el
// UploadDocumentResponse — si ZipKey viene vacío, revisar ErrorMessageList (la DIAN rechazó
// el ZIP en validaciones iniciales, antes de encolarlo). El resultado de validación real se
// consulta después con GetStatusZip(ZipKey).
func (c *Client) SendTestSetAsync(fileName string, content []byte, testSetID string) (*UploadDocumentResponse, error) {
	var result struct {
		Result UploadDocumentResponse `xml:"SendTestSetAsyncResult"`
	}
	err := c.call("SendTestSetAsync", func(body *etree.Element) {
		el := body.CreateElement("wcf:SendTestSetAsync")
		el.CreateElement("wcf:fileName").SetText(fileName)
		el.CreateElement("wcf:contentFile").SetText(base64.StdEncoding.EncodeToString(content))
		el.CreateElement("wcf:testSetId").SetText(testSetID)
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// SendBillSync envía un ZIP con un único documento UBL y devuelve el resultado de
// validación de inmediato (proceso síncrono, sección 7.10 del Anexo Técnico 1.9) — a
// diferencia de SendTestSetAsync/SendBillAsync, no hay ZipKey ni paso posterior de consulta;
// el propio DianResponse ya trae StatusCode/IsValid.
func (c *Client) SendBillSync(fileName string, content []byte) (*DianResponse, error) {
	var result struct {
		Result DianResponse `xml:"SendBillSyncResult"`
	}
	err := c.call("SendBillSync", func(body *etree.Element) {
		el := body.CreateElement("wcf:SendBillSync")
		el.CreateElement("wcf:fileName").SetText(fileName)
		el.CreateElement("wcf:contentFile").SetText(base64.StdEncoding.EncodeToString(content))
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// SendBillAsync envía un ZIP con uno o varios documentos UBL de forma asíncrona (sección 7.8
// del Anexo Técnico 1.9) — mismo patrón que SendTestSetAsync (devuelve ZipKey, el resultado
// real se consulta después con GetStatusZip), pero sin testSetId: es para envío normal, no
// para el set de pruebas de habilitación.
func (c *Client) SendBillAsync(fileName string, content []byte) (*UploadDocumentResponse, error) {
	var result struct {
		Result UploadDocumentResponse `xml:"SendBillAsyncResult"`
	}
	err := c.call("SendBillAsync", func(body *etree.Element) {
		el := body.CreateElement("wcf:SendBillAsync")
		el.CreateElement("wcf:fileName").SetText(fileName)
		el.CreateElement("wcf:contentFile").SetText(base64.StdEncoding.EncodeToString(content))
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// GetStatusZip consulta el resultado de validación de un ZIP enviado por SendBillAsync o
// SendTestSetAsync. Un ZIP puede contener varios documentos, por eso devuelve un slice.
func (c *Client) GetStatusZip(trackID string) ([]DianResponse, error) {
	var result struct {
		Result struct {
			Items []DianResponse `xml:"DianResponse"`
		} `xml:"GetStatusZipResult"`
	}
	err := c.call("GetStatusZip", func(body *etree.Element) {
		el := body.CreateElement("wcf:GetStatusZip")
		el.CreateElement("wcf:trackId").SetText(trackID)
	}, &result)
	if err != nil {
		return nil, err
	}
	return result.Result.Items, nil
}

// GetStatus consulta el resultado de validación de un único documento enviado por
// SendBillSync.
func (c *Client) GetStatus(trackID string) (*DianResponse, error) {
	var result struct {
		Result DianResponse `xml:"GetStatusResult"`
	}
	err := c.call("GetStatus", func(body *etree.Element) {
		el := body.CreateElement("wcf:GetStatus")
		el.CreateElement("wcf:trackId").SetText(trackID)
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// GetNumberingRange consulta los rangos de numeración autorizados por la DIAN para un
// emisor + software dados. accountCode y accountCodeT son el NIT del emisor (en integración
// directa son iguales); softwareCode es el UUID del software registrado.
// Devuelve todos los rangos activos e históricos que la DIAN tiene para ese par
// emisor/software, cada uno con su resolución, prefijo, from/to, fechas de vigencia y la
// clave técnica (TechnicalKey) necesaria para calcular el CUFE.
func (c *Client) GetNumberingRange(accountCode, accountCodeT, softwareCode string) (*NumberRangeResponseList, error) {
	var result struct {
		Result NumberRangeResponseList `xml:"GetNumberingRangeResult"`
	}
	err := c.call("GetNumberingRange", func(body *etree.Element) {
		el := body.CreateElement("wcf:GetNumberingRange")
		el.CreateElement("wcf:accountCode").SetText(accountCode)
		el.CreateElement("wcf:accountCodeT").SetText(accountCodeT)
		el.CreateElement("wcf:softwareCode").SetText(softwareCode)
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// SendNominaSync envía un ZIP con un NominaIndividual (o NominaIndividualDeAjuste) firmado
// y devuelve el resultado de validación de forma síncrona (Anexo Técnico Nómina, sección 9.7).
// A diferencia de SendBillSync, solo recibe contentFile (sin fileName).
func (c *Client) SendNominaSync(content []byte) (*DianResponse, error) {
	var result struct {
		Result DianResponse `xml:"SendNominaSyncResult"`
	}
	err := c.call("SendNominaSync", func(body *etree.Element) {
		el := body.CreateElement("wcf:SendNominaSync")
		el.CreateElement("wcf:contentFile").SetText(base64.StdEncoding.EncodeToString(content))
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// SendNominaSyncTestSet envía un ZIP de nómina al set de pruebas de habilitación.
// Combina la lógica de SendNominaSync con el testSetId requerido para habilitación.
func (c *Client) SendNominaSyncTestSet(content []byte, testSetID string) (*DianResponse, error) {
	var result struct {
		Result DianResponse `xml:"SendNominaSyncResult"`
	}
	err := c.call("SendNominaSync", func(body *etree.Element) {
		el := body.CreateElement("wcf:SendNominaSync")
		el.CreateElement("wcf:contentFile").SetText(base64.StdEncoding.EncodeToString(content))
		el.CreateElement("wcf:testSetId").SetText(testSetID)
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// GetAcquirer consulta el registro de intercambio/notificación de la DIAN para un tercero —
// ver AcquirerResponse. Ayuda opcional al capturar un NIT, nunca bloqueante: un resultado vacío
// (sin ReceiverName/ReceiverEmail) es normal y esperado para la mayoría de identificaciones.
func (c *Client) GetAcquirer(identificationType, identificationNumber string) (*AcquirerResponse, error) {
	var result struct {
		Result AcquirerResponse `xml:"GetAcquirerResult"`
	}
	err := c.call("GetAcquirer", func(body *etree.Element) {
		el := body.CreateElement("wcf:GetAcquirer")
		el.CreateElement("wcf:identificationType").SetText(identificationType)
		el.CreateElement("wcf:identificationNumber").SetText(identificationNumber)
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result.Result, nil
}
