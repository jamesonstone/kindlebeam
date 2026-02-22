BINARY := kindlebeam

.PHONY: build run test clean fmt lint

build:
	go build -o ./bin/kindlebeam/$(BINARY) .

run:
	go run .

test:
	go test ./...

clean:
	rm -f $(BINARY)

fmt:
	gofmt -w ./

lint:
	golangci-lint run ./... || true
