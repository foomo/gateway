// Standalone page frontend. Used by the kind integration test.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/foomo/gateway/example/internal/pagefrontend"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")

	flag.Parse()

	log.Printf("pagefrontend listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, pagefrontend.Handler()))
}
