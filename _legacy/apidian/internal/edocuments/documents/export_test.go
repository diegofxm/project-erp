package documents

// AggregateTaxesForTest expone aggregateTaxes (no exportada) para los tests externos del
// paquete (service_test.go, package documents_test) — ver
// TestCreateInvoiceDraft_AggregatesMixedTaxRatesSeparately, sección 9.46.
var AggregateTaxesForTest = aggregateTaxes
