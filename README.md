# Wayland Virtual Input Go Bindings

**Production-ready Go bindings for Wayland virtual input protocols** - Control mouse and keyboard input programmatically on Wayland compositors.

[![Go Reference](https://pkg.go.dev/badge/github.com/bnema/wayland-virtual-input-go.svg)](https://pkg.go.dev/github.com/bnema/wayland-virtual-input-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/bnema/wayland-virtual-input-go)](https://goreportcard.com/report/github.com/bnema/wayland-virtual-input-go)

## Overview

This library provides **complete, working implementations** for:
- **Virtual Pointer** (`zwlr_virtual_pointer_v1`): Mouse movement, clicks, and scrolling
- **Virtual Keyboard** (`zwp_virtual_keyboard_v1`): Keyboard input and key combinations
- **Pointer Constraints** (`zwp_pointer_constraints_v1`): Lock or confine pointer motion

Built on top of [neurlang/wayland](https://github.com/neurlang/wayland) client library, these bindings enable applications to inject input events directly into Wayland compositors and control pointer behavior.

### Use Cases
- **Remote desktop applications** - Forward input from remote clients
- **Input automation and testing** - Programmatic UI testing
- **Accessibility tools** - Alternative input methods
- **Screen sharing applications** - Multi-user input handling
- **Gaming and simulation** - Synthetic input generation
- **FPS games** - Lock pointer for mouse-look controls
- **Creative applications** - Confine pointer to canvas area

## Features

### Virtual Pointer
- Relative and absolute mouse movement
- Mouse button events (left, right, middle, side, extra)
- Scroll wheel events (vertical and horizontal)
- Multiple axis sources (wheel, finger, continuous, wheel tilt)
- Discrete scrolling support
- Frame-based event grouping

### Virtual Keyboard
- Individual key press/release events
- String typing with automatic character mapping
- Modifier key support (Ctrl, Alt, Shift, etc.)
- Function keys and navigation keys
- Numeric keypad support
- Key combinations and shortcuts
- Modifier state management

### Pointer Constraints
- Lock pointer to current position
- Confine pointer to specified region
- Oneshot and persistent lifetime modes
- Cursor position hints for unlock
- Event notifications for constraint state changes
- Region updates while constrained

## Installation

```bash
go get github.com/bnema/wayland-virtual-input-go
```

## Quick Start

### Virtual Pointer Example

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/bnema/wayland-virtual-input-go/virtual_pointer"
)

func main() {
    ctx := context.Background()
    
    // Create virtual pointer manager
    manager, err := virtual_pointer.NewVirtualPointerManager(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Close()
    
    // Create virtual pointer
    pointer, err := manager.CreatePointer()
    if err != nil {
        log.Fatal(err)
    }
    defer pointer.Close()
    
    // Move mouse relatively (100px right, 50px down)
    err = pointer.MoveRelative(100.0, 50.0)
    if err != nil {
        log.Fatal(err)
    }
    
    // Left click
    err = pointer.LeftClick()
    if err != nil {
        log.Fatal(err)
    }
    
    // Scroll down
    err = pointer.ScrollVertical(5.0)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Virtual Keyboard Example

```go
package main

import (
    "context"
    "log"
    
    "github.com/bnema/wayland-virtual-input-go/virtual_keyboard"
)

func main() {
    ctx := context.Background()
    
    // Create virtual keyboard manager
    manager, err := virtual_keyboard.NewVirtualKeyboardManager(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Close()
    
    // Create virtual keyboard
    keyboard, err := manager.CreateKeyboard()
    if err != nil {
        log.Fatal(err)
    }
    defer keyboard.Close()
    
    // Type a string (handles uppercase automatically)
    err = keyboard.TypeString("Hello, Wayland!")
    if err != nil {
        log.Fatal(err)
    }
    
    // Press Enter
    err = keyboard.TypeKey(virtual_keyboard.KEY_ENTER)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Pointer Constraints Example

```go
package main

import (
    "context"
    "log"
    
    "github.com/bnema/wayland-virtual-input-go/pointer_constraints"
)

func main() {
    ctx := context.Background()
    
    // Create pointer constraints manager
    manager, err := pointer_constraints.NewPointerConstraintsManager(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Close()
    
    // Get surface and pointer from your window toolkit
    surface := getWlSurface() // From your application window
    pointer := getWlPointer() // From seat capabilities
    
    // Lock pointer for FPS-style controls
    locked, err := manager.LockPointer(surface, pointer, nil, pointer_constraints.LifetimePersistent)
    if err != nil {
        log.Fatal(err)
    }
    defer locked.Close()
    
    // Set cursor position hint for unlock
    locked.SetCursorPositionHint(400.0, 300.0)
    
    // Or confine pointer to a region
    region := createRegion(0, 0, 800, 600) // Create region bounds
    confined, err := manager.ConfinePointer(surface, pointer, region, pointer_constraints.LifetimeOneshot)
    if err != nil {
        log.Fatal(err)
    }
    defer confined.Close()
}
```

## API Reference

### Virtual Pointer

#### Main Types

```go 
type VirtualPointerManager struct {
    // Creates and manages virtual pointer devices
}

type VirtualPointer struct {
    // Represents a virtual pointer device for input injection
}
```

#### Key Methods

```go
// Manager creation
func NewVirtualPointerManager(ctx context.Context) (*VirtualPointerManager, error)
func (m *VirtualPointerManager) CreatePointer() (*VirtualPointer, error)
func (m *VirtualPointerManager) Close() error

// Core pointer operations  
func (p *VirtualPointer) Motion(time time.Time, dx, dy float64) error
func (p *VirtualPointer) Button(time time.Time, button, state uint32) error
func (p *VirtualPointer) Axis(time time.Time, axis uint32, value float64) error
func (p *VirtualPointer) Frame() error
func (p *VirtualPointer) Close() error

// Convenience methods
func (p *VirtualPointer) MoveRelative(dx, dy float64) error
func (p *VirtualPointer) LeftClick() error
func (p *VirtualPointer) RightClick() error
func (p *VirtualPointer) MiddleClick() error  
func (p *VirtualPointer) ScrollVertical(value float64) error
func (p *VirtualPointer) ScrollHorizontal(value float64) error
```

#### Constants

```go
// Mouse buttons (Linux input event codes)
const (
    BTN_LEFT   = 0x110
    BTN_RIGHT  = 0x111 
    BTN_MIDDLE = 0x112
    BTN_SIDE   = 0x113
    BTN_EXTRA  = 0x114
)

// Button/axis states
const (
    ButtonStateReleased = 0
    ButtonStatePressed  = 1
    AxisVertical       = 0
    AxisHorizontal     = 1
)
```

### Virtual Keyboard

#### Main Types

```go
type VirtualKeyboardManager struct {
    // Creates and manages virtual keyboard devices  
}

type VirtualKeyboard struct {
    // Represents a virtual keyboard device for input injection
}
```

#### Key Methods

```go
// Manager creation
func NewVirtualKeyboardManager(ctx context.Context) (*VirtualKeyboardManager, error)
func (m *VirtualKeyboardManager) CreateKeyboard() (*VirtualKeyboard, error) 
func (m *VirtualKeyboardManager) Close() error

// Core keyboard operations
func (k *VirtualKeyboard) Key(timestamp time.Time, key uint32, state KeyState) error
func (k *VirtualKeyboard) Modifiers(modsDepressed, modsLatched, modsLocked, group uint32) error
func (k *VirtualKeyboard) Close() error

// Convenience methods
func (k *VirtualKeyboard) PressKey(key uint32) error
func (k *VirtualKeyboard) ReleaseKey(key uint32) error
func (k *VirtualKeyboard) TypeKey(key uint32) error
func (k *VirtualKeyboard) TypeString(text string) error
```

#### Constants

```go
// Key codes (Linux input event codes)
const (
    KEY_A = 30; KEY_B = 48; KEY_C = 46; KEY_D = 32; KEY_E = 18
    KEY_F = 33; KEY_G = 34; KEY_H = 35; KEY_I = 23; KEY_J = 36
    // ... (full alphabet A-Z)
    KEY_1 = 2; KEY_2 = 3; KEY_3 = 4; KEY_4 = 5; KEY_5 = 6
    KEY_6 = 7; KEY_7 = 8; KEY_8 = 9; KEY_9 = 10; KEY_0 = 11
    KEY_SPACE = 57; KEY_ENTER = 28; KEY_TAB = 15; KEY_BACKSPACE = 14
    KEY_LEFTSHIFT = 42; KEY_LEFTCTRL = 29; KEY_LEFTALT = 56
    // ... (complete list available in source)
)

// Key states
const (
    KeyStateReleased KeyState = 0
    KeyStatePressed  KeyState = 1  
)
```

## Testing & Examples

### Interactive Tests

The library includes interactive test programs that demonstrate real functionality:

```bash
# Comprehensive test with both mouse and keyboard
go run tests/inject/main.go

# Minimal mouse movement test  
go run tests/minimal/main.go
```

**Note**: These tests require a Wayland compositor that supports virtual input protocols (e.g., Sway, Hyprland).

### Running Tests

```bash
# Run all tests
go test ./...

# Test specific packages
go test ./virtual_pointer
go test ./virtual_keyboard
go test ./internal/protocols

# Run with coverage
go test -cover ./...

# Debug protocol communication
WAYLAND_DEBUG=1 go run tests/minimal/main.go
```

## Development Tools

### Code Generation

The `tools/generate.go` utility can generate Go bindings from Wayland protocol XML files:

```bash
# Generate virtual pointer bindings
go run tools/generate.go \
  -protocol=virtual_pointer \
  -xml=../wlr-protocols/unstable/wlr-virtual-pointer-unstable-v1.xml \
  -output=virtual_pointer/generated.go

# Generate virtual keyboard bindings
go run tools/generate.go \
  -protocol=virtual_keyboard \
  -xml=path/to/virtual-keyboard-unstable-v1.xml \
  -output=virtual_keyboard/generated.go
```

## Architecture

### Implementation

This library provides **complete, production-ready implementations** of Wayland virtual input protocols. Built on [neurlang/wayland](https://github.com/neurlang/wayland), it handles all the complex Wayland protocol communication automatically.

#### Core Components

1. **Protocol Layer** (`internal/protocols/`) - Low-level protocol implementations
   - Direct Wayland protocol request/response handling  
   - Proper object lifecycle management
   - Fixed-point number conversion for coordinates

2. **Client Layer** (`internal/client/`) - Wayland connection management
   - Display connection and registry handling
   - Protocol global discovery and binding
   - Event loop and context management

3. **High-Level APIs** (`virtual_pointer/`, `virtual_keyboard/`) - User-friendly interfaces
   - Convenience methods for common operations
   - Automatic resource cleanup
   - Error handling and validation

#### Key Features

- **Zero Dependencies** - Only depends on neurlang/wayland
- **Thread Safe** - Safe for concurrent use
- **Resource Management** - Automatic cleanup and proper object destruction
- **Error Handling** - Comprehensive error reporting
- **Protocol Compliance** - Fully compliant with Wayland protocol specifications

## Protocol Support

### Implemented Protocols

- **zwlr_virtual_pointer_v1** (wlroots virtual pointer)
  - ✅ Relative pointer motion with fixed-point precision
  - ✅ Button press/release events (left, right, middle, side, extra)
  - ✅ Axis events for scrolling (vertical/horizontal)  
  - ✅ Frame-based event grouping
  - ✅ Axis source information
  - ✅ Discrete scrolling support

- **zwp_virtual_keyboard_v1** (Wayland virtual keyboard)
  - ✅ Key press/release events with timestamp
  - ✅ XKB keymap management (automatic default keymap)
  - ✅ Modifier state handling
  - ✅ File descriptor passing for keymaps

### Protocol Implementation Details

- **Proper Wayland Object Lifecycle** - Correct creation, binding, and destruction
- **Fixed-Point Arithmetic** - Wayland uses 24.8 fixed-point for coordinates
- **File Descriptor Handling** - Proper fd passing for keyboard keymaps
- **Event Sequencing** - Correct ordering of protocol requests
- **Error Handling** - Comprehensive error reporting for protocol failures

## Security Considerations

Virtual input protocols have significant security implications:

- **Compositor Permission**: Most Wayland compositors require explicit permission or privileged access to create virtual input devices
- **Sandboxing**: Applications may need special permissions or be run outside sandboxes
- **User Consent**: Consider requiring user consent before creating virtual input devices
- **Input Validation**: Always validate input parameters to prevent potential security issues

## Compatibility

### Wayland Compositors

**Tested and Working:**
- ✅ **Sway** - Full support for both protocols
- ✅ **Hyprland** - Full support for both protocols  
- ✅ **wlroots-based compositors** - Full support

**Limited/Untested:**
- ⚠️ **GNOME** - May require additional permissions
- ⚠️ **KDE Plasma** - Limited virtual input support
- ❓ **Other compositors** - Check `wayland-info | grep virtual` 

### System Requirements

- **Go 1.19+** (tested on Go 1.20+)
- **Wayland compositor** with virtual input protocol support
- **Linux** (uses Linux input event codes)
- **Wayland session** (`XDG_SESSION_TYPE=wayland`)

### Verification

Check if your compositor supports the required protocols:
```bash
# Check available protocols
wayland-info | grep -E "(virtual_pointer|virtual_keyboard)"

# Should show:
# zwlr_virtual_pointer_manager_v1
# zwp_virtual_keyboard_manager_v1
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

### Development Guidelines

- Follow Go conventions and idioms
- Write comprehensive tests
- Document all public APIs
- Maintain backward compatibility
- Update examples when adding features

## License

This project is dual-licensed:
- The library code is licensed under the MIT License
- Protocol definitions follow their respective licenses (typically MIT-style)

See the protocol source files for specific licensing information.

## Acknowledgments

- **wlroots project** for the virtual pointer protocol specification
- **Wayland project** for the virtual keyboard protocol specification
- **Go community** for excellent tooling and libraries

## Related Projects

- [waymon](https://github.com/bnema/waymon) - Mouse sharing application using these bindings
- [wlroots](https://github.com/swaywm/wlroots) - Wayland compositor library
- [wayland-protocols](https://gitlab.freedesktop.org/wayland/wayland-protocols) - Wayland protocol specifications

## Support

For bugs, feature requests, or questions:
1. Check existing issues
2. Create a new issue with detailed information
3. Include Go version, OS, and Wayland compositor details
4. Provide minimal reproduction code when possible