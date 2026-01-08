.PHONY: build install clean test run release

BINARY_NAME=obsidiantui
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

install: build
	cp $(BINARY_NAME) $(HOME)/.local/bin/

clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/

test:
	go test -v ./...

run: build
	./$(BINARY_NAME)

# Cross compilation
build-all: build-darwin build-linux build-windows

build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .

build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 .

build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .

# Release with goreleaser
release:
	goreleaser release --clean

release-snapshot:
	goreleaser release --snapshot --clean
