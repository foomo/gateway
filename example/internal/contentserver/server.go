// Package contentserver provides a mock contentserver that resolves URIs to
// (mimeType, contentID) pairs. It implements the bare HTTP shape the
// foomo/contentserver client expects: POST a JSON body with a "URI" field,
// receive a JSON envelope `{"Reply": SiteContent}`.
package contentserver

import (
	"encoding/json"
	"log"
	"net/http"

	cscontent "github.com/foomo/contentserver/content"
)

// DefaultContentTree is the demo content tree used by both the integrated
// example and the kind-deployed binary.
var DefaultContentTree = map[string]*cscontent.SiteContent{
	"/": {
		Status: cscontent.StatusOk,
		URI:    "/",
		Item:   &cscontent.Item{ID: "home", MimeType: "application/x-sandbox-page"},
	},
	"/about": {
		Status: cscontent.StatusOk,
		URI:    "/about",
		Item:   &cscontent.Item{ID: "about", MimeType: "application/x-sandbox-page"},
	},
	"/news/article-1": {
		Status: cscontent.StatusOk,
		URI:    "/news/article-1",
		Item:   &cscontent.Item{ID: "news-1", MimeType: "application/x-sandbox-news"},
	},
}

type response struct {
	Reply *cscontent.SiteContent `json:"Reply"`
}

// Handler returns an http.Handler backed by the given content tree. Pass nil
// to use DefaultContentTree.
func Handler(tree map[string]*cscontent.SiteContent) http.Handler {
	if tree == nil {
		tree = DefaultContentTree
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URI string `json:"URI"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		sc, ok := tree[req.URI]
		if !ok {
			sc = &cscontent.SiteContent{Status: cscontent.StatusNotFound}
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(response{Reply: sc}); err != nil {
			log.Printf("contentserver mock: encode error: %v", err)
		}
	})
}
