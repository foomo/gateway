package handler

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"

	"go.uber.org/zap"

	"github.com/foomo/gateway/pkg/gateway"
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
	mimeProxies map[gateway.MimeType]*serviceProxy
	errorProxy  *serviceProxy
	sitemapURLs []string
	robotsTXT   []string
}

type serviceProxy struct {
	proxy *httputil.ReverseProxy
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
	h.state.Store(&state{mimeProxies: map[gateway.MimeType]*serviceProxy{}})

	return h
}

// Apply rebuilds routing state from the given specs atomically.
func (h *Handler) Apply(specs []gateway.Spec) {
	st := &state{
		mimeProxies: map[gateway.MimeType]*serviceProxy{},
	}

	for _, spec := range specs {
		proxyURL, err := url.Parse(string(spec.Service))
		if err != nil || proxyURL.Scheme == "" {
			h.l.Warn("skipping invalid service URL",
				zap.String("service", string(spec.Service)),
			)

			continue
		}

		sp := &serviceProxy{
			proxy: httputil.NewSingleHostReverseProxy(proxyURL),
		}

		if spec.Sitemap != "" {
			st.sitemapURLs = append(st.sitemapURLs, spec.Sitemap)
		}

		if spec.AddToRobots != "" {
			st.robotsTXT = append(st.robotsTXT, spec.AddToRobots)
		}

		if spec.ErrorFrontend {
			st.errorProxy = sp

			continue
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

	switch r.URL.Path {
	case "/robots.txt":
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(strings.Join(st.robotsTXT, "\n")))

		return
	case "/sitemap.xml":
		h.serveSitemap(w, st)

		return
	case "/gateway/status":
		h.serveStatus(w, st)

		return
	}

	mimeType, contentID, err := h.resolver.Resolve(r)
	if err != nil {
		h.l.Error("resolver error",
			zap.String("uri", r.URL.Path),
			zap.Error(err),
		)
		h.serveError(w, r, st, http.StatusInternalServerError)

		return
	}

	if mimeType == "" {
		h.serveError(w, r, st, http.StatusNotFound)

		return
	}

	mt := gateway.MimeType(mimeType)

	sp, ok := st.mimeProxies[mt]
	if !ok {
		h.l.Warn("no proxy registered for mime type",
			zap.String("mimeType", string(mt)),
			zap.String("uri", r.URL.Path),
		)
		h.serveError(w, r, st, http.StatusNotFound)

		return
	}

	r.Header.Set(HeaderContentID, contentID)
	r.Header.Set(HeaderMimeType, string(mt))
	sp.proxy.ServeHTTP(w, r)
}

// Watch ranges over gateway events, maintains the full spec collection, and calls Apply on each change.
// It returns when the events channel is closed. Run it in a goroutine.
func (h *Handler) Watch(events <-chan gateway.Event) {
	specs := map[gateway.Service]gateway.Spec{}

	for event := range events {
		switch event.Type {
		case gateway.EventAdd, gateway.EventUpdate:
			specs[event.Gateway.Spec.Service] = event.Gateway.Spec
		case gateway.EventDelete:
			delete(specs, event.Gateway.Spec.Service)
		}

		all := make([]gateway.Spec, 0, len(specs))
		for _, s := range specs {
			all = append(all, s)
		}

		h.Apply(all)
	}
}

func (h *Handler) serveError(w http.ResponseWriter, r *http.Request, st *state, code int) {
	if st.errorProxy != nil {
		r.Header.Set(HeaderErrorCode, strconv.Itoa(code))
		st.errorProxy.proxy.ServeHTTP(w, r)

		return
	}

	http.Error(w, http.StatusText(code), code)
}

type sitemapIndex struct {
	XMLName  xml.Name     `xml:"sitemapindex"`
	XMLNS    string       `xml:"xmlns,attr"`
	Sitemaps []sitemapLoc `xml:"sitemap"`
}

type sitemapLoc struct {
	Loc string `xml:"loc"`
}

func (h *Handler) serveSitemap(w http.ResponseWriter, st *state) {
	idx := sitemapIndex{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	for _, u := range st.sitemapURLs {
		idx.Sitemaps = append(idx.Sitemaps, sitemapLoc{Loc: u})
	}

	w.Header().Set("Content-Type", "application/xml")
	_, _ = fmt.Fprint(w, xml.Header)

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	_ = enc.Encode(idx)
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
