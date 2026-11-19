VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)

.PHONY: build test lint release verify

verify:
	@if grep -RInE 'net\.Listen|ListenAndServe|http\.Listen|opentelemetry|prometheus|sentry' --include='*.go' cmd internal; then \
		echo "forbidden runtime/network/telemetry code found"; exit 1; \
	fi

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/db-adm-tui ./cmd/db-adm-tui

test:
	go test -race ./...

lint:
	go vet ./...

release: verify
	rm -rf dist
	mkdir -p dist/db-adm-tui_$(VERSION)_linux_amd64
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o dist/db-adm-tui_$(VERSION)_linux_amd64/db-adm-tui ./cmd/db-adm-tui
	tar -C dist/db-adm-tui_$(VERSION)_linux_amd64 -czf dist/db-adm-tui_$(VERSION)_linux_amd64.tar.gz db-adm-tui
	mkdir -p dist/db-adm-tui_$(VERSION)_linux_arm64
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o dist/db-adm-tui_$(VERSION)_linux_arm64/db-adm-tui ./cmd/db-adm-tui
	tar -C dist/db-adm-tui_$(VERSION)_linux_arm64 -czf dist/db-adm-tui_$(VERSION)_linux_arm64.tar.gz db-adm-tui
