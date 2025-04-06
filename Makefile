.PHONY: lint lint-fix

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run ./...

lint-fix:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run --fix ./...
