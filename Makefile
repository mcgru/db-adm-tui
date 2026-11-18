.PHONY: build test lint

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/db-adm-tui ./cmd/db-adm-tui

test:
	go test -race ./...

lint:
	go vet ./...
