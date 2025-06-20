package protocols

import (
	"github.com/bnema/wlturbo/wl"
)

// Protocol interface names
const (
	VirtualPointerManagerInterface = "zwlr_virtual_pointer_manager_v1"
	VirtualPointerInterface        = "zwlr_virtual_pointer_v1"
)

// VirtualPointerManager manages virtual pointer objects
type VirtualPointerManager struct {
	wl.BaseProxy
}

// NewVirtualPointerManager creates a new virtual pointer manager
func NewVirtualPointerManager(ctx *wl.Context) *VirtualPointerManager {
	manager := &VirtualPointerManager{}
	// Set the context properly
	manager.SetContext(ctx)
	ctx.Register(manager)
	return manager
}

// CreateVirtualPointer creates a new virtual pointer
func (m *VirtualPointerManager) CreateVirtualPointer(seat *wl.Seat) (*VirtualPointer, error) {
	// Allocate ID for the new pointer object
	pointerID := m.Context().AllocateID()
	
	pointer := &VirtualPointer{}
	pointer.SetContext(m.Context())
	pointer.SetID(pointerID)
	m.Context().Register(pointer)
	
	// Opcode 0: create_virtual_pointer
	const opcode = 0
	
	// The neurlang/wayland library expects the object itself for new_id parameters
	err := m.Context().SendRequest(m, opcode, seat, pointer)
	if err != nil {
		m.Context().Unregister(pointer)
		return nil, err
	}
	
	return pointer, nil
}

// CreateVirtualPointerWithOutput creates a new virtual pointer with output (v2)
func (m *VirtualPointerManager) CreateVirtualPointerWithOutput(seat *wl.Seat, output *wl.Output) (*VirtualPointer, error) {
	// Allocate ID for the new pointer object
	pointerID := m.Context().AllocateID()
	
	pointer := &VirtualPointer{}
	pointer.SetContext(m.Context())
	pointer.SetID(pointerID)
	m.Context().Register(pointer)
	
	// Opcode 2: create_virtual_pointer_with_output (since version 2)
	const opcode = 2
	
	err := m.Context().SendRequest(m, opcode, seat, output, pointer)
	if err != nil {
		m.Context().Unregister(pointer)
		return nil, err
	}
	
	return pointer, nil
}

// Destroy destroys the virtual pointer manager
func (m *VirtualPointerManager) Destroy() error {
	// Opcode 1: destroy
	const opcode = 1
	
	err := m.Context().SendRequest(m, opcode)
	m.Context().Unregister(m)
	return err
}

// Dispatch handles incoming events (virtual pointer manager has no events)
func (m *VirtualPointerManager) Dispatch(_ *wl.Event) {
	// Virtual pointer manager has no events
}

// VirtualPointer represents a virtual pointer device
type VirtualPointer struct {
	wl.BaseProxy
}

// NewVirtualPointer creates a new virtual pointer
func NewVirtualPointer(ctx *wl.Context) *VirtualPointer {
	pointer := &VirtualPointer{}
	// Set the context properly
	pointer.SetContext(ctx)
	ctx.Register(pointer)
	return pointer
}

// Motion sends a relative pointer motion event
func (p *VirtualPointer) Motion(time uint32, dx, dy wl.Fixed) error {
	// Opcode 0: motion
	const opcode = 0
	return p.Context().SendRequest(p, opcode, time, dx, dy)
}

// MotionAbsolute sends an absolute pointer motion event
func (p *VirtualPointer) MotionAbsolute(time, x, y, xExtent, yExtent uint32) error {
	// Opcode 1: motion_absolute
	const opcode = 1
	return p.Context().SendRequest(p, opcode, time, x, y, xExtent, yExtent)
}

// Button sends a button press/release event
func (p *VirtualPointer) Button(time, button, state uint32) error {
	// Opcode 2: button
	const opcode = 2
	return p.Context().SendRequest(p, opcode, time, button, state)
}

// Axis sends a scroll event
func (p *VirtualPointer) Axis(time, axis uint32, value wl.Fixed) error {
	// Opcode 3: axis
	const opcode = 3
	return p.Context().SendRequest(p, opcode, time, axis, value)
}

// Frame indicates the end of a pointer event sequence
func (p *VirtualPointer) Frame() error {
	// Opcode 4: frame
	const opcode = 4
	return p.Context().SendRequest(p, opcode)
}

// AxisSource sets the axis source
func (p *VirtualPointer) AxisSource(axisSource uint32) error {
	// Opcode 5: axis_source
	const opcode = 5
	return p.Context().SendRequest(p, opcode, axisSource)
}

// AxisStop sends an axis stop event
func (p *VirtualPointer) AxisStop(time, axis uint32) error {
	// Opcode 6: axis_stop
	const opcode = 6
	return p.Context().SendRequest(p, opcode, time, axis)
}

// AxisDiscrete sends a discrete axis event
func (p *VirtualPointer) AxisDiscrete(time, axis uint32, value wl.Fixed, discrete int32) error {
	// Opcode 7: axis_discrete
	const opcode = 7
	return p.Context().SendRequest(p, opcode, time, axis, value, discrete)
}

// Destroy destroys the virtual pointer
func (p *VirtualPointer) Destroy() error {
	// Opcode 8: destroy
	const opcode = 8
	err := p.Context().SendRequest(p, opcode)
	p.Context().Unregister(p)
	return err
}

// Dispatch handles incoming events (virtual pointer has no events)
func (p *VirtualPointer) Dispatch(_ *wl.Event) {
	// Virtual pointer has no events
}