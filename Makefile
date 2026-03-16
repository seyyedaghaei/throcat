.PHONY: build test install lint

BINARY := throcat
PKG := ./cmd/throcat

build:
	go build -o $(BINARY) $(PKG)

test:
	go test ./...

install:
	go install $(PKG)

lint:
	golangci-lint run ./...
