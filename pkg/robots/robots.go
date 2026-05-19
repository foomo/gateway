package robots

import (
	"net/http"
	"strings"
)

// Serve writes a concatenated robots.txt from the given entries to w.
func Serve(w http.ResponseWriter, entries []string) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(strings.Join(entries, "\n")))
}
