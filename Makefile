.PHONY: test lint build release-patch release-minor

test:
	go test ./...

lint:
	golangci-lint run --config .golangci.yml

build:
	go build ./...

release-patch:
	@latest=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	next=$$(echo $$latest | awk -F. '{printf "%s.%s.%d", $$1, $$2, $$3+1}'); \
	echo "Tagging $$next"; \
	git tag $$next && git push origin $$next

release-minor:
	@latest=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	next=$$(echo $$latest | awk -F. '{printf "%s.%d.0", $$1, $$2+1}'); \
	echo "Tagging $$next"; \
	git tag $$next && git push origin $$next
