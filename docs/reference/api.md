# Go API

The `github.com/foomo/gateway/pkg/gateway` package provides Go types matching the Gateway CRD and a listener for watching CRD changes.

## Types

### Gateway

Represents a service registration in the gateway.

```go
type Gateway struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitzero"`
    Spec              Spec `json:"spec"`
}
```

### Spec

Defines the desired state of a Gateway resource.

```go
type Spec struct {
    Service     Service  `json:"service"`
    Sitemap     string   `json:"sitemap,omitempty"`
    AddToRobots string   `json:"addToRobots,omitempty"`
    Expose      []Expose `json:"expose,omitempty"`
}
```

### Expose

Defines an exposed path configuration.

```go
type Expose struct {
    Description          string                `json:"description,omitempty"`
    CMSApp               string                `json:"cmsApp,omitempty"`
    Paths                []Path                `json:"paths,omitempty"`
    CmsMimetypes         []MimeType            `json:"cmsMimetypes,omitempty"`
    InternalAccessGroups []InternalAccessGroup  `json:"internalAccessGroups,omitempty"`
    StripBasePath        bool                  `json:"stripBasePath,omitempty"`
}
```

### Custom string types

| Type | Description |
|------|-------------|
| `Service` | Backend service identifier (min length: 1). |
| `Path` | URI path. |
| `MimeType` | MIME type string. |
| `InternalAccessGroup` | Internal access group identifier. |

## Events

### EventType

```go
type EventType string

const (
    EventAdd    EventType = "add"
    EventUpdate EventType = "update"
    EventDelete EventType = "delete"
)
```

### Event

```go
type Event struct {
    Type    EventType
    Gateway Gateway
}
```

## Functions

### GVR

Returns the `schema.GroupVersionResource` for the Gateway CRD.

```go
func GVR() schema.GroupVersionResource
```

### Listen

Watches for Gateway CRD changes in the given namespace and streams events to the returned channel. The channel is closed when the context is canceled.

```go
func Listen(ctx context.Context, client dynamic.Interface, namespace string) (<-chan Event, error)
```

**Parameters:**
- `ctx` — controls the lifetime of the listener. Cancel to stop watching and close the channel.
- `client` — a Kubernetes dynamic client.
- `namespace` — the namespace to watch for Gateway resources.

**Returns:**
- A read-only channel of `Event` values.
- An error if the informer cannot be set up.
