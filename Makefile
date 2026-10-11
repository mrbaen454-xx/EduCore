.PHONY: run build test test-race fmt vet db-up db-down db-version db-compose-up db-compose-down

# Default port
APP_PORT ?= 8080

run:
	go run cmd/api/main.go

build:
	go build -o bin/educore.exe cmd/api/main.go

test:
	go test -v ./...

test-race:
	go test -race ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

db-up:
	go run ./cmd/migrate up

db-down:
	go run ./cmd/migrate down -steps 1

db-version:
	go run ./cmd/migrate version

db-compose-up:
	docker compose up -d postgres

db-compose-down:
	docker compose down

