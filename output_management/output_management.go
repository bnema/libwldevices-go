// Package output_management provides high-level Go bindings for the wlr-output-management Wayland protocol.
//
// This package allows applications to query and monitor output device configuration
// including monitor position, size, scale, and other properties.
package output_management

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bnema/libwldevices-go/internal/client"
	"github.com/bnema/libwldevices-go/internal/protocols"
	"github.com/bnema/wlturbo/wl"
)

// OutputManager manages output configuration and monitoring
type OutputManager struct {
	client    *client.Client
	manager   *protocols.OutputManager
	heads     map[uint32]*OutputHead
	mu        sync.RWMutex
	serial    uint32
	handlers  OutputHandlers
	hasSerial bool
	serialCh  chan struct{}
}

// OutputHandlers contains callback functions for output events
type OutputHandlers struct {
	// OnHeadAdded is called when a new output head is detected
	OnHeadAdded func(head *OutputHead)
	// OnHeadRemoved is called when an output head is removed
	OnHeadRemoved func(head *OutputHead)
	// OnConfigurationChanged is called when output configuration changes
	OnConfigurationChanged func(heads []*OutputHead)
}

// OutputHead represents a physical output device (monitor)
type OutputHead struct {
	ID           uint32
	Name         string
	Description  string
	Make         string
	Model        string
	SerialNumber string
	Enabled      bool
	Position     Position
	Size         Size
	PhysicalSize Size // Physical size in millimeters
	Mode         *OutputMode
	CurrentMode  *OutputMode
	Scale        float64
	Transform    Transform
	head         *protocols.OutputHead
	modes        []*OutputMode
}

// Position represents the position of an output in the global compositor space
type Position struct {
	X int32
	Y int32
}

// Size represents the size of an output
type Size struct {
	Width  int32
	Height int32
}

// OutputMode represents a display mode
type OutputMode struct {
	Width     int32
	Height    int32
	Refresh   int32 // in mHz
	Preferred bool
	mode      *protocols.OutputMode
}

// Transform represents output transformation
type Transform int32

// Transform constants for output rotation and flipping
const (
	TransformNormal         Transform = iota // No transformation
	Transform90                              // 90 degree clockwise rotation
	Transform180                             // 180 degree rotation
	Transform270                             // 270 degree clockwise rotation
	TransformFlipped                         // Horizontal flip
	TransformFlipped90                       // Horizontal flip + 90 degree rotation
	TransformFlipped180                      // Horizontal flip + 180 degree rotation
	TransformFlipped270                      // Horizontal flip + 270 degree rotation
)

// NewOutputManager creates a new output manager
func NewOutputManager(ctx context.Context) (*OutputManager, error) {
	// fmt.Println("[DEBUG] Creating output manager...")
	
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
	// fmt.Println("[DEBUG] Client created successfully")

	// Check if output manager protocol is available using the client's detection
	// fmt.Printf("[DEBUG] Checking HasOutputManager: %v\n", c.HasOutputManager())
	// fmt.Printf("[DEBUG] OutputManagerName: %d\n", c.GetOutputManagerName())
	if !c.HasOutputManager() {
		c.Close()
		return nil, fmt.Errorf("zwlr_output_manager_v1 not available - compositor may not support wlr-output-management protocol")
	}
	// fmt.Println("[DEBUG] Output manager protocol is available")

	om := &OutputManager{
		client:   c,
		heads:    make(map[uint32]*OutputHead),
		serialCh: make(chan struct{}, 1),
	}

	// Use the output manager name from the client
	managerName := c.GetOutputManagerName()
	registry := c.GetRegistry()
	context := c.GetContext()
	// fmt.Printf("[DEBUG] Output manager name: %d\n", managerName)

	// Create and bind output manager
	om.manager = protocols.NewOutputManager(context)
	// fmt.Printf("[DEBUG] Created output manager proxy with ID: %d\n", om.manager.ID())

	err := registry.Bind(managerName, protocols.OutputManagerInterface, 4, om.manager)
	if err != nil {
		return nil, fmt.Errorf("failed to bind output manager: %w", err)
	}
	// fmt.Printf("[DEBUG] Bound to output manager successfully, ID: %d\n", om.manager.ID())

	// Set up event handlers
	om.manager.SetHeadHandler(om.handleHead)
	om.manager.SetDoneHandler(om.handleDone)
	om.manager.SetFinishedHandler(om.handleFinished)
	// fmt.Println("[DEBUG] Event handlers set up")

	// Start event processing in background
	go func() {
		// fmt.Println("[DEBUG] Starting event dispatch loop...")
		for {
			if err := c.GetDisplay().Dispatch(); err != nil {
				// Connection closed or error occurred
				// fmt.Printf("[DEBUG] Dispatch error: %v\n", err)
				return
			}
		}
	}()

	// Force a roundtrip to get initial events
	// fmt.Println("[DEBUG] Performing roundtrip...")
	_ = c.GetDisplay().Roundtrip() // Ignore roundtrip errors during initialization

	// Wait for initial configuration to be received with context support
	// fmt.Println("[DEBUG] Waiting for initial configuration...")
	select {
	case <-om.serialCh:
		// Initial configuration received
		// fmt.Println("[DEBUG] Initial configuration received")
	case <-time.After(5 * time.Second):
		// fmt.Println("[DEBUG] Timeout waiting for initial configuration")
		_ = om.Close()
		return nil, fmt.Errorf("timeout waiting for initial output configuration")
	case <-ctx.Done():
		_ = om.Close()
		return nil, ctx.Err()
	}

	return om, nil
}

