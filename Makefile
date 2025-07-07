SHELL=/bin/bash
BINARY_NAME=dotxt
MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_ROOT := $(dir $(MAKEFILE_PATH))

test:
	cd $(PROJECT_ROOT) && go test ./...

build:
	cd $(PROJECT_ROOT) && go build -o $(BINARY_NAME) main.go

clean:
	cd $(PROJECT_ROOT) && go clean && rm -f $(BINARY_NAME)

clean-cache:
	cd $(PROJECT_ROOT) && go clean -cache && go clean -testcache

.PHONY: test build clean clean-cache
