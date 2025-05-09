SHELL=/bin/bash
BINARY_NAME=to-dotxt
MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_ROOT := $(dir $(MAKEFILE_PATH))

build:
	cd $(PROJECT_ROOT) && go build -o $(BINARY_NAME) main.go

clean:
	cd $(PROJECT_ROOT) && go clean && rm -f $(BINARY_NAME)

.PHONY: build clean
