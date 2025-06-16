// Package wlclient provides a minimal Wayland client implementation for virtual input protocols.
//
// This package implements just enough of the Wayland protocol to support virtual pointer
// and keyboard input injection. It's not a full Wayland client library.
package wlclient

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// Fixed represents a 24.8 fixed-point number
type Fixed int32

// Float64 converts Fixed to float64
func (f Fixed) Float64() float64 {
	return float64(f) / 256.0
}

// NewFixed creates a Fixed from float64
func NewFixed(v float64) Fixed {
	return Fixed(v * 256.0)
}

// Object represents a Wayland object
type Object interface {
	ID() uint32
}

// Display represents a connection to the Wayland display
type Display struct {
	conn      net.Conn
	fd        int
	objects   sync.Map // map[uint32]Object
	nextID    uint32
	sendMu    sync.Mutex
	recvMu    sync.Mutex
	listeners sync.Map // map[uint32]map[uint16][]func([]byte)
	
	// Core objects
	registry *Registry
	
	// Error state
	lastError     error
	lastErrorCode uint32
	lastErrorObj  uint32
}

// Registry represents the global registry
type Registry struct {
	id       uint32
	display  *Display
	globals  map[uint32]Global
	mu       sync.RWMutex
	handlers map[string]GlobalHandler
}

// Global represents a global object
type Global struct {
	Name      uint32
	Interface string
	Version   uint32
}

// GlobalHandler is called when a global is announced
type GlobalHandler func(registry *Registry, name uint32, version uint32)

// Connect connects to the Wayland display
func Connect(socketPath string) (*Display, error) {
	if socketPath == "" {
		socketPath = os.Getenv("WAYLAND_DISPLAY")
		if socketPath == "" {
			socketPath = "wayland-0"
		}
	}

	// Resolve socket path
	if !filepath.IsAbs(socketPath) {
		runDir := os.Getenv("XDG_RUNTIME_DIR")
		if runDir == "" {
			return nil, errors.New("XDG_RUNTIME_DIR not set")
		}
		socketPath = filepath.Join(runDir, socketPath)
	}

	// Connect to socket
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Wayland: %w", err)
	}

	// Get file descriptor for advanced operations
	file, err := conn.(*net.UnixConn).File()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get socket fd: %w", err)
	}
	fd := int(file.Fd())
	file.Close() // We only need the fd

	d := &Display{
		conn:   conn,
		fd:     fd,
		nextID: 2, // 1 is reserved for wl_display
	}

	// Register display object (ID 1)
	d.objects.Store(uint32(1), d)
	
	// Initialize registry
	d.registry = &Registry{
		id:       d.allocateID(),
		display:  d,
		globals:  make(map[uint32]Global),
		handlers: make(map[string]GlobalHandler),
	}
	d.objects.Store(d.registry.id, d.registry)

	// Get registry
	if err := d.getRegistry(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get registry: %w", err)
	}

	// Do initial roundtrip to populate registry
	if err := d.Roundtrip(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("initial roundtrip failed: %w", err)
	}

	return d, nil
}

// Close closes the display connection
func (d *Display) Close() error {
	return d.conn.Close()
}

// ID returns the display's object ID (always 1)
func (d *Display) ID() uint32 {
	return 1
}

// allocateID allocates a new object ID
func (d *Display) allocateID() uint32 {
	return atomic.AddUint32(&d.nextID, 1) - 1
}

// SendRequest sends a request to the compositor
func (d *Display) SendRequest(objectID uint32, opcode uint16, args ...interface{}) error {
	d.sendMu.Lock()
	defer d.sendMu.Unlock()

	var buf bytes.Buffer

	// Write header placeholder
	header := make([]byte, 8)
	buf.Write(header)

	// Marshal arguments
	for _, arg := range args {
		if err := d.marshalArg(&buf, arg); err != nil {
			return fmt.Errorf("failed to marshal argument: %w", err)
		}
	}

	// Update header with actual size
	size := uint32(buf.Len())
	binary.LittleEndian.PutUint32(header[0:4], objectID)
	binary.LittleEndian.PutUint32(header[4:8], (size&0xffff)|uint32(opcode)<<16)
	
	// Update buffer with correct header
	data := buf.Bytes()
	copy(data[0:8], header)

	// Send message
	_, err := d.conn.Write(data)
	return err
}

