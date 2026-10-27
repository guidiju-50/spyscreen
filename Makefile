.PHONY: build-macos test test-race clean

BIN := bin/screenshare
MAIN := ./cmd/screenshare

build-macos:
	@./scripts/build-macos.sh

test:
	go test ./...

test-race:
	go test -race ./...

clean:
	rm -rf bin/
