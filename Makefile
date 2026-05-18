.PHONY: build test lint install clean

build:
	go build -o bin/nextcloud-pp-cli ./cmd/nextcloud-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/nextcloud-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -o bin/nextcloud-pp-mcp ./cmd/nextcloud-pp-mcp

install-mcp:
	go install ./cmd/nextcloud-pp-mcp

build-all: build build-mcp
