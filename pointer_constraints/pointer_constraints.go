// Package pointer_constraints provides Go bindings for the pointer-constraints-unstable-v1 Wayland protocol.
//
// This protocol specifies a set of interfaces used for adding constraints to the motion of a pointer.
// Possible constraints include confining pointer motions to a given region, or locking it to its current position.
//
// # Basic Usage
//
//	// Create constraint manager
//	manager := NewPointerConstraintsManager(display, registry)
//	
//	// Lock pointer to current position (exclusive capture)
//	lockedPointer := manager.LockPointer(surface, pointer, region, lifetime)
//	
//	// Or confine pointer to a region
//	confinedPointer := manager.ConfinePointer(surface, pointer, region, lifetime)
//
// # Protocol Specification
//
// Based on pointer-constraints-unstable-v1 from Wayland protocols.
// Supported by most Wayland compositors including Hyprland, Sway, and wlroots-based compositors.
package pointer_constraints

import (
	"context"
	"fmt"

	"github.com/bnema/libwldevices-go/internal/client"
	"github.com/bnema/libwldevices-go/internal/protocols"
	"github.com/bnema/wlturbo/wl"
)

// Lifetime constants for pointer constraints
const (
	LIFETIME_ONESHOT    = 1 // Constraint destroyed on pointer unlock/unconfine
	LIFETIME_PERSISTENT = 2 // Constraint persists across pointer unlock/unconfine
	
	// Alternative names used in examples
	LifetimeOneshot    = LIFETIME_ONESHOT
	LifetimePersistent = LIFETIME_PERSISTENT
)

// Error constants for pointer constraints
const (
	ERROR_ALREADY_CONSTRAINED = 1 // Pointer constraint already requested on that surface
)

// PointerConstraintsManager manages pointer constraints
type PointerConstraintsManager struct {
	client  *client.Client
	manager *protocols.PointerConstraintsManager
}

// LockedPointer represents a locked pointer constraint
type LockedPointer struct {
	manager *PointerConstraintsManager
	locked  *protocols.LockedPointer
}

// ConfinedPointer represents a confined pointer constraint
type ConfinedPointer struct {
	manager  *PointerConstraintsManager
	confined *protocols.ConfinedPointer
}

// PointerConstraintsError represents errors that can occur with pointer constraints operations.
type PointerConstraintsError struct {
	Code    int
	Message string
}

func (e *PointerConstraintsError) Error() string {
	return fmt.Sprintf("pointer constraints error %d: %s", e.Code, e.Message)
}

// NewPointerConstraintsManager creates a new pointer constraints manager.
func NewPointerConstraintsManager(ctx context.Context) (*PointerConstraintsManager, error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Create Wayland client with timeout
	type clientResult struct {
		client *client.Client
		err    error
	}
	
	clientCh := make(chan clientResult, 1)
	go func() {
		c, err := client.NewClient()
		clientCh <- clientResult{client: c, err: err}
	}()
	
	// Wait for client creation or context cancellation
	var c *client.Client
	select {
	case result := <-clientCh:
		if result.err != nil {
			return nil, fmt.Errorf("failed to create client: %w", result.err)
		}
		c = result.client
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled during client creation: %w", ctx.Err())
	}

	// Check if pointer constraints protocol is available using the client's detection
	if !c.HasPointerConstraints() {
		_ = c.Close()
		return nil, fmt.Errorf("zwp_pointer_constraints_v1 not available - compositor may not support pointer-constraints protocol")
	}

	pcm := &PointerConstraintsManager{
		client: c,
	}

	// Check context before binding
	select {
	case <-ctx.Done():
		_ = c.Close()
		return nil, fmt.Errorf("context cancelled before binding: %w", ctx.Err())
	default:
	}
	
	// Use the constraints manager name from the client
	managerName := c.GetConstraintsManagerName()
	registry := c.GetRegistry()
	wayland_context := c.GetContext()

	// Create and bind pointer constraints manager using detected name
	pcm.manager = protocols.NewPointerConstraintsManager(wayland_context)
	err := registry.Bind(managerName, protocols.PointerConstraintsInterface, 1, pcm.manager)
	if err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("failed to bind pointer constraints manager: %w", err)
	}

	return pcm, nil
}

// Close closes the pointer constraints manager
func (pcm *PointerConstraintsManager) Close() error {
	if pcm.manager != nil {
		_ = pcm.manager.Destroy()
	}
	if pcm.client != nil {
		return pcm.client.Close()
	}
	return nil
}

// Destroy destroys the pointer constraints manager
func (pcm *PointerConstraintsManager) Destroy() error {
	return pcm.Close()
}

// LockPointer locks the pointer to its current position
func (pcm *PointerConstraintsManager) LockPointer(surface interface{}, pointer interface{}, region interface{}, lifetime uint32) (*LockedPointer, error) {
	if pcm.manager == nil {
		return nil, &PointerConstraintsError{
			Code:    -1,
			Message: "manager not connected",
		}
	}

	if lifetime != LIFETIME_ONESHOT && lifetime != LIFETIME_PERSISTENT {
		return nil, &PointerConstraintsError{
			Code:    -1,
			Message: "invalid lifetime value",
		}
	}

	// Convert interfaces to proper Wayland types
	wlSurface, ok := surface.(*wl.Surface)
	if !ok && surface != nil {
		return nil, &PointerConstraintsError{
			Code:    -1,
			Message: "surface must be a *wl.Surface",
		}
	}

	wlPointer, ok := pointer.(*wl.Pointer)
	if !ok && pointer != nil {
		return nil, &PointerConstraintsError{
			Code:    -1,
			Message: "pointer must be a *wl.Pointer",
		}
	}

	wlRegion, ok := region.(*wl.Region)
	if !ok && region != nil {
		return nil, &PointerConstraintsError{
			Code:    -1,
			Message: "region must be a *wl.Region",
		}
	}

	locked, err := pcm.manager.LockPointer(wlSurface, wlPointer, wlRegion, lifetime)
	if err != nil {
		return nil, fmt.Errorf("failed to lock pointer: %w", err)
	}

	return &LockedPointer{
		manager: pcm,
		locked:  locked,
	}, nil
}

