// Package errorfrontend renders an HTML error page. The gateway routes here
// when no mime-type proxy matches; the error code arrives via X-Error-Code.
package errorfrontend

import (
	"fmt"
	"net/http"

	"github.com/foomo/gateway/pkg/handler"
)

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.Header.Get(handler.HeaderErrorCode)

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body>\n")
		fmt.Fprintf(w, "<h1>Error frontend</h1>\n")
		fmt.Fprintf(w, "<p>Error code: %s</p>\n", code)
		fmt.Fprintf(w, "<p>URI: %s</p>\n", r.URL.Path)
		fmt.Fprintf(w, "</body></html>\n")
	})
}
