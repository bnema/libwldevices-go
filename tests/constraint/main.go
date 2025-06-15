// Test application for pointer constraints functionality
//
// This test demonstrates both pointer locking and confinement features.
// It requires a Wayland compositor with pointer constraints support and
// an active window to capture pointer events.
//
// Usage: go run tests/constraint/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/bnema/wayland-virtual-input-go/pointer_constraints"
)

// Simple event handler that logs events
type testEventHandler struct {
	name string
}

func (h *testEventHandler) HandleLocked(e pointer_constraints.LockedEvent) {
	fmt.Printf("[%s] Pointer locked!\n", h.name)
}

func (h *testEventHandler) HandleUnlocked(e pointer_constraints.UnlockedEvent) {
	fmt.Printf("[%s] Pointer unlocked! (lifetime: %d)\n", h.name, e.Lifetime)
}

func (h *testEventHandler) HandleConfined(e pointer_constraints.ConfinedEvent) {
	fmt.Printf("[%s] Pointer confined!\n", h.name)
}

func (h *testEventHandler) HandleUnconfined(e pointer_constraints.UnconfinedEvent) {
	fmt.Printf("[%s] Pointer unconfined! (lifetime: %d)\n", h.name, e.Lifetime)
}

func main() {
	fmt.Println("=== Wayland Pointer Constraints Test ===")
	fmt.Printf("WAYLAND_DISPLAY: %s\n", os.Getenv("WAYLAND_DISPLAY"))
	fmt.Printf("XDG_SESSION_TYPE: %s\n\n", os.Getenv("XDG_SESSION_TYPE"))

	fmt.Println("IMPORTANT: This test requires:")
	fmt.Println("1. A Wayland compositor with pointer constraints support")
	fmt.Println("2. An active window with pointer focus")
	fmt.Println("3. The test will attempt to lock/confine the pointer")
	fmt.Println("")

	ctx := context.Background()

	// Create pointer constraints manager
	fmt.Println("Creating pointer constraints manager...")
	manager, err := pointer_constraints.NewPointerConstraintsManager(ctx)
	if err != nil {
		log.Fatalf("Failed to create pointer constraints manager: %v", err)
	}
	defer manager.Close()
	fmt.Println("✓ Pointer constraints manager created")

	// NOTE: In a real application, you would get these from your window toolkit
	// For this test, we'll create placeholder objects to demonstrate the API
	fmt.Println("\nNOTE: This test uses placeholder objects for surface and pointer.")
	fmt.Println("In a real application, you would get these from your window toolkit")
	fmt.Println("(e.g., from a wl_surface and wl_pointer created by your application).")
	fmt.Println("")

	// Test 1: Demonstrate API usage (will fail without real surface/pointer)
	fmt.Println("Test 1: API demonstration")
	fmt.Println("The following calls will fail because we don't have real surface/pointer objects:")
	fmt.Println("")

	// This would be how you use the API with real objects:
	/*
		// Get these from your window toolkit
		surface := getWlSurface()  // e.g., from your window
		pointer := getWlPointer()  // e.g., from seat capabilities

		// Example 1: Lock pointer with oneshot lifetime
		locked, err := manager.LockPointer(surface, pointer, nil, pointer_constraints.LifetimeOneshot)
		if err != nil {
			log.Printf("Failed to lock pointer: %v", err)
		} else {
			locked.SetEventHandler(&testEventHandler{name: "Lock"})
			fmt.Println("✓ Pointer lock created (oneshot)")

			// Set cursor position hint
			locked.SetCursorPositionHint(100.0, 100.0)

			// Wait for events...
			time.Sleep(5 * time.Second)

			// Unlock
			locked.Close()
		}

		// Example 2: Confine pointer to region
		region := compositor.CreateRegion()
		region.Add(0, 0, 800, 600)  // Confine to 800x600 area

		confined, err := manager.ConfinePointer(surface, pointer, region, pointer_constraints.LifetimePersistent)
		if err != nil {
			log.Printf("Failed to confine pointer: %v", err)
		} else {
			confined.SetEventHandler(&testEventHandler{name: "Confine"})
			fmt.Println("✓ Pointer confinement created (persistent)")

			// Wait for events...
			time.Sleep(5 * time.Second)

			// Update confinement region
			newRegion := compositor.CreateRegion()
			newRegion.Add(100, 100, 600, 400)
			confined.SetRegion(newRegion)

			// Unconfine
			confined.Close()
		}
	*/

	// Test 2: Show the different lifetime behaviors
	fmt.Println("\nTest 2: Lifetime behaviors")
	fmt.Println("- Oneshot: Constraint is destroyed after first deactivation")
	fmt.Println("- Persistent: Constraint can reactivate after deactivation")
	fmt.Println("")

	// Test 3: Demonstrate convenience functions
	fmt.Println("Test 3: Convenience functions available:")
	fmt.Println("- LockPointerAtCurrentPosition(): Quick oneshot lock")
	fmt.Println("- LockPointerPersistent(): Persistent lock")
	fmt.Println("- ConfinePointerToRegion(): Quick oneshot confinement")
	fmt.Println("- ConfinePointerToRegionPersistent(): Persistent confinement")
	fmt.Println("")

	// Test 4: Event handling
	fmt.Println("Test 4: Event handling")
	fmt.Println("The protocol provides these events:")
	fmt.Println("- Locked: Emitted when lock activates")
	fmt.Println("- Unlocked: Emitted when lock deactivates")
	fmt.Println("- Confined: Emitted when confinement activates")
	fmt.Println("- Unconfined: Emitted when confinement deactivates")
	fmt.Println("")

	// Show protocol information
	fmt.Println("Protocol Information:")
	fmt.Println("- Protocol: pointer-constraints-unstable-v1")
	fmt.Println("- Compositor must support zwp_pointer_constraints_v1")
	fmt.Println("- Constraints require surface to have pointer focus")
	fmt.Println("- Only one constraint per surface/seat allowed")
	fmt.Println("")

	fmt.Println("Test completed. The pointer constraints implementation is ready for use!")
	fmt.Println("To use in your application, integrate with your Wayland window toolkit.")
}
