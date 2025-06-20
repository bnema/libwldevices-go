// Package keyboard_shortcuts_inhibitor provides Go bindings for the keyboard-shortcuts-inhibit-unstable-v1 Wayland protocol.
//
// This protocol specifies a way for a client to request the compositor to ignore its own keyboard shortcuts
// for a given seat, so that all key events from that seat get forwarded to a surface.
//
// # Basic Usage
//
//	// Create inhibitor manager
//	manager := NewKeyboardShortcutsInhibitorManager(display, registry)
//	
//	// Inhibit shortcuts for a surface
//	inhibitor := manager.InhibitShortcuts(surface, seat)
//	
//	// Later, destroy the inhibitor to restore shortcuts
//	inhibitor.Destroy()
//
// # Protocol Specification
//
// Based on keyboard-shortcuts-inhibit-unstable-v1 from Wayland protocols.
// Supported by most Wayland compositors including Hyprland, Sway, and wlroots-based compositors.
package keyboard_shortcuts_inhibitor

import (
	"context"
	"fmt"
)

// Error constants for keyboard shortcuts inhibitor
const (
	ERROR_ALREADY_INHIBITED = 1 // Shortcuts already inhibited on this surface
)

// KeyboardShortcutsInhibitorManager represents the zwp_keyboard_shortcuts_inhibit_manager_v1 interface.
// A global interface to inhibit keyboard shortcuts for specific surfaces.
type KeyboardShortcutsInhibitorManager interface {
	// Destroy destroys the keyboard shortcuts inhibitor manager.
	Destroy() error

	// InhibitShortcuts creates a keyboard shortcuts inhibitor for a surface.
	// The inhibitor instructs the compositor to ignore its own keyboard shortcuts
	// when the associated surface has keyboard focus.
	InhibitShortcuts(surface interface{}, seat interface{}) (KeyboardShortcutsInhibitor, error)
}

// KeyboardShortcutsInhibitor represents the zwp_keyboard_shortcuts_inhibitor_v1 interface.
// A keyboard shortcuts inhibitor instructs the compositor to ignore its own keyboard shortcuts
// when the associated surface has keyboard focus.
type KeyboardShortcutsInhibitor interface {
	// Destroy destroys the keyboard shortcuts inhibitor.
	// Keyboard shortcuts will be restored for the surface.
	Destroy() error
}

// KeyboardShortcutsInhibitorError represents errors that can occur with keyboard shortcuts inhibitor operations.
type KeyboardShortcutsInhibitorError struct {
	Code    int
	Message string
}

func (e *KeyboardShortcutsInhibitorError) Error() string {
	return fmt.Sprintf("keyboard shortcuts inhibitor error %d: %s", e.Code, e.Message)
}

// Implementation structs (these would be implemented by the actual Wayland client library)

// keyboardShortcutsInhibitorManager is the concrete implementation of KeyboardShortcutsInhibitorManager.
type keyboardShortcutsInhibitorManager struct {
	// This would contain the actual Wayland client connection and manager object
	// For now, we provide a stub implementation
	connected bool
}

// NewKeyboardShortcutsInhibitorManager creates a new keyboard shortcuts inhibitor manager.
// In a real implementation, this would connect to the Wayland compositor
// and bind to the zwp_keyboard_shortcuts_inhibit_manager_v1 global.
func NewKeyboardShortcutsInhibitorManager(ctx context.Context) (KeyboardShortcutsInhibitorManager, error) {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// This is a stub implementation - in reality, this would:
	// 1. Connect to the Wayland display with context timeout
	// 2. Get the registry
	// 3. Bind to zwp_keyboard_shortcuts_inhibit_manager_v1
	// 4. Return the manager object
	
	return &keyboardShortcutsInhibitorManager{
		connected: true,
	}, nil
}

func (m *keyboardShortcutsInhibitorManager) Destroy() error {
	if !m.connected {
		return &KeyboardShortcutsInhibitorError{
			Code:    -1,
			Message: "manager not connected",
		}
	}

	m.connected = false
	return nil
}

func (m *keyboardShortcutsInhibitorManager) InhibitShortcuts(surface interface{}, seat interface{}) (KeyboardShortcutsInhibitor, error) {
	if !m.connected {
		return nil, &KeyboardShortcutsInhibitorError{
			Code:    -1,
			Message: "manager not connected",
		}
	}

	if surface == nil {
		return nil, &KeyboardShortcutsInhibitorError{
			Code:    -1,
			Message: "surface cannot be nil",
		}
	}

	if seat == nil {
		return nil, &KeyboardShortcutsInhibitorError{
			Code:    -1,
			Message: "seat cannot be nil",
		}
	}

	// This would actually create the inhibitor object via Wayland protocol
	return &keyboardShortcutsInhibitor{
		manager: m,
		active:  true,
		surface: surface,
		seat:    seat,
	}, nil
}

// keyboardShortcutsInhibitor is the concrete implementation of KeyboardShortcutsInhibitor.
type keyboardShortcutsInhibitor struct {
	manager *keyboardShortcutsInhibitorManager
	active  bool
	surface interface{}
	seat    interface{}
}

func (i *keyboardShortcutsInhibitor) Destroy() error {
	if !i.active {
		return &KeyboardShortcutsInhibitorError{
			Code:    -1,
			Message: "inhibitor not active",
		}
	}

	// This would send the actual destroy request to the Wayland compositor
	i.active = false
	return nil
}

// Convenience functions for common operations

// CreateTemporaryInhibitor creates an inhibitor that can be easily destroyed later.
// This is useful for temporary exclusive keyboard access.
func CreateTemporaryInhibitor(manager KeyboardShortcutsInhibitorManager, surface interface{}, seat interface{}) (KeyboardShortcutsInhibitor, error) {
	return manager.InhibitShortcuts(surface, seat)
}

// InhibitorStatus represents the status of a keyboard shortcuts inhibitor.
type InhibitorStatus struct {
	Active  bool
	Surface interface{}
	Seat    interface{}
}

// GetStatus returns the current status of the inhibitor.
func GetStatus(inhibitor KeyboardShortcutsInhibitor) InhibitorStatus {
	if impl, ok := inhibitor.(*keyboardShortcutsInhibitor); ok {
		return InhibitorStatus{
			Active:  impl.active,
			Surface: impl.surface,
			Seat:    impl.seat,
		}
	}
	return InhibitorStatus{Active: false}
}