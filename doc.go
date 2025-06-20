// Package libwldevices provides Go bindings for Wayland device protocols.
//
// This library implements **complete, production-ready** Go bindings for virtual input,
// pointer constraints, and output management Wayland protocols. Built on the high-performance
// WLTurbo Wayland client library, it enables applications to inject input events,
// manage pointer behavior, and monitor display configuration in Wayland compositors
// without requiring root privileges.
//
// # Supported Protocols
//
// **Fully Implemented:**
// • zwlr_virtual_pointer_v1: Mouse input injection (relative motion, buttons, scrolling)
// • zwp_virtual_keyboard_v1: Keyboard input injection (keys, modifiers, text typing)
// • zwp_pointer_constraints_v1: Pointer locking and confinement for gaming/applications
// • zwlr_output_management_v1: Real-time monitor detection and configuration
//
// **Planned (Interface Ready):**
// • zwp_relative_pointer_v1: High-precision relative mouse movement
// • zwp_keyboard_shortcuts_inhibit_v1: Disable compositor shortcuts
//
// # Compositor Compatibility
//
// **Tested and Working:**
// • Hyprland (full support - all protocols)
//
// **Expected to Work (wlroots-based):**
// • Sway, River, Wayfire, Hikari, dwl
// • Other wlroots compositors
//
// **Limited/No Support:**
// • GNOME (limited virtual input support)
// • KDE Plasma (limited protocol support)
// • Weston (protocol availability varies)
//
// # Security Model
//
// Virtual input protocols operate at the user level without requiring root privileges.
// The Wayland compositor controls access and implements security policies:
//
// • **User Consent**: Applications may need user permission for virtual input
// • **Sandboxing**: May require running outside strict sandboxes (Flatpak, etc.)
// • **Input Validation**: All parameters are validated to prevent security issues
// • **Resource Management**: Proper cleanup prevents resource leaks
//
// # Basic Usage
//
// **Context-Aware Operations:**
// All operations support context.Context for cancellation and timeouts.
//
// Virtual Pointer (Mouse):
//
//	import (
//		"context"
//		"time"
//		"github.com/bnema/libwldevices-go/virtual_pointer"
//	)
//
//	// Create manager with timeout
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	manager, err := virtual_pointer.NewVirtualPointerManager(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer manager.Close()
//
//	pointer, err := manager.CreatePointer()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer pointer.Close()
//
//	// Move mouse cursor (relative movement)
//	pointer.MoveRelative(10.0, 5.0)
//
//	// Click buttons
//	pointer.LeftClick()
//	pointer.RightClick()
//	pointer.MiddleClick()
//
//	// Scroll (positive = down/right, negative = up/left)
//	pointer.ScrollVertical(5.0)
//	pointer.ScrollHorizontal(-3.0)
//
// Virtual Keyboard:
//
//	import "github.com/bnema/libwldevices-go/virtual_keyboard"
//
//	// Create manager and keyboard with context
//	manager, err := virtual_keyboard.NewVirtualKeyboardManager(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer manager.Close()
//
//	keyboard, err := manager.CreateKeyboard()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer keyboard.Close()
//
//	// Type text (handles uppercase/lowercase automatically)
//	keyboard.TypeString("Hello, Wayland!")
//
//	// Press individual keys
//	keyboard.PressKey(virtual_keyboard.KEY_ENTER)
//	keyboard.TypeKey(virtual_keyboard.KEY_TAB)
//
//	// Key combinations (Ctrl+C example)
//	keyboard.PressKey(virtual_keyboard.KEY_LEFTCTRL)
//	keyboard.PressKey(virtual_keyboard.KEY_C)
//	keyboard.ReleaseKey(virtual_keyboard.KEY_C)
//	keyboard.ReleaseKey(virtual_keyboard.KEY_LEFTCTRL)
//
// Pointer Constraints:
//
//	import "github.com/bnema/libwldevices-go/pointer_constraints"
//
//	// Create manager
//	manager, err := pointer_constraints.NewPointerConstraintsManager(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer manager.Close()
//
//	// Get surface and pointer from your window toolkit
//	surface := getWlSurface() // From your application window
//	pointer := getWlPointer() // From seat capabilities
//
//	// Lock pointer for FPS-style mouse look controls
//	locked, err := manager.LockPointer(surface, pointer, nil, pointer_constraints.LifetimePersistent)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer locked.Close()
//
//	// Set cursor position hint for when unlocked
//	locked.SetCursorPositionHint(400.0, 300.0)
//
//	// Confine pointer to a rectangular region
//	region := createRegion(0, 0, 800, 600) // Create region bounds
//	confined, err := manager.ConfinePointer(surface, pointer, region, pointer_constraints.LifetimeOneshot)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer confined.Close()
//
// Output Management:
//
//	import "github.com/bnema/libwldevices-go/output_management"
//
//	// Create manager
//	manager, err := output_management.NewOutputManager(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer manager.Close()
//
//	// Set up event handlers for real-time updates
//	manager.SetHandlers(output_management.OutputHandlers{
//		OnHeadAdded: func(head *output_management.OutputHead) {
//			fmt.Printf("Monitor connected: %s\n", head.Name)
//		},
//		OnHeadRemoved: func(head *output_management.OutputHead) {
//			fmt.Printf("Monitor disconnected: %s\n", head.Name)
//		},
//		OnConfigurationChanged: func(heads []*output_management.OutputHead) {
//			fmt.Printf("Display configuration changed: %d monitors\n", len(heads))
//		},
//	})
//
//	// Get all monitors
//	heads := manager.GetHeads()
//	for _, head := range heads {
//		if head.Enabled {
//			fmt.Printf("Monitor: %s (%dx%d) at (%d,%d) scale %.2f\n",
//				head.Name, head.Mode.Width, head.Mode.Height,
//				head.Position.X, head.Position.Y, head.Scale)
//		}
//	}
//
//	// Find primary monitor
//	primary := manager.GetPrimaryHead()
//	if primary != nil {
//		fmt.Printf("Primary monitor: %s\n", primary.Name)
//	}
//
// # Architecture
//
// Built on **WLTurbo** (https://github.com/bnema/wlturbo) - a high-performance,
// zero-allocation Wayland client library optimized for sub-microsecond latency.
//
// **Core Components:**
// • **Protocol Layer** (internal/protocols/) - Low-level Wayland protocol bindings
// • **Client Layer** (internal/client/) - Connection and registry management
// • **High-Level APIs** - User-friendly interfaces with automatic resource management
//
// **Key Features:**
// • **Context Support**: Cancellation and timeout handling for all operations
// • **Resource Management**: Automatic cleanup and proper object lifecycle
// • **Error Handling**: Comprehensive error reporting with context
// • **Protocol Compliance**: Fully compliant with Wayland specifications
// • **Performance**: Built on zero-allocation, sub-microsecond client library
// • **Production Ready**: Complete implementations, not just stubs
//
// # Thread Safety and Performance
//
// The current implementation is **not thread-safe**. All operations should be
// performed from the same goroutine that manages the Wayland event loop.
//
// **Performance Characteristics:**
// • Built on WLTurbo for sub-microsecond latency
// • Zero-allocation hot paths for gaming applications
// • Optimized for 8000Hz gaming devices (mice/keyboards)
// • Lock-free event dispatching with object pools
//
// **For gaming/real-time applications**: Consider using WLTurbo directly
// for maximum performance.
//
// # Error Handling
//
// All methods return errors for comprehensive error handling:
//
// **Connection Errors:**
// • Wayland display connection failures
// • Context cancellation during setup
// • Timeout during protocol negotiation
//
// **Protocol Errors:**
// • Protocol not supported by compositor
// • Version mismatch or binding failures
// • Invalid object state or parameters
//
// **Resource Errors:**
// • Failed resource allocation
// • Cleanup failures (generally safe to ignore)
//
// # Installation
//
//	go get github.com/bnema/libwldevices-go
//
// # Examples and Testing
//
// **Interactive Examples:**
//	go run examples/virtual_pointer/main.go
//	go run examples/virtual_keyboard/main.go
//	go run examples/monitors_output/main.go
//
// **Tests:**
//	go test ./...
//	go test -v ./virtual_pointer
//
// **Note**: Examples and tests require a Wayland compositor with protocol support.
//
// # Related Projects
//
// • **WLTurbo**: High-performance Wayland client (https://github.com/bnema/wlturbo)
// • **Waymon**: Mouse sharing application using these bindings
//
// # Support and Compatibility
//
// **System Requirements:**
// • Linux with Wayland session (XDG_SESSION_TYPE=wayland)
// • Go 1.24+ (tested on Go 1.24)
// • Wayland compositor with virtual input support
//
// **Verification:**
//	wayland-info | grep -E "(virtual_pointer|virtual_keyboard|pointer_constraints|output_management)"
//
// Should show the required protocol interfaces.
package libwldevices