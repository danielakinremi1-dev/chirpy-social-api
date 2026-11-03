# DB_URL comes from the environment, falling back to the value in .env.
DB_URL ?= $(shell sed -n 's/^DB_URL=//p' .env 2>/dev/null | tr -d '"')

MIGRATIONS := sql/schema
GOLANGCI_LINT_VERSION := v2.15.0

.PHONY: run build test test-integration cover lint fmt migrate-up migrate-down migrate-status sqlc docker-up docker-down psql

run:
	go run ./cmd/chirpy

build:
	go build -o chirpy ./cmd/chirpy

test:
	go test -race ./...

# Needs Docker: starts Postgres with testcontainers.
test-integration:
	go test -race -tags=integration ./...

cover:
	go test -tags=integration -coverpkg=./internal/... -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

lint:
	@test -z "$$(gofmt -l .)" || (echo "Run 'make fmt' to format:"; gofmt -l .; exit 1)
	go vet -tags=integration ./...
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...

fmt:
	gofmt -w .

migrate-up:
	goose -dir $(MIGRATIONS) postgres "$(DB_URL)" up

migrate-down:
	goose -dir $(MIGRATIONS) postgres "$(DB_URL)" down

migrate-status:
	goose -dir $(MIGRATIONS) postgres "$(DB_URL)" status

sqlc:
	sqlc generate

docker-up:
	docker compose up --build

docker-down:
	docker compose down

psql:
	psql "$(DB_URL)"
