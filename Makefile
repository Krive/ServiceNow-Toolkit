.PHONY: all build test lint

all: build test lint

build:
	go build -o bin/servicenowtoolkit ./cmd/servicenowtoolkit

test:
	go test ./... -v

lint:
	golangci-lint run

clean:
	rm -rf bin coverage.out
