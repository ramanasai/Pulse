.PHONY: build run fmt tidy clean test release

BINARY := bin/pulse
LDFLAGS := -s -w -X 'main.version=$(shell git describe --tags --always --dirty)'

# Default build (static binary, stripped)
build:
	GOFLAGS="-trimpath" CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./...

build-local:
	GOFLAGS="-trimpath" CGO_ENABLED=0 go build -ldflags "-s -w -X 'main.version=v0.1.0-dirty'" -o bin/pulse .

# Run directly (dev mode)
run:
	go run ./...

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


snapshot:
	goreleaser release --snapshot --clean

# # Real release (publishes GitHub release; normally done by CI)
# release:
# 	goreleaser release --clean