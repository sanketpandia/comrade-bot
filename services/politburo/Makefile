.PHONY: build run test fmt generate ci

REPO_ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/../..)

build:
	go build -buildvcs=false ./cmd/politburo

run:
	go run ./cmd/politburo

test:
	go test ./...

ci: generate test build

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './internal/api/generated/*')

generate:
	$(MAKE) -C $(REPO_ROOT)/cicd generate-politburo generate-infinite-flight
