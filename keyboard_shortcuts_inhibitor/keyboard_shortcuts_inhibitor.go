// Package keyboard_shortcuts_inhibitor provides Go bindings for the keyboard-shortcuts-inhibit-unstable-v1 Wayland protocol.
//
// This protocol specifies a way for a client to request the compositor to ignore its own keyboard shortcuts
// for a given seat, so that all key events from that seat get forwarded to a surface.
//
// # Basic Usage
//
//	ctx := context.Background()
//	manager, err := NewKeyboardShortcutsInhibitorManager(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer manager.Close()
//
//	// Inhibit shortcuts for a surface and seat
//	inhibitor, err := manager.InhibitShortcuts(surface, seat)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer inhibitor.Destroy()
//
//	// Observe whether the compositor actually activated the inhibitor
//	if inhibitor.Active() {
//		// shortcuts are inhibited
//	}
//
// # Protocol Specification
//
// Based on keyboard-shortcuts-inhibit-unstable-v1 from Wayland protocols.
// Supported by most Wayland compositors including Hyprland, Sway, and wlroots-based compositors.
package keyboard_shortcuts_inhibitor

import (
	"context"
	"fmt"
	"sync"

	"github.com/bnema/libwldevices-go/internal/client"
	"github.com/bnema/libwldevices-go/internal/protocols"
	"github.com/bnema/wlturbo/wl"
)

// Error constants for keyboard shortcuts inhibitor
const (
	// ERROR_ALREADY_INHIBITED is the protocol error the compositor raises when
	// shortcuts are already inhibited for the given surface and seat. The
	// compositor reports it through a wl_display error on the inhibitor object.
	ERROR_ALREADY_INHIBITED = protocols.ERROR_ALREADY_INHIBITED
)

// KeyboardShortcutsInhibitorManager represents the zwp_keyboard_shortcuts_inhibit_manager_v1 interface.
// A global interface to inhibit keyboard shortcuts for specific surfaces.
type KeyboardShortcutsInhibitorManager struct {
	client  *client.Client
	manager *protocols.KeyboardShortcutsInhibitManager

	// ownsClient is false when the connection was supplied by the caller, so
	// closing the manager must not close someone else's connection.
	ownsClient bool

	mu        sync.Mutex
	destroyed bool
}

// NewKeyboardShortcutsInhibitorManagerWithClient binds the inhibitor manager on
// an existing connection.
//
// inhibit_shortcuts references a surface and a seat, and a compositor rejects
// objects that do not belong to the connection that sent the request. The
// constructor without a client opens a private connection and can therefore
// only be used with objects it created itself; callers that own a surface must
// use this constructor with their own connection.
func NewKeyboardShortcutsInhibitorManagerWithClient(c *client.Client) (*KeyboardShortcutsInhibitorManager, error) {
	if c == nil {
		return nil, fmt.Errorf("nil client")
	}
	if !c.HasKeyboardShortcutsInhibit() {
		return nil, fmt.Errorf("zwp_keyboard_shortcuts_inhibit_manager_v1 not available - compositor may not support keyboard-shortcuts-inhibit protocol")
	}

	manager := protocols.NewKeyboardShortcutsInhibitManager(c.GetContext())
	if err := c.GetRegistry().Bind(
		c.GetKeyboardShortcutsInhibitManagerName(),
		protocols.KeyboardShortcutsInhibitManagerInterface,
		1,
		manager,
	); err != nil {
		return nil, fmt.Errorf("failed to bind keyboard shortcuts inhibitor manager: %w", err)
	}

	return &KeyboardShortcutsInhibitorManager{
		client:     c,
		manager:    manager,
		ownsClient: false,
	}, nil
}

// KeyboardShortcutsInhibitor represents the zwp_keyboard_shortcuts_inhibitor_v1 interface.
// A keyboard shortcuts inhibitor instructs the compositor to ignore its own keyboard shortcuts
// when the associated surface has keyboard focus.
type KeyboardShortcutsInhibitor struct {
	manager   *KeyboardShortcutsInhibitorManager
	inhibitor *protocols.KeyboardShortcutsInhibitor
	surface   *wl.Surface
	seat      *wl.Seat

	mu               sync.Mutex
	active           bool
	destroyed        bool
	activeHandlers   []func()
	inactiveHandlers []func()
}

// KeyboardShortcutsInhibitorError represents errors that can occur with keyboard shortcuts inhibitor operations.
type KeyboardShortcutsInhibitorError struct {
	Code    int
	Message string
}

func (e *KeyboardShortcutsInhibitorError) Error() string {
	return fmt.Sprintf("keyboard shortcuts inhibitor error %d: %s", e.Code, e.Message)
}

// NewKeyboardShortcutsInhibitorManager creates a new keyboard shortcuts inhibitor manager.
// It connects to the Wayland compositor and binds the zwp_keyboard_shortcuts_inhibit_manager_v1 global.
func NewKeyboardShortcutsInhibitorManager(ctx context.Context) (*KeyboardShortcutsInhibitorManager, error) {
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

	// Fail clearly when the compositor does not advertise the global.
	if !c.HasKeyboardShortcutsInhibit() {
		_ = c.Close()
		return nil, fmt.Errorf("zwp_keyboard_shortcuts_inhibit_manager_v1 not available - compositor may not support keyboard-shortcuts-inhibit protocol")
	}

	// Check context before binding
	select {
	case <-ctx.Done():
		_ = c.Close()
		return nil, fmt.Errorf("context cancelled before binding: %w", ctx.Err())
	default:
	}

	manager := protocols.NewKeyboardShortcutsInhibitManager(c.GetContext())
	if err := c.GetRegistry().Bind(
		c.GetKeyboardShortcutsInhibitManagerName(),
		protocols.KeyboardShortcutsInhibitManagerInterface,
		1,
		manager,
	); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("failed to bind keyboard shortcuts inhibitor manager: %w", err)
	}

	return &KeyboardShortcutsInhibitorManager{
		client:     c,
		manager:    manager,
		ownsClient: true,
	}, nil
}

