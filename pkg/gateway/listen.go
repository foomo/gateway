package gateway

import (
	"context"
	"encoding/json"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

// EventType represents the type of change to a Gateway resource.
type EventType string

const (
	EventAdd    EventType = "add"
	EventUpdate EventType = "update"
	EventDelete EventType = "delete"
)

// Event represents a change to a Gateway resource.
type Event struct {
	Type    EventType
	Gateway Gateway
}

// GVR returns the GroupVersionResource for Gateway CRDs.
func GVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    Group,
		Version:  Version,
		Resource: Resource,
	}
}

// Listen watches for Gateway CRD changes and sends events to the returned channel.
// The channel is closed when the context is canceled.
func Listen(ctx context.Context, client dynamic.Interface, namespace string) (<-chan Event, error) {
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		client,
		30*time.Second,
		namespace,
		nil,
	)

	informer := factory.ForResource(GVR()).Informer()

	ch := make(chan Event, 64)

	reg, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if gw, ok := toGateway(obj); ok {
				ch <- Event{Type: EventAdd, Gateway: gw}
			}
		},
		UpdateFunc: func(_, newObj any) {
			if gw, ok := toGateway(newObj); ok {
				ch <- Event{Type: EventUpdate, Gateway: gw}
			}
		},
		DeleteFunc: func(obj any) {
			if gw, ok := toGateway(obj); ok {
				ch <- Event{Type: EventDelete, Gateway: gw}
			}
		},
	})
	if err != nil {
		close(ch)
		return nil, err
	}

	factory.Start(ctx.Done())
	factory.WaitForCacheSync(ctx.Done())

	go func() {
		<-ctx.Done()

		_ = informer.RemoveEventHandler(reg)

		close(ch)
	}()

	return ch, nil
}

func toGateway(obj any) (Gateway, bool) {
	data, err := json.Marshal(obj)
	if err != nil {
		return Gateway{}, false
	}

	var gw Gateway
	if err := json.Unmarshal(data, &gw); err != nil {
		return Gateway{}, false
	}

	return gw, true
}
