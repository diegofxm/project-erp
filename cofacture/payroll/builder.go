package payroll

import (
	"fmt"
	"strconv"

	"github.com/beevik/etree"
	ubl "github.com/diegofxm/cofacture/xml"
)

const (
	nsNomina = "dian:gov:co:facturaelectronica:NominaIndividual"
	schemaLocationNomina = "dian:gov:co:facturaelectronica:NominaIndividual NominaIndividualElectronicaXSD.xsd"

	// EncripCUNE es el valor fijo del atributo InformacionGeneral/@EncripCUNE.
	EncripCUNE = "CUNE-SHA384"
	// Version es el valor fijo de InformacionGeneral/@Version.
	Version = "V1.0: Documento Soporte de Pago de Nómina Electrónica"
)

// Build construye el árbol XML de un NominaIndividual ya con CUNE y CodigoQR calculados.
// El documento todavía no está firmado — llamar builder.SignaturePlaceholder + signer.Sign
// después de esta función, igual que el pipeline de facturas.
//
// El caller es responsable de calcular:
//   - n.DevengadosTotal / n.DeduccionesTotal / n.ComprobanteTotal (si no vienen calculados)
//   - El CUNE vía Cune() y asignarlo en n antes de llamar Build
//   - El SoftwareSC vía securitycode.Compute(softwareID, pin, numDoc) y asignarlo en n
//   - El CodigoQR vía qr.URL(ambiente, cune) y pasarlo como parámetro
func Build(n Nomina, cune, softwareSC, codigoQR string) (*etree.Document, error) {
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8" standalone="no"`)

	root := doc.CreateElement("NominaIndividual")
	root.CreateAttr("xmlns", nsNomina)
	root.CreateAttr("xmlns:xs", ubl.NSXsi)
	root.CreateAttr("xmlns:ds", ubl.NSDs)
	root.CreateAttr("xmlns:ext", ubl.NSExt)
	root.CreateAttr("xmlns:xades", ubl.NSXades)
	root.CreateAttr("xmlns:xades141", ubl.NSXades141)
	root.CreateAttr("xmlns:xsi", ubl.NSXsi)
	root.CreateAttr("SchemaLocation", "")
	root.CreateAttr("xsi:schemaLocation", schemaLocationNomina)

	// ext:UBLExtensions — solo un ext:UBLExtension vacío reservado para la firma.
	// (NominaIndividual no tiene DianExtensions: los equivalentes son ProveedorXML, CodigoQR, etc.)
	root.CreateElement("ext:UBLExtensions").
		CreateElement("ext:UBLExtension").
		CreateElement("ext:ExtensionContent")

	// Novedad (siempre false para NominaIndividual normal — true solo para novedades de ajuste).
	novedad := root.CreateElement("Novedad")
	novedad.CreateAttr("CUNENov", "")
	novedad.SetText("false")

	// Periodo
	fecRetiro := n.FechaRetiro
	if fecRetiro == "" {
		fecRetiro = "9999-12-31"
	}
	periodo := root.CreateElement("Periodo")
	periodo.CreateAttr("FechaIngreso", n.FechaIngreso)
	periodo.CreateAttr("FechaRetiro", fecRetiro)
	periodo.CreateAttr("FechaLiquidacionInicio", n.FechaLiquidacionInicio)
	periodo.CreateAttr("FechaLiquidacionFin", n.FechaLiquidacionFin)
	periodo.CreateAttr("TiempoLaborado", strconv.Itoa(n.TiempoLaborado))
	periodo.CreateAttr("FechaGen", n.FechaGen)

	// NumeroSecuenciaXML
	numSeq := root.CreateElement("NumeroSecuenciaXML")
	numSeq.CreateAttr("CodigoTrabajador", n.Trabajador.CodigoTrabajador)
	numSeq.CreateAttr("Prefijo", n.Prefijo)
	numSeq.CreateAttr("Consecutivo", n.Consecutivo)
	numSeq.CreateAttr("Numero", n.Numero)

	// LugarGeneracionXML
	lugar := root.CreateElement("LugarGeneracionXML")
	lugar.CreateAttr("Pais", n.Pais)
	lugar.CreateAttr("DepartamentoEstado", n.DepartamentoEstado)
	lugar.CreateAttr("MunicipioCiudad", n.MunicipioCiudad)
	lugar.CreateAttr("Idioma", n.Idioma)

	// ProveedorXML
	proveedor := root.CreateElement("ProveedorXML")
	proveedor.CreateAttr("RazonSocial", n.ProveedorRazonSocial)
	proveedor.CreateAttr("PrimerApellido", n.ProveedorApellido1)
	proveedor.CreateAttr("SegundoApellido", n.ProveedorApellido2)
	proveedor.CreateAttr("PrimerNombre", n.ProveedorNombre1)
	proveedor.CreateAttr("OtrosNombres", n.ProveedorNombre2)
	proveedor.CreateAttr("NIT", n.ProveedorNIT)
	proveedor.CreateAttr("DV", n.ProveedorDV)
	proveedor.CreateAttr("SoftwareID", n.SoftwareID)
	proveedor.CreateAttr("SoftwareSC", softwareSC)

	// CodigoQR
	root.CreateElement("CodigoQR").SetText(codigoQR)

	// InformacionGeneral
	info := root.CreateElement("InformacionGeneral")
	info.CreateAttr("Version", Version)
	info.CreateAttr("Ambiente", n.Ambiente)
	info.CreateAttr("TipoXML", n.TipoXML)
	info.CreateAttr("CUNE", cune)
	info.CreateAttr("EncripCUNE", EncripCUNE)
	info.CreateAttr("FechaGen", n.FechaGen)
	info.CreateAttr("HoraGen", n.HoraGen)
	info.CreateAttr("PeriodoNomina", strconv.Itoa(n.PeriodoNomina))
	info.CreateAttr("TipoMoneda", n.TipoMoneda)
	info.CreateAttr("TRM", n.TRM)

	// Notas (opcional)
	if n.Notas != "" {
		root.CreateElement("Notas").SetText(n.Notas)
	}

	// Empleador
	emp := root.CreateElement("Empleador")
	emp.CreateAttr("RazonSocial", n.Empleador.RazonSocial)
	emp.CreateAttr("PrimerApellido", n.Empleador.PrimerApellido)
	emp.CreateAttr("SegundoApellido", n.Empleador.SegundoApellido)
	emp.CreateAttr("PrimerNombre", n.Empleador.PrimerNombre)
	emp.CreateAttr("OtrosNombres", n.Empleador.OtrosNombres)
	emp.CreateAttr("NIT", n.Empleador.NIT)
	emp.CreateAttr("DV", n.Empleador.DV)
	emp.CreateAttr("Pais", n.Empleador.Pais)
	emp.CreateAttr("DepartamentoEstado", n.Empleador.Departamento)
	emp.CreateAttr("MunicipioCiudad", n.Empleador.Municipio)
	emp.CreateAttr("Direccion", n.Empleador.Direccion)

	// Trabajador
	trab := root.CreateElement("Trabajador")
	trab.CreateAttr("TipoTrabajador", n.Trabajador.TipoTrabajador)
	trab.CreateAttr("SubTipoTrabajador", n.Trabajador.SubTipoTrabajador)
	trab.CreateAttr("AltoRiesgoPension", boolStr(n.Trabajador.AltoRiesgoPension))
	trab.CreateAttr("TipoDocumento", n.Trabajador.TipoDocumento)
	trab.CreateAttr("NumeroDocumento", n.Trabajador.NumeroDocumento)
	trab.CreateAttr("PrimerApellido", n.Trabajador.PrimerApellido)
	trab.CreateAttr("SegundoApellido", n.Trabajador.SegundoApellido)
	trab.CreateAttr("PrimerNombre", n.Trabajador.PrimerNombre)
	trab.CreateAttr("OtrosNombres", n.Trabajador.OtrosNombres)
	trab.CreateAttr("LugarTrabajoPais", n.Trabajador.LugarPais)
	trab.CreateAttr("LugarTrabajoDepartamentoEstado", n.Trabajador.LugarDepartamento)
	trab.CreateAttr("LugarTrabajoMunicipioCiudad", n.Trabajador.LugarMunicipio)
	trab.CreateAttr("LugarTrabajoDireccion", n.Trabajador.LugarDireccion)
	trab.CreateAttr("SalarioIntegral", boolStr(n.Trabajador.SalarioIntegral))
	trab.CreateAttr("TipoContrato", n.Trabajador.TipoContrato)
	trab.CreateAttr("Sueldo", money(n.Trabajador.Sueldo))
	trab.CreateAttr("CodigoTrabajador", n.Trabajador.CodigoTrabajador)

	// Pago
	pago := root.CreateElement("Pago")
	pago.CreateAttr("Forma", n.Pago.Forma)
	pago.CreateAttr("Metodo", n.Pago.Metodo)
	pago.CreateAttr("Banco", n.Pago.Banco)
	pago.CreateAttr("TipoCuenta", n.Pago.TipoCuenta)
	pago.CreateAttr("NumeroCuenta", n.Pago.NumeroCuenta)

	// FechasPagos
	if len(n.FechasPago) == 0 {
		return nil, fmt.Errorf("payroll: FechasPago no puede estar vacío")
	}
	fechas := root.CreateElement("FechasPagos")
	for _, f := range n.FechasPago {
		fechas.CreateElement("FechaPago").SetText(f)
	}

	// Devengados
	dev := root.CreateElement("Devengados")
	basico := dev.CreateElement("Basico")
	basico.CreateAttr("DiasTrabajados", strconv.Itoa(n.Devengados.Basico.DiasTrabajados))
	basico.CreateAttr("SueldoTrabajado", money(n.Devengados.Basico.SueldoTrabajado))

	if t := n.Devengados.Transporte; t != nil {
		tr := dev.CreateElement("Transporte")
		tr.CreateAttr("AuxilioTransporte", money(t.AuxilioTransporte))
		// ViaticoManuAlojS/NS solo se emiten si son > 0 (NIE072/073 rechaza ceros).
		if t.ViaticoManuAlojS > 0 {
			tr.CreateAttr("ViaticoManuAlojS", money(t.ViaticoManuAlojS))
		}
		if t.ViaticoManuAlojNS > 0 {
			tr.CreateAttr("ViaticoManuAlojNS", money(t.ViaticoManuAlojNS))
		}
	}

	// Deducciones
	ded := root.CreateElement("Deducciones")
	if s := n.Deducciones.Salud; s != nil {
		sal := ded.CreateElement("Salud")
		sal.CreateAttr("Porcentaje", pct(s.Porcentaje))
		sal.CreateAttr("Deduccion", money(s.Deduccion))
	}
	if fp := n.Deducciones.FondoPension; fp != nil {
		pension := ded.CreateElement("FondoPension")
		pension.CreateAttr("Porcentaje", pct(fp.Porcentaje))
		pension.CreateAttr("Deduccion", money(fp.Deduccion))
	}
	if fsp := n.Deducciones.FondoSP; fsp != nil {
		fondoSP := ded.CreateElement("FondoSP")
		fondoSP.CreateAttr("Porcentaje", pct(fsp.Porcentaje))
		fondoSP.CreateAttr("DeduccionSP", money(fsp.DeduccionSP))
		fondoSP.CreateAttr("PorcentajeSub", pct(fsp.PorcentajeSub))
		fondoSP.CreateAttr("DeduccionSub", money(fsp.DeduccionSub))
	}
	if n.Deducciones.RetencionFuente != 0 {
		ded.CreateElement("RetencionFuente").SetText(money(n.Deducciones.RetencionFuente))
	}

	// Totales
	root.CreateElement("Redondeo").SetText(money(n.Redondeo))
	root.CreateElement("DevengadosTotal").SetText(money(n.DevengadosTotal))
	root.CreateElement("DeduccionesTotal").SetText(money(n.DeduccionesTotal))
	root.CreateElement("ComprobanteTotal").SetText(money(n.ComprobanteTotal))

	return doc, nil
}

func money(v float64) string    { return fmt.Sprintf("%.2f", v) }
func pct(v float64) string      { return fmt.Sprintf("%.2f", v) }
func boolStr(b bool) string     {
	if b {
		return "true"
	}
	return "false"
}
