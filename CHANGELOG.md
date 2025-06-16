# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2024-12-XX

### 🎉 Major Release: Complete Working Implementation

This release transforms the library from stub implementations to a **fully functional, production-ready** Wayland virtual input library.

### Added

#### Core Functionality
- **Complete Wayland Protocol Implementation** - Full zwlr_virtual_pointer_v1 and zwp_virtual_keyboard_v1 support
- **Production-Ready Codebase** - Built on neurlang/wayland for robust Wayland communication
- **Thread-Safe Operations** - Safe for concurrent use in multi-threaded applications
- **Automatic Resource Management** - Proper object lifecycle and cleanup

#### Virtual Pointer Features
- ✅ **Relative mouse movement** with fixed-point precision (24.8 format)
- ✅ **Mouse button events** (left, right, middle, side, extra buttons)
- ✅ **Scroll wheel support** (vertical and horizontal)
- ✅ **Frame-based event grouping** for atomic operations
- ✅ **Convenience methods** for common operations (LeftClick, RightClick, MoveRelative, etc.)

#### Virtual Keyboard Features  
- ✅ **Key press/release events** with proper timing
- ✅ **String typing** with automatic uppercase/lowercase handling
- ✅ **XKB keymap management** with default keymap creation
- ✅ **Modifier key support** (Ctrl, Alt, Shift, etc.)
- ✅ **File descriptor handling** for keymap communication
- ✅ **Complete key code definitions** (Linux input event codes)

#### Developer Experience
- **Comprehensive Examples** - Working examples for all functionality
- **Interactive Tests** - Real demonstrations with visual feedback
- **Complete Documentation** - Updated README with working API examples
- **Error Handling** - Comprehensive error reporting and validation
- **Debug Support** - WAYLAND_DEBUG integration for protocol debugging

### Technical Implementation

#### Architecture
- **Protocol Layer** (`internal/protocols/`) - Low-level Wayland protocol handling
- **Client Layer** (`internal/client/`) - Connection and registry management  
- **High-Level APIs** (`virtual_pointer/`, `virtual_keyboard/`) - User-friendly interfaces

#### Key Technical Fixes
- **Fixed parameter types** - Proper handling of Wayland proxy objects vs. IDs
- **File descriptor passing** - Correct uintptr handling for keyboard keymaps
- **Fixed-point arithmetic** - Proper 24.8 fixed-point coordinate conversion
- **Protocol compliance** - Full adherence to Wayland protocol specifications
- **Event sequencing** - Correct ordering of protocol requests and responses

### Examples and Tests

#### New Examples (`examples/`)
- `mouse_simple.go` - Basic mouse control demonstration
- `keyboard_simple.go` - Basic keyboard input demonstration  
- `combined_demo.go` - Advanced automation scenario with both input types
- Complete README with troubleshooting guide

#### Enhanced Tests (`tests/`)
- `inject/main.go` - Comprehensive integration test for both protocols
- `minimal/main.go` - Minimal test for debugging protocol issues
- Proper test documentation and usage instructions

### Breaking Changes

⚠️ **This is a major version bump due to significant API changes**

#### API Changes
- **Method names**: `Destroy()` → `Close()` for consistency
- **Manager creation**: Now requires `context.Context` parameter
- **Error handling**: All methods now return proper error values
- **Resource management**: Automatic cleanup with defer-friendly Close() methods

#### Migration Guide
```go
// Old (v1.x - stub implementation)
manager := NewVirtualPointerManager(display, registry)
pointer := manager.CreateVirtualPointer(seat)
pointer.Destroy()

// New (v2.x - working implementation)
manager, err := NewVirtualPointerManager(ctx)
if err != nil {
    log.Fatal(err)
}
defer manager.Close()

pointer, err := manager.CreatePointer()
if err != nil {
    log.Fatal(err)
}
defer pointer.Close()
```

### Compatibility

#### Tested Compositors
- ✅ **Sway** - Full support verified
- ✅ **Hyprland** - Full support verified
- ✅ **wlroots-based compositors** - Full support expected

#### System Requirements
- **Go 1.19+** (tested on Go 1.20+)
- **Wayland compositor** with virtual input protocol support
- **Linux** (uses Linux input event codes)
- **Active Wayland session** (`XDG_SESSION_TYPE=wayland`)

### Dependencies

- **Added**: `github.com/neurlang/wayland v0.2.1` - Core Wayland client library
- **Removed**: All stub/mock implementations

### Documentation

- **Complete README rewrite** - Reflects working implementation
- **API documentation** - Updated with real method signatures
- **Usage examples** - All examples now work with real compositors
- **Troubleshooting guide** - Help for common setup issues
- **Protocol specifications** - Detailed implementation coverage

### Testing

Run the interactive tests to verify functionality:

```bash
# Comprehensive test (both mouse and keyboard)
go run tests/inject/main.go

# Minimal mouse test
go run tests/minimal/main.go

# Debug protocol communication
WAYLAND_DEBUG=1 go run tests/minimal/main.go
```

### Acknowledgments

This major release was made possible by:
- **neurlang/wayland** project for providing excellent Wayland client bindings
- **wlroots** project for virtual pointer protocol specification
- **Wayland** project for virtual keyboard protocol specification
- **Community testing** on various wlroots-based compositors

---

## [1.x.x] - Previous Versions

Previous versions contained stub implementations for API design and testing purposes. 
See git history for details of pre-2.0 releases.