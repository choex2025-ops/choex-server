.PHONY: run build test

run:
	go run cmd/server/main.go

build:
	go build -o bin/server cmd/server/main.go

test:
	go test ./... -v

lint:
	golangci-lint run ./...
