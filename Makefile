.PHONY: test test-integration generate-protocols vet

generate-protocols:
	GOWORK=off go generate ./internal/protocols

test:
	CGO_ENABLED=0 GOWORK=off go test ./...

test-integration:
	GOWORK=off go test -v ./test/integration

vet:
	CGO_ENABLED=0 GOWORK=off go vet ./...
