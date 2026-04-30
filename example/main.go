// Example demonstrates foomo/gateway routing without any external dependencies.
//
// It spins up three local services:
//   - a mock contentserver (resolves URIs to mime types)
//   - a page frontend  (handles application/x-sandbox-page)
//   - an error frontend (handles 404 / 500)
//
// Then it wires the gateway handler, calls Apply with two specs, and serves on :8080.
//
// Run: go run main.go
//
// Try it:
//
//	curl -i http://localhost:8080/
//	curl -i http://localhost:8080/about
//	curl -i http://localhost:8080/news/article-1
//	curl -i http://localhost:8080/missing
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	"go.uber.org/zap"

	csclient "github.com/foomo/contentserver/client"
	cscontent "github.com/foomo/contentserver/content"
	csrequests "github.com/foomo/contentserver/requests"
	"github.com/foomo/gateway/pkg/gateway"
	"github.com/foomo/gateway/pkg/handler"
)

// contentTree defines the URIs the mock contentserver knows about.
var contentTree = map[string]*cscontent.SiteContent{
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

// csResponse mirrors the envelope the contentserver HTTP client expects.
type csResponse struct {
	Reply *cscontent.SiteContent `json:"Reply"`
}

func mockContentserver() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URI string `json:"URI"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		sc, ok := contentTree[req.URI]
		if !ok {
			sc = &cscontent.SiteContent{Status: cscontent.StatusNotFound}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(csResponse{Reply: sc}); err != nil {
			log.Printf("contentserver mock: encode error: %v", err)
		}
	}))
}

// csResolver wraps the contentserver client to implement handler.Resolver.
type csResolver struct {
	cs *csclient.Client
}

func (r *csResolver) Resolve(req *http.Request) (string, string, error) {
	sc, err := r.cs.GetContent(req.Context(), &csrequests.Content{URI: req.URL.Path})
	if err != nil {
		return "", "", err
	}

	if sc.Status != cscontent.StatusOk || sc.Item == nil {
		return "", "", nil
	}

	return sc.Item.MimeType, sc.Item.ID, nil
}

// svc returns the full URL of an httptest server as a gateway.Service.
func svc(srv *httptest.Server) gateway.Service {
	return gateway.Service(srv.URL)
}

func pageFrontend() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentID := r.Header.Get(handler.HeaderContentID)
		mimeType := r.Header.Get(handler.HeaderMimeType)

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body>\n")
		fmt.Fprintf(w, "<h1>Page frontend</h1>\n")
		fmt.Fprintf(w, "<p>URI: %s</p>\n", r.URL.Path)
		fmt.Fprintf(w, "<p>Content-ID: %s</p>\n", contentID)
		fmt.Fprintf(w, "<p>Mime-Type: %s</p>\n", mimeType)
		fmt.Fprintf(w, "</body></html>\n")
	}))
}

func newsFrontend() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentID := r.Header.Get(handler.HeaderContentID)
		mimeType := r.Header.Get(handler.HeaderMimeType)

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body>\n")
		fmt.Fprintf(w, "<h1>News frontend</h1>\n")
		fmt.Fprintf(w, "<p>URI: %s</p>\n", r.URL.Path)
		fmt.Fprintf(w, "<p>Content-ID: %s</p>\n", contentID)
		fmt.Fprintf(w, "<p>Mime-Type: %s</p>\n", mimeType)
		fmt.Fprintf(w, "</body></html>\n")
	}))
}

func errorFrontend() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.Header.Get(handler.HeaderErrorCode)

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<html><body>\n")
		fmt.Fprintf(w, "<h1>Error frontend</h1>\n")
		fmt.Fprintf(w, "<p>Error code: %s</p>\n", code)
		fmt.Fprintf(w, "<p>URI: %s</p>\n", r.URL.Path)
		fmt.Fprintf(w, "</body></html>\n")
	}))
}

func main() {
	cs := mockContentserver()
	page := pageFrontend()
	news := newsFrontend()
	errSvc := errorFrontend()

	defer cs.Close()
	defer page.Close()
	defer news.Close()
	defer errSvc.Close()

	l, _ := zap.NewDevelopment()

	l.Info("mock contentserver", zap.String("url", cs.URL))
	l.Info("page frontend", zap.String("url", page.URL))
	l.Info("news frontend", zap.String("url", news.URL))
	l.Info("error frontend", zap.String("url", errSvc.URL))

	csClient, err := csclient.NewHTTPClient(cs.URL)
	if err != nil {
		log.Fatal(err)
	}

	h := handler.New(l, &csResolver{cs: csClient})

	h.Apply([]gateway.Spec{
		{
			Service:     svc(page),
			Sitemap:     "https://example.com/sitemap.xml",
			AddToRobots: "User-agent: *\nDisallow: /admin",
			Expose: []gateway.Expose{
				{CmsMimetypes: []gateway.MimeType{"application/x-sandbox-page"}},
			},
		},
		{
			Service: svc(news),
			Expose: []gateway.Expose{
				{CmsMimetypes: []gateway.MimeType{"application/x-sandbox-news"}},
			},
		},
		{
			Service:       svc(errSvc),
			ErrorFrontend: true,
		},
	})

	l.Info("gateway listening", zap.String("addr", ":8080"))
	l.Info("try: curl -i http://localhost:8080/")
	l.Info("try: curl -i http://localhost:8080/about")
	l.Info("try: curl -i http://localhost:8080/news/article-1")
	l.Info("try: curl -i http://localhost:8080/missing")
	l.Info("try: curl -i http://localhost:8080/robots.txt")
	l.Info("try: curl -i http://localhost:8080/sitemap.xml")
	l.Info("try: curl -i http://localhost:8080/gateway/status")

	log.Fatal(http.ListenAndServe(":8080", h))
}
