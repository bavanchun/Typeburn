BINARY := typeburn
BIN_DIR := ./bin

# Version metadata injected into internal/version via -ldflags -X. Mirrors the
# ldflags in .goreleaser.yaml so local `make build` binaries report the same
# shape as released ones. VERSION falls back to "dev" outside a git checkout.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MODULE := github.com/bavanchun/Typeburn/internal/version
LDFLAGS := -s -w \
	-X $(MODULE).Version=$(VERSION) \
	-X $(MODULE).Commit=$(COMMIT) \
	-X $(MODULE).Date=$(DATE)

.PHONY: build run test test-race lint fmt clean version snapshot release

build:
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(BINARY) .

run:
	go run .

test:
	go test ./...

test-race:
	go test ./... -race -count=1

lint:
	@echo "==> gofmt"
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "$$out"; exit 1; fi
	@echo "==> go vet"
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -rf $(BIN_DIR)

# Build then print the resolved banner — quick check that ldflags landed.
version: build
	@$(BIN_DIR)/$(BINARY) --version

# Local dry-run: builds + archives into dist/. Proves the build/archive/ldflags
# path ONLY (no publish/auth/release-notes — that is Phase 5's disposable-tag run).
snapshot:
	goreleaser release --snapshot --clean

# Full publish. CI-only (release.yml); requires a tag + GITHUB_TOKEN.
release:
	goreleaser release --clean
