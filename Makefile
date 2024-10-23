.PHONY: test
test:
	go test ./... -v

.PHONY: lint
lint:
	golangci-lint run
	
	golines -m 120 -w pkg/
	golines -m 120 -w events/

	gofumpt -w .

.PHONY:
sync:
	git submodule update --init --recursive

update-submodules:
	git submodule update --remote --merge && git submodule sync
