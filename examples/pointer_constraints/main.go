// Example demonstrating pointer constraints usage in a Wayland application
//
// This example shows how to integrate pointer constraints with your Wayland application.
// It provides code snippets that you can adapt for your specific window toolkit.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bnema/wayland-virtual-input-go/pointer_constraints"
	"github.com/neurlang/wayland/wl"
)

// CustomEventHandler demonstrates handling constraint events in your application
type CustomEventHandler struct {
	app *Application
}

func (h *CustomEventHandler) HandleLocked(e pointer_constraints.LockedEvent) {
	log.Println("Pointer locked - hiding cursor and capturing input")
	h.app.onPointerLocked()
}

func (h *CustomEventHandler) HandleUnlocked(e pointer_constraints.UnlockedEvent) {
	log.Println("Pointer unlocked - showing cursor")
	h.app.onPointerUnlocked()

	if e.Lifetime == pointer_constraints.LifetimeOneshot {
		log.Println("Lock was oneshot - constraint is now defunct")
	}
}

func (h *CustomEventHandler) HandleConfined(e pointer_constraints.ConfinedEvent) {
	log.Println("Pointer confined to region")
	h.app.onPointerConfined()
}

func (h *CustomEventHandler) HandleUnconfined(e pointer_constraints.UnconfinedEvent) {
	log.Println("Pointer no longer confined")
	h.app.onPointerUnconfined()
}

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

	// Set event handler
	handler := &CustomEventHandler{app: app}
	lock.SetEventHandler(handler)

	// Store the lock
	app.currentLock = lock

	// Set hint for where cursor should appear when unlocked
	// (e.g., center of window)
	lock.SetCursorPositionHint(400.0, 300.0)

	return nil
}

// Example 2: Drawing application with canvas boundaries
func (app *Application) confineToCanvas(x, y, width, height int32) error {
	// Create region for canvas area
	region, err := app.compositor.CreateRegion()
	if err != nil {
		return fmt.Errorf("failed to create region: %w", err)
	}
	region.Add(x, y, width, height)

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

	// Set event handler
	handler := &CustomEventHandler{app: app}
	confinement.SetEventHandler(handler)

	app.currentConfinement = confinement
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
	region.Add(scrollMargin, scrollMargin,
		windowWidth-2*scrollMargin, windowHeight-2*scrollMargin)

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

	confinement.SetEventHandler(&CustomEventHandler{app: app})
	app.currentConfinement = confinement

	return nil
}

// Example 4: Toggle pointer lock with hotkey
func (app *Application) togglePointerLock() {
	if app.currentLock != nil && app.currentLock.IsActive() {
		// Unlock pointer
		app.currentLock.Close()
		app.currentLock = nil
		log.Println("Pointer unlocked")
	} else {
		// Lock pointer
		lock, err := pointer_constraints.LockPointerAtCurrentPosition(
			app.constraintManager,
			app.surface,
			app.pointer,
		)
		if err != nil {
			log.Printf("Failed to lock pointer: %v", err)
			return
		}

		lock.SetEventHandler(&CustomEventHandler{app: app})
		app.currentLock = lock
		log.Println("Pointer locked")
	}
}

// Callbacks for constraint state changes
func (app *Application) onPointerLocked() {
	// Hide cursor sprite
	// Start capturing relative motion events
	// Update UI to show locked state
}

func (app *Application) onPointerUnlocked() {
	// Show cursor sprite
	// Stop relative motion capture
	// Update UI to show unlocked state
}

func (app *Application) onPointerConfined() {
	// Update UI to show confined state
	// Maybe show visual boundaries
}

func (app *Application) onPointerUnconfined() {
	// Update UI to show unconfined state
	// Remove visual boundaries
}

func main() {
	fmt.Println("=== Pointer Constraints Integration Example ===")
	fmt.Println()
	fmt.Println("This example demonstrates how to integrate pointer constraints")
	fmt.Println("into your Wayland application. The code shows common use cases:")
	fmt.Println()
	fmt.Println("1. FPS game controls (pointer locking)")
	fmt.Println("2. Drawing application (confine to canvas)")
	fmt.Println("3. RTS edge scrolling (confinement with zones)")
	fmt.Println("4. Toggle lock with hotkey")
	fmt.Println()
	fmt.Println("To use these examples:")
	fmt.Println("1. Get wl.Surface from your window")
	fmt.Println("2. Get wl.Pointer from seat capabilities")
	fmt.Println("3. Create constraint manager")
	fmt.Println("4. Apply constraints as needed")
	fmt.Println()

	// Show how to create the manager
	ctx := context.Background()
	manager, err := pointer_constraints.NewPointerConstraintsManager(ctx)
	if err != nil {
		log.Printf("Note: %v", err)
		log.Println("This is expected if running outside a Wayland session")
	} else {
		defer manager.Close()
		log.Println("✓ Pointer constraints manager created successfully")
	}

	fmt.Println()
	fmt.Println("Key points:")
	fmt.Println("- Constraints only activate when surface has pointer focus")
	fmt.Println("- Only one constraint per surface/seat at a time")
	fmt.Println("- Compositor decides when to activate constraints")
	fmt.Println("- Use event handlers to track constraint state")
	fmt.Println("- Remember to close constraints when done")
}
