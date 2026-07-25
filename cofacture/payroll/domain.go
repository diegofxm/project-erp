// Package payroll construye, firma y envía documentos de Nómina Individual Electrónica
// (NominaIndividual) al servicio web de la DIAN (SendNominaSync).
//
// No genera sus propios clientes SOAP — delega en cofacture/soap, cuyo Client ya sabe
// hablar con WcfDianCustomerServices. Sí contiene su propio builder XML porque la
// estructura de NominaIndividual no es UBL 2.1 y no comparte los helpers de builder/.
package payroll

// Nomina reúne todos los datos necesarios para generar, firmar y enviar
// un NominaIndividual o NominaIndividualDeAjuste.
type Nomina struct {
	// Secuencia y tipo.
	Consecutivo string // e.g. "1"
	Prefijo     string // e.g. "NE"
	Numero      string // número completo = Prefijo + Consecutivo

	TipoXML string // "102" NominaIndividual, "103" Ajuste-Reemplazar, "104" Ajuste-Eliminar

	// Ambiente DIAN: "1" producción, "2" habilitación.
	Ambiente string

	// Fechas y horas del periodo y de generación del documento.
	FechaGen               string // "2024-01-31"
	HoraGen                string // "09:00:00-05:00"
	FechaIngreso           string // "2020-01-01"
	FechaRetiro            string // "" vacío = no aplica
	FechaLiquidacionInicio string // "2024-01-01"
	FechaLiquidacionFin    string // "2024-01-31"
	TiempoLaborado         int    // días laborados en el periodo

	// Lugar de generación.
	Pais              string // "CO"
	DepartamentoEstado string // código DIAN, e.g. "11"
	MunicipioCiudad   string // código DIAN, e.g. "001"
	Idioma            string // ISO 639-1, e.g. "es"

	// Metadatos del comprobante.
	PeriodoNomina int    // 1=Semanal 2=Decenal 3=Quincenal 4=Mensual ...
	TipoMoneda    string // "COP"
	TRM           string // "1.00" (tasa de cambio; "1.00" para COP→COP)
	Notas         string // texto libre opcional

	// Software propio / PT.
	SoftwareID string // UUID registrado en la DIAN
	PIN        string // PIN numérico del software

	// Proveedor XML (quien genera el documento — empresa o PT).
	// Para personas naturales usar PrimerApellido/PrimerNombre; para empresas RazonSocial.
	ProveedorNIT         string
	ProveedorDV          string
	ProveedorRazonSocial string
	ProveedorApellido1   string
	ProveedorApellido2   string
	ProveedorNombre1     string
	ProveedorNombre2     string // OtrosNombres

	// Fechas de pago de la nómina.
	FechasPago []string // []"2024-01-31"

	Empleador   Empleador
	Trabajador  Trabajador
	Pago        Pago
	Devengados  Devengados
	Deducciones Deducciones

	// Totales (si son cero se calculan a partir de Devengados/Deducciones en Build).
	Redondeo          float64
	DevengadosTotal   float64
	DeduccionesTotal  float64
	ComprobanteTotal  float64
}

// Empleador identifica a la empresa o persona que paga la nómina.
// Para personas naturales usar PrimerApellido/PrimerNombre/etc.; para empresas usar RazonSocial.
type Empleador struct {
	RazonSocial     string
	PrimerApellido  string
	SegundoApellido string
	PrimerNombre    string
	OtrosNombres    string
	NIT             string
	DV              string
	Pais            string // "CO"
	Departamento    string // código DIAN
	Municipio       string // código DIAN
	Direccion       string
}

// Trabajador identifica al empleado.
type Trabajador struct {
	TipoTrabajador    string // "01" empleado, etc.
	SubTipoTrabajador string // "00" no aplica, "01" dependiente, etc.
	AltoRiesgoPension bool
	TipoDocumento     string // "13" CC, "31" NIT, etc.
	NumeroDocumento   string
	PrimerApellido    string
	SegundoApellido   string
	PrimerNombre      string
	OtrosNombres      string
	LugarPais         string // "CO"
	LugarDepartamento string // código DIAN
	LugarMunicipio    string // código DIAN
	LugarDireccion    string
	SalarioIntegral   bool
	TipoContrato      string  // "1" fijo, "2" indefinido, "3" aprendizaje, "4" practicante
	Sueldo            float64 // salario base mensual
	CodigoTrabajador  string  // código interno del empleado
}

// Pago describe el método de pago de la nómina al trabajador.
type Pago struct {
	Forma        string // "1" contado, "2" crédito
	Metodo       string // "10" efectivo, "42" transferencia bancaria, etc.
	Banco        string
	TipoCuenta   string // "AHORRO", "CORRIENTE"
	NumeroCuenta string
}

// Devengados agrupa los conceptos de ingreso del trabajador en el periodo.
type Devengados struct {
	Basico     Basico
	Transporte *Transporte // nil = no aplica
}

// Basico es el sueldo básico del periodo (obligatorio).
type Basico struct {
	DiasTrabajados  int
	SueldoTrabajado float64
}

// Transporte cubre el auxilio de transporte y viáticos.
type Transporte struct {
	AuxilioTransporte float64
	ViaticoManuAlojS  float64 // salariales
	ViaticoManuAlojNS float64 // no salariales
}

// Deducciones agrupa los descuentos al trabajador en el periodo.
type Deducciones struct {
	Salud           *DeduccionPct // nil = no aplica
	FondoPension    *DeduccionPct // nil = no aplica
	FondoSP         *FondoSP     // nil = no aplica (solo para salarios > 4 SMLMV)
	RetencionFuente float64
}

// DeduccionPct es una deducción definida por porcentaje y monto.
type DeduccionPct struct {
	Porcentaje float64
	Deduccion  float64
}

// FondoSP es la deducción al Fondo de Solidaridad Pensional.
type FondoSP struct {
	Porcentaje    float64
	DeduccionSP   float64
	PorcentajeSub float64
	DeduccionSub  float64
}
