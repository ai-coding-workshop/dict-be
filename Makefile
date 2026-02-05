.PHONY: build run fmt vet test tidy clean

BINARY_NAME := dict-be
CMD_PATH := ./cmd/$(BINARY_NAME)

build:
	go build -o $(BINARY_NAME) $(CMD_PATH)

run:
	go run $(CMD_PATH)

fmt:
	gofmt -w $$(go list -f '{{.Dir}}' ./...)

vet:
	go vet ./...

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -f $(BINARY_NAME)
