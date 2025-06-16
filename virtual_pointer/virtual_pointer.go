// Package virtual_pointer provides Go bindings for the wlr-virtual-pointer-unstable-v1 Wayland protocol.
//
// This protocol allows clients to emulate a physical pointer device, enabling mouse input injection
// into Wayland compositors without requiring root privileges. This is a complete, working 
// implementation built on neurlang/wayland.
//
// # Basic Usage
//
//	// Create manager and pointer
//	ctx := context.Background()
//	manager, err := NewVirtualPointerManager(ctx)
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
//	pointer.MoveRelative(100.0, 50.0)
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
//	// Manual control with timestamps
//	pointer.Motion(time.Now(), 10.0, 5.0)
//	pointer.Frame()
//
// # Protocol Specification
//
// Based on wlr-virtual-pointer-unstable-v1 from wlroots project.
// Supported by Hyprland, Sway, and other wlroots-based compositors.
package virtual_pointer

import (
	"context"
	"fmt"
	"time"

	"github.com/bnema/wayland-virtual-input-go/internal/client"
	"github.com/bnema/wayland-virtual-input-go/internal/protocols"
	"github.com/neurlang/wayland/wl"
)

// Button constants for mouse buttons
const (
	BTN_LEFT   = 0x110
	BTN_RIGHT  = 0x111
	BTN_MIDDLE = 0x112
	BTN_SIDE   = 0x113
	BTN_EXTRA  = 0x114
)

// Button state constants
const (
	BUTTON_STATE_RELEASED = 0
	BUTTON_STATE_PRESSED  = 1
)

// Axis constants (from wl_pointer)
const (
	AXIS_VERTICAL_SCROLL   = 0
	AXIS_HORIZONTAL_SCROLL = 1
)

// Axis source constants (from wl_pointer)
const (
	AXIS_SOURCE_WHEEL      = 0
	AXIS_SOURCE_FINGER     = 1
	AXIS_SOURCE_CONTINUOUS = 2
	AXIS_SOURCE_WHEEL_TILT = 3
)

// ButtonState represents the state of a button
type ButtonState uint32

const (
	ButtonStateReleased ButtonState = 0
	ButtonStatePressed  ButtonState = 1
)

// Axis represents a scroll axis
type Axis uint32

const (
	AxisVertical   Axis = 0
	AxisHorizontal Axis = 1
)

// AxisSource represents the source of axis events
type AxisSource uint32

const (
	AxisSourceWheel      AxisSource = 0
	AxisSourceFinger     AxisSource = 1
	AxisSourceContinuous AxisSource = 2
	AxisSourceWheelTilt  AxisSource = 3
)

// VirtualPointerManager manages virtual pointer devices
type VirtualPointerManager struct {
	client  *client.Client
	manager *protocols.VirtualPointerManager
}

// VirtualPointer represents a virtual pointer device
type VirtualPointer struct {
	pointer *protocols.VirtualPointer
}

// floatToFixed converts a float64 to wayland fixed point
func floatToFixed(val float64) wl.Fixed {
	return wl.Fixed(val * 256.0)
}

// NewVirtualPointerManager creates a new virtual pointer manager
func NewVirtualPointerManager(ctx context.Context) (*VirtualPointerManager, error) {
	// Create Wayland client
	c, err := client.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Wayland client: %w", err)
	}
	
	// Check if virtual pointer protocol is available
	if !c.HasVirtualPointer() {
		c.Close()
		return nil, fmt.Errorf("zwlr_virtual_pointer_manager_v1 not available")
	}
	
	// Create the manager proxy
	manager := protocols.NewVirtualPointerManager(c.GetContext())
	
	// Bind to the global
	name := c.GetPointerManagerName()
	err = c.GetRegistry().Bind(name, protocols.VirtualPointerManagerInterface, 1, manager)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to bind virtual pointer manager: %w", err)
	}
	
	// Sync to ensure binding is complete
	sync, err := c.GetDisplay().Sync()
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to sync: %w", err)
	}
	
	err = c.GetContext().RunTill(sync)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to wait for sync: %w", err)
	}
	
	return &VirtualPointerManager{
		client:  c,
		manager: manager,
	}, nil
}

// CreatePointer creates a new virtual pointer device
func (m *VirtualPointerManager) CreatePointer() (*VirtualPointer, error) {
	// Create virtual pointer using the current seat
	pointer, err := m.manager.CreateVirtualPointer(m.client.GetSeat())
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual pointer: %w", err)
	}
	
	return &VirtualPointer{
		pointer: pointer,
	}, nil
}

