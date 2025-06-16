.PHONY: test-inject test-minimal clean debug

test-inject:
	@echo "Running injection test..."
	cd tests/inject && go run main.go

test-minimal:
	@echo "Running minimal test..."
	cd tests/minimal && go run main.go

debug-inject:
	@echo "Running injection test with Wayland debug..."
	cd tests/inject && WAYLAND_DEBUG=1 go run main.go

debug-minimal:
	@echo "Running minimal test with Wayland debug..."
	cd tests/minimal && WAYLAND_DEBUG=1 go run main.go

clean:
	rm -f keyboard_example mouse_example

help:
	@echo "Available targets:"
	@echo "  make test-inject   - Run the comprehensive injection test"
	@echo "  make test-minimal  - Run the minimal test"
	@echo "  make debug-inject  - Run injection test with WAYLAND_DEBUG=1"
	@echo "  make debug-minimal - Run minimal test with WAYLAND_DEBUG=1"
	@echo "  make clean         - Remove built binaries"