.PHONY: build test fmt

build:
	go build -o ./bin/temporal-start_dev ./cmd/temporal-start_dev

test:
	go test ./...

fmt:
	go fmt ./...
