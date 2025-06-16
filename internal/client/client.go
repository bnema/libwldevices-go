// Package client provides Wayland client connection management for virtual input protocols
package client

import (
	"fmt"
	"sync"

	"github.com/neurlang/wayland/wl"
)

// Client manages the Wayland connection and protocol objects
type Client struct {
	display    *wl.Display
	registry   *wl.Registry
	seat       *wl.Seat
	context    *wl.Context
	
	// Protocol globals
<<<<<<< HEAD
	pointerManager   uint32
	keyboardManager  uint32
	
	mu sync.Mutex
=======
	pointerManager     uint32
	keyboardManager    uint32
	constraintsManager uint32

	mu      sync.Mutex
>>>>>>> 82885fa (feat: add pointer constraints protocol implementation)
	globals map[uint32]string
}

// NewClient creates a new Wayland client
func NewClient() (*Client, error) {
	display, err := wl.Connect("")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Wayland: %w", err)
	}
	
	client := &Client{
		display: display,
		context: display.Context(),
		globals: make(map[uint32]string),
	}
	
	// Get registry
	registry, err := display.GetRegistry()
	if err != nil {
		return nil, fmt.Errorf("failed to get registry: %w", err)
	}
	client.registry = registry
	
	// Set up registry listener
	registry.AddGlobalHandler(client)
	registry.AddGlobalRemoveHandler(client)
	
	// Get initial globals
	sync, err := display.Sync()
	if err != nil {
		return nil, fmt.Errorf("failed to sync: %w", err)
	}
	
	// Wait for sync
	err = client.context.RunTill(sync)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for sync: %w", err)
	}
	
	return client, nil
}

// HandleRegistryGlobal implements wl.RegistryGlobalHandler
func (c *Client) HandleRegistryGlobal(event wl.RegistryGlobalEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.globals[event.Name] = event.Interface
	
	switch event.Interface {
	case "wl_seat":
		// Bind to seat for virtual input
		seat := wl.NewSeat(c.context)
		err := c.registry.Bind(event.Name, event.Interface, event.Version, seat)
		if err == nil {
			c.seat = seat
		}
		
	case "zwlr_virtual_pointer_manager_v1":
		c.pointerManager = event.Name
		
	case "zwp_virtual_keyboard_manager_v1":
		c.keyboardManager = event.Name

	case "zwp_pointer_constraints_v1":
		c.constraintsManager = event.Name
	}
}

// HandleRegistryGlobalRemove implements wl.RegistryGlobalRemoveHandler
func (c *Client) HandleRegistryGlobalRemove(event wl.RegistryGlobalRemoveEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.globals, event.Name)
}

// HasVirtualPointer returns true if virtual pointer protocol is available
func (c *Client) HasVirtualPointer() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pointerManager != 0
}

// HasVirtualKeyboard returns true if virtual keyboard protocol is available
func (c *Client) HasVirtualKeyboard() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.keyboardManager != 0
}

// GetRegistry returns the Wayland registry
func (c *Client) GetRegistry() *wl.Registry {
	return c.registry
}

// GetDisplay returns the Wayland display
func (c *Client) GetDisplay() *wl.Display {
	return c.display
}

// GetContext returns the Wayland context
func (c *Client) GetContext() *wl.Context {
	return c.context
}

// GetSeat returns the Wayland seat
func (c *Client) GetSeat() *wl.Seat {
	return c.seat
}

// GetPointerManagerName returns the name ID for the virtual pointer manager
func (c *Client) GetPointerManagerName() uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pointerManager
}

// GetKeyboardManagerName returns the name ID for the virtual keyboard manager  
func (c *Client) GetKeyboardManagerName() uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.keyboardManager
}

// HasPointerConstraints returns true if pointer constraints protocol is available
func (c *Client) HasPointerConstraints() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.constraintsManager != 0
}

// GetConstraintsManagerName returns the name ID for the pointer constraints manager
func (c *Client) GetConstraintsManagerName() uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.constraintsManager
}

// Close closes the Wayland connection
func (c *Client) Close() error {
	if c.context != nil {
		return c.context.Close()
	}
	return nil
}