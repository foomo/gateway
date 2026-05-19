// Package kind_test exercises the foomo/gateway library end-to-end against a
// real kind cluster. The test process plays the role of the in-process gateway,
// connecting to the apiserver via $KUBECONFIG and proxying to in-cluster
// backends via NodePorts that kind has mapped onto the host.
//
// Prerequisites (managed by `make test.kind`):
//   - kind cluster from ../kind-config.yaml (NodePorts 30001..30004 on host)
//   - foomo-gateway-example:test image loaded into the cluster
//   - manifests in ./manifests/ applied + Deployments Available
//   - CRD config/crd/foomo.org_gateways.yaml applied + Established
package kind_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	csclient "github.com/foomo/contentserver/client"
	cscontent "github.com/foomo/contentserver/content"
	csrequests "github.com/foomo/contentserver/requests"
	"github.com/foomo/gateway/pkg/gateway"
	"github.com/foomo/gateway/pkg/handler"
	testingx "github.com/foomo/go/testing"
	tagx "github.com/foomo/go/testing/tag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	contentserverURL = "http://localhost:30001"
	pagefrontendURL  = "http://localhost:30002"
	newsfrontendURL  = "http://localhost:30003"
	errorfrontendURL = "http://localhost:30004"

	eventually = 30 * time.Second
	tick       = 100 * time.Millisecond
)

// loadKubeconfig resolves a *rest.Config from $KUBECONFIG or ~/.kube/config.
func loadKubeconfig(t *testing.T) *rest.Config {
	t.Helper()

	path := os.Getenv("KUBECONFIG")
	if path == "" {
		home, err := os.UserHomeDir()
		require.NoError(t, err)

		path = filepath.Join(home, ".kube", "config")
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", path)
	require.NoError(t, err, "load kubeconfig (KUBECONFIG=%q)", path)

	return cfg
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

// fixture wires gateway.Listen + handler.Watch + a *Handler against the kind
// cluster. Returned cancel stops Listen and waits for Watch to drain.
type fixture struct {
	dyn      dynamic.Interface
	handler  *handler.Handler
	cancel   func()
	watchEnd <-chan struct{}
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	cfg := loadKubeconfig(t)

	dyn, err := dynamic.NewForConfig(cfg)
	require.NoError(t, err)

	csCli, err := csclient.NewHTTPClient(contentserverURL)
	require.NoError(t, err)

	h := handler.New(zap.NewNop(), &csResolver{cs: csCli})

	ctx, cancel := context.WithCancel(context.Background())

	events, err := gateway.Listen(ctx, dyn, "default", gateway.GVR())
	if err != nil {
		cancel()
		t.Fatalf("gateway.Listen: %v", err)
	}

	watchEnd := make(chan struct{})

	go func() {
		handler.Watch(h, events)
		close(watchEnd)
	}()

	t.Cleanup(func() {
		cancel()
		<-watchEnd
	})

	return &fixture{dyn: dyn, handler: h, cancel: cancel, watchEnd: watchEnd}
}

// dial issues a request against the in-process gateway handler and returns the
// recorder. The handler proxies to in-cluster backends via the NodePort URLs
// embedded in each Gateway CR's spec.service.
func (f *fixture) dial(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil)
	f.handler.ServeHTTP(rec, req)

	return rec
}

// TestKind_PreAppliedRouting verifies the manifests applied by `make kind.deploy`
// flow through Listen → Watch → Apply correctly: requests to known URIs reach
// the right backend; unknown URIs hit the error frontend.
func TestKind_PreAppliedRouting(t *testing.T) {
	testingx.Tags(t, tagx.Skip, tagx.Integration)
	f := newFixture(t)

	// Page frontend (URI "/" → application/x-sandbox-page).
	require.Eventually(t, func() bool {
		rec := f.dial(t, "/")

		return rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), "Page frontend")
	}, eventually, tick, "page routing never became active")

	// Same mime type, different URI.
	rec := f.dial(t, "/about")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Page frontend")

	// News frontend (URI "/news/article-1" → application/x-sandbox-news).
	require.Eventually(t, func() bool {
		rec := f.dial(t, "/news/article-1")

		return rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), "News frontend")
	}, eventually, tick, "news routing never became active")

	// Unknown URI → error frontend (errorFrontend: true on a Gateway CR).
	require.Eventually(t, func() bool {
		rec := f.dial(t, "/missing")

		return strings.Contains(rec.Body.String(), "Error frontend")
	}, eventually, tick, "error frontend never became active")
}