// InhibitShortcuts creates a keyboard shortcuts inhibitor for a surface and seat.
// The inhibitor instructs the compositor to ignore its own keyboard shortcuts
// when the associated surface has keyboard focus.
func (m *KeyboardShortcutsInhibitorManager) InhibitShortcuts(surface *wl.Surface, seat *wl.Seat) (*KeyboardShortcutsInhibitor, error) {
	m.mu.Lock()
	destroyed := m.destroyed
	m.mu.Unlock()

	if m.manager == nil || destroyed {
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

	child, err := m.manager.InhibitShortcuts(surface, seat)
	if err != nil {
		return nil, fmt.Errorf("failed to inhibit shortcuts: %w", err)
	}

	inhibitor := &KeyboardShortcutsInhibitor{
		manager:   m,
		inhibitor: child,
		surface:   surface,
		seat:      seat,
	}
	child.OnActive(func() { inhibitor.setActive(true) })
	child.OnInactive(func() { inhibitor.setActive(false) })

	return inhibitor, nil
}

// Roundtrip flushes pending requests and processes incoming events. Protocol
// errors reported by the compositor (for example already_inhibited) surface
// here as a transport display error rather than being swallowed.
func (m *KeyboardShortcutsInhibitorManager) Roundtrip() error {
	if m.client == nil {
		return &KeyboardShortcutsInhibitorError{
			Code:    -1,
			Message: "manager not connected",
		}
	}
	return m.client.GetDisplay().Roundtrip()
}

// Close destroys the keyboard shortcuts inhibitor manager and closes the
// connection. It is safe to call once; a second call reports an error.
func (m *KeyboardShortcutsInhibitorManager) Close() error {
	m.mu.Lock()
	if m.destroyed {
		m.mu.Unlock()
		return &KeyboardShortcutsInhibitorError{
			Code:    -1,
			Message: "manager already destroyed",
		}
	}
	m.destroyed = true
	m.mu.Unlock()

	var firstErr error
	if m.manager != nil {
		if err := m.manager.Destroy(); err != nil {
			firstErr = err
		}
	}
	if m.ownsClient && m.client != nil {
		if err := m.client.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Destroy destroys the keyboard shortcuts inhibitor manager.
func (m *KeyboardShortcutsInhibitorManager) Destroy() error {
	return m.Close()
}

// Surface returns the surface this inhibitor is associated with.
func (i *KeyboardShortcutsInhibitor) Surface() *wl.Surface {
	return i.surface
}

// Seat returns the seat this inhibitor is associated with.
func (i *KeyboardShortcutsInhibitor) Seat() *wl.Seat {
	return i.seat
}

// Active reports whether the compositor currently has shortcuts inhibited on
// behalf of the surface. It becomes true when the compositor sends the
// protocol's active event and false when it sends inactive.
func (i *KeyboardShortcutsInhibitor) Active() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.active
}

// OnActive registers a handler called whenever the compositor activates the
// inhibitor. The handler must not block.
func (i *KeyboardShortcutsInhibitor) OnActive(handler func()) {
	if handler == nil {
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.activeHandlers = append(i.activeHandlers, handler)
}

// OnInactive registers a handler called whenever the compositor restores its
// own shortcuts. The handler must not block.
func (i *KeyboardShortcutsInhibitor) OnInactive(handler func()) {
	if handler == nil {
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.inactiveHandlers = append(i.inactiveHandlers, handler)
}

// setActive updates the observed state and notifies the relevant handlers.
func (i *KeyboardShortcutsInhibitor) setActive(active bool) {
	i.mu.Lock()
	i.active = active
	var handlers []func()
	if active {
		handlers = append(handlers, i.activeHandlers...)
	} else {
		handlers = append(handlers, i.inactiveHandlers...)
	}
	i.mu.Unlock()

	for _, handler := range handlers {
		handler()
	}
}

// Destroy destroys the keyboard shortcuts inhibitor. Keyboard shortcuts will be
// restored for the surface. A second Destroy reports an error.
func (i *KeyboardShortcutsInhibitor) Destroy() error {
	i.mu.Lock()
	if i.destroyed || i.inhibitor == nil {
		i.mu.Unlock()
		return &KeyboardShortcutsInhibitorError{
			Code:    -1,
			Message: "inhibitor not active",
		}
	}
	i.destroyed = true
	i.mu.Unlock()

	return i.inhibitor.Destroy()
}

// Convenience functions for common operations

// CreateTemporaryInhibitor creates an inhibitor that can be easily destroyed later.
// This is useful for temporary exclusive keyboard access.
func CreateTemporaryInhibitor(manager *KeyboardShortcutsInhibitorManager, surface *wl.Surface, seat *wl.Seat) (*KeyboardShortcutsInhibitor, error) {
	return manager.InhibitShortcuts(surface, seat)
}

// InhibitorStatus represents the status of a keyboard shortcuts inhibitor.
type InhibitorStatus struct {
	Active  bool
	Surface *wl.Surface
	Seat    *wl.Seat
}

// GetStatus returns the current status of the inhibitor.
func GetStatus(inhibitor *KeyboardShortcutsInhibitor) InhibitorStatus {
	if inhibitor == nil {
		return InhibitorStatus{Active: false}
	}
	return InhibitorStatus{
		Active:  inhibitor.Active(),
		Surface: inhibitor.Surface(),
		Seat:    inhibitor.Seat(),
	}
}
