# AGENTS.md

Guidance for AI agents and contributors working on this repository.

## Repository Overview

This is a [Traefik](https://traefik.io) middleware plugin written in Go. It intercepts HTTP responses, checks for configured headers, and triggers a Cloudflare cache purge when those headers are detected. Traefik loads the plugin at runtime via [Yaegi](https://github.com/traefik/yaegi), a Go interpreter — the code is **not compiled** into a binary.

## Directory Layout

```
.
├── main.go              # Plugin entry point: Config, CreateConfig(), New(), ServeHTTP()
├── cf-helper.go         # Cloudflare API integration (PurgeCache function)
├── main_test.go         # Unit tests
├── go.mod               # Go module definition (no external dependencies)
├── .traefik.yml         # Traefik plugin catalog manifest
├── .golangci.yml        # Linter configuration
├── Makefile             # Build/test/lint targets
├── docker-compose.yaml  # Local dev environment (Traefik + whoami)
├── config/
│   └── middleware.yaml  # Dynamic middleware config for local testing
├── .assets/
│   └── icon.png         # Plugin icon for Traefik catalog
├── .github/workflows/
│   ├── main.yml         # CI: lint + test + yaegi validation
│   └── go-cross.yml     # Cross-platform test matrix
├── readme.md            # Project README
└── LICENSE              # Apache 2.0
```

## Build, Test, and Lint Commands

```bash
# Run linter and tests (default target)
make

# Run tests only
make test
# Or: go test -v -cover ./...

# Run linter only
make lint

# Run Yaegi interpreter tests (requires yaegi installed)
make yaegi_test

# Start local dev environment
docker compose up -d
```

## Coding Conventions

### Traefik Plugin Constraints

These are **mandatory** for the plugin to be loadable by Traefik/Yaegi:

- The package must export exactly these symbols:
  - `type Config struct { ... }`
  - `func CreateConfig() *Config`
  - `func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error)`
- **No external dependencies** — Yaegi can only interpret stdlib. All imports must be from the Go standard library.
- The `go.mod` must exist at the repo root with the correct module path (`github.com/tilak999/cf-cache-buster`). The Go package name is `cf_cache_buster`.
- Dependencies, if any, must be vendored (though currently there are none).

### Go Style

- Follow standard Go conventions (`gofmt`, `go vet`).
- The `.golangci.yml` enables nearly all linters — run `make lint` before committing.
- Use `log.Logger` for logging (no `fmt.Println` or `os.Stdout.WriteString`).
- Error messages should be lowercase, without trailing punctuation.
- Internal types (not part of the plugin API) should be unexported.

### Error Handling

- Never use `log.Fatal` or `os.Exit` — these kill the entire Traefik process.
- Return errors from `New()` for configuration problems.
- Log and return from goroutines (like `PurgeCache`) — don't panic.

### HTTP Client Usage

- Use a dedicated `http.Client` with a timeout, not `http.DefaultClient`.
- Use `http.NewRequestWithContext` for cancellation support.
- Always close response bodies in a defer.

## CI Pipelines

### `main.yml` (Main Process)
Runs on push to `master` and pull requests. Executes lint, unit tests, and Yaegi interpreter validation.

### `go-cross.yml` (Go Matrix)
Cross-platform test matrix across Ubuntu, macOS, and Windows with Go 1.22 and latest.

## Key Files to Understand

1. **[main.go](main.go)** — Start here. Contains the plugin's public API (`Config`, `CreateConfig`, `New`) and the `ServeHTTP` middleware logic.
2. **[cf-helper.go](cf-helper.go)** — The Cloudflare API call. Called asynchronously from `ServeHTTP` when headers are detected.
3. **[.traefik.yml](.traefik.yml)** — Plugin manifest. The `testData` section must contain valid config that allows `New()` to succeed, or the plugin will be rejected by the Traefik catalog.

## Testing

- Tests are in `main_test.go` using `httptest.NewRecorder`.
- The test package is `cf_cache_buster_test` (external test package).
- Tests cover: header detection, empty config errors, missing credentials, no-match scenarios, and status code proxying.
- The `PurgeCache` function makes real HTTP calls to Cloudflare — tests that trigger it use dummy credentials that will fail silently.
