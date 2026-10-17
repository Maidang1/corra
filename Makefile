.PHONY: dev build test

# MyGo defaults to CGO_ENABLED=0; AppKit rich editing requires cgo on macOS.
ifeq ($(shell go env GOOS),darwin)
export CGO_ENABLED := 1
endif

dev:
	go tool mygo dev

build:
	go tool mygo build

test:
	go test ./...

# AI-friendly file read/write CLI (no cgo needed).
.PHONY: aitool
aitool:
	CGO_ENABLED=0 go build -o bin/aitool ./cmd/aitool