// GetHeads returns all currently detected output heads
func (om *OutputManager) GetHeads() []*OutputHead {
	if om == nil {
		return nil
	}

	om.mu.RLock()
	defer om.mu.RUnlock()

	heads := make([]*OutputHead, 0, len(om.heads))
	for _, head := range om.heads {
		heads = append(heads, head)
	}
	return heads
}

// GetEnabledHeads returns only enabled output heads
func (om *OutputManager) GetEnabledHeads() []*OutputHead {
	if om == nil {
		return nil
	}

	om.mu.RLock()
	defer om.mu.RUnlock()

	var heads []*OutputHead
	for _, head := range om.heads {
		if head.Enabled {
			heads = append(heads, head)
		}
	}
	return heads
}

// GetHeadByName returns an output head by name
func (om *OutputManager) GetHeadByName(name string) *OutputHead {
	if om == nil {
		return nil
	}

	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, head := range om.heads {
		if head.Name == name {
			return head
		}
	}
	return nil
}

// GetPrimaryHead returns the output at position (0,0)
func (om *OutputManager) GetPrimaryHead() *OutputHead {
	if om == nil {
		return nil
	}

	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, head := range om.heads {
		if head.Enabled && head.Position.X == 0 && head.Position.Y == 0 {
			return head
		}
	}

	// Fallback to first enabled head
	for _, head := range om.heads {
		if head.Enabled {
			return head
		}
	}

	return nil
}

// GetHeadAtPoint returns the output head that contains the given point
func (om *OutputManager) GetHeadAtPoint(x, y int32) *OutputHead {
	if om == nil {
		return nil
	}

	om.mu.RLock()
	defer om.mu.RUnlock()

	for _, head := range om.heads {
		if head.Enabled && head.ContainsPoint(x, y) {
			return head
		}
	}
	return nil
}

// SetHandlers sets the event handlers for output events
func (om *OutputManager) SetHandlers(handlers OutputHandlers) {
	if om == nil {
		return
	}

	om.mu.Lock()
	om.handlers = handlers
	om.mu.Unlock()
}

// Close cleans up the output manager
func (om *OutputManager) Close() error {
	if om == nil {
		return nil
	}

	if om.manager != nil {
		_ = om.manager.Stop()
		_ = om.manager.Destroy()
	}

	if om.client != nil {
		return om.client.Close()
	}

	return nil
}

// Event handlers
func (om *OutputManager) handleHead(head *protocols.OutputHead) {
	// fmt.Printf("[DEBUG] handleHead called with head ID: %d\n", head.ID())
	if head == nil {
		return
	}

	om.mu.Lock()
	defer om.mu.Unlock()

	// Get a unique ID for this head
	headID := uint32(head.ID())

	outputHead := &OutputHead{
		ID:    headID,
		head:  head,
		modes: make([]*OutputMode, 0),
		Scale: 1.0, // Default scale
	}

	// Set up head event handlers
	head.SetNameHandler(func(name string) {
		om.mu.Lock()
		outputHead.Name = name
		om.mu.Unlock()
	})

	head.SetDescriptionHandler(func(description string) {
		outputHead.Description = description
	})

	head.SetPhysicalSizeHandler(func(width, height int32) {
		outputHead.PhysicalSize = Size{Width: width, Height: height}
	})

	head.SetEnabledHandler(func(enabled int32) {
		outputHead.Enabled = enabled != 0
	})

	head.SetPositionHandler(func(x, y int32) {
		outputHead.Position = Position{X: x, Y: y}
	})

	head.SetModeHandler(func(mode *protocols.OutputMode) {
		om := &OutputMode{
			mode: mode,
		}

		mode.SetSizeHandler(func(width, height int32) {
			om.Width = width
			om.Height = height
		})

		mode.SetRefreshHandler(func(refresh int32) {
			om.Refresh = refresh
		})

		mode.SetPreferredHandler(func() {
			om.Preferred = true
		})

		outputHead.modes = append(outputHead.modes, om)

		// Set as default mode if preferred
		if om.Preferred && outputHead.Mode == nil {
			outputHead.Mode = om
		}
	})

	head.SetCurrentModeHandler(func(mode *protocols.OutputMode) {
		// Find the mode in our list
		for _, m := range outputHead.modes {
			if m.mode == mode {
				outputHead.CurrentMode = m
				outputHead.Mode = m
				break
			}
		}
	})

	head.SetScaleHandler(func(scale wl.Fixed) {
		outputHead.Scale = float64(scale) / 256.0
		if outputHead.Scale == 0 {
			outputHead.Scale = 1.0 // Default to 1.0 if not set
		}
	})

	head.SetTransformHandler(func(transform int32) {
		outputHead.Transform = Transform(transform)
	})

	head.SetMakeHandler(func(makeStr string) {
		outputHead.Make = makeStr
	})

	head.SetModelHandler(func(model string) {
		outputHead.Model = model
	})

	head.SetSerialNumberHandler(func(serial string) {
		outputHead.SerialNumber = serial
	})

	head.SetFinishedHandler(func() {
		// Head is being removed
		delete(om.heads, outputHead.ID)
		if om.handlers.OnHeadRemoved != nil {
			om.handlers.OnHeadRemoved(outputHead)
		}
	})

	om.heads[uint32(head.ID())] = outputHead
}

