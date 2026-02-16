.PHONY: build test lint clean

build:
	go build ./...

test:
	go test ./core/... ./broker/... ./internal/... -v -race

test-all:
	go test ./... -v -race

lint:
	go vet ./...

clean:
	go clean ./...

tidy:
	go mod tidy
