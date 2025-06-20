// Package protocols provides low-level Wayland protocol implementations for virtual input
package protocols

import (
	"github.com/bnema/wlturbo/wl"
)

// Protocol interface names for pointer constraints
const (
	PointerConstraintsInterface = "zwp_pointer_constraints_v1"
	LockedPointerInterface      = "zwp_locked_pointer_v1"
	ConfinedPointerInterface    = "zwp_confined_pointer_v1"
)

// Error codes
const (
	ErrorAlreadyConstrained = 1
)

// Lifetime enum
const (
	LifetimeOneshot    = 1
	LifetimePersistent = 2
)

// PointerConstraintsManager manages pointer constraints
type PointerConstraintsManager struct {
	wl.BaseProxy
}

// NewPointerConstraintsManager creates a new pointer constraints manager
func NewPointerConstraintsManager(ctx *wl.Context) *PointerConstraintsManager {
	manager := &PointerConstraintsManager{}
	ctx.Register(manager)
	return manager
}

// LockPointer creates a locked pointer
func (m *PointerConstraintsManager) LockPointer(surface *wl.Surface, pointer *wl.Pointer, region *wl.Region, lifetime uint32) (*LockedPointer, error) {
	locked := NewLockedPointer(m.Context())

	// Opcode 1: lock_pointer
	const opcode = 1

	// Handle nil arguments
	var surfaceProxy wl.Proxy
	if surface != nil {
		surfaceProxy = surface
	}

	var pointerProxy wl.Proxy
	if pointer != nil {
		pointerProxy = pointer
	}

	var regionProxy wl.Proxy
	if region != nil {
		regionProxy = region
	}

	err := m.Context().SendRequest(m, opcode, locked, surfaceProxy, pointerProxy, regionProxy, lifetime)
	if err != nil {
		m.Context().Unregister(locked)
		return nil, err
	}

	return locked, nil
}

// ConfinePointer creates a confined pointer
func (m *PointerConstraintsManager) ConfinePointer(surface *wl.Surface, pointer *wl.Pointer, region *wl.Region, lifetime uint32) (*ConfinedPointer, error) {
	confined := NewConfinedPointer(m.Context())

	// Opcode 2: confine_pointer
	const opcode = 2

	// Handle nil arguments
	var surfaceProxy wl.Proxy
	if surface != nil {
		surfaceProxy = surface
	}

	var pointerProxy wl.Proxy
	if pointer != nil {
		pointerProxy = pointer
	}

	var regionProxy wl.Proxy
	if region != nil {
		regionProxy = region
	}

	err := m.Context().SendRequest(m, opcode, confined, surfaceProxy, pointerProxy, regionProxy, lifetime)
	if err != nil {
		m.Context().Unregister(confined)
		return nil, err
	}

	return confined, nil
}

// Destroy destroys the pointer constraints manager
func (m *PointerConstraintsManager) Destroy() error {
	// Opcode 0: destroy
	const opcode = 0
	err := m.Context().SendRequest(m, opcode)
	m.Context().Unregister(m)
	return err
}

// Dispatch handles incoming events
func (m *PointerConstraintsManager) Dispatch(_ *wl.Event) {
	// Pointer constraints manager has no events
}

// LockedPointer represents a locked pointer
type LockedPointer struct {
	wl.BaseProxy
	handler LockedPointerHandler
}

// LockedPointerHandler handles locked pointer events
type LockedPointerHandler interface {
	HandleLocked(*LockedPointer)
	HandleUnlocked(*LockedPointer)
}

// NewLockedPointer creates a new locked pointer
func NewLockedPointer(ctx *wl.Context) *LockedPointer {
	locked := &LockedPointer{}
	ctx.Register(locked)
	return locked
}

// SetHandler sets the event handler
func (l *LockedPointer) SetHandler(handler LockedPointerHandler) {
	l.handler = handler
}

// SetCursorPositionHint sets the cursor position hint
func (l *LockedPointer) SetCursorPositionHint(surfaceX, surfaceY float64) error {
	// Opcode 1: set_cursor_position_hint
	const opcode = 1
	return l.Context().SendRequest(l, opcode, wl.Fixed(surfaceX*256.0), wl.Fixed(surfaceY*256.0))
}

// SetRegion updates the lock region
func (l *LockedPointer) SetRegion(region *wl.Region) error {
	// Opcode 2: set_region
	const opcode = 2

	// Handle nil region
	var regionProxy wl.Proxy
	if region != nil {
		regionProxy = region
	}

	return l.Context().SendRequest(l, opcode, regionProxy)
}

// Destroy destroys the locked pointer
func (l *LockedPointer) Destroy() error {
	// Opcode 0: destroy
	const opcode = 0
	err := l.Context().SendRequest(l, opcode)
	l.Context().Unregister(l)
	return err
}

// Dispatch handles incoming events
func (l *LockedPointer) Dispatch(event *wl.Event) {
	if l.handler == nil {
		return
	}

	switch event.Opcode {
	case 0: // locked
		l.handler.HandleLocked(l)
	case 1: // unlocked
		l.handler.HandleUnlocked(l)
	}
}

// ConfinedPointer represents a confined pointer
type ConfinedPointer struct {
	wl.BaseProxy
	handler ConfinedPointerHandler
}

// ConfinedPointerHandler handles confined pointer events
type ConfinedPointerHandler interface {
	HandleConfined(*ConfinedPointer)
	HandleUnconfined(*ConfinedPointer)
}

// NewConfinedPointer creates a new confined pointer
func NewConfinedPointer(ctx *wl.Context) *ConfinedPointer {
	confined := &ConfinedPointer{}
	ctx.Register(confined)
	return confined
}

// SetHandler sets the event handler
func (c *ConfinedPointer) SetHandler(handler ConfinedPointerHandler) {
	c.handler = handler
}

// SetRegion updates the confinement region
func (c *ConfinedPointer) SetRegion(region *wl.Region) error {
	// Opcode 1: set_region
	const opcode = 1

	// Handle nil region
	var regionProxy wl.Proxy
	if region != nil {
		regionProxy = region
	}

	return c.Context().SendRequest(c, opcode, regionProxy)
}

// Destroy destroys the confined pointer
func (c *ConfinedPointer) Destroy() error {
	// Opcode 0: destroy
	const opcode = 0
	err := c.Context().SendRequest(c, opcode)
	c.Context().Unregister(c)
	return err
}

// Dispatch handles incoming events
func (c *ConfinedPointer) Dispatch(event *wl.Event) {
	if c.handler == nil {
		return
	}

	switch event.Opcode {
	case 0: // confined
		c.handler.HandleConfined(c)
	case 1: // unconfined
		c.handler.HandleUnconfined(c)
	}
}
