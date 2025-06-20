package protocols

import (
	"github.com/bnema/wlturbo/wl"
)

// Protocol interface names
const (
	OutputManagerInterface           = "zwlr_output_manager_v1"
	OutputHeadInterface              = "zwlr_output_head_v1"
	OutputModeInterface              = "zwlr_output_mode_v1"
	OutputConfigurationInterface     = "zwlr_output_configuration_v1"
	OutputConfigurationHeadInterface = "zwlr_output_configuration_head_v1"
)

// OutputManager manages output configuration
type OutputManager struct {
	wl.BaseProxy
	headHandler     func(*OutputHead)
	doneHandler     func(uint32)
	finishedHandler func()
}

// NewOutputManager creates a new output manager
func NewOutputManager(ctx *wl.Context) *OutputManager {
	manager := &OutputManager{}
	manager.SetContext(ctx)
	return manager
}

// SetHeadHandler sets the handler for new head events
func (m *OutputManager) SetHeadHandler(handler func(*OutputHead)) {
	m.headHandler = handler
}

// SetDoneHandler sets the handler for done events
func (m *OutputManager) SetDoneHandler(handler func(uint32)) {
	m.doneHandler = handler
}

// SetFinishedHandler sets the handler for finished events
func (m *OutputManager) SetFinishedHandler(handler func()) {
	m.finishedHandler = handler
}

// CreateConfiguration creates a new output configuration
func (m *OutputManager) CreateConfiguration(serial uint32) (*OutputConfiguration, error) {
	config := NewOutputConfiguration(m.Context())

	// Opcode 0: create_configuration
	const opcode = 0

	err := m.Context().SendRequest(m, opcode, config, serial)
	if err != nil {
		m.Context().Unregister(config)
		return nil, err
	}

	return config, nil
}

// Stop stops receiving events
func (m *OutputManager) Stop() error {
	// Opcode 1: stop
	const opcode = 1
	return m.Context().SendRequest(m, opcode)
}

// Destroy destroys the output manager
func (m *OutputManager) Destroy() error {
	m.Context().Unregister(m)
	return nil
}

// Dispatch handles incoming events
func (m *OutputManager) Dispatch(event *wl.Event) {
	// fmt.Printf("[DEBUG] OutputManager.Dispatch called with opcode %d\n", event.Opcode)
	switch event.Opcode {
	case 0: // head
		// For new_id in events, the ID is sent by the server
		// Let me check if the bytes might be in a different order
		// if len(event.Data()) >= 4 {
		// 	fmt.Printf("[DEBUG] Head event raw bytes: [")
		// 	for i := 0; i < len(event.Data()); i++ {
		// 		fmt.Printf("%02x ", event.Data()[i])
		// 	}
		// 	fmt.Printf("]\n")
		//
		// 	// What if we read it as big-endian?
		// 	bigEndianID := binary.BigEndian.Uint32(event.Data()[0:4])
		// 	littleEndianID := binary.LittleEndian.Uint32(event.Data()[0:4])
		// 	fmt.Printf("[DEBUG] Big-endian read: %d (0x%x), Little-endian read: %d (0x%x)\n",
		// 		bigEndianID, bigEndianID, littleEndianID, littleEndianID)
		// }
		headID := event.Uint32()
		// fmt.Printf("[DEBUG] Creating new head with ID %d (0x%x)\n", headID, headID)

		// Accept the ID as-is - the server might be using a special range
		// if headID > 1000000 {
		// 	fmt.Printf("[DEBUG] Note: Head ID %d (0x%x) is in a special range\n", headID, headID)
		// }
		head := NewOutputHead(m.Context())
		head.SetID(headID)
		head.SetContext(m.Context())
		m.Context().Register(head)
		if m.headHandler != nil {
			m.headHandler(head)
		}
	case 1: // done
		if m.doneHandler != nil {
			serial := event.Uint32()
			m.doneHandler(serial)
		}
	case 2: // finished
		if m.finishedHandler != nil {
			m.finishedHandler()
		}
		m.Context().Unregister(m)
	}
}