// ConfinePointer confines the pointer to a region
func (pcm *PointerConstraintsManager) ConfinePointer(surface interface{}, pointer interface{}, region interface{}, lifetime uint32) (*ConfinedPointer, error) {
	if pcm.manager == nil {
		return nil, &PointerConstraintsError{
			Code:    -1,
			Message: "manager not connected",
		}
	}

	if lifetime != LIFETIME_ONESHOT && lifetime != LIFETIME_PERSISTENT {
		return nil, &PointerConstraintsError{
			Code:    -1,
			Message: "invalid lifetime value",
		}
	}

	// Convert interfaces to proper Wayland types
	wlSurface, ok := surface.(*wl.Surface)
	if !ok && surface != nil {
		return nil, &PointerConstraintsError{
			Code:    -1,
			Message: "surface must be a *wl.Surface",
		}
	}

	wlPointer, ok := pointer.(*wl.Pointer)
	if !ok && pointer != nil {
		return nil, &PointerConstraintsError{
			Code:    -1,
			Message: "pointer must be a *wl.Pointer",
		}
	}

	wlRegion, ok := region.(*wl.Region)
	if !ok && region != nil {
		return nil, &PointerConstraintsError{
			Code:    -1,
			Message: "region must be a *wl.Region",
		}
	}

	confined, err := pcm.manager.ConfinePointer(wlSurface, wlPointer, wlRegion, lifetime)
	if err != nil {
		return nil, fmt.Errorf("failed to confine pointer: %w", err)
	}

	return &ConfinedPointer{
		manager:  pcm,
		confined: confined,
	}, nil
}

// LockedPointer methods

// Destroy destroys the locked pointer object
func (lp *LockedPointer) Destroy() error {
	if lp.locked != nil {
		return lp.locked.Destroy()
	}
	return nil
}

// SetCursorPositionHint provides a hint about where the cursor should be positioned
func (lp *LockedPointer) SetCursorPositionHint(surfaceX, surfaceY float64) error {
	if lp.locked != nil {
		return lp.locked.SetCursorPositionHint(surfaceX, surfaceY)
	}
	return &PointerConstraintsError{
		Code:    -1,
		Message: "locked pointer not active",
	}
}

// SetRegion sets the region used to confine the pointer
func (lp *LockedPointer) SetRegion(region interface{}) error {
	if lp.locked == nil {
		return &PointerConstraintsError{
			Code:    -1,
			Message: "locked pointer not active",
		}
	}

	wlRegion, ok := region.(*wl.Region)
	if !ok && region != nil {
		return &PointerConstraintsError{
			Code:    -1,
			Message: "region must be a *wl.Region",
		}
	}

	return lp.locked.SetRegion(wlRegion)
}

// ConfinedPointer methods

// Destroy destroys the confined pointer object
func (cp *ConfinedPointer) Destroy() error {
	if cp.confined != nil {
		return cp.confined.Destroy()
	}
	return nil
}

// SetRegion sets the region used to confine the pointer
func (cp *ConfinedPointer) SetRegion(region interface{}) error {
	if cp.confined == nil {
		return &PointerConstraintsError{
			Code:    -1,
			Message: "confined pointer not active",
		}
	}

	wlRegion, ok := region.(*wl.Region)
	if !ok && region != nil {
		return &PointerConstraintsError{
			Code:    -1,
			Message: "region must be a *wl.Region",
		}
	}

	return cp.confined.SetRegion(wlRegion)
}

// Convenience functions for common operations

// LockPointerAtCurrentPosition locks the pointer at its current position with oneshot lifetime.
func LockPointerAtCurrentPosition(manager *PointerConstraintsManager, surface interface{}, pointer interface{}) (*LockedPointer, error) {
	return manager.LockPointer(surface, pointer, nil, LIFETIME_ONESHOT)
}

// LockPointerPersistent locks the pointer at its current position with persistent lifetime.
func LockPointerPersistent(manager *PointerConstraintsManager, surface interface{}, pointer interface{}) (*LockedPointer, error) {
	return manager.LockPointer(surface, pointer, nil, LIFETIME_PERSISTENT)
}

// ConfinePointerToRegion confines the pointer to a specific region with oneshot lifetime.
func ConfinePointerToRegion(manager *PointerConstraintsManager, surface interface{}, pointer interface{}, region interface{}) (*ConfinedPointer, error) {
	return manager.ConfinePointer(surface, pointer, region, LIFETIME_ONESHOT)
}

// globalHandler is a helper type for handling registry globals
type globalHandler struct {
	found   *bool
	name    *uint32
	version *uint32
}

// HandleRegistryGlobal implements the RegistryGlobalHandler interface
func (h *globalHandler) HandleRegistryGlobal(event wl.RegistryGlobalEvent) {
	if event.Interface == protocols.PointerConstraintsInterface {
		*h.found = true
		*h.name = event.Name
		*h.version = event.Version
	}
}