// marshalArg marshals a single argument
func (d *Display) marshalArg(buf *bytes.Buffer, arg interface{}) error {
	switch v := arg.(type) {
	case uint32:
		return binary.Write(buf, binary.LittleEndian, v)
	case int32:
		return binary.Write(buf, binary.LittleEndian, v)
	case Fixed:
		return binary.Write(buf, binary.LittleEndian, int32(v))
	case string:
		// String format: length (including null) + string + null + padding
		strlen := len(v) + 1
		if err := binary.Write(buf, binary.LittleEndian, uint32(strlen)); err != nil {
			return err
		}
		buf.WriteString(v)
		buf.WriteByte(0)
		// Pad to 32-bit boundary
		padding := (4 - (strlen % 4)) % 4
		for i := 0; i < padding; i++ {
			buf.WriteByte(0)
		}
	case []byte:
		// Array format: length + data + padding
		arrlen := len(v)
		if err := binary.Write(buf, binary.LittleEndian, uint32(arrlen)); err != nil {
			return err
		}
		buf.Write(v)
		// Pad to 32-bit boundary
		padding := (4 - (arrlen % 4)) % 4
		for i := 0; i < padding; i++ {
			buf.WriteByte(0)
		}
	case Object:
		if v != nil {
			return binary.Write(buf, binary.LittleEndian, v.ID())
		} else {
			return binary.Write(buf, binary.LittleEndian, uint32(0))
		}
	case nil:
		// Null object
		return binary.Write(buf, binary.LittleEndian, uint32(0))
	default:
		return fmt.Errorf("unsupported argument type: %T", arg)
	}
	return nil
}

// Dispatch reads and dispatches events
func (d *Display) Dispatch() error {
	d.recvMu.Lock()
	defer d.recvMu.Unlock()

	// Read header
	header := make([]byte, 8)
	if _, err := io.ReadFull(d.conn, header); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	objectID := binary.LittleEndian.Uint32(header[0:4])
	sizeOpcode := binary.LittleEndian.Uint32(header[4:8])
	size := sizeOpcode & 0xffff
	opcode := uint16(sizeOpcode >> 16)

	// Read message body
	var body []byte
	if size > 8 {
		body = make([]byte, size-8)
		if _, err := io.ReadFull(d.conn, body); err != nil {
			return fmt.Errorf("failed to read body: %w", err)
		}
	}

	// Handle display events specially
	if objectID == 1 {
		return d.handleDisplayEvent(opcode, body)
	}

	// Dispatch to object
	if listeners, ok := d.listeners.Load(objectID); ok {
		if opcodeMap, ok := listeners.(map[uint16][]func([]byte)); ok {
			if handlers, ok := opcodeMap[opcode]; ok {
				for _, handler := range handlers {
					handler(body)
				}
			}
		}
	}

	return nil
}

// handleDisplayEvent handles events on the display object
func (d *Display) handleDisplayEvent(opcode uint16, data []byte) error {
	switch opcode {
	case 0: // error
		if len(data) < 12 {
			return errors.New("invalid error event")
		}
		objectID := binary.LittleEndian.Uint32(data[0:4])
		code := binary.LittleEndian.Uint32(data[4:8])
		msgLen := binary.LittleEndian.Uint32(data[8:12])
		
		var message string
		if msgLen > 0 && len(data) >= 12+int(msgLen) {
			message = string(data[12 : 12+msgLen-1]) // -1 to remove null terminator
		}
		
		d.lastError = fmt.Errorf("protocol error: object %d, code %d: %s", objectID, code, message)
		d.lastErrorCode = code
		d.lastErrorObj = objectID
		return d.lastError
		
	case 1: // delete_id
		if len(data) < 4 {
			return errors.New("invalid delete_id event")
		}
		id := binary.LittleEndian.Uint32(data[0:4])
		d.objects.Delete(id)
	}
	
	return nil
}

