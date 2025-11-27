SHELL := /bin/bash -e

.DEFAULT_GOAL := help

.PHONY: all
all: test lint build ## Run all the tests, linters and build the project

.PHONY: clean
clean: ## Clean the working directory from binaries, coverage
	rm -f kubectl-api_resource_versions tmp/coverage/*
	rm -rf dist
	find . -type f -executable -name '*.test' -delete

.PHONY: build
build: ## Build the project (resulting binary is written to kubectl-api_resource_versions)
	@echo "🛠️ building the project …"
	goreleaser build --auto-snapshot --clean --single-target --output kubectl-api_resource_versions

.PHONY: test
test: buildable test-unit test-examples benchmark test-flakiness ## Run all the tests (unit & benchmark)

.PHONY: buildable
buildable: ## Check if the project is buildable
	@echo "👷🏽 checking if the project is buildable, it may take a while to download dependencies …"
	go build -o /dev/null -v ./...

.PHONY: test-unit
test-unit: tmp/coverage ## Run the unit tests
	@echo "🧪 running the unit tests, it may take a few minutes to build with race detection …"
	go test -v -timeout 10s -race -skip '^Example' -coverprofile=tmp/coverage/cover.out \
		./...

.PHONY: test-examples
test-examples: tmp/coverage ## Run the testable examples
	@echo "🧪 running the testable examples …"
	go test -v -run '^Example' -coverprofile=tmp/coverage/example.out \
		./...

.PHONY: benchmark
benchmark: tmp/coverage ## Run the benchmarks
	@echo "🧪 running the benchmarks …"
	go test -v -run '^$$' -bench '^Benchmark' -coverprofile=tmp/coverage/benchmark.out \
		./...

.PHONY: test-flakiness
test-flakiness: tmp/coverage ## Run the unit tests with a high count to ensure they are not flaky
	@echo "🧪 running the unit tests with a high count to ensure they are not flaky …"
	go test -timeout 2m -count 1000 -failfast -skip '^Example' ./...

.PHONY: lint
lint: ## Run the linters
	@echo "🔍 running the linters, this may take a few minutes …"
	pre-commit run --all-files

tmp/coverage:
	mkdir -p tmp/coverage

# Implements this pattern for autodocumenting Makefiles:
# https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
#
# Picks up all comments that start with a ## and are at the end of a target definition line.
help:
	@grep -h -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
