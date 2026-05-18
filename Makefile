BINARY := monkeytype-tui
BIN_DIR := ./bin

.PHONY: build run test test-race lint fmt clean

build:
	go build -o $(BIN_DIR)/$(BINARY) .

run:
	go run .

test:
	go test ./...

test-race:
	go test ./... -race -count=1

lint:
	@echo "==> gofmt"
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "$$out"; exit 1; fi
	@echo "==> go vet"
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -rf $(BIN_DIR)
