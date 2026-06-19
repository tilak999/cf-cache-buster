# Traefik Cache Purge Plugin

[![Build Status](https://github.com/tilak999/traefikplugin/workflows/Main/badge.svg?branch=master)](https://github.com/tilak999/traefikplugin/actions)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.22-blue)

A [Traefik](https://traefik.io) middleware plugin that detects specific response headers from your backend and automatically triggers a [Cloudflare cache purge](https://developers.cloudflare.com/cache/how-to/purge-cache/purge-by-hostname/) for the requested host.

## How It Works

```
Client → Traefik → [This Plugin] → Backend
                         │
                         ├─ Intercepts the response
                         ├─ Checks for configured headers
                         └─ If found → async Cloudflare cache purge
```

1. A request passes through Traefik to your backend service.
2. The plugin inspects the **response headers** from the backend.
3. If any of the configured headers are present, the plugin fires an **asynchronous** Cloudflare API call to purge the cache for that host.
4. The response is passed through to the client unmodified.

This is useful when your backend signals that content has changed (e.g., via an `x-invalidate-cache` header) and the CDN cache should be refreshed.

## Configuration Reference

| Field             | Type       | Required | Default | Description                                                                 |
|-------------------|------------|----------|---------|-----------------------------------------------------------------------------|
| `headers`         | `[]string` | ✅       | `[]`    | Response headers to watch for. If any are present, a cache purge is fired.  |
| `cloudflarezone`  | `string`   | ✅       | `""`    | Your Cloudflare Zone ID.                                                    |
| `cloudflaretoken` | `string`   | ✅       | `""`    | Cloudflare API token with cache purge permissions.                          |
| `dryrun`          | `bool`     | ❌       | `false` | When `true`, logs detected headers and API responses without suppressing purge. Useful for debugging. |

## Installation

Add the plugin to your Traefik **static configuration**:

```yaml
# Static configuration
experimental:
  plugins:
    cachePurge:
      moduleName: github.com/tilak999/traefikplugin
      version: v0.0.7  # Use the latest tag
```

Then configure it as middleware in your **dynamic configuration**:

```yaml
# Dynamic configuration
http:
  middlewares:
    cache-purge:
      plugin:
        cachePurge:
          cloudflarezone: "your-zone-id"
          cloudflaretoken: "your-api-token"
          headers:
            - x-invalidate-cache
          dryrun: false

  routers:
    my-router:
      rule: Host(`example.com`)
      service: my-service
      entryPoints:
        - web
      middlewares:
        - cache-purge

  services:
    my-service:
      loadBalancer:
        servers:
          - url: http://127.0.0.1:5000
```

## Docker Compose Quick Start

A ready-to-use `docker-compose.yaml` is included for local development and testing. It sets up:

- **Traefik v3** with the plugin loaded in local mode
- **whoami** as a test backend service

```bash
docker compose up -d
```

The Traefik dashboard is available at `http://localhost:8080`, and the test service at `http://localhost`.

Edit `config/middleware.yaml` to configure the plugin middleware with your Cloudflare credentials.

## Local Development

### Prerequisites

- Go ≥ 1.22
- [golangci-lint](https://golangci-lint.run/) (for linting)
- Docker & Docker Compose (for integration testing)

### Running Tests

```bash
# Run all tests with coverage
make test

# Or directly
go test -v -cover ./...
```

### Linting

```bash
make lint
```

### Local Plugin Mode

To test with Traefik locally, place the plugin source at:

```
./plugins-local/src/github.com/tilak999/traefikplugin/
```

And use the local plugin configuration:

```yaml
# Static configuration
experimental:
  localPlugins:
    headerDetection:
      moduleName: github.com/tilak999/traefikplugin
```

The included `docker-compose.yaml` handles this automatically by volume-mounting the project directory.

## License

[Apache License 2.0](LICENSE)