// OutputHead represents an output device
type OutputHead struct {
	wl.BaseProxy
	nameHandler         func(string)
	descriptionHandler  func(string)
	physicalSizeHandler func(int32, int32)
	modeHandler         func(*OutputMode)
	enabledHandler      func(int32)
	currentModeHandler  func(*OutputMode)
	positionHandler     func(int32, int32)
	transformHandler    func(int32)
	scaleHandler        func(wl.Fixed)
	makeHandler         func(string)
	modelHandler        func(string)
	serialNumberHandler func(string)
	adaptiveSyncHandler func(uint32)
	finishedHandler     func()
}

// NewOutputHead creates a new output head
func NewOutputHead(ctx *wl.Context) *OutputHead {
	head := &OutputHead{}
	head.SetContext(ctx)
	return head
}

// SetNameHandler sets the handler for name events
func (h *OutputHead) SetNameHandler(handler func(string)) {
	h.nameHandler = handler
}

// SetDescriptionHandler sets the handler for description events
func (h *OutputHead) SetDescriptionHandler(handler func(string)) {
	h.descriptionHandler = handler
}

// SetPhysicalSizeHandler sets the handler for physical size events
func (h *OutputHead) SetPhysicalSizeHandler(handler func(int32, int32)) {
	h.physicalSizeHandler = handler
}

// SetModeHandler sets the handler for mode events
func (h *OutputHead) SetModeHandler(handler func(*OutputMode)) {
	h.modeHandler = handler
}

// SetEnabledHandler sets the handler for enabled events
func (h *OutputHead) SetEnabledHandler(handler func(int32)) {
	h.enabledHandler = handler
}

// SetCurrentModeHandler sets the handler for current mode events
func (h *OutputHead) SetCurrentModeHandler(handler func(*OutputMode)) {
	h.currentModeHandler = handler
}

// SetPositionHandler sets the handler for position events
func (h *OutputHead) SetPositionHandler(handler func(int32, int32)) {
	h.positionHandler = handler
}

// SetTransformHandler sets the handler for transform events
func (h *OutputHead) SetTransformHandler(handler func(int32)) {
	h.transformHandler = handler
}

// SetScaleHandler sets the handler for scale events
func (h *OutputHead) SetScaleHandler(handler func(wl.Fixed)) {
	h.scaleHandler = handler
}

// SetMakeHandler sets the handler for make events
func (h *OutputHead) SetMakeHandler(handler func(string)) {
	h.makeHandler = handler
}

// SetModelHandler sets the handler for model events
func (h *OutputHead) SetModelHandler(handler func(string)) {
	h.modelHandler = handler
}

// SetSerialNumberHandler sets the handler for serial number events
func (h *OutputHead) SetSerialNumberHandler(handler func(string)) {
	h.serialNumberHandler = handler
}

// SetAdaptiveSyncHandler sets the handler for adaptive sync events
func (h *OutputHead) SetAdaptiveSyncHandler(handler func(uint32)) {
	h.adaptiveSyncHandler = handler
}

// SetFinishedHandler sets the handler for finished events
func (h *OutputHead) SetFinishedHandler(handler func()) {
	h.finishedHandler = handler
}

// Release releases the output head
func (h *OutputHead) Release() error {
	// Opcode 0: release (since version 3)
	const opcode = 0
	err := h.Context().SendRequest(h, opcode)
	h.Context().Unregister(h)
	return err
}

// Destroy destroys the output head
func (h *OutputHead) Destroy() error {
	h.Context().Unregister(h)
	return nil
}