// TestKind_DynamicCreate adds a new Gateway CR at runtime and asserts the
// gateway picks it up: a fresh URI (resolved by the contentserver) routes to
// an existing backend via a new mime type.
func TestKind_DynamicCreate(t *testing.T) {
	testingx.Tags(t, tagx.Skip, tagx.Integration)
	f := newFixture(t)

	// Wait for baseline routing to be active first, otherwise a fast Eventually
	// might assert before the watcher has even cached the pre-applied CRs.
	require.Eventually(t, func() bool {
		return f.dial(t, "/").Code == http.StatusOK
	}, eventually, tick)

	gvr := gateway.GVR()
	name := "extra-page-" + uniqueSuffix()

	cr := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": gateway.Group + "/" + gateway.Version,
			"kind":       "Gateway",
			"metadata": map[string]any{
				"name":      name,
				"namespace": "default",
			},
			"spec": map[string]any{
				// Reuse the page frontend but bind it to a brand-new mime type
				// the contentserver doesn't currently advertise. The CR going
				// through the apiserver is the load-bearing assertion here.
				"service": pagefrontendURL,
				"expose": []any{
					map[string]any{
						"cmsMimetypes": []any{"application/x-sandbox-extra"},
					},
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := f.dyn.Resource(gvr).Namespace("default").Create(ctx, cr, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_ = f.dyn.Resource(gvr).Namespace("default").Delete(ctx, name, metav1.DeleteOptions{})
	})

	// Assert the new CR landed in handler state by querying /gateway/status.
	require.Eventually(t, func() bool {
		rec := f.dial(t, "/gateway/status")

		return rec.Code == http.StatusOK &&
			strings.Contains(rec.Body.String(), "application/x-sandbox-extra")
	}, eventually, tick, "new mime type never appeared in /gateway/status")
}

// TestKind_DynamicDelete deletes the news Gateway CR and asserts /news/article-1
// stops resolving to the news frontend (falls back to the error frontend).
// Recreates the CR in t.Cleanup so the test is rerunnable.
func TestKind_DynamicDelete(t *testing.T) {
	testingx.Tags(t, tagx.Skip, tagx.Integration)
	f := newFixture(t)

	gvr := gateway.GVR()

	// Capture the live CR so we can faithfully restore it in cleanup.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	original, err := f.dyn.Resource(gvr).Namespace("default").Get(ctx, "news", metav1.GetOptions{})
	require.NoError(t, err)

	// Wait for baseline news routing to be active first.
	require.Eventually(t, func() bool {
		rec := f.dial(t, "/news/article-1")

		return rec.Code == http.StatusOK && strings.Contains(rec.Body.String(), "News frontend")
	}, eventually, tick, "news routing never became active before delete")

	require.NoError(t, f.dyn.Resource(gvr).Namespace("default").Delete(ctx, "news", metav1.DeleteOptions{}))

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Strip server-managed fields before recreating.
		clone := original.DeepCopy()
		unstructured.RemoveNestedField(clone.Object, "metadata", "resourceVersion")
		unstructured.RemoveNestedField(clone.Object, "metadata", "uid")
		unstructured.RemoveNestedField(clone.Object, "metadata", "creationTimestamp")
		unstructured.RemoveNestedField(clone.Object, "metadata", "generation")
		unstructured.RemoveNestedField(clone.Object, "metadata", "managedFields")

		_, _ = f.dyn.Resource(gvr).Namespace("default").Create(ctx, clone, metav1.CreateOptions{})
	})

	// After delete, /news/article-1 should fall through to the error frontend
	// (no matching mime-type proxy).
	require.Eventually(t, func() bool {
		rec := f.dial(t, "/news/article-1")

		return strings.Contains(rec.Body.String(), "Error frontend")
	}, eventually, tick, "news routing never reverted after delete")
}

// uniqueSuffix produces a short, monotonic suffix for resource names within a
// single test run to avoid collisions between sequential or retried tests.
func uniqueSuffix() string {
	return strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")
}
