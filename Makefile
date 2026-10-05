BIN    := stomp-phone
GOOS   := linux
GOARCH := amd64

.PHONY: all help build lint tidy clean

all: help

help:
	@echo "build       build $(BIN)"
	@echo "lint        golangci-lint fmt + golangci-lint run"
	@echo "tidy        go mod tidy"
	@echo "clean       remove build artifacts"

build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags="-s -w" -trimpath -o $(BIN) .
	upx --best --lzma $(BIN)

lint:
	golangci-lint fmt
	golangci-lint run

tidy:
	go mod tidy

clean:
	rm -vf $(BIN)
