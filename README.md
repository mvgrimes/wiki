# Wiki

Wiki is a Go web application scaffold with a CLI workflow modeled after the sharefile template conventions.

## Project Layout

```text
cmd/                  # Cobra CLI commands
internal/
  server/             # Echo server setup and routes
  web/                # templates and web assets
migrations/           # goose migrations
sql/                  # schema and sqlc query definitions
```

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
