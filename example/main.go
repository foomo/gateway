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
	"net/http"
	"net/http/httptest"

	"go.uber.org/zap"

	csclient "github.com/foomo/contentserver/client"
	cscontent "github.com/foomo/contentserver/content"
	csrequests "github.com/foomo/contentserver/requests"

	"github.com/foomo/gateway/example/internal/contentserver"
	"github.com/foomo/gateway/example/internal/errorfrontend"
	"github.com/foomo/gateway/example/internal/newsfrontend"
	"github.com/foomo/gateway/example/internal/pagefrontend"
	"github.com/foomo/gateway/pkg/gateway"
	"github.com/foomo/gateway/pkg/handler"
)

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

func main() {
	cs := httptest.NewServer(contentserver.Handler(nil))
	page := httptest.NewServer(pagefrontend.Handler())
	news := httptest.NewServer(newsfrontend.Handler())
	errSvc := httptest.NewServer(errorfrontend.Handler())

	defer cs.Close()
	defer page.Close()
	defer news.Close()
	defer errSvc.Close()

	l, _ := zap.NewDevelopment()

	l.Info("mock contentserver", zap.String("url", cs.URL))
	l.Info("page frontend", zap.String("url", page.URL))
	l.Info("news frontend", zap.String("url", news.URL))
	l.Info("error frontend", zap.String("url", errSvc.URL))

	csCli, err := csclient.NewHTTPClient(cs.URL)
	if err != nil {
		panic(err)
	}

	h := handler.New(l, &csResolver{cs: csCli})

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

	if err := http.ListenAndServe(":8080", h); err != nil {
		panic(err)
	}
}
