all: test

test:
	@go test -cover ./...
