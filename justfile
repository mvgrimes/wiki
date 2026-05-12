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

sqlc:
  sqlc generate
