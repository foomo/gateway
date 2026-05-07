// Package newsfrontend renders a tiny HTML page identifying itself as the news
// frontend. See pagefrontend for the same shape.
package newsfrontend

import (
	"fmt"
	"net/http"

	"github.com/foomo/gateway/pkg/handler"
)

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentID := r.Header.Get(handler.HeaderContentID)
		mimeType := r.Header.Get(handler.HeaderMimeType)

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body>\n")
		fmt.Fprintf(w, "<h1>News frontend</h1>\n")
		fmt.Fprintf(w, "<p>URI: %s</p>\n", r.URL.Path)
		fmt.Fprintf(w, "<p>Content-ID: %s</p>\n", contentID)
		fmt.Fprintf(w, "<p>Mime-Type: %s</p>\n", mimeType)
		fmt.Fprintf(w, "</body></html>\n")
	})
}
