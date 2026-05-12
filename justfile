APP := "wiki"

set shell := ["bash", "-euo", "pipefail", "-c"]

build:
  go build -o {{APP}} .

run:
  go run ./... server

migrate:
  go run ./... migrate up

bootstrap:
  go run ./... bootstrap

lint:
  go vet ./...
  golangci-lint run ./...

test:
  go test ./...

test-unit:
  go test ./cmd/... ./internal/auth/... ./internal/config/... ./internal/server/...

test-integration:
  go test ./internal/db/...

test-auth:
  go test -cover ./internal/auth/...

test-migrations:
  go test -run TestMigrations ./internal/db/...

sqlc:
  sqlc generate

goose-check:
  go run ./... migrate status

quality-gate:
  just lint
  just sqlc
  just goose-check
  just test-unit
  just test-integration
  just test-auth
  just test-migrations
