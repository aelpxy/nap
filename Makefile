.PHONY: build run clean install release

BINARY_NAME=yap
VERSION ?= 0.1.0-alpha
BUILD_TIME=$(shell date -u '+%Y-%m-%d-%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)

build:
	@echo "building $(VERSION)..."
	@go build -o $(BINARY_NAME) .
	@echo "build complete: ./$(BINARY_NAME)"

build-prod:
	@echo "building binary $(VERSION)..."
	@go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) .
	@ls -lh $(BINARY_NAME)
	@echo "build complete: ./$(BINARY_NAME)"

release:
	@./build.sh

run:
	@go run main.go

clean:
	@echo "cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf dist/
	@echo "Clean complete"

install:
	@echo "installing..."
	@go install
	@echo "install complete"