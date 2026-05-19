// Standalone mock contentserver. Used by the kind integration test.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/foomo/gateway/example/internal/contentserver"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")

	flag.Parse()

	log.Printf("contentserver listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, contentserver.Handler(nil)))
}
