.PHONY: clean lint test build

default: clean lint test build

lint:
	golangci-lint run

clean:
	rm -rf cover.out

test: clean
	go test -v -cover ./...

build: clean
	CGO_ENABLED=0 go build -trimpath -ldflags '-w -s'
