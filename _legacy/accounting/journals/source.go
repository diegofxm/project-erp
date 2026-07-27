package journals

// Tipos de documento fuente que pueden originar un asiento contable.
// Se almacenan en source_document_type de journal_entries para discriminar
// a qué módulo/tabla pertenece el source_document_id.
const (
	// Documentos electrónicos del módulo apidian
	SourceFE = "FE"  // Factura Electrónica de Venta
	SourceNC = "NC"  // Nota Crédito de Venta
	SourceND = "ND"  // Nota Débito de Venta
	SourceDS = "DS"  // Documento Soporte (compras a no obligados)
	SourceNA = "NA"  // Nota de Ajuste de Documento Soporte

	// Módulos ERP futuros
	SourceNOM = "NOM" // Nómina (payroll)
	SourceINV = "INV" // Movimiento de inventario
	SourceOC  = "OC"  // Orden de compra / recepción de mercancía
	SourceLC  = "LC"  // Legalización de gastos / viáticos
	SourceAF  = "AF"  // Activo fijo (compra del bien)
)
