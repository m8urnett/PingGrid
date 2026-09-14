BINARY_NAME := pg
ifeq ($(OS),Windows_NT)
  GIT_COMMIT := $(shell git rev-parse --short HEAD 2>NUL || echo dev)
  BUILD_DATE := $(shell git show -s --format=%%cI HEAD 2>NUL || echo unknown)
else
  GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
  BUILD_DATE := $(shell git show -s --format=%cI HEAD 2>/dev/null || echo unknown)
endif

LDFLAGS := -ldflags "\
	-s -w \
	-X github.com/m8urnett/PingGrid/internal/version.GitCommit=$(GIT_COMMIT) \
	-X github.com/m8urnett/PingGrid/internal/version.BuildDate=$(BUILD_DATE)"

.PHONY: build prepare-windows-resource build-windows build-linux build-darwin build-all package-source run test test-race test-coverage vet lint fmt tidy deps verify-deps vuln check ci audit install clean

build: build-windows

prepare-windows-resource:
	windres pg_windows_amd64.rc -O coff -o resource_windows_amd64.syso

build-windows: prepare-windows-resource
	go build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME).exe .

build-linux:
ifeq ($(OS),Windows_NT)
	cmd /c "set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ."
	cmd /c "set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=arm64&& go build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ."
	cmd /c "set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=arm&& set GOARM=7&& go build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-armv7 ."
else
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-armv7 .
endif

build-darwin:
ifeq ($(OS),Windows_NT)
	cmd /c "set CGO_ENABLED=0&& set GOOS=darwin&& set GOARCH=arm64&& go build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ."
	cmd /c "set CGO_ENABLED=0&& set GOOS=darwin&& set GOARCH=amd64&& go build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ."
else
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 .
endif

build-all: build-windows build-linux build-darwin package-source

package-source:
	git archive --format=zip --prefix=PingGrid-source/ -o bin/PingGrid-source.zip HEAD

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
	go fmt ./...
	-goimports -w .

tidy:
	go mod tidy

deps:
	go mod download


verify-deps: deps
	go mod tidy -diff
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

ci: verify-deps fmt vet lint test test-race vuln

audit: ci
