# Protocol paths - these match the standard locations
WL_PROTOCOL_DIR ?= /usr/share/wayland-protocols
POINTER_CONSTRAINTS_XML = $(WL_PROTOCOL_DIR)/unstable/pointer-constraints/pointer-constraints-unstable-v1.xml

.PHONY: test test-unit test-inject test-minimal clean debug generate-protocols

generate-protocols: generate-pointer-constraints

generate-pointer-constraints:
	@echo "Generating pointer constraints protocol bindings..."
	@if [ ! -f "$(POINTER_CONSTRAINTS_XML)" ]; then \
		echo "Error: pointer-constraints-unstable-v1.xml not found at $(POINTER_CONSTRAINTS_XML)"; \
		echo "Make sure wayland-protocols is installed on your system"; \
		exit 1; \
	fi
	@echo "Using protocol from: $(POINTER_CONSTRAINTS_XML)"
	go run tools/generate.go \
		-protocol=pointer_constraints \
		-xml=$(POINTER_CONSTRAINTS_XML) \
		-output=internal/protocols/pointer_constraints.go \
		-package=protocols

# Unit tests - safe to run, no real input injection
test-unit:
	@echo "Running unit tests (safe - no real input injection)..."
	go test ./virtual_pointer ./virtual_keyboard ./pointer_constraints ./keyboard_shortcuts_inhibitor -v

# All tests including unit tests
test: test-unit
	@echo "Unit tests completed successfully"

test-inject:
	@echo "Running injection test..."
	cd tests/inject && go run main.go

test-minimal:
	@echo "Running minimal test..."
	cd tests/minimal && go run main.go

test-constraint:
	@echo "Running pointer constraints test..."
	cd tests/constraint && go run main.go

debug-inject:
	@echo "Running injection test with Wayland debug..."
	cd tests/inject && WAYLAND_DEBUG=1 go run main.go

debug-minimal:
	@echo "Running minimal test with Wayland debug..."
	cd tests/minimal && WAYLAND_DEBUG=1 go run main.go

clean:
	rm -f keyboard_example mouse_example

help:
	@echo "🔒 SAFE TESTING (recommended):"
	@echo "  make test            - Run unit tests (SAFE - no real input injection)"
	@echo "  make test-unit       - Run unit tests (SAFE - no real input injection)"
	@echo ""
	@echo "🔧 DEVELOPMENT:"
	@echo "  make generate-protocols      - Generate protocol bindings from system wayland-protocols"
	@echo "  make generate-pointer-constraints - Generate pointer constraints protocol bindings"
	@echo "  make clean           - Remove built binaries"
	@echo ""
	@echo "⚠️  DANGEROUS INTEGRATION TESTS (WILL CONTROL YOUR REAL MOUSE/KEYBOARD!):"
	@echo "  make test-inject     - Run the comprehensive injection test (MOVES REAL MOUSE & TYPES!)"
	@echo "  make test-minimal    - Run the minimal test (MOVES REAL MOUSE!)"
	@echo "  make test-constraint - Run the pointer constraints test"
	@echo "  make debug-inject    - Run injection test with WAYLAND_DEBUG=1 (MOVES REAL MOUSE & TYPES!)"
	@echo "  make debug-minimal   - Run minimal test with WAYLAND_DEBUG=1 (MOVES REAL MOUSE!)"
	@echo ""
	@echo "💡 To enable real input tests in unit tests: WAYLAND_TEST_REAL_INPUT=1 make test-unit"