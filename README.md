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
- `pnpm test:js`

You can also use `just` shortcuts from `justfile` for common workflows.

## Test Matrix

The test strategy is organized by layer so contributors can choose fast local checks or full validation gates:

- unit: package-level behavior checks for CLI, auth middleware, server routes, and config (`just test-unit`)
- integration: DB + migration behavior against an ephemeral sqlite database (`just test-integration`)
- e2e-lite: route-level HTTP behavior through server tests (`go test ./internal/server/...`)

## Quality Gates

Run `just quality-gate` before marking feature work complete. This gate enforces:

- static checks: `just lint`
- SQL generation validity: `just sqlc`
- migration command health: `just goose-check`
- required tests: `just test-unit`, `just test-integration`, `just test-auth`, `just test-migrations`

Coverage targets for critical packages:

- `internal/auth`: >= 80% (`go test -cover ./internal/auth/...`)
- `internal/db` migration/integration path: >= 60% (`go test -cover ./internal/db/...`)
- wiki service route path (`internal/server`): >= 70% (`go test -cover ./internal/server/...`)

Auth and migration-related tickets must not be closed if `just test-auth` or `just test-migrations` fail.

## Release Validation Checklist

Before a release cut, validate:

- security: auth middleware behavior, access control checks, dependency/audit review
- regressions: run `just quality-gate` and manually verify core wiki edit/upload paths
- database safety: run migration status and test migration path from clean DB

## Wiki Edit UI Manual Checklist

- Open `/wiki/<slug>/edit` and confirm the live preview updates while typing (no full reload).
- Use the file picker to upload an image and a non-image file; verify progress updates and success state.
- Drag and drop a file onto the upload dropzone; verify success state and inserted markdown snippet.
- Force an upload error (for example, with an unsupported file type) and verify the error message appears.
- Repeat the picker and drag/drop checks in a mobile browser simulator and confirm behavior remains usable.
