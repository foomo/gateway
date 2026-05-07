// Standalone news frontend. Used by the kind integration test.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/foomo/gateway/example/internal/newsfrontend"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")

	flag.Parse()

	log.Printf("newsfrontend listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, newsfrontend.Handler()))
}
