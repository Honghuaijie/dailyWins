GOHOSTOS := $(shell go env GOHOSTOS)
GOPATH   := $(shell go env GOPATH)
VERSION  ?= $(shell git describe --tags --always 2>/dev/null || echo dev)

SERVICE_NAME ?= dailyWins-service
MODULE_NAME  ?= dailyWins-service
BINARY_NAME  ?= $(SERVICE_NAME)
CMD_PATH     ?= ./cmd/$(SERVICE_NAME)
CONFIG_DIR   ?= ./configs
CONFIG_ABS   := $(abspath $(CONFIG_DIR))
OUTPUT_DIR   ?= ./api/all-pb-go
env          ?=
PROTOC_GEN_GO_VERSION ?= v1.36.6
PROTOC_GEN_GO_GRPC_VERSION ?= v1.5.1
PROTOC_GEN_GO_HTTP_VERSION ?= v2.9.0
PROTOC_GEN_OPENAPI_VERSION ?= v0.7.0
WIRE_VERSION ?= v0.7.0
KRATOS_VERSION ?= v2.9.2
PROTOC_GEN_VALIDATE_VERSION ?= v1.2.1

ifeq ($(GOHOSTOS), windows)
	Git_Bash = $(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
	INTERNAL_PROTO_FILES = $(shell $(Git_Bash) -c "find internal -name '*.proto' | sed 's|^internal/||'")
	API_PROTO_FILES      = $(shell $(Git_Bash) -c "find api -path 'api/all-pb-go' -prune -o -name '*.proto' -print | sed 's|^api/||'")
else
	INTERNAL_PROTO_FILES = $(shell find internal -name '*.proto' | sed 's|^internal/||')
	API_PROTO_FILES      = $(shell find api -path 'api/all-pb-go' -prune -o -name '*.proto' -print | sed 's|^api/||')
endif

.PHONY: validate
# validate proto files
validate:
	@if [ -z "$(strip $(API_PROTO_FILES))" ]; then \
		echo "No API proto files found under ./api"; \
		exit 0; \
	fi
	protoc --proto_path=./api \
	       --proto_path=./third_party \
	       --go_out=paths=source_relative:./api \
	       --validate_out=paths=source_relative,lang=go:./api \
	       $(API_PROTO_FILES)

.PHONY: init
# init env tools
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)
	go install github.com/go-kratos/kratos/cmd/kratos/v2@$(KRATOS_VERSION)
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@$(PROTOC_GEN_GO_HTTP_VERSION)
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@$(PROTOC_GEN_OPENAPI_VERSION)
	go install github.com/google/wire/cmd/wire@$(WIRE_VERSION)
	go install github.com/envoyproxy/protoc-gen-validate@$(PROTOC_GEN_VALIDATE_VERSION)

.PHONY: config
# generate internal config proto
config:
	@if [ -z "$(strip $(INTERNAL_PROTO_FILES))" ]; then \
		echo "No internal proto files found under ./internal"; \
		exit 0; \
	fi
	protoc --proto_path=./internal \
	       --proto_path=./third_party \
	       --go_out=paths=source_relative:./internal \
	       $(INTERNAL_PROTO_FILES)

.PHONY: api
# generate api pb/go-http/go-grpc/openapi files
api:
	@if [ -z "$(strip $(API_PROTO_FILES))" ]; then \
		echo "No API proto files found under ./api"; \
		exit 0; \
	fi
	@echo "Generating pb files to $(OUTPUT_DIR) ..."
	@mkdir -p $(OUTPUT_DIR)
	@find ./api -type f -name "*.proto" | while read proto_file; do \
		case "$$proto_file" in ./api/all-pb-go/*) continue ;; esac; \
		version_dir=$$(echo "$$proto_file" | grep -oE '/v[0-9]+' | tail -n 1 | tr -d '/'); \
		if [ -z "$$version_dir" ]; then version_dir="v1"; fi; \
		mkdir -p "$(OUTPUT_DIR)/$$version_dir"; \
		rel_path=$$(echo "$$proto_file" | sed 's|^\./api/||'); \
		proto_input=$$(echo "$$proto_file" | sed 's|^\./api/||'); \
		protoc --proto_path=./api \
		       --proto_path=./third_party \
		       --go_out=. \
		       --go_opt=module=$(MODULE_NAME) \
		       --go-http_out=. \
		       --go-http_opt=module=$(MODULE_NAME) \
		       --go-grpc_out=. \
		       --go-grpc_opt=module=$(MODULE_NAME) \
		       "$$proto_input"; \
		echo "Generated: $(OUTPUT_DIR)/$$version_dir"; \
	done
	protoc --proto_path=./api \
	       --proto_path=./third_party \
	       --openapi_out=fq_schema_naming=true,default_response=false:. \
	       $(API_PROTO_FILES)

.PHONY: wire
# generate wire
wire:
	@if [ ! -d "$(CMD_PATH)" ]; then \
		echo "cmd path not found: $(CMD_PATH)"; \
		exit 1; \
	fi
	@echo "Generating wire in $(CMD_PATH) ..."
	cd $(CMD_PATH) && wire

.PHONY: run
# run service
run:
	$(MAKE) api
	$(MAKE) config
	$(MAKE) wire
	kratos run -- -conf $(CONFIG_ABS) -env $(env)

.PHONY: build
# build service binary
build:
	@mkdir -p bin/
	go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/$(BINARY_NAME) $(CMD_PATH)

.PHONY: test-biz
# run biz tests
test-biz:
	go test ./internal/biz -v

.PHONY: test
# run template tests
test:
	@if [ -f ./tests/project_generation_security_test.sh ]; then \
		bash ./tests/project_generation_security_test.sh; \
	fi
	go test ./internal/... -v

.PHONY: generate
# run go generate and tidy
generate:
	go generate ./...
	go mod tidy

.PHONY: all
# run all generators
all:
	$(MAKE) api
	$(MAKE) config
	$(MAKE) wire
	$(MAKE) generate

.PHONY: print-vars
# print template variables
print-vars:
	@echo "SERVICE_NAME=$(SERVICE_NAME)"
	@echo "MODULE_NAME=$(MODULE_NAME)"
	@echo "BINARY_NAME=$(BINARY_NAME)"
	@echo "CMD_PATH=$(CMD_PATH)"
	@echo "CONFIG_DIR=$(CONFIG_DIR)"
	@echo "CONFIG_ABS=$(CONFIG_ABS)"
	@echo "OUTPUT_DIR=$(OUTPUT_DIR)"
	@echo "env=$(env)"

.PHONY: help
# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo '  make [target]'
	@echo ''
	@echo 'Variables (override by make VAR=value):'
	@echo '  SERVICE_NAME BINARY_NAME CMD_PATH CONFIG_DIR OUTPUT_DIR env'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
		helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
