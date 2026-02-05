.PHONY: build run fmt vet test tidy clean release

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

release:
	mkdir -p dist/linux-amd64 dist/macos-arm64 dist/windows-amd64
	GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/$(BINARY_NAME) $(CMD_PATH)
	GOOS=darwin GOARCH=arm64 go build -o dist/macos-arm64/$(BINARY_NAME) $(CMD_PATH)
	GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/$(BINARY_NAME).exe $(CMD_PATH)
