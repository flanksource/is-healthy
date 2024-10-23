.PHONY: test
test:
	go test ./... -v

.PHONY: lint
lint:
	golangci-lint run
	
	if command -v golines > /dev/null 2>&1; then \
		golines -m 120 -w pkg/; \
		golines -m 120 -w events/; \
	fi