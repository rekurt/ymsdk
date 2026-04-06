.PHONY: test test-race test-cover lint lint-fix build vet tidy check release-patch release-minor

## Development

test: ## Run tests
	go test ./...

test-race: ## Run tests with race detector
	go test -race -count=1 ./...

test-cover: ## Run tests with coverage report
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out
	@echo "\nHTML report: go tool cover -html=coverage.out"

lint: ## Run linter
	golangci-lint run --config .golangci.yml

lint-fix: ## Run linter with auto-fix
	golangci-lint run --config .golangci.yml --fix

build: ## Build all packages
	go build ./...

vet: ## Run go vet
	go vet ./...

tidy: ## Tidy go.mod
	go mod tidy

check: lint test-race build vet ## Run all checks (lint + test + build + vet)

## Releasing

release-patch: ## Tag and push a patch release (v0.0.X)
	@latest=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	next=$$(echo $$latest | awk -F. '{printf "%s.%s.%d", $$1, $$2, $$3+1}'); \
	echo "Tagging $$next"; \
	git tag $$next && git push origin $$next

release-minor: ## Tag and push a minor release (v0.X.0)
	@latest=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	next=$$(echo $$latest | awk -F. '{printf "%s.%d.0", $$1, $$2+1}'); \
	echo "Tagging $$next"; \
	git tag $$next && git push origin $$next

## Help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
