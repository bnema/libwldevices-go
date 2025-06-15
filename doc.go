// Package wayland_virtual_input_go provides Go bindings for Wayland virtual input protocols.
//
// This library implements Go bindings for the wlr-virtual-pointer-unstable-v1 and
// virtual-keyboard-unstable-v1 Wayland protocols, enabling applications to inject
// mouse and keyboard events into Wayland compositors without requiring root privileges.
//
// # Supported Protocols
//
// • wlr-virtual-pointer-unstable-v1: Mouse input injection (motion, buttons, scroll)
// • virtual-keyboard-unstable-v1: Keyboard input injection (keys, modifiers, text)
// • pointer-constraints-unstable-v1: Exclusive pointer capture and constraints
// • keyboard-shortcuts-inhibit-unstable-v1: Keyboard shortcut inhibition
//
// # Compositor Compatibility
//
// This library is designed for and tested with wlroots-based compositors:
// • Hyprland (full support)
// • Sway (full support)
// • Other wlroots compositors (generally supported)
//
// Note: GNOME and KDE have limited or no support for these protocols.
//
// # Security Model
//
// Virtual input protocols work at the user level without requiring root privileges.
// The Wayland compositor controls access and can implement security policies.
// Most wlroots-based compositors allow virtual input devices by default.
//
// # Basic Usage
//
// Virtual Pointer (Mouse):
//
//	import "github.com/bnema/wayland-virtual-input-go/virtual_pointer"
//
//	// Create manager and pointer
//	manager := virtual_pointer.NewVirtualPointerManager(display, registry)
//	pointer := manager.CreateVirtualPointer(seat)
//
//	// Move mouse cursor
//	pointer.Motion(timestamp, 10.0, 5.0)
//	pointer.Frame()
//
//	// Click left button
//	pointer.LeftClick()
//
// Virtual Keyboard:
//
//	import "github.com/bnema/wayland-virtual-input-go/virtual_keyboard"
//
//	// Create manager and keyboard
//	manager := virtual_keyboard.NewVirtualKeyboardManager(display, registry)
//	keyboard := manager.CreateVirtualKeyboard(seat)
//
//	// Type text
//	keyboard.TypeString("Hello, World!")
//
//	// Press key combination
//	keyboard.ModifierPress(virtual_keyboard.MOD_CTRL)
//	keyboard.Key(virtual_keyboard.KEY_C, virtual_keyboard.KEY_STATE_PRESSED)
//
// # Architecture
//
// The library provides high-level Go interfaces that wrap the low-level Wayland
// protocol messages. It includes:
//
// • Protocol-compliant message generation
// • Convenient wrapper functions for common operations
// • Proper error handling and resource management
// • Comprehensive documentation and examples
//
// # Thread Safety
//
// The current implementation is not thread-safe. All operations should be
// performed from the same goroutine that manages the Wayland event loop.
//
// # Error Handling
//
// All methods return errors for proper error handling. Common error conditions include:
// • Wayland connection failures
// • Protocol not supported by compositor
// • Invalid parameters or state
//
// See the examples/ directory for complete working examples.
package wayland_virtual_input_go