// Roundtrip performs a synchronous roundtrip to the compositor
func (d *Display) Roundtrip() error {
	// Create callback
	callbackID := d.allocateID()
	done := make(chan error, 1)
	
	// Register callback listener
	d.AddListener(callbackID, 0, func(data []byte) {
		d.objects.Delete(callbackID)
		done <- nil
	})
	
	// Send sync request (opcode 0)
	if err := d.SendRequest(1, 0, callbackID); err != nil {
		return err
	}
	
	// Process events until callback fires
	for {
		if err := d.Dispatch(); err != nil {
			return err
		}
		
		select {
		case err := <-done:
			return err
		default:
			// Continue dispatching
		}
	}
}

// AddListener adds an event listener for an object
func (d *Display) AddListener(objectID uint32, opcode uint16, handler func([]byte)) {
	// Load or create listener map for object
	listeners, _ := d.listeners.LoadOrStore(objectID, make(map[uint16][]func([]byte)))
	opcodeMap := listeners.(map[uint16][]func([]byte))
	
	// Add handler
	opcodeMap[opcode] = append(opcodeMap[opcode], handler)
}

// getRegistry gets the global registry
func (d *Display) getRegistry() error {
	// Add registry listeners
	d.AddListener(d.registry.id, 0, d.registry.handleGlobal)
	d.AddListener(d.registry.id, 1, d.registry.handleGlobalRemove)
	
	// Send get_registry request (opcode 1)
	return d.SendRequest(1, 1, d.registry.id)
}

// Registry returns the global registry
func (d *Display) Registry() *Registry {
	return d.registry
}

// ID returns the registry's object ID
func (r *Registry) ID() uint32 {
	return r.id
}

// handleGlobal handles global announcements
func (r *Registry) handleGlobal(data []byte) {
	if len(data) < 12 {
		return
	}
	
	name := binary.LittleEndian.Uint32(data[0:4])
	ifaceLen := binary.LittleEndian.Uint32(data[4:8])
	
	if len(data) < 12+int(ifaceLen) {
		return
	}
	
	iface := string(data[8 : 8+ifaceLen-1]) // -1 to remove null terminator
	version := binary.LittleEndian.Uint32(data[8+ifaceLen:])
	
	// Store global
	r.mu.Lock()
	r.globals[name] = Global{
		Name:      name,
		Interface: iface,
		Version:   version,
	}
	r.mu.Unlock()
	
	// Call handler if registered
	if handler, ok := r.handlers[iface]; ok {
		handler(r, name, version)
	}
}

// handleGlobalRemove handles global removal
func (r *Registry) handleGlobalRemove(data []byte) {
	if len(data) < 4 {
		return
	}
	
	name := binary.LittleEndian.Uint32(data[0:4])
	
	r.mu.Lock()
	delete(r.globals, name)
	r.mu.Unlock()
}

// AddHandler adds a handler for a specific interface
func (r *Registry) AddHandler(iface string, handler GlobalHandler) {
	r.handlers[iface] = handler
}

// Bind binds to a global object
func (r *Registry) Bind(name uint32, iface string, version uint32) (uint32, error) {
	newID := r.display.allocateID()
	
	// Marshal interface string and new_id
	var buf bytes.Buffer
	
	// name (4 bytes)
	binary.Write(&buf, binary.LittleEndian, name)
	
	// interface string with length
	ifaceBytes := []byte(iface)
	binary.Write(&buf, binary.LittleEndian, uint32(len(ifaceBytes)+1))
	buf.Write(ifaceBytes)
	buf.WriteByte(0)
	// Pad to 32-bit boundary
	padding := (4 - ((len(ifaceBytes)+1) % 4)) % 4
	for i := 0; i < padding; i++ {
		buf.WriteByte(0)
	}
	
	// version (4 bytes)
	binary.Write(&buf, binary.LittleEndian, version)
	
	// new_id (4 bytes)
	binary.Write(&buf, binary.LittleEndian, newID)
	
	// Send bind request (opcode 0)
	if err := r.display.SendRequest(r.id, 0, buf.Bytes()); err != nil {
		return 0, err
	}
	
	return newID, nil
}

// GetGlobals returns all announced globals
func (r *Registry) GetGlobals() map[uint32]Global {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	globals := make(map[uint32]Global)
	for k, v := range r.globals {
		globals[k] = v
	}
	return globals
}

// FindGlobal finds a global by interface name
func (r *Registry) FindGlobal(iface string) (Global, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	for _, global := range r.globals {
		if global.Interface == iface {
			return global, true
		}
	}
	return Global{}, false
}