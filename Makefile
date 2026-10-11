.PHONY: run build test test-race fmt vet

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