// Motion sends a relative motion event
func (p *VirtualPointer) Motion(timestamp time.Time, dx, dy float64) error {
	timeMs := uint32(timestamp.UnixNano() / 1000000)
	return p.pointer.Motion(timeMs, floatToFixed(dx), floatToFixed(dy))
}

// MotionAbsolute sends an absolute motion event
func (p *VirtualPointer) MotionAbsolute(timestamp time.Time, x, y uint32, xExtent, yExtent uint32) error {
	timeMs := uint32(timestamp.UnixNano() / 1000000)
	return p.pointer.MotionAbsolute(timeMs, x, y, xExtent, yExtent)
}

// Button sends a button press/release event
func (p *VirtualPointer) Button(timestamp time.Time, button uint32, state ButtonState) error {
	timeMs := uint32(timestamp.UnixNano() / 1000000)
	return p.pointer.Button(timeMs, button, uint32(state))
}

// Axis sends a scroll event
func (p *VirtualPointer) Axis(timestamp time.Time, axis Axis, value float64) error {
	timeMs := uint32(timestamp.UnixNano() / 1000000)
	return p.pointer.Axis(timeMs, uint32(axis), floatToFixed(value))
}

// Frame indicates the end of a pointer event sequence
func (p *VirtualPointer) Frame() error {
	return p.pointer.Frame()
}

// AxisSource sets the axis source for subsequent axis events
func (p *VirtualPointer) AxisSource(source AxisSource) error {
	return p.pointer.AxisSource(uint32(source))
}

// AxisStop sends an axis stop event
func (p *VirtualPointer) AxisStop(timestamp time.Time, axis Axis) error {
	timeMs := uint32(timestamp.UnixNano() / 1000000)
	return p.pointer.AxisStop(timeMs, uint32(axis))
}

// AxisDiscrete sends a discrete axis event
func (p *VirtualPointer) AxisDiscrete(timestamp time.Time, axis Axis, value float64, discrete int32) error {
	timeMs := uint32(timestamp.UnixNano() / 1000000)
	return p.pointer.AxisDiscrete(timeMs, uint32(axis), floatToFixed(value), discrete)
}

// Close releases the virtual pointer device
func (p *VirtualPointer) Close() error {
	return p.pointer.Destroy()
}

// Close releases the virtual pointer manager
func (m *VirtualPointerManager) Close() error {
	if m.manager != nil {
		m.manager.Destroy()
	}
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}

// Convenience methods for common operations

// MoveRelative moves the pointer by the specified amount
func (p *VirtualPointer) MoveRelative(dx, dy float64) error {
	if err := p.Motion(time.Now(), dx, dy); err != nil {
		return err
	}
	return p.Frame()
}

// LeftClick performs a left mouse button click
func (p *VirtualPointer) LeftClick() error {
	now := time.Now()
	if err := p.Button(now, BTN_LEFT, ButtonStatePressed); err != nil {
		return err
	}
	if err := p.Button(now, BTN_LEFT, ButtonStateReleased); err != nil {
		return err
	}
	return p.Frame()
}

// RightClick performs a right mouse button click
func (p *VirtualPointer) RightClick() error {
	now := time.Now()
	if err := p.Button(now, BTN_RIGHT, ButtonStatePressed); err != nil {
		return err
	}
	if err := p.Button(now, BTN_RIGHT, ButtonStateReleased); err != nil {
		return err
	}
	return p.Frame()
}

// MiddleClick performs a middle mouse button click
func (p *VirtualPointer) MiddleClick() error {
	now := time.Now()
	if err := p.Button(now, BTN_MIDDLE, ButtonStatePressed); err != nil {
		return err
	}
	if err := p.Button(now, BTN_MIDDLE, ButtonStateReleased); err != nil {
		return err
	}
	return p.Frame()
}

// ScrollVertical scrolls vertically by the specified amount
func (p *VirtualPointer) ScrollVertical(amount float64) error {
	if err := p.Axis(time.Now(), AxisVertical, amount); err != nil {
		return err
	}
	return p.Frame()
}

// ScrollHorizontal scrolls horizontally by the specified amount
func (p *VirtualPointer) ScrollHorizontal(amount float64) error {
	if err := p.Axis(time.Now(), AxisHorizontal, amount); err != nil {
		return err
	}
	return p.Frame()
}