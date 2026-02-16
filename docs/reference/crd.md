# Gateway CRD

The Gateway Custom Resource Definition registers a service in the application gateway.

## Resource metadata

| Field | Value |
|-------|-------|
| Group | `foomo.org` |
| Version | `v1` |
| Kind | `Gateway` |
| Plural | `gateways` |
| Short name | `gw` |
| Scope | Namespaced |

## Spec fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `service` | `string` | Yes | Backend service identifier (min length: 1). |
| `sitemap` | `string` | No | URL to the service's sitemap. |
| `addToRobots` | `string` | No | Content to add to the gateway's robots.txt. |
| `expose` | `[]Expose` | No | List of exposed path configurations. |

## Expose fields

| Field | Type | Description |
|-------|------|-------------|
| `description` | `string` | Description shown in the gateway API. |
| `cmsApp` | `string` | Named application in the CMS. |
| `paths` | `[]string` | URL paths to register. Lookup is automatically ordered from longest to shortest. |
| `cmsMimetypes` | `[]string` | Content server MIME types. |
| `internalAccessGroups` | `[]string` | Restricts access to these internal groups. |
| `stripBasePath` | `bool` | When `true`, the base path is stripped from forwarded requests and set as the `x-base-path` header. |

## Example

```yaml
apiVersion: foomo.org/v1
kind: Gateway
metadata:
  name: my-service
  namespace: default
spec:
  service: my-service
  sitemap: https://example.com/sitemap.xml
  addToRobots: "User-agent: *\nAllow: /api/"
  expose:
    - description: Public API
      paths:
        - /api/v1
      stripBasePath: true
    - description: CMS pages
      cmsApp: my-app
      paths:
        - /content
      cmsMimetypes:
        - text/html
      internalAccessGroups:
        - editors
```

## Installation

The CRD manifest is located at `config/crd/foomo.org_gateways.yaml` and is auto-generated from Go types using [controller-gen](https://book.kubebuilder.io/reference/controller-gen).

```bash
kubectl apply -f config/crd/foomo.org_gateways.yaml
```

To regenerate the CRD after modifying the Go types:

```bash
make generate
```