func (om *OutputManager) handleDone(serial uint32) {
	// fmt.Printf("[DEBUG] handleDone called with serial: %d\n", serial)
	om.mu.Lock()
	isFirst := !om.hasSerial
	om.serial = serial
	om.hasSerial = true
	handlers := om.handlers
	om.mu.Unlock()

	// Signal that we have received the initial configuration
	if isFirst {
		select {
		case om.serialCh <- struct{}{}:
		default:
		}
	}

	// Configuration is complete, notify handlers
	if handlers.OnConfigurationChanged != nil {
		heads := om.GetHeads()
		handlers.OnConfigurationChanged(heads)
	}

	// Notify about new heads (only on first done after binding)
	if handlers.OnHeadAdded != nil && isFirst {
		om.mu.RLock()
		for _, head := range om.heads {
			handlers.OnHeadAdded(head)
		}
		om.mu.RUnlock()
	}
}

func (om *OutputManager) handleFinished() {
	// fmt.Println("[DEBUG] handleFinished called")
	// Manager is being destroyed
	om.mu.Lock()
	defer om.mu.Unlock()

	om.manager = nil
}

// Helper functions

// Bounds returns the bounding rectangle of the output head
func (h *OutputHead) Bounds() (x1, y1, x2, y2 int32) {
	x1 = h.Position.X
	y1 = h.Position.Y
	switch {
	case h.Mode != nil:
		x2 = x1 + h.Mode.Width
		y2 = y1 + h.Mode.Height
	case h.CurrentMode != nil:
		x2 = x1 + h.CurrentMode.Width
		y2 = y1 + h.CurrentMode.Height
	default:
		x2 = x1 + h.Size.Width
		y2 = y1 + h.Size.Height
	}
	return x1, y1, x2, y2
}

// Contains checks if a point is within this output
func (h *OutputHead) Contains(x, y int32) bool {
	if !h.Enabled {
		return false
	}
	x1, y1, x2, y2 := h.Bounds()
	return x >= x1 && x < x2 && y >= y1 && y < y2
}

// IsPrimary returns true if this output is at position (0,0)
func (h *OutputHead) IsPrimary() bool {
	return h.Enabled && h.Position.X == 0 && h.Position.Y == 0
}

// ContainsPoint returns true if the given point is within this output's bounds
func (h *OutputHead) ContainsPoint(x, y int32) bool {
	x1, y1, x2, y2 := h.Bounds()
	return x >= x1 && x < x2 && y >= y1 && y < y2
}

// GetModes returns all available modes for this output head
func (h *OutputHead) GetModes() []*OutputMode {
	if h == nil {
		return nil
	}
	return h.modes
}

// GetRefreshRate returns the refresh rate in Hz
func (m *OutputMode) GetRefreshRate() float64 {
	return float64(m.Refresh) / 1000.0
}

// String returns a string representation of the transform
func (t Transform) String() string {
	switch t {
	case TransformNormal:
		return "normal"
	case Transform90:
		return "90"
	case Transform180:
		return "180"
	case Transform270:
		return "270"
	case TransformFlipped:
		return "flipped"
	case TransformFlipped90:
		return "flipped-90"
	case TransformFlipped180:
		return "flipped-180"
	case TransformFlipped270:
		return "flipped-270"
	default:
		return "unknown"
	}
}
