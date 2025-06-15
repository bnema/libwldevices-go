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
)

// Lifetime constants for pointer constraints
const (
	LIFETIME_ONESHOT    = 1 // Constraint destroyed on pointer unlock/unconfine
	LIFETIME_PERSISTENT = 2 // Constraint persists across pointer unlock/unconfine
)

// Error constants for pointer constraints
const (
	ERROR_ALREADY_CONSTRAINED = 1 // Pointer constraint already requested on that surface
)

// PointerConstraintsManager represents the zwp_pointer_constraints_v1 interface.
// The global interface exposing pointer constraining functionality.
type PointerConstraintsManager interface {
	// Destroy destroys the pointer constraints manager.
	Destroy() error

	// LockPointer locks the pointer to its current position.
	// The locked pointer will not move until an unlock request is sent.
	LockPointer(surface interface{}, pointer interface{}, region interface{}, lifetime uint32) (LockedPointer, error)

	// ConfinePointer confines the pointer to a region.
	// The pointer will be confined to the region defined by the given region object.
	ConfinePointer(surface interface{}, pointer interface{}, region interface{}, lifetime uint32) (ConfinedPointer, error)
}

// LockedPointer represents the zwp_locked_pointer_v1 interface.
// The locked pointer interface allows a client to lock the cursor position.
type LockedPointer interface {
	// Destroy destroys the locked pointer object.
	Destroy() error

	// SetCursorPositionHint provides a hint about where the cursor should be positioned.
	SetCursorPositionHint(surfaceX, surfaceY float64) error

	// SetRegion sets the region used to confine the pointer.
	SetRegion(region interface{}) error
}

// ConfinedPointer represents the zwp_confined_pointer_v1 interface.
// The confined pointer interface allows a client to confine the cursor to a region.
type ConfinedPointer interface {
	// Destroy destroys the confined pointer object.
	Destroy() error

	// SetRegion sets the region used to confine the pointer.
	SetRegion(region interface{}) error
}

// PointerConstraintsError represents errors that can occur with pointer constraints operations.
type PointerConstraintsError struct {
	Code    int
	Message string
}

func (e *PointerConstraintsError) Error() string {
	return fmt.Sprintf("pointer constraints error %d: %s", e.Code, e.Message)
}

// Implementation structs (these would be implemented by the actual Wayland client library)

// pointerConstraintsManager is the concrete implementation of PointerConstraintsManager.
type pointerConstraintsManager struct {
	// This would contain the actual Wayland client connection and manager object
	// For now, we provide a stub implementation
	connected bool
}

// NewPointerConstraintsManager creates a new pointer constraints manager.
// In a real implementation, this would connect to the Wayland compositor
// and bind to the zwp_pointer_constraints_v1 global.
func NewPointerConstraintsManager(ctx context.Context) (PointerConstraintsManager, error) {
	// This is a stub implementation - in reality, this would:
	// 1. Connect to the Wayland display
	// 2. Get the registry
	// 3. Bind to zwp_pointer_constraints_v1
	// 4. Return the manager object
	
	return &pointerConstraintsManager{
		connected: true,
	}, nil
}

func (m *pointerConstraintsManager) Destroy() error {
	if !m.connected {
		return &PointerConstraintsError{
			Code:    -1,
			Message: "manager not connected",
		}
	}

	m.connected = false
	return nil
}

func (m *pointerConstraintsManager) LockPointer(surface interface{}, pointer interface{}, region interface{}, lifetime uint32) (LockedPointer, error) {
	if !m.connected {
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

	// This would actually create the locked pointer object via Wayland protocol
	return &lockedPointer{
		manager: m,
		active:  true,
	}, nil
}

func (m *pointerConstraintsManager) ConfinePointer(surface interface{}, pointer interface{}, region interface{}, lifetime uint32) (ConfinedPointer, error) {
	if !m.connected {
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

	// This would actually create the confined pointer object via Wayland protocol
	return &confinedPointer{
		manager: m,
		active:  true,
	}, nil
}

// lockedPointer is the concrete implementation of LockedPointer.
type lockedPointer struct {
	manager *pointerConstraintsManager
	active  bool
}

func (l *lockedPointer) Destroy() error {
	if !l.active {
		return &PointerConstraintsError{
			Code:    -1,
			Message: "locked pointer not active",
		}
	}

	l.active = false
	return nil
}

func (l *lockedPointer) SetCursorPositionHint(surfaceX, surfaceY float64) error {
	if !l.active {
		return &PointerConstraintsError{
			Code:    -1,
			Message: "locked pointer not active",
		}
	}

	// This would send the actual cursor position hint request to the Wayland compositor
	return nil
}

func (l *lockedPointer) SetRegion(region interface{}) error {
	if !l.active {
		return &PointerConstraintsError{
			Code:    -1,
			Message: "locked pointer not active",
		}
	}

	// This would send the actual set region request to the Wayland compositor
	return nil
}

// confinedPointer is the concrete implementation of ConfinedPointer.
type confinedPointer struct {
	manager *pointerConstraintsManager
	active  bool
}

func (c *confinedPointer) Destroy() error {
	if !c.active {
		return &PointerConstraintsError{
			Code:    -1,
			Message: "confined pointer not active",
		}
	}

	c.active = false
	return nil
}

func (c *confinedPointer) SetRegion(region interface{}) error {
	if !c.active {
		return &PointerConstraintsError{
			Code:    -1,
			Message: "confined pointer not active",
		}
	}

	// This would send the actual set region request to the Wayland compositor
	return nil
}

// Convenience functions for common operations

// LockPointerAtCurrentPosition locks the pointer at its current position with oneshot lifetime.
func LockPointerAtCurrentPosition(manager PointerConstraintsManager, surface interface{}, pointer interface{}) (LockedPointer, error) {
	return manager.LockPointer(surface, pointer, nil, LIFETIME_ONESHOT)
}

// LockPointerPersistent locks the pointer at its current position with persistent lifetime.
func LockPointerPersistent(manager PointerConstraintsManager, surface interface{}, pointer interface{}) (LockedPointer, error) {
	return manager.LockPointer(surface, pointer, nil, LIFETIME_PERSISTENT)
}

// ConfinePointerToRegion confines the pointer to a specific region with oneshot lifetime.
func ConfinePointerToRegion(manager PointerConstraintsManager, surface interface{}, pointer interface{}, region interface{}) (ConfinedPointer, error) {
	return manager.ConfinePointer(surface, pointer, region, LIFETIME_ONESHOT)
}