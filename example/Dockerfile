# Multi-stage build producing one image with all four mock-service entrypoints.
# Each k8s Deployment selects its binary via `command: ["/<name>"]`.
#
# IMPORTANT: build context is the repo ROOT, not example/, because example/go.mod
# uses `replace github.com/foomo/gateway => ../` to consume the in-tree library.
#   docker build -f example/Dockerfile -t foomo-gateway-example:test .
FROM golang:1.26.4-alpine AS build
WORKDIR /src

# Module manifests first for better layer caching.
COPY go.mod go.sum ./
COPY example/go.mod example/go.sum ./example/

# Download both modules' dependencies up-front.
RUN go mod download
WORKDIR /src/example
RUN go mod download
WORKDIR /src

# Source files. The replace directive in example/go.mod points at /src/pkg,
# so the root packages must also be copied into the build stage.
COPY pkg ./pkg
COPY example ./example

WORKDIR /src/example
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/contentserver  ./cmd/contentserver  \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/pagefrontend   ./cmd/pagefrontend   \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/newsfrontend   ./cmd/newsfrontend   \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/errorfrontend  ./cmd/errorfrontend

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/contentserver  /contentserver
COPY --from=build /out/pagefrontend   /pagefrontend
COPY --from=build /out/newsfrontend   /newsfrontend
COPY --from=build /out/errorfrontend  /errorfrontend
EXPOSE 8080
