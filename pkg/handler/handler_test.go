package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/foomo/gateway/pkg/gateway"
	"github.com/foomo/gateway/pkg/handler"
)

// resolveFunc adapts a plain function to the Resolver interface.
type resolveFunc func(r *http.Request) (string, string, error)

func (f resolveFunc) Resolve(r *http.Request) (string, string, error) { return f(r) }

// routeResolver returns a Resolver backed by a static URI → (mimeType, contentID) map.
func routeResolver(routes map[string][2]string) handler.Resolver {
	return resolveFunc(func(r *http.Request) (string, string, error) {
		if v, ok := routes[r.URL.Path]; ok {
			return v[0], v[1], nil
		}

		return "", "", nil
	})
}

// newMockFrontend returns an httptest.Server that captures the last request headers.
func newMockFrontend() (*httptest.Server, *http.Header) {
	captured := http.Header{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Clone()

		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "ok")
	}))

	return srv, &captured
}

func newHandler(t *testing.T, resolver handler.Resolver) *handler.Handler {
	t.Helper()

	return handler.New(zap.NewNop(), resolver)
}

func newRequest(method, path string) *http.Request {
	return httptest.NewRequestWithContext(context.Background(), method, path, nil)
}

// svc returns the full URL of an httptest server as a gateway.Service.
func svc(srv *httptest.Server) gateway.Service {
	return gateway.Service(srv.URL)
}

func TestServeHTTP_MimeTypeRouting(t *testing.T) {
	t.Parallel()

	resolver := routeResolver(map[string][2]string{
		"/page": {"text/page", "item-1"},
	})

	frontend, headers := newMockFrontend()
	defer frontend.Close()

	h := newHandler(t, resolver)
	h.Apply([]gateway.Spec{
		{
			Service: svc(frontend),
			Expose:  []gateway.Expose{{CmsMimetypes: []gateway.MimeType{"text/page"}}},
		},
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(http.MethodGet, "/page"))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "item-1", headers.Get(handler.HeaderContentID))
	assert.Equal(t, "text/page", headers.Get(handler.HeaderMimeType))
}

func TestServeHTTP_UnknownMimeType_RoutesToErrorFrontend(t *testing.T) {
	t.Parallel()

	resolver := routeResolver(map[string][2]string{
		"/unknown": {"text/unknown", "item-2"},
	})

	errFrontend, headers := newMockFrontend()
	defer errFrontend.Close()

	h := newHandler(t, resolver)
	h.Apply([]gateway.Spec{
		{
			Service:       svc(errFrontend),
			ErrorFrontend: true,
		},
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(http.MethodGet, "/unknown"))

	assert.Equal(t, "404", headers.Get(handler.HeaderErrorCode))
}

func TestServeHTTP_ResolverNotFound_RoutesToErrorFrontend(t *testing.T) {
	t.Parallel()

	errFrontend, headers := newMockFrontend()
	defer errFrontend.Close()

	h := newHandler(t, routeResolver(nil))
	h.Apply([]gateway.Spec{
		{
			Service:       svc(errFrontend),
			ErrorFrontend: true,
		},
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(http.MethodGet, "/missing"))

	assert.Equal(t, "404", headers.Get(handler.HeaderErrorCode))
}

func TestServeHTTP_NoErrorFrontend_PlainHTTPError(t *testing.T) {
	t.Parallel()

	h := newHandler(t, routeResolver(nil))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(http.MethodGet, "/missing"))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestServeHTTP_RobotsTXT(t *testing.T) {
	t.Parallel()

	h := newHandler(t, routeResolver(nil))
	h.Apply([]gateway.Spec{
		{Service: "http://svc", AddToRobots: "User-agent: *\nDisallow: /admin"},
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(http.MethodGet, "/robots.txt"))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Disallow: /admin")
}

func TestServeHTTP_Sitemap(t *testing.T) {
	t.Parallel()

	h := newHandler(t, routeResolver(nil))
	h.Apply([]gateway.Spec{
		{Service: "http://svc", Sitemap: "https://example.com/sitemap.xml"},
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(http.MethodGet, "/sitemap.xml"))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "https://example.com/sitemap.xml")
}

func TestWatch(t *testing.T) {
	t.Parallel()

	resolver := routeResolver(map[string][2]string{
		"/page": {"text/page", "item-1"},
	})

	frontend, _ := newMockFrontend()
	defer frontend.Close()

	h := newHandler(t, resolver)

	events := make(chan gateway.Event, 1)
	events <- gateway.Event{
		Type: gateway.EventAdd,
		Gateway: gateway.Gateway{
			Spec: gateway.Spec{
				Service: svc(frontend),
				Expose:  []gateway.Expose{{CmsMimetypes: []gateway.MimeType{"text/page"}}},
			},
		},
	}

	close(events)

	handler.Watch(h, events)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newRequest(http.MethodGet, "/page"))

	assert.Equal(t, http.StatusOK, rec.Code)
}
