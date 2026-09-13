# Developer entry points. The package graph is the source of truth for what
# exists; these targets only wrap the common commands.

.PHONY: test test-unit test-integration generate-protocols fmt vet clean

# Regenerate the generator fixture's committed bindings. The golden file is
# internal/protocoltest/bindings.go and `go test ./scanner` fails when it drifts.
generate-protocols:
	go run ./scanner/cmd/wayland-scanner \
		-p protocoltest \
		-o internal/protocoltest/bindings.go \
		scanner/testdata/protocol_fixture.xml

# Every package except the Docker-based compositor fixture.
test-unit:
	go test -race $$(go list ./... | grep -v /test/integration)

# Headless wlroots compositor fixture: builds the pinned image and runs the
# standalone consumer against the real compositor.
test-integration:
	bash test/integration/run.sh

test: test-unit

fmt:
	gofmt -w $$(git ls-files '*.go')

vet:
	go vet ./...

clean:
	rm -rf bin
