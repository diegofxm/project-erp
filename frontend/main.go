// Command devui sirve un mini frontend de pruebas (HTML+CSS+JS planos, sin build step, sin
// dependencias) para probar api-dian a mano sin Postman. Es deliberadamente desechable — el
// frontend real (React u otro) es un proyecto aparte, este módulo no tiene relación con
// api-dian ni con cofacture, solo le habla por HTTP como cualquier otro cliente.
package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

func main() {
	addr := flag.String("addr", ":5500", "dirección a escuchar")
	flag.Parse()

	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("devui escuchando en http://localhost%s — api-dian debe correr aparte (go run ./cmd/server, puerto 8080)", *addr)
	log.Fatal(http.ListenAndServe(*addr, http.FileServer(http.FS(sub))))
}
