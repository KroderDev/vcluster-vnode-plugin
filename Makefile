.PHONY: tidy vet test lint build docker-build clean

all: tidy vet lint test build

tidy:
	go mod tidy

vet:
	go vet ./...

test:
	go test ./... -v -coverprofile=coverage.out -covermode=atomic

lint:
	golangci-lint run ./...

build:
	go build -o bin/

docker-build:
	docker build -t ghcr.io/kroderdev/vcluster-vnode-plugin:dev -f Dockerfile .

clean:
	rm -f bin/