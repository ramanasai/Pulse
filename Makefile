.PHONY: build build-local run fmt tidy clean test release install dev

BINARY := bin/pulse

# Package path where version vars are defined
PKG := github.com/ramanasai/pulse

# ldflags for version metadata
LDFLAGS := -s -w \
	-X '$(PKG).buildVersion=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)' \
	-X '$(PKG).buildCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo none)' \
	-X '$(PKG).buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)'

all: build-local


build-local:
	@mkdir -p bin
	GOFLAGS="-trimpath" go build -ldflags "$(LDFLAGS)" -o $(BINARY) .
	@chmod +x $(BINARY)
	@echo "✅ Built local binary at $(BINARY)"

build:
	@mkdir -p bin
	GOFLAGS="-trimpath" CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) .
	@chmod +x $(BINARY)
	@echo "✅ Built static binary at $(BINARY)"

# Run directly (dev mode)
run:
	go run .

# Format code
fmt:
	gofmt -s -w .

# Tidy modules
tidy:
	go mod tidy

# Unit tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin dist

# Release cross-platform (requires goreleaser or xgo, here use simple Go cross builds)
release: clean
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/pulse-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/pulse-darwin-amd64 .
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/pulse-linux-amd64 .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/pulse-linux-arm64 .
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/pulse-windows-amd64.exe .


# Install locally
install: build-local
	@echo "📦 Installing $(BINARY) to /usr/local/bin..."
	@sudo cp $(BINARY) /usr/local/bin/pulse
	@echo "✅ Installed pulse to /usr/local/bin"

# Development (build with race detector)
dev:
	@mkdir -p bin
	go build -race -o $(BINARY)-dev .
	@chmod +x $(BINARY)-dev
	@echo "🔧 Built development binary at $(BINARY)-dev"

snapshot:
	goreleaser release --snapshot --clean

# # Real release (publishes GitHub release; normally done by CI)
# release:
# 	goreleaser release --clean