# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`github.com/foomo/gateway` is a Go library (no binaries) for creating application gateways. Go 1.25, module-only. Uses `github.com/stretchr/testify` for testing.

## Common Commands

```bash
make test          # Run tests (uses -tags=safe, outputs coverage.out)
make test.race     # Run tests with race detector
make test.update   # Run tests with -update flag (golden/snapshot updates)
make lint          # Run golangci-lint
make lint.fix      # Run golangci-lint with --fix
make fmt           # Format code (golangci-lint fmt)
make tidy          # go mod tidy
make generate      # go generate ./...
```

Run a single test: `go test -tags=safe -run TestName ./path/to/package`

Prerequisites: `mise` (tool manager) and `lefthook` (git hooks) must be installed. Run `mise install` to set up tool versions.

## Linting

- golangci-lint v2 config with `default: all` (all linters enabled, specific ones disabled)
- Build tag `safe` is required for both linting and testing
- Formatters: `gofmt` and `goimports`
- Pre-commit hook auto-formats staged `.go` files via `golangci-lint fmt`

## CI Pipeline

CI runs on push to `main` and PRs:
1. `make tidy` + `make generate` + `make fmt` — must produce no diffs
2. `make lint`
3. `make test`

## Conventions

- Commit messages follow Conventional Commits (`feat:`, `fix:`, `docs:`, `refactor:`, etc.)
- Releases triggered by `v*.*.*` tags via goreleaser (library-only, no binaries)
- Documentation site uses VitePress + Bun (`make docs` to serve locally)
