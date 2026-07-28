package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strconv"
	"sync/atomic"

	"github.com/foomo/gateway/pkg/gateway"
	"github.com/foomo/gateway/pkg/robots"
	"github.com/foomo/gateway/pkg/sitemap"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	HeaderContentID = "X-Content-Id"
	HeaderMimeType  = "X-Mime-Type"
	HeaderErrorCode = "X-Error-Code"
)

// Resolver resolves an incoming request to a mime type and content ID.
// Return empty mimeType (with no error) when the content does not exist or is not accessible.
type Resolver interface {
	Resolve(r *http.Request) (mimeType, contentID string, err error)
}

type state struct {
	mimeProxies map[gateway.MimeType]*httputil.ReverseProxy
	errorProxy  *httputil.ReverseProxy
	sitemapURLs []string
	robotsTXT   []string
}

// Handler routes HTTP requests to registered frontend services based on mime-type resolution.
type Handler struct {
	resolver Resolver
	l        *zap.Logger
	state    atomic.Pointer[state]
}

// New creates a Handler backed by the given Resolver.
func New(l *zap.Logger, resolver Resolver) *Handler {
	h := &Handler{
		resolver: resolver,
		l:        l,
	}
	h.state.Store(&state{mimeProxies: map[gateway.MimeType]*httputil.ReverseProxy{}})

	return h
}

// Apply rebuilds routing state from the given specs atomically.
func (h *Handler) Apply(specs []gateway.Spec) {
	st := &state{
		mimeProxies: map[gateway.MimeType]*httputil.ReverseProxy{},
	}

	for _, spec := range specs {
		proxyURL, err := url.Parse(string(spec.Service))
		if err != nil || proxyURL.Scheme == "" {
			h.l.Warn("skipping invalid service URL",
				zap.String("service", string(spec.Service)),
			)

			continue
		}

		sp := httputil.NewSingleHostReverseProxy(proxyURL)
		sp.Transport = otelhttp.NewTransport(sp.Transport)

		if spec.Sitemap != "" {
			st.sitemapURLs = append(st.sitemapURLs, spec.Sitemap)
		}

		if spec.AddToRobots != "" {
			st.robotsTXT = append(st.robotsTXT, spec.AddToRobots)
		}

		if spec.ErrorFrontend {
			st.errorProxy = sp
		}

		for _, expose := range spec.Expose {
			for _, mt := range expose.CmsMimetypes {
				st.mimeProxies[mt] = sp
			}
		}
	}

	h.state.Store(st)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	st := h.state.Load()
	span := trace.SpanFromContext(r.Context())

	switch r.URL.Path {
	case "/robots.txt":
		span.SetStatus(codes.Ok, "robots")
		robots.Serve(w, st.robotsTXT)

		return
	case "/sitemap.xml":
		span.SetStatus(codes.Ok, "sitemap")
		sitemap.Serve(w, st.sitemapURLs)

		return
	case "/gateway/status":
		span.SetStatus(codes.Ok, "status")
		h.serveStatus(w, st)

		return
	}

	mimeType, contentID, err := h.resolver.Resolve(r)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		h.serveError(w, r, st, http.StatusInternalServerError)

		return
	} else if mimeType == "" {
		span.SetStatus(codes.Error, "empty mime type")
		h.serveError(w, r, st, http.StatusNotFound)

		return
	}

	span.SetAttributes(
		attribute.String("mimeType", mimeType),
		attribute.String("contentId", contentID),
	)

	sp, ok := st.mimeProxies[gateway.MimeType(mimeType)]
	if !ok {
		span.SetStatus(codes.Error, "missing mime type proxy")
		h.serveError(w, r, st, http.StatusNotFound)

		return
	}

	r.Header.Set(HeaderContentID, contentID)
	r.Header.Set(HeaderMimeType, mimeType)
	sp.ServeHTTP(w, r)
}

func (h *Handler) serveError(w http.ResponseWriter, r *http.Request, st *state, code int) {
	if st.errorProxy != nil {
		r.Header.Set(HeaderErrorCode, strconv.Itoa(code))
		st.errorProxy.ServeHTTP(w, r)

		return
	}

	http.Error(w, http.StatusText(code), code)
}

func (h *Handler) serveStatus(w http.ResponseWriter, st *state) {
	mimes := make([]string, 0, len(st.mimeProxies))
	for mt := range st.mimeProxies {
		mimes = append(mimes, string(mt))
	}

	sort.Strings(mimes)

	type statusResponse struct {
		MimeTypes   []string `json:"mimeTypes"`
		ErrorProxy  bool     `json:"errorProxy"`
		SitemapURLs []string `json:"sitemapURLs"`
	}

	out, err := json.Marshal(statusResponse{
		MimeTypes:   mimes,
		ErrorProxy:  st.errorProxy != nil,
		SitemapURLs: st.sitemapURLs,
	})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}
