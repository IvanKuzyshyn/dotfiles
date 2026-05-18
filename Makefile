GO ?= go
PKG := ./...
BIN := bin/dot
VERSION ?= dev

.PHONY: build test test-int test-snapshot snapshot lint fmt fmt-check tidy clean

build:
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o $(BIN) ./cmd/dot

test:
	$(GO) test $(PKG)

test-int:
	$(GO) test -tags=integration $(PKG)

test-snapshot:
	$(GO) test -tags=teatest ./internal/tui/...

snapshot:
	$(GO) test -tags=teatest ./internal/tui/... -update

lint:
	golangci-lint run

fmt:
	gofumpt -w . && goimports -w .

fmt-check:
	@out=$$(gofumpt -l . && goimports -l .); \
	if [ -n "$$out" ]; then echo "$$out"; exit 1; fi

tidy:
	$(GO) mod tidy

clean:
	rm -rf bin dist
