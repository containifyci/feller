.PHONY: lint test build

lint:
	golangci-lint run -v --fix ./...

test:
	go test ./...

build:
	go build -o feller main.go