// Example demonstrating pointer constraints usage in a Wayland application
//
// This example shows how to integrate pointer constraints with your Wayland application.
// It provides code snippets that you can adapt for your specific window toolkit.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bnema/libwldevices-go/pointer_constraints"
	"github.com/bnema/wlturbo/wl"
)

// Application represents your Wayland application
type Application struct {
	// Your window toolkit components would go here
	surface    *wl.Surface
	pointer    *wl.Pointer
	compositor *wl.Compositor

	// Pointer constraints
	constraintManager  *pointer_constraints.PointerConstraintsManager
	currentLock        *pointer_constraints.LockedPointer
	currentConfinement *pointer_constraints.ConfinedPointer
}

// Example 1: First-person game camera control
func (app *Application) enableFPSControls() error {
	// Lock pointer for FPS-style mouse look
	lock, err := app.constraintManager.LockPointer(
		app.surface,
		app.pointer,
		nil, // No region restriction
		pointer_constraints.LifetimePersistent,
	)
	if err != nil {
		return fmt.Errorf("failed to lock pointer: %w", err)
	}

	// Store the lock
	app.currentLock = lock

	// Set hint for where cursor should appear when unlocked
	// (e.g., center of window)
	err = lock.SetCursorPositionHint(400.0, 300.0)
	if err != nil {
		log.Printf("Warning: failed to set cursor position hint: %v", err)
	}

	log.Println("Pointer locked for FPS controls")
	return nil
}

// Example 2: Drawing application with canvas boundaries
func (app *Application) confineToCanvas(x, y, width, height int32) error {
	// Create region for canvas area
	region, err := app.compositor.CreateRegion()
	if err != nil {
		return fmt.Errorf("failed to create region: %w", err)
	}
	if err := region.Add(x, y, width, height); err != nil {
		return fmt.Errorf("failed to add region: %w", err)
	}

	// Confine pointer to canvas
	confinement, err := app.constraintManager.ConfinePointer(
		app.surface,
		app.pointer,
		region,
		pointer_constraints.LifetimePersistent,
	)
	if err != nil {
		return fmt.Errorf("failed to confine pointer: %w", err)
	}

	app.currentConfinement = confinement
	log.Printf("Pointer confined to canvas area (%d,%d %dx%d)", x, y, width, height)
	return nil
}

// Example 3: RTS game edge scrolling
func (app *Application) setupEdgeScrolling() error {
	// Define scroll zones (10px from each edge)
	scrollMargin := int32(10)
	windowWidth := int32(1920)
	windowHeight := int32(1080)

	// Create region that excludes the scroll zones
	region, err := app.compositor.CreateRegion()
	if err != nil {
		return fmt.Errorf("failed to create region: %w", err)
	}
	if err := region.Add(scrollMargin, scrollMargin,
		windowWidth-2*scrollMargin, windowHeight-2*scrollMargin); err != nil {
		return fmt.Errorf("failed to add region: %w", err)
	}

	// Use oneshot confinement - releases when user wants to scroll
	confinement, err := app.constraintManager.ConfinePointer(
		app.surface,
		app.pointer,
		region,
		pointer_constraints.LifetimeOneshot,
	)
	if err != nil {
		return fmt.Errorf("failed to setup edge scrolling: %w", err)
	}

	app.currentConfinement = confinement
	log.Println("Edge scrolling setup complete")
	return nil
}

// Example 4: Toggle pointer lock
func (app *Application) togglePointerLock() error {
	if app.currentLock != nil {
		// Unlock pointer
		err := app.currentLock.Destroy()
		if err != nil {
			log.Printf("Warning: failed to destroy lock: %v", err)
		}
		app.currentLock = nil
		log.Println("Pointer unlocked")
	} else {
		// Lock pointer using convenience function
		lock, err := pointer_constraints.LockPointerAtCurrentPosition(
			app.constraintManager,
			app.surface,
			app.pointer,
		)
		if err != nil {
			return fmt.Errorf("failed to lock pointer: %w", err)
		}

		app.currentLock = lock
		log.Println("Pointer locked")
	}
	return nil
}

// Example 5: Confine pointer to a specific region
func (app *Application) confineToRegion(x, y, width, height int32) error {
	// Create the region
	region, err := app.compositor.CreateRegion()
	if err != nil {
		return fmt.Errorf("failed to create region: %w", err)
	}
	if err := region.Add(x, y, width, height); err != nil {
		return fmt.Errorf("failed to add region: %w", err)
	}

	// Use convenience function to confine pointer
	confinement, err := pointer_constraints.ConfinePointerToRegion(
		app.constraintManager,
		app.surface,
		app.pointer,
		region,
	)
	if err != nil {
		return fmt.Errorf("failed to confine pointer to region: %w", err)
	}

	app.currentConfinement = confinement
	log.Printf("Pointer confined to region (%d,%d %dx%d)", x, y, width, height)
	return nil
}

// Example 6: Update confinement region
func (app *Application) updateConfinementRegion(x, y, width, height int32) error {
	if app.currentConfinement == nil {
		return fmt.Errorf("no active confinement to update")
	}

	// Create new region
	region, err := app.compositor.CreateRegion()
	if err != nil {
		return fmt.Errorf("failed to create region: %w", err)
	}
	if err := region.Add(x, y, width, height); err != nil {
		return fmt.Errorf("failed to add region: %w", err)
	}

	// Update the confinement region
	err = app.currentConfinement.SetRegion(region)
	if err != nil {
		return fmt.Errorf("failed to update confinement region: %w", err)
	}

	log.Printf("Confinement region updated to (%d,%d %dx%d)", x, y, width, height)
	return nil
}

