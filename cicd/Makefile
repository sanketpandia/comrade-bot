.PHONY: generate generate-politburo generate-infinite-flight generate-bot-client \
	check test build ci openapi-view openapi-stop

REPO_ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/..)
GO_TOOL = cd $(REPO_ROOT)/cicd/tools && go tool
POLITBURO_DIR = $(REPO_ROOT)/services/politburo
BOT_DIR = $(REPO_ROOT)/services/comrade-bot-discord
OPENAPI_DIR = $(REPO_ROOT)/openapi
COMPOSE_DEV = $(REPO_ROOT)/infra/dev/docker-compose.dev.yml

generate: generate-politburo generate-infinite-flight generate-bot-client

generate-politburo:
	mkdir -p $(POLITBURO_DIR)/internal/api/generated/politburo
	$(GO_TOOL) oapi-codegen -config $(REPO_ROOT)/cicd/codegen/politburo-oapi-codegen.yaml $(OPENAPI_DIR)/politburo.yaml

generate-infinite-flight:
	mkdir -p $(POLITBURO_DIR)/internal/api/generated/infiniteflight
	$(GO_TOOL) oapi-codegen -config $(REPO_ROOT)/cicd/codegen/infinite-flight-oapi-codegen.yaml $(OPENAPI_DIR)/infinite-flight/openapi.yaml

generate-bot-client:
	cd $(BOT_DIR) && npm run api:generate

check: generate
	cd $(POLITBURO_DIR) && go test ./...

test:
	cd $(POLITBURO_DIR) && go test ./...
	cd $(BOT_DIR) && npm test

build:
	cd $(POLITBURO_DIR) && go build -buildvcs=false ./cmd/politburo
	cd $(BOT_DIR) && npm run build

ci: generate test build

openapi-view: generate-politburo generate-infinite-flight
	docker compose -f $(COMPOSE_DEV) up -d swagger-editor
	@echo "Politburo API: http://localhost:8081"

openapi-stop:
	docker compose -f $(COMPOSE_DEV) stop swagger-editor
