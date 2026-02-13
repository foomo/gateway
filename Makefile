.DEFAULT_GOAL:=help
-include .makerc

# --- Config -----------------------------------------------------------------

# Newline hack for error output
define br


endef

# --- Targets -----------------------------------------------------------------

# This allows us to accept extra arguments
%: .mise .lefthook
	@:

.PHONY: .lefthook
# Configure git hooks for lefthook
.lefthook:
	@lefthook install --reset-hooks-path

.PHONY: .mise
# Install dependencies
.mise:
ifeq (, $(shell command -v mise))
	$(error $(br)$(br)Please ensure you have 'mise' installed and activated!$(br)$(br)  $$ brew update$(br)  $$ brew install mise$(br)$(br)See the documentation: https://mise.jdx.dev/getting-started.html)
endif
	@mise install
.PHONY: doc

### Tasks

.PHONY: tidy
## Run go mod tidy
tidy:
	@echo "〉running go mod tidy"
	@go mod tidy

.PHONY: generate
## Run go generate
generate:
	@echo "〉running go generate"
	@go generate ./...

.PHONY: test
## Run tests
test:
	@echo "〉running tests"
	@GO_TEST_TAGS=-skip go test -tags=safe -coverprofile=coverage.out ./...

.PHONY: test.race
## Run tests with `-race` flag
test.race:
	@echo "〉running tests with -race flag"
	@GO_TEST_TAGS=-skip,race go test -tags=safe -coverprofile=coverage.out -update ./...

.PHONY: test.update
## Run tests with `-update` flag
test.update:
	@echo "〉running tests with -update flag"
	@GO_TEST_TAGS=-skip go test -tags=safe -update -coverprofile=coverage.out -update ./...

.PHONY: fmt
## Run format
fmt:
	@echo "〉formatting files"
	@golangci-lint fmt ./...

.PHONY: lint
## Run linter
lint:
	@echo "〉linting files"
	@golangci-lint run

.PHONY: lint.fix
## Fix lint violations
lint.fix:
	@echo "〉fixing lint errors"
	@golangci-lint run --fix

.PHONY: outdated
## Show outdated direct dependencies
outdated:
	@echo "〉listing outdated dependencies"
	@go list -u -m -json all | go-mod-outdated -update -direct

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
## Show help text
help:
	@echo ""
	@echo "Welcome to the gateway!"
	@echo "\nUsage:\n  make [task]"
	@awk '{ \
		if($$0 ~ /^### /){ \
			if(help) printf "%-23s %s\n\n", cmd, help; help=""; \
			printf "\n%s:\n", substr($$0,5); \
		} else if($$0 ~ /^[a-zA-Z0-9._-]+:/){ \
			cmd = substr($$0, 1, index($$0, ":")-1); \
			if(help) printf "  %-23s %s\n", cmd, help; help=""; \
		} else if($$0 ~ /^##/){ \
			help = help ? help "\n                          " substr($$0,3) : substr($$0,3); \
		} else if(help){ \
			print "\n                        " help "\n"; help=""; \
		} \
	}' $(MAKEFILE_LIST)