// Dispatch handles incoming events
func (h *OutputHead) Dispatch(event *wl.Event) {
	switch event.Opcode {
	case 0: // name
		if h.nameHandler != nil {
			name := event.String()
			h.nameHandler(name)
		}
	case 1: // description
		if h.descriptionHandler != nil {
			description := event.String()
			h.descriptionHandler(description)
		}
	case 2: // physical_size
		if h.physicalSizeHandler != nil {
			width := event.Int32()
			height := event.Int32()
			h.physicalSizeHandler(width, height)
		}
	case 3: // mode
		proxy := event.NewID()
		mode := NewOutputMode(h.Context())
		mode.SetID(proxy.ID())
		mode.SetContext(h.Context())
		h.Context().Register(mode)
		if h.modeHandler != nil {
			h.modeHandler(mode)
		}
	case 4: // enabled
		if h.enabledHandler != nil {
			enabled := event.Int32()
			h.enabledHandler(enabled)
		}
	case 5: // current_mode
		if h.currentModeHandler != nil {
			// Get the proxy ID and look it up in the context
			_ = event.Uint32() // proxyId
			// TODO: Need a way to lookup proxies by ID from context
			// For now, we'll skip this handler
		}
	case 6: // position
		if h.positionHandler != nil {
			x := event.Int32()
			y := event.Int32()
			h.positionHandler(x, y)
		}
	case 7: // transform
		if h.transformHandler != nil {
			transform := event.Int32()
			h.transformHandler(transform)
		}
	case 8: // scale
		if h.scaleHandler != nil {
			rawScale := event.Uint32()
		// Safe conversion: Wayland scale values are typically small positive numbers
		if rawScale > 0x7FFFFFFF {
			return // Invalid scale value, skip
		}
		scale := wl.Fixed(rawScale)
			h.scaleHandler(scale)
		}
	case 9: // finished
		if h.finishedHandler != nil {
			h.finishedHandler()
		}
		h.Context().Unregister(h)
	case 10: // make (since version 2)
		if h.makeHandler != nil {
			makeStr := event.String()
			h.makeHandler(makeStr)
		}
	case 11: // model (since version 2)
		if h.modelHandler != nil {
			model := event.String()
			h.modelHandler(model)
		}
	case 12: // serial_number (since version 2)
		if h.serialNumberHandler != nil {
			serial := event.String()
			h.serialNumberHandler(serial)
		}
	case 13: // adaptive_sync (since version 4)
		if h.adaptiveSyncHandler != nil {
			state := event.Uint32()
			h.adaptiveSyncHandler(state)
		}
	}
}

// OutputMode represents an output mode
type OutputMode struct {
	wl.BaseProxy
	sizeHandler      func(int32, int32)
	refreshHandler   func(int32)
	preferredHandler func()
	finishedHandler  func()
}

// NewOutputMode creates a new output mode
func NewOutputMode(ctx *wl.Context) *OutputMode {
	mode := &OutputMode{}
	mode.SetContext(ctx)
	return mode
}

// SetSizeHandler sets the handler for size events
func (m *OutputMode) SetSizeHandler(handler func(int32, int32)) {
	m.sizeHandler = handler
}

// SetRefreshHandler sets the handler for refresh events
func (m *OutputMode) SetRefreshHandler(handler func(int32)) {
	m.refreshHandler = handler
}

// SetPreferredHandler sets the handler for preferred events
func (m *OutputMode) SetPreferredHandler(handler func()) {
	m.preferredHandler = handler
}

// SetFinishedHandler sets the handler for finished events
func (m *OutputMode) SetFinishedHandler(handler func()) {
	m.finishedHandler = handler
}

// Release releases the output mode
func (m *OutputMode) Release() error {
	// Opcode 0: release (since version 3)
	const opcode = 0
	err := m.Context().SendRequest(m, opcode)
	m.Context().Unregister(m)
	return err
}

// Destroy destroys the output mode
func (m *OutputMode) Destroy() error {
	m.Context().Unregister(m)
	return nil
}

// Dispatch handles incoming events
func (m *OutputMode) Dispatch(event *wl.Event) {
	switch event.Opcode {
	case 0: // size
		if m.sizeHandler != nil {
			width := event.Int32()
			height := event.Int32()
			m.sizeHandler(width, height)
		}
	case 1: // refresh
		if m.refreshHandler != nil {
			refresh := event.Int32()
			m.refreshHandler(refresh)
		}
	case 2: // preferred
		if m.preferredHandler != nil {
			m.preferredHandler()
		}
	case 3: // finished
		if m.finishedHandler != nil {
			m.finishedHandler()
		}
		m.Context().Unregister(m)
	}
}

// OutputConfiguration represents an output configuration
type OutputConfiguration struct {
	wl.BaseProxy
	succeededHandler func()
	failedHandler    func()
	cancelledHandler func()
}

// NewOutputConfiguration creates a new output configuration
func NewOutputConfiguration(ctx *wl.Context) *OutputConfiguration {
	config := &OutputConfiguration{}
	config.SetContext(ctx)
	return config
}

// SetSucceededHandler sets the handler for succeeded events
func (c *OutputConfiguration) SetSucceededHandler(handler func()) {
	c.succeededHandler = handler
}

