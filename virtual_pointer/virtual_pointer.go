// Package virtual_pointer provides Go bindings for the wlr-virtual-pointer-unstable-v1 Wayland protocol.
//
// This protocol allows clients to emulate a physical pointer device. The requests are mostly
// mirror opposites of those specified in wl_pointer.
//
// Protocol specification: wlr-virtual-pointer-unstable-v1
package virtual_pointer

import (
	"context"
	"fmt"
	"time"
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

// Axis constants for scroll events
const (
	AXIS_VERTICAL_SCROLL   = 0
	AXIS_HORIZONTAL_SCROLL = 1
)

// Axis source constants
const (
	AXIS_SOURCE_WHEEL      = 0
	AXIS_SOURCE_FINGER     = 1
	AXIS_SOURCE_CONTINUOUS = 2
	AXIS_SOURCE_WHEEL_TILT = 3
)

// VirtualPointerManager represents the zwlr_virtual_pointer_manager_v1 interface.
// This object allows clients to create individual virtual pointer objects.
type VirtualPointerManager interface {
	// CreateVirtualPointer creates a new virtual pointer.
	// The optional seat is a suggestion to the compositor.
	CreateVirtualPointer(seat interface{}) (VirtualPointer, error)

	// CreateVirtualPointerWithOutput creates a new virtual pointer with output mapping.
	// The seat and output arguments are optional. If the seat argument is set, the
	// compositor should assign the input device to the requested seat. If the output
	// argument is set, the compositor should map the input device to the requested output.
	CreateVirtualPointerWithOutput(seat interface{}, output interface{}) (VirtualPointer, error)

	// Destroy destroys the virtual pointer manager.
	Destroy() error
}

// VirtualPointer represents the zwlr_virtual_pointer_v1 interface.
// This protocol allows clients to emulate a physical pointer device.
type VirtualPointer interface {
	// Motion sends a pointer relative motion event.
	// The pointer has moved by a relative amount to the previous request.
	// Values are in the global compositor space.
	Motion(time time.Time, dx, dy float64) error

	// MotionAbsolute sends a pointer absolute motion event.
	// The pointer has moved in an absolute coordinate frame.
	// Value of x can range from 0 to xExtent, value of y can range from 0 to yExtent.
	MotionAbsolute(time time.Time, x, y, xExtent, yExtent uint32) error

	// Button sends a button press or release event.
	Button(time time.Time, button uint32, state uint32) error

	// ButtonPress is a convenience method for pressing a button.
	ButtonPress(button uint32) error

	// ButtonRelease is a convenience method for releasing a button.
	ButtonRelease(button uint32) error

	// Axis sends a scroll and other axis event.
	Axis(time time.Time, axis uint32, value float64) error

	// AxisSource sends axis source information for scroll and other axis events.
	AxisSource(axisSource uint32) error

	// AxisStop sends a stop notification for scroll and other axes.
	AxisStop(time time.Time, axis uint32) error

	// AxisDiscrete sends discrete step information for scroll and other axes.
	// This event allows the client to extend data normally sent using the axis
	// event with discrete value.
	AxisDiscrete(time time.Time, axis uint32, value float64, discrete int32) error

	// Frame indicates the set of events that logically belong together.
	// This should be called after a sequence of related pointer events.
	Frame() error

	// Destroy destroys the virtual pointer object.
	Destroy() error
}

// VirtualPointerError represents errors that can occur with virtual pointer operations.
type VirtualPointerError struct {
	Code    int
	Message string
}

func (e *VirtualPointerError) Error() string {
	return fmt.Sprintf("virtual pointer error %d: %s", e.Code, e.Message)
}

// Error codes for virtual pointer
const (
	ERROR_INVALID_AXIS        = 0
	ERROR_INVALID_AXIS_SOURCE = 1
)

// Implementation structs (these would be implemented by the actual Wayland client library)

// virtualPointerManager is the concrete implementation of VirtualPointerManager.
type virtualPointerManager struct {
	// This would contain the actual Wayland client connection and manager object
	// For now, we provide a stub implementation
	connected bool
}

// NewVirtualPointerManager creates a new virtual pointer manager.
// In a real implementation, this would connect to the Wayland compositor
// and bind to the zwlr_virtual_pointer_manager_v1 global.
func NewVirtualPointerManager(ctx context.Context) (VirtualPointerManager, error) {
	// This is a stub implementation - in reality, this would:
	// 1. Connect to the Wayland display
	// 2. Get the registry
	// 3. Bind to zwlr_virtual_pointer_manager_v1
	// 4. Return the manager object
	
	return &virtualPointerManager{
		connected: true,
	}, nil
}

func (m *virtualPointerManager) CreateVirtualPointer(seat interface{}) (VirtualPointer, error) {
	if !m.connected {
		return nil, &VirtualPointerError{
			Code:    -1,
			Message: "manager not connected",
		}
	}

	// This would actually create the virtual pointer object via Wayland protocol
	return &virtualPointer{
		manager: m,
		active:  true,
	}, nil
}

func (m *virtualPointerManager) CreateVirtualPointerWithOutput(seat interface{}, output interface{}) (VirtualPointer, error) {
	if !m.connected {
		return nil, &VirtualPointerError{
			Code:    -1,
			Message: "manager not connected",
		}
	}

	// This would actually create the virtual pointer object with output mapping
	return &virtualPointer{
		manager: m,
		active:  true,
	}, nil
}

func (m *virtualPointerManager) Destroy() error {
	if !m.connected {
		return &VirtualPointerError{
			Code:    -1,
			Message: "manager not connected",
		}
	}

	m.connected = false
	return nil
}

// virtualPointer is the concrete implementation of VirtualPointer.
type virtualPointer struct {
	manager *virtualPointerManager
	active  bool
}

func (p *virtualPointer) Motion(time time.Time, dx, dy float64) error {
	if !p.active {
		return &VirtualPointerError{
			Code:    -1,
			Message: "pointer not active",
		}
	}

	// This would send the actual motion request to the Wayland compositor
	// For now, we just validate the parameters
	return nil
}

func (p *virtualPointer) MotionAbsolute(time time.Time, x, y, xExtent, yExtent uint32) error {
	if !p.active {
		return &VirtualPointerError{
			Code:    -1,
			Message: "pointer not active",
		}
	}

	if x > xExtent || y > yExtent {
		return &VirtualPointerError{
			Code:    -1,
			Message: "coordinates out of bounds",
		}
	}

	// This would send the actual motion_absolute request to the Wayland compositor
	return nil
}

func (p *virtualPointer) Button(time time.Time, button uint32, state uint32) error {
	if !p.active {
		return &VirtualPointerError{
			Code:    -1,
			Message: "pointer not active",
		}
	}

	if state != BUTTON_STATE_PRESSED && state != BUTTON_STATE_RELEASED {
		return &VirtualPointerError{
			Code:    -1,
			Message: "invalid button state",
		}
	}

	// This would send the actual button request to the Wayland compositor
	return nil
}

func (p *virtualPointer) ButtonPress(button uint32) error {
	return p.Button(time.Now(), button, BUTTON_STATE_PRESSED)
}

func (p *virtualPointer) ButtonRelease(button uint32) error {
	return p.Button(time.Now(), button, BUTTON_STATE_RELEASED)
}

func (p *virtualPointer) Axis(time time.Time, axis uint32, value float64) error {
	if !p.active {
		return &VirtualPointerError{
			Code:    -1,
			Message: "pointer not active",
		}
	}

	if axis != AXIS_VERTICAL_SCROLL && axis != AXIS_HORIZONTAL_SCROLL {
		return &VirtualPointerError{
			Code:    ERROR_INVALID_AXIS,
			Message: "invalid axis",
		}
	}

	// This would send the actual axis request to the Wayland compositor
	return nil
}

func (p *virtualPointer) AxisSource(axisSource uint32) error {
	if !p.active {
		return &VirtualPointerError{
			Code:    -1,
			Message: "pointer not active",
		}
	}

	validSources := []uint32{AXIS_SOURCE_WHEEL, AXIS_SOURCE_FINGER, AXIS_SOURCE_CONTINUOUS, AXIS_SOURCE_WHEEL_TILT}
	valid := false
	for _, source := range validSources {
		if axisSource == source {
			valid = true
			break
		}
	}

	if !valid {
		return &VirtualPointerError{
			Code:    ERROR_INVALID_AXIS_SOURCE,
			Message: "invalid axis source",
		}
	}

	// This would send the actual axis_source request to the Wayland compositor
	return nil
}

func (p *virtualPointer) AxisStop(time time.Time, axis uint32) error {
	if !p.active {
		return &VirtualPointerError{
			Code:    -1,
			Message: "pointer not active",
		}
	}

	if axis != AXIS_VERTICAL_SCROLL && axis != AXIS_HORIZONTAL_SCROLL {
		return &VirtualPointerError{
			Code:    ERROR_INVALID_AXIS,
			Message: "invalid axis",
		}
	}

	// This would send the actual axis_stop request to the Wayland compositor
	return nil
}

func (p *virtualPointer) AxisDiscrete(time time.Time, axis uint32, value float64, discrete int32) error {
	if !p.active {
		return &VirtualPointerError{
			Code:    -1,
			Message: "pointer not active",
		}
	}

	if axis != AXIS_VERTICAL_SCROLL && axis != AXIS_HORIZONTAL_SCROLL {
		return &VirtualPointerError{
			Code:    ERROR_INVALID_AXIS,
			Message: "invalid axis",
		}
	}

	// This would send the actual axis_discrete request to the Wayland compositor
	return nil
}

func (p *virtualPointer) Frame() error {
	if !p.active {
		return &VirtualPointerError{
			Code:    -1,
			Message: "pointer not active",
		}
	}

	// This would send the actual frame request to the Wayland compositor
	return nil
}

func (p *virtualPointer) Destroy() error {
	if !p.active {
		return &VirtualPointerError{
			Code:    -1,
			Message: "pointer not active",
		}
	}

	p.active = false
	return nil
}

// Convenience functions for common operations

// Click performs a complete click operation (press + release + frame).
func Click(pointer VirtualPointer, button uint32) error {
	if err := pointer.ButtonPress(button); err != nil {
		return err
	}
	if err := pointer.ButtonRelease(button); err != nil {
		return err
	}
	return pointer.Frame()
}

// Scroll performs a scroll operation with the given axis and value.
func Scroll(pointer VirtualPointer, axis uint32, value float64) error {
	now := time.Now()
	if err := pointer.AxisSource(AXIS_SOURCE_WHEEL); err != nil {
		return err
	}
	if err := pointer.Axis(now, axis, value); err != nil {
		return err
	}
	return pointer.Frame()
}

// ScrollVertical performs a vertical scroll operation.
func ScrollVertical(pointer VirtualPointer, value float64) error {
	return Scroll(pointer, AXIS_VERTICAL_SCROLL, value)
}

// ScrollHorizontal performs a horizontal scroll operation.
func ScrollHorizontal(pointer VirtualPointer, value float64) error {
	return Scroll(pointer, AXIS_HORIZONTAL_SCROLL, value)
}

// MoveRelative performs a relative mouse movement followed by a frame.
func MoveRelative(pointer VirtualPointer, dx, dy float64) error {
	if err := pointer.Motion(time.Now(), dx, dy); err != nil {
		return err
	}
	return pointer.Frame()
}

// MoveAbsolute performs an absolute mouse movement followed by a frame.
func MoveAbsolute(pointer VirtualPointer, x, y, xExtent, yExtent uint32) error {
	if err := pointer.MotionAbsolute(time.Now(), x, y, xExtent, yExtent); err != nil {
		return err
	}
	return pointer.Frame()
}