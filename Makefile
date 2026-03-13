VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY  := dirloc
LDFLAGS := -ldflags "-X github.com/dirloc/dirloc/cmd.Version=$(VERSION)"

.PHONY: build install test lint clean

build:
	go build $(LDFLAGS) -o $(BINARY) .

install:
	go install $(LDFLAGS) .

test:
	go test ./... -v

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)
