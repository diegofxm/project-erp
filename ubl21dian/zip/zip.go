// Package zip comprime los documentos UBL firmados en el paquete que exigen los
// webservices de recepción de la DIAN (SendBillSync/SendBillAsync, Anexo Técnico 1.9,
// secciones 6.5.7/6.5.8/7.8/7.10).
package zip

import (
	"archive/zip"
	"bytes"
	"fmt"
)

// File es un documento a incluir en el paquete comprimido.
type File struct {
	Name    string
	Content []byte
}

// Build comprime los documentos dados en un único archivo ZIP, en el orden recibido
// (un slice en vez de un map: el orden debe ser determinístico, no depender de la
// iteración de un map).
//
// SendBillSync exige exactamente un documento; SendBillAsync admite hasta 50 (sección
// 6.5.8). Esa validación de cantidad es responsabilidad de quien llama — soap, no zip —
// porque depende de qué operación se vaya a usar, algo que este paquete no conoce.
func Build(files []File) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, f := range files {
		entry, err := w.Create(f.Name)
		if err != nil {
			return nil, fmt.Errorf("zip: crear entrada %s: %w", f.Name, err)
		}
		if _, err := entry.Write(f.Content); err != nil {
			return nil, fmt.Errorf("zip: escribir entrada %s: %w", f.Name, err)
		}
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("zip: cerrar: %w", err)
	}
	return buf.Bytes(), nil
}
