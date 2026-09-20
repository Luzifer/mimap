default:

lint:
	golangci-lint run ./...

test:
	go test -v -cover ./...

.PHONY: test
