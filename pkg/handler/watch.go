package handler

import "github.com/foomo/gateway/pkg/gateway"

// Watch ranges over gateway events, maintains the full spec collection, and calls Apply on each change.
// It returns when the events channel is closed. Run it in a goroutine.
func Watch(h *Handler, events <-chan gateway.Event) {
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
