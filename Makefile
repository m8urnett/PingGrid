BINARY_NAME := PingGrid

.PHONY: build run test test-race test-coverage vet lint fmt tidy deps install clean check

build:
	go build -o $(BINARY_NAME) .

run:
	go run .

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

install:
	go install .

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe coverage.out coverage.html

check: fmt vet lint test
