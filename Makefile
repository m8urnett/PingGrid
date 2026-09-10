BINARY_NAME := pg
VERSION := 1.0.0
BUILD_NUM := 46
ifeq ($(OS),Windows_NT)
  GIT_COMMIT := $(shell git rev-parse --short HEAD 2>NUL || echo dev)
  BUILD_DATE := $(shell powershell -NoProfile -Command "(Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')" 2>NUL || echo unknown)
else
  GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
  BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || echo unknown)
endif

LDFLAGS := -ldflags "\
	-s -w \
	-X github.com/m8urnett/PingGrid/internal/version.GitCommit=$(GIT_COMMIT) \
	-X github.com/m8urnett/PingGrid/internal/version.BuildDate=$(BUILD_DATE)"

.PHONY: build build-windows build-linux build-darwin build-all run test test-race test-coverage vet lint fmt tidy deps verify-deps vuln check ci audit install clean

build: build-windows

build-windows:
	go build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME).exe .

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 .

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 .

build-all: build-windows build-linux build-darwin

run:
	go run -trimpath $(LDFLAGS) .

test:
	go test ./...

test-race:
	go test -race ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

vet:
	go vet ./...

lint:
	golangci-lint run

fmt:
	gofmt -s -w .
	-goimports -w .

tidy:
	go mod tidy

deps:
	go mod download

verify-deps: tidy deps
	go mod verify

vuln:
	govulncheck ./...

install:
	go install $(LDFLAGS) .

clean:
ifeq ($(OS),Windows_NT)
	-@del /q /f $(BINARY_NAME).exe bin\$(BINARY_NAME)* coverage.out coverage.html grid.png 2>NUL || exit 0
else
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe bin/$(BINARY_NAME)* coverage.out coverage.html grid.png
endif

check: fmt vet test

ci: verify-deps fmt vet lint test vuln

audit: ci
