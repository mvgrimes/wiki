# Wiki

Wiki is a Go web application scaffold with a CLI workflow modeled after the sharefile template conventions.

## Project Layout

```text
cmd/                  # Cobra CLI commands
internal/
  server/             # Echo server setup and routes
  web/                # templates and web assets
    templates/pages/  # URL-oriented page templates (e.g. "/" -> pages/home)
migrations/           # goose migrations
sql/                  # schema and sqlc query definitions
```

## HTTP and Template Routing

- `GET /healthz` and `GET /readyz` are unauthenticated health endpoints.
- `GET /` renders the `pages/home` template with the base `layout` template.
- Static assets are served from `/assets/*` using `internal/web/assets`.
- Template loading strategy:
  - development (`WIKI_ENVIRONMENT=development`): load templates from disk (`internal/web/templates`) for fast edit/reload.
  - non-development: load templates from embedded filesystem for reproducible deploys.

## Quickstart

1) Install dependencies:

```bash
go mod tidy
```

2) Run migrations:

```bash
go run ./... migrate up
```

3) Bootstrap local dev state:

```bash
go run ./... bootstrap
```

4) Start server:

```bash
go run ./... server
```

5) Run tests:

```bash
go test ./...
```

## Developer Commands

- `go run ./... server`
- `go run ./... migrate up`
- `go run ./... migrate down`
- `go run ./... migrate status`
- `go run ./... bootstrap`
- `go test ./...`
- `go vet ./...`
- `golangci-lint run ./...`
- `sqlc generate`

You can also use `just` shortcuts from `justfile` for common workflows.