// Clean up all constraints
func (app *Application) cleanup() {
	if app.currentLock != nil {
		if err := app.currentLock.Destroy(); err != nil {
			log.Printf("Warning: failed to destroy lock: %v", err)
		}
		app.currentLock = nil
	}

	if app.currentConfinement != nil {
		if err := app.currentConfinement.Destroy(); err != nil {
			log.Printf("Warning: failed to destroy confinement: %v", err)
		}
		app.currentConfinement = nil
	}

	if app.constraintManager != nil {
		if err := app.constraintManager.Close(); err != nil {
			log.Printf("Warning: failed to close constraint manager: %v", err)
		}
	}
}

func demonstrateAPI() {
	fmt.Println("=== Pointer Constraints API Demonstration ===")
	fmt.Println()

	// Show how to create the manager
	ctx := context.Background()
	manager, err := pointer_constraints.NewPointerConstraintsManager(ctx)
	if err != nil {
		log.Printf("Note: %v", err)
		log.Println("This is expected if running outside a Wayland session")
		fmt.Println()
		fmt.Println("In a real application, you would:")
		fmt.Println("1. Get wl.Surface from your window toolkit")
		fmt.Println("2. Get wl.Pointer from seat capabilities")
		fmt.Println("3. Create regions as needed")
		fmt.Println("4. Apply constraints using the manager")
		return
	}
	defer func() { _ = manager.Close() }()

	log.Println("✓ Pointer constraints manager created successfully")

	// Test basic functionality (without real Wayland objects)
	fmt.Println()
	fmt.Println("Testing API with nil arguments (should handle gracefully):")

	// Test lock pointer with invalid arguments
	_, err = manager.LockPointer(nil, nil, nil, pointer_constraints.LifetimeOneshot)
	if err != nil {
		fmt.Printf("  LockPointer with nil args: %v\n", err)
	}

	// Test confine pointer with invalid arguments
	_, err = manager.ConfinePointer(nil, nil, nil, pointer_constraints.LifetimePersistent)
	if err != nil {
		fmt.Printf("  ConfinePointer with nil args: %v\n", err)
	}

	// Test convenience functions
	_, err = pointer_constraints.LockPointerAtCurrentPosition(manager, nil, nil)
	if err != nil {
		fmt.Printf("  LockPointerAtCurrentPosition: %v\n", err)
	}

	_, err = pointer_constraints.LockPointerPersistent(manager, nil, nil)
	if err != nil {
		fmt.Printf("  LockPointerPersistent: %v\n", err)
	}

	_, err = pointer_constraints.ConfinePointerToRegion(manager, nil, nil, nil)
	if err != nil {
		fmt.Printf("  ConfinePointerToRegion: %v\n", err)
	}

	fmt.Println("✓ API responds correctly to invalid arguments")
}

func main() {
	fmt.Println("Pointer Constraints Integration Example")
	fmt.Println("=====================================")
	fmt.Println()
	fmt.Println("This example demonstrates how to integrate pointer constraints")
	fmt.Println("into your Wayland application. The code shows common use cases:")
	fmt.Println()
	fmt.Println("1. FPS game controls (pointer locking)")
	fmt.Println("2. Drawing application (confine to canvas)")
	fmt.Println("3. RTS edge scrolling (confinement with zones)")
	fmt.Println("4. Toggle lock functionality")
	fmt.Println("5. Dynamic region updates")
	fmt.Println()

	demonstrateAPI()

	fmt.Println()
	fmt.Println("Key Integration Points:")
	fmt.Println("======================")
	fmt.Println()
	fmt.Println("1. **Lifetime Management**")
	fmt.Println("   - LifetimeOneshot: Constraint destroyed on first unlock/unconfine")
	fmt.Println("   - LifetimePersistent: Constraint persists across state changes")
	fmt.Println()
	fmt.Println("2. **Error Handling**")
	fmt.Println("   - Always check errors when creating constraints")
	fmt.Println("   - Handle ERROR_ALREADY_CONSTRAINED gracefully")
	fmt.Println("   - Constraints may fail if surface doesn't have focus")
	fmt.Println()
	fmt.Println("3. **Resource Cleanup**")
	fmt.Println("   - Call Destroy() on constraints when done")
	fmt.Println("   - Close the manager when application exits")
	fmt.Println("   - Check for nil before calling methods")
	fmt.Println()
	fmt.Println("4. **Integration with Window Toolkit**")
	fmt.Println("   - Get wl.Surface from your window")
	fmt.Println("   - Get wl.Pointer from seat capabilities")
	fmt.Println("   - Create wl.Region objects for confinement areas")
	fmt.Println("   - Handle constraint activation based on focus events")
	fmt.Println()
	fmt.Println("5. **Best Practices**")
	fmt.Println("   - Only one constraint per surface/seat at a time")
	fmt.Println("   - Constraints only activate when surface has pointer focus")
	fmt.Println("   - Compositor decides when to actually activate constraints")
	fmt.Println("   - Provide user feedback about constraint state")
	fmt.Println("   - Always provide an escape mechanism (hotkey, etc.)")
	fmt.Println()
	fmt.Println("Example Usage in Your Application:")
	fmt.Println("=================================")
	fmt.Println(`
// Initialize
ctx := context.Background()
manager, err := pointer_constraints.NewPointerConstraintsManager(ctx)
if err != nil {
    log.Fatal(err)
}
defer manager.Close()

// Lock pointer for FPS controls
lock, err := manager.LockPointer(surface, pointer, nil, 
    pointer_constraints.LifetimePersistent)
if err != nil {
    log.Printf("Failed to lock pointer: %v", err)
    return
}

// Set cursor position hint
lock.SetCursorPositionHint(centerX, centerY)

// Later, unlock
lock.Destroy()
`)
	fmt.Println()
	fmt.Println("For complete integration examples, see the Application struct methods above.")
}