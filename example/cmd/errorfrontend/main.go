// Standalone error frontend. Used by the kind integration test.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/foomo/gateway/example/internal/errorfrontend"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")

	flag.Parse()

	log.Printf("errorfrontend listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, errorfrontend.Handler()))
}
