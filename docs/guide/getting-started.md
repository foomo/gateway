# Getting Started

## Installation

```bash
go get github.com/foomo/gateway
```

## Overview

The `gateway` package provides a Kubernetes Custom Resource Definition (CRD) for registering services in an application gateway, along with a Go client for watching CRD changes in real time.

## Applying the CRD

Install the Gateway CRD into your cluster:

```bash
kubectl apply -f config/crd/foomo.org_gateways.yaml
```

## Creating a Gateway resource

Define a Gateway resource to register a service:

```yaml
apiVersion: foomo.org/v1
kind: Gateway
metadata:
  name: my-service
  namespace: default
spec:
  service: my-service
  sitemap: https://example.com/sitemap.xml
  expose:
    - description: Public API
      paths:
        - /api/v1
      stripBasePath: true
    - description: CMS content
      cmsApp: my-cms-app
      paths:
        - /content
      cmsMimetypes:
        - text/html
      internalAccessGroups:
        - editors
```

Apply it:

```bash
kubectl apply -f my-gateway.yaml
```

## Listening for changes

Use the `Listen` function to watch for Gateway CRD events:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/foomo/gateway/pkg/gateway"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

func main() {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	events, err := gateway.Listen(ctx, client, "default")
	if err != nil {
		log.Fatal(err)
	}

	for event := range events {
		switch event.Type {
		case gateway.EventAdd:
			fmt.Printf("added: %s/%s\n", event.Gateway.Namespace, event.Gateway.Name)
		case gateway.EventUpdate:
			fmt.Printf("updated: %s/%s\n", event.Gateway.Namespace, event.Gateway.Name)
		case gateway.EventDelete:
			fmt.Printf("deleted: %s/%s\n", event.Gateway.Namespace, event.Gateway.Name)
		}
	}
}
```

The returned channel is closed when the context is canceled.
