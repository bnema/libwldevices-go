# Wayland Virtual Input Go Bindings

Go bindings for Wayland virtual input protocols, providing programmatic control over mouse and keyboard input in Wayland compositors.

## Overview

This library provides Go interfaces and implementations for:
- **Virtual Pointer** (`wlr-virtual-pointer-unstable-v1`): Mouse movement, clicks, and scrolling
- **Virtual Keyboard** (`virtual-keyboard-unstable-v1`): Keyboard input and key combinations

These bindings allow applications to simulate user input events as if they came from physical input devices, which is useful for:
- Remote desktop applications
- Input automation and testing
- Accessibility tools
- Screen sharing applications
- Input event forwarding/routing

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
    
    "github.com/bnema/wayland-virtual-input-go/virtual_pointer"
)

func main() {
    ctx := context.Background()
    
    // Create virtual pointer manager
    manager, err := virtual_pointer.NewVirtualPointerManager(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Destroy()
    
    // Create virtual pointer
    pointer, err := manager.CreateVirtualPointer(nil)
    if err != nil {
        log.Fatal(err)
    }
    defer pointer.Destroy()
    
    // Move mouse relatively
    virtual_pointer.MoveRelative(pointer, 100, 50)
    
    // Click left mouse button
    virtual_pointer.Click(pointer, virtual_pointer.BTN_LEFT)
    
    // Scroll vertically
    virtual_pointer.ScrollVertical(pointer, 10.0)
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
    defer manager.Destroy()
    
    // Create virtual keyboard
    keyboard, err := manager.CreateVirtualKeyboard(nil)
    if err != nil {
        log.Fatal(err)
    }
    defer keyboard.Destroy()
    
    // Type a string
    virtual_keyboard.TypeString(keyboard, "Hello, Wayland!")
    
    // Press Enter
    virtual_keyboard.TypeKey(keyboard, virtual_keyboard.KEY_ENTER)
    
    // Key combination (Ctrl+C)
    virtual_keyboard.KeyCombo(keyboard, virtual_keyboard.MOD_CTRL, virtual_keyboard.KEY_C)
}
```

## API Reference

### Virtual Pointer

#### Constants

```go
// Mouse buttons
const (
    BTN_LEFT   = 0x110
    BTN_RIGHT  = 0x111
    BTN_MIDDLE = 0x112
    BTN_SIDE   = 0x113
    BTN_EXTRA  = 0x114
)

// Button states
const (
    BUTTON_STATE_RELEASED = 0
    BUTTON_STATE_PRESSED  = 1
)

// Scroll axes
const (
    AXIS_VERTICAL_SCROLL   = 0
    AXIS_HORIZONTAL_SCROLL = 1
)

// Axis sources
const (
    AXIS_SOURCE_WHEEL      = 0
    AXIS_SOURCE_FINGER     = 1
    AXIS_SOURCE_CONTINUOUS = 2
    AXIS_SOURCE_WHEEL_TILT = 3
)
```

#### Interfaces

```go
type VirtualPointerManager interface {
    CreateVirtualPointer(seat interface{}) (VirtualPointer, error)
    CreateVirtualPointerWithOutput(seat, output interface{}) (VirtualPointer, error)
    Destroy() error
}

type VirtualPointer interface {
    Motion(time time.Time, dx, dy float64) error
    MotionAbsolute(time time.Time, x, y, xExtent, yExtent uint32) error
    Button(time time.Time, button, state uint32) error
    ButtonPress(button uint32) error
    ButtonRelease(button uint32) error
    Axis(time time.Time, axis uint32, value float64) error
    AxisSource(axisSource uint32) error
    AxisStop(time time.Time, axis uint32) error
    AxisDiscrete(time time.Time, axis uint32, value float64, discrete int32) error
    Frame() error
    Destroy() error
}
```

#### Convenience Functions

```go
// Complete click operation
func Click(pointer VirtualPointer, button uint32) error

// Scroll operations
func ScrollVertical(pointer VirtualPointer, value float64) error
func ScrollHorizontal(pointer VirtualPointer, value float64) error

// Movement operations
func MoveRelative(pointer VirtualPointer, dx, dy float64) error
func MoveAbsolute(pointer VirtualPointer, x, y, xExtent, yExtent uint32) error
```

### Virtual Keyboard

#### Constants

```go
// Key codes (Linux input event codes)
const (
    KEY_A         = 30
    KEY_B         = 48
    KEY_C         = 46
    // ... (full alphabet and symbols)
    KEY_SPACE     = 57
    KEY_ENTER     = 28
    KEY_LEFTCTRL  = 29
    KEY_LEFTSHIFT = 42
    KEY_LEFTALT   = 56
    // ... (function keys, arrows, etc.)
)

// Key states
const (
    KEY_STATE_RELEASED = 0
    KEY_STATE_PRESSED  = 1
)

// Modifiers
const (
    MOD_SHIFT = 1 << 0
    MOD_CAPS  = 1 << 1
    MOD_CTRL  = 1 << 2
    MOD_ALT   = 1 << 3
    MOD_NUM   = 1 << 4
    MOD_LOGO  = 1 << 6
)
```

#### Interfaces

```go
type VirtualKeyboardManager interface {
    CreateVirtualKeyboard(seat interface{}) (VirtualKeyboard, error)
    Destroy() error
}

type VirtualKeyboard interface {
    Keymap(format uint32, fd *os.File, size uint32) error
    Key(time uint32, key, state uint32) error
    KeyPress(key uint32) error
    KeyRelease(key uint32) error
    Modifiers(modsDepressed, modsLatched, modsLocked, group uint32) error
    Destroy() error
}
```

#### Convenience Functions

```go
// Type individual key
func TypeKey(keyboard VirtualKeyboard, key uint32) error

// Type strings
func TypeString(keyboard VirtualKeyboard, text string) error

// Modifier management
func SetModifiers(keyboard VirtualKeyboard, modifiers uint32) error
func PressModifiers(keyboard VirtualKeyboard, modifiers uint32) error
func ReleaseModifiers(keyboard VirtualKeyboard, modifiers uint32) error

// Key combinations
func KeyCombo(keyboard VirtualKeyboard, modifiers uint32, key uint32) error
```

## Examples

See the `examples/` directory for complete working examples:
- `mouse_move.go`: Comprehensive mouse control demonstration
- `keyboard_input.go`: Keyboard input and text typing examples

Run examples:
```bash
go run examples/mouse_move.go
go run examples/keyboard_input.go
```

## Testing

Run the test suite:
```bash
go test ./...
```

Run tests for specific packages:
```bash
go test ./virtual_pointer
go test ./virtual_keyboard
```

Run tests with coverage:
```bash
go test -cover ./...
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

### Current Implementation

This library currently provides **stub implementations** for demonstration and testing purposes. The interfaces and APIs are designed to match the Wayland protocol specifications, but the actual Wayland communication is not implemented.

### Integration with Wayland Clients

To use these bindings in a real Wayland environment, you would need to integrate them with a Wayland client library such as:
- [go-wayland](https://github.com/neurlang/wayland) - Pure Go Wayland client library
- [wayland-go](https://github.com/rajveermalviya/wayland-go) - Go bindings for libwayland
- Custom CGO bindings to libwayland-client

### Real Implementation Requirements

A complete implementation would need to:
1. Connect to the Wayland display server
2. Get the global registry
3. Bind to the virtual input protocol globals
4. Send protocol requests over the Wayland socket
5. Handle protocol events and errors
6. Manage object lifecycle properly

## Protocol Support

### Supported Protocols

- **wlr-virtual-pointer-unstable-v1** (Version 2)
  - Relative and absolute pointer motion
  - Button events
  - Axis events with source information
  - Discrete scrolling
  - Frame-based event grouping

- **virtual-keyboard-unstable-v1** (Version 1)
  - Key press/release events
  - Keymap management
  - Modifier state handling

### Protocol Sources

The protocol specifications are based on:
- [wlroots protocols](https://github.com/swaywm/wlroots/tree/master/protocol) for virtual pointer
- [Wayland protocols](https://gitlab.freedesktop.org/wayland/wayland-protocols) for virtual keyboard

## Security Considerations

Virtual input protocols have significant security implications:

- **Compositor Permission**: Most Wayland compositors require explicit permission or privileged access to create virtual input devices
- **Sandboxing**: Applications may need special permissions or be run outside sandboxes
- **User Consent**: Consider requiring user consent before creating virtual input devices
- **Input Validation**: Always validate input parameters to prevent potential security issues

## Compatibility

### Wayland Compositors

Virtual input protocol support varies by compositor:

| Compositor | Virtual Pointer | Virtual Keyboard | Notes |
|------------|----------------|------------------|-------|
| wlroots-based | ✅ | ✅ | Sway, Hyprland, etc. |
| GNOME | ⚠️ | ⚠️ | Limited support |
| KDE | ⚠️ | ⚠️ | Limited support |
| Others | ❓ | ❓ | Check individual support |

### Go Versions

- Requires Go 1.19 or later
- Tested on Go 1.20+

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