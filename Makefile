.DEFAULT_GOAL:=help
-include .makerc

# --- Config -----------------------------------------------------------------

GOMODS=$(shell find . -type f -name go.mod)
KIND_CLUSTER  ?= foomo-gateway
EXAMPLE_IMAGE ?= foomo-gateway-example:test
# Newline hack for error output
define br


endef

# --- Targets -----------------------------------------------------------------

# This allows us to accept extra arguments
%: .mise .lefthook go.work
	@:

.PHONY: .mise
# Install dependencies
.mise:
ifeq (, $(shell command -v mise))
	$(error $(br)$(br)Please ensure you have 'mise' installed and activated!$(br)$(br)  $$ brew update$(br)  $$ brew install mise$(br)$(br)See the documentation: https://mise.jdx.dev/getting-started.html)
endif
	@mise install

.PHONY: .lefthook
# Configure git hooks for lefthook
.lefthook:
	@lefthook install --reset-hooks-path

# Ensure go.work file
go.work:
	@echo "〉initializing go work"
	@go work init
	@go work use -r .
	@go work sync

### Tasks

.PHONY: check
## Run lint & tests
check: tidy generate lint test.race audit

.PHONY: lint
## Run linter
lint:
	@echo "〉golangci-lint run"
	@$(foreach mod,$(GOMODS), (cd $(dir $(mod)) && echo "📂 $(dir $(mod))" && golangci-lint run) &&) true

.PHONY: lint.fix
## Fix lint violations
lint.fix:
	@echo "〉golangci-lint run fix"
	@$(foreach mod,$(GOMODS), (cd $(dir $(mod)) && echo "📂 $(dir $(mod))" && golangci-lint run --fix) &&) true

.PHONY: generate
## Run go generate
generate:
	@echo "〉go generate"
	@go generate work

.PHONY: test
## Run tests
test:
	@echo "〉go test"
	@GO_TEST_TAGS=-skip go test -coverprofile=coverage.out -tags=safe work

.PHONY: test.race
## Run tests with -race
test.race:
	@echo "〉go test -race"
	@GO_TEST_TAGS=-skip go test -coverprofile=coverage.out -tags=safe -race work

.PHONY: test.update
## Run tests with -update
test.update:
	@echo "〉go test -update"
	@GO_TEST_TAGS=-skip go test -tags=safe -update -coverprofile=coverage.out -update work

### Dependencies

.PHONY: tidy
## Run go mod tidy (root + submodules)
tidy:
	@echo "〉go mod tidy"
	@$(foreach mod,$(GOMODS), (cd $(dir $(mod)) && echo "📂 $(dir $(mod))" && go mod tidy) &&) true
	@go work sync

.PHONY: outdated
## Show outdated direct dependencies
outdated:
	@echo "〉go mod outdated"
	@go list -u -m -json all | go-mod-outdated -update -direct

.PHONY: upgrade
## Show outdated direct dependencies
upgrade:
	@echo "〉go mod upgrade"
	@go list -u -m -f '{{if and (not .Indirect) .Update}}{{.Path}}{{end}}' all | xargs -n1 -I{} go get {}@latest
	@$(MAKE) tidy

### Example

.PHONY: example
## Run the example (no Docker required)
example:
	@echo "〉running example"
	@cd example && go run main.go

### Kind integration

.PHONY: test.kind
## Run kind integration tests (cluster + image + manifests + go test)
test.kind: kind.up kind.deploy
	@echo "〉go test (kind)"
	@GO_TEST_TAGS=integration go test -C test/kind -tags=safe,kind -count=1 ./...

.PHONY: kind.up
## Create the kind cluster if it doesn't already exist
kind.up:
	@if ! kind get clusters | grep -qx "$(KIND_CLUSTER)"; then \
		echo "〉kind create cluster $(KIND_CLUSTER)"; \
		kind create cluster --name="$(KIND_CLUSTER)" --config=test/kind/kind-config.yaml; \
	else \
		echo "〉kind cluster $(KIND_CLUSTER) already exists"; \
	fi

.PHONY: kind.deploy
## Build the example image, load it into kind, apply CRD + manifests
kind.deploy:
	@echo "〉docker build $(EXAMPLE_IMAGE)"
	@docker build -f example/Dockerfile -t $(EXAMPLE_IMAGE) .
	@echo "〉kind load docker-image"
	@kind load docker-image $(EXAMPLE_IMAGE) --name="$(KIND_CLUSTER)"
	@echo "〉apply CRD + wait for Established"
	@kubectl apply -f config/crd/foomo.org_gateways.yaml
	@kubectl wait --for=condition=Established crd/gateways.foomo.org --timeout=30s
	@echo "〉apply backend manifests"
	@kubectl apply -f test/kind/manifests/
	@kubectl -n default wait --for=condition=Available --timeout=120s \
		deployment/contentserver deployment/pagefrontend \
		deployment/newsfrontend deployment/errorfrontend

.PHONY: test.kind.down
## Delete the kind cluster created by test.kind
test.kind.down:
	@kind delete cluster --name="$(KIND_CLUSTER)"

### Documentation

.PHONY: docs
## Open docs
docs:
	@echo "〉starting docs"
	@cd docs && bun install && bun run dev

.PHONY: docs.build
## Open docs
docs.build:
	@echo "〉building docs"
	@cd docs && bun install && bun run build

.PHONY: godocs
## Open go docs
godocs:
	@echo "〉starting go docs"
	@go doc -http

### Utils

.PHONY: help
# https://patorjk.com/software/taag/#p=display&f=Tmplr&t=gateway&x=none&v=4&h=4&w=80&we=false
## Show help text
help: g=\033[0;32m
help: b=\033[0;34m
help: w=\033[0;90m
help: e=\033[0m
help:
	@echo "$(g)"
	@echo "┏┓┏┓╋┏┓┓┏┏┏┓┓┏"
	@echo "┗┫┗┻┗┗ ┗┻┛┗┻┗┫"
	@echo " ┛           ┛"
	@echo "with ❤ foomo by bestbytes"
	@echo "$(e)"
	@echo "$(b)Usage:$(e)\n  make [task]"
	@awk '{ \
		if($$0 ~ /^### /){ \
			if(help) printf "  %-21s $(w)%s$(e)\n\n", cmd, help; help=""; \
			printf "$(b)\n%s:$(e)\n", substr($$0,5); \
		} else if($$0 ~ /^[a-zA-Z0-9._-]+:/){ \
			cmd = substr($$0, 1, index($$0, ":")-1); \
			if(help) printf "  %-21s $(w)%s$(e)\n", cmd, help; help=""; \
		} else if($$0 ~ /^##/){ \
			help = help ? help "\n                        " substr($$0,3) : substr($$0,3); \
		} else if(help){ \
			print "\n                        $(w)" help "$(e)\n"; help=""; \
		} \
	}' $(MAKEFILE_LIST)
	@echo ""

