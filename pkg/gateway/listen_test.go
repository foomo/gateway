package gateway_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/foomo/gateway/pkg/gateway"
)

func newFakeClient(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	scheme := runtime.NewScheme()

	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			gateway.GVR(): "GatewayList",
		},
		objects...,
	)
}

func newUnstructuredGateway(name, namespace, service string, paths []string) *unstructured.Unstructured {
	pathsIface := make([]any, len(paths))
	for i, p := range paths {
		pathsIface[i] = p
	}

	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": gateway.Group + "/" + gateway.Version,
			"kind":       "Gateway",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]any{
				"service": service,
				"sitemap": "",
				"expose": []any{
					map[string]any{
						"paths": pathsIface,
					},
				},
			},
		},
	}
}

func receiveEvent(t *testing.T, ch <-chan gateway.Event, timeout time.Duration) gateway.Event {
	t.Helper()

	select {
	case event := <-ch:
		return event
	case <-time.After(timeout):
		t.Fatal("timed out waiting for event")

		return gateway.Event{}
	}
}

func TestListenAdd(t *testing.T) {
	t.Parallel()

	client := newFakeClient()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := gateway.Listen(ctx, client, "default")
	require.NoError(t, err)

	obj := newUnstructuredGateway("test-svc", "default", "http://test-svc.default.svc.cluster.local", []string{"/api"})
	_, err = client.Resource(gateway.GVR()).Namespace("default").Create(ctx, obj, metav1.CreateOptions{})
	require.NoError(t, err)

	event := receiveEvent(t, ch, 5*time.Second)
	assert.Equal(t, gateway.EventAdd, event.Type)
	assert.Equal(t, "test-svc", event.Gateway.Name)
	assert.Equal(t, gateway.Service("http://test-svc.default.svc.cluster.local"), event.Gateway.Spec.Service)
	assert.Equal(t, []gateway.Path{"/api"}, event.Gateway.Spec.Expose[0].Paths)
}

func TestListenUpdate(t *testing.T) {
	t.Parallel()

	client := newFakeClient()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := gateway.Listen(ctx, client, "default")
	require.NoError(t, err)

	// Create the object first.
	obj := newUnstructuredGateway("test-svc", "default", "http://test-svc.default.svc.cluster.local", []string{"/api"})
	_, err = client.Resource(gateway.GVR()).Namespace("default").Create(ctx, obj, metav1.CreateOptions{})
	require.NoError(t, err)

	// Drain add event.
	receiveEvent(t, ch, 5*time.Second)

	// Update the object.
	updated := newUnstructuredGateway("test-svc", "default", "http://test-svc.default.svc.cluster.local", []string{"/api", "/health"})
	_, err = client.Resource(gateway.GVR()).Namespace("default").Update(ctx, updated, metav1.UpdateOptions{})
	require.NoError(t, err)

	event := receiveEvent(t, ch, 5*time.Second)
	assert.Equal(t, gateway.EventUpdate, event.Type)
	assert.Equal(t, []gateway.Path{"/api", "/health"}, event.Gateway.Spec.Expose[0].Paths)
}

func TestListenDelete(t *testing.T) {
	t.Parallel()

	client := newFakeClient()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := gateway.Listen(ctx, client, "default")
	require.NoError(t, err)

	// Create the object first.
	obj := newUnstructuredGateway("test-svc", "default", "http://test-svc.default.svc.cluster.local", []string{"/api"})
	_, err = client.Resource(gateway.GVR()).Namespace("default").Create(ctx, obj, metav1.CreateOptions{})
	require.NoError(t, err)

	// Drain add event.
	receiveEvent(t, ch, 5*time.Second)

	// Delete the object.
	err = client.Resource(gateway.GVR()).Namespace("default").Delete(ctx, "test-svc", metav1.DeleteOptions{})
	require.NoError(t, err)

	event := receiveEvent(t, ch, 5*time.Second)
	assert.Equal(t, gateway.EventDelete, event.Type)
	assert.Equal(t, "test-svc", event.Gateway.Name)
}

func TestListenContextCancel(t *testing.T) {
	t.Parallel()

	client := newFakeClient()

	ctx, cancel := context.WithCancel(context.Background())

	ch, err := gateway.Listen(ctx, client, "default")
	require.NoError(t, err)

	cancel()

	// Channel should be closed after context cancellation.
	select {
	case _, ok := <-ch:
		assert.False(t, ok, "channel should be closed")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}