// SetFailedHandler sets the handler for failed events
func (c *OutputConfiguration) SetFailedHandler(handler func()) {
	c.failedHandler = handler
}

// SetCancelledHandler sets the handler for cancelled events
func (c *OutputConfiguration) SetCancelledHandler(handler func()) {
	c.cancelledHandler = handler
}

// EnableHead enables a head
func (c *OutputConfiguration) EnableHead(head *OutputHead) (*OutputConfigurationHead, error) {
	configHead := NewOutputConfigurationHead(c.Context())

	// Opcode 0: enable_head
	const opcode = 0

	err := c.Context().SendRequest(c, opcode, configHead, head)
	if err != nil {
		c.Context().Unregister(configHead)
		return nil, err
	}

	return configHead, nil
}

// DisableHead disables a head
func (c *OutputConfiguration) DisableHead(head *OutputHead) error {
	// Opcode 1: disable_head
	const opcode = 1
	return c.Context().SendRequest(c, opcode, head)
}

// Apply applies the configuration
func (c *OutputConfiguration) Apply() error {
	// Opcode 2: apply
	const opcode = 2
	return c.Context().SendRequest(c, opcode)
}

// Test tests the configuration
func (c *OutputConfiguration) Test() error {
	// Opcode 3: test
	const opcode = 3
	return c.Context().SendRequest(c, opcode)
}

// Destroy destroys the output configuration
func (c *OutputConfiguration) Destroy() error {
	// Opcode 4: destroy
	const opcode = 4
	err := c.Context().SendRequest(c, opcode)
	c.Context().Unregister(c)
	return err
}

// Dispatch handles incoming events
func (c *OutputConfiguration) Dispatch(event *wl.Event) {
	switch event.Opcode {
	case 0: // succeeded
		if c.succeededHandler != nil {
			c.succeededHandler()
		}
	case 1: // failed
		if c.failedHandler != nil {
			c.failedHandler()
		}
	case 2: // cancelled
		if c.cancelledHandler != nil {
			c.cancelledHandler()
		}
	}
}

// OutputConfigurationHead represents a head configuration
type OutputConfigurationHead struct {
	wl.BaseProxy
}

// NewOutputConfigurationHead creates a new output configuration head
func NewOutputConfigurationHead(ctx *wl.Context) *OutputConfigurationHead {
	head := &OutputConfigurationHead{}
	head.SetContext(ctx)
	return head
}

// SetMode sets the mode
func (h *OutputConfigurationHead) SetMode(mode *OutputMode) error {
	// Opcode 0: set_mode
	const opcode = 0
	return h.Context().SendRequest(h, opcode, mode)
}

// SetCustomMode sets a custom mode
func (h *OutputConfigurationHead) SetCustomMode(width, height, refresh int32) error {
	// Opcode 1: set_custom_mode
	const opcode = 1
	return h.Context().SendRequest(h, opcode, width, height, refresh)
}

// SetPosition sets the position
func (h *OutputConfigurationHead) SetPosition(x, y int32) error {
	// Opcode 2: set_position
	const opcode = 2
	return h.Context().SendRequest(h, opcode, x, y)
}

// SetTransform sets the transform
func (h *OutputConfigurationHead) SetTransform(transform int32) error {
	// Opcode 3: set_transform
	const opcode = 3
	return h.Context().SendRequest(h, opcode, transform)
}

// SetScale sets the scale
func (h *OutputConfigurationHead) SetScale(scale wl.Fixed) error {
	// Opcode 4: set_scale
	const opcode = 4
	return h.Context().SendRequest(h, opcode, scale)
}

// SetAdaptiveSync sets adaptive sync state (since version 4)
func (h *OutputConfigurationHead) SetAdaptiveSync(state uint32) error {
	// Opcode 5: set_adaptive_sync
	const opcode = 5
	return h.Context().SendRequest(h, opcode, state)
}

// Destroy destroys the output configuration head
func (h *OutputConfigurationHead) Destroy() error {
	h.Context().Unregister(h)
	return nil
}

// Dispatch handles incoming events (output configuration head has no events)
func (h *OutputConfigurationHead) Dispatch(event *wl.Event) {
	// Output configuration head has no events
}
