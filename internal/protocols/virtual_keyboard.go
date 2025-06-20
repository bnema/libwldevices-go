package protocols

import (
	"fmt"
	"syscall"

	"github.com/bnema/wlturbo/wl"
)

// Protocol interface names for virtual keyboard
const (
	VirtualKeyboardManagerInterface = "zwp_virtual_keyboard_manager_v1"
	VirtualKeyboardInterface        = "zwp_virtual_keyboard_v1"
)

// VirtualKeyboardManager manages virtual keyboard objects
type VirtualKeyboardManager struct {
	wl.BaseProxy
}

// NewVirtualKeyboardManager creates a new virtual keyboard manager
func NewVirtualKeyboardManager(ctx *wl.Context) *VirtualKeyboardManager {
	manager := &VirtualKeyboardManager{}
	// Set the context properly
	manager.SetContext(ctx)
	// Note: Manager ID will be set by Registry.Bind
	return manager
}

// CreateVirtualKeyboard creates a new virtual keyboard
func (m *VirtualKeyboardManager) CreateVirtualKeyboard(seat *wl.Seat) (*VirtualKeyboard, error) {
	keyboard := NewVirtualKeyboard(m.Context())

	// Opcode 0: create_virtual_keyboard
	const opcode = 0

	err := m.Context().SendRequest(m, opcode, seat, keyboard)
	if err != nil {
		m.Context().Unregister(keyboard)
		return nil, err
	}

	return keyboard, nil
}

// Destroy destroys the virtual keyboard manager (no destructor in protocol)
func (m *VirtualKeyboardManager) Destroy() error {
	m.Context().Unregister(m)
	return nil
}

// Dispatch handles incoming events (manager has no events)
func (m *VirtualKeyboardManager) Dispatch(event *wl.Event) {
	// Virtual keyboard manager has no events
}

// VirtualKeyboard represents a virtual keyboard device
type VirtualKeyboard struct {
	wl.BaseProxy
}

// NewVirtualKeyboard creates a new virtual keyboard
func NewVirtualKeyboard(ctx *wl.Context) *VirtualKeyboard {
	keyboard := &VirtualKeyboard{}
	// Set the context properly
	keyboard.SetContext(ctx)
	// Allocate and set ID before registering
	id := ctx.AllocateID()
	keyboard.SetID(id)
	ctx.Register(keyboard)
	return keyboard
}

// Keymap sets the keyboard mapping
func (k *VirtualKeyboard) Keymap(format uint32, fd int, size uint32) error {
	// Opcode 0: keymap
	const opcode = 0

	// Debug: verify fd is valid
	if fd < 0 {
		return fmt.Errorf("invalid file descriptor: %d", fd)
	}

	// File descriptors must be sent via SendRequestWithFDs
	// The fd argument is passed as uintptr for neurlang/wayland compatibility
	return k.Context().SendRequestWithFDs(k, opcode, []int{fd}, format, uintptr(fd), size)
}

// Key sends a key press/release event
func (k *VirtualKeyboard) Key(time, key, state uint32) error {
	// Opcode 1: key
	const opcode = 1

	// The virtual keyboard protocol expects raw evdev key codes, NOT XKB key codes
	// Do NOT add 8 - that's only for XKB keysyms, not for virtual keyboard input
	return k.Context().SendRequest(k, opcode, time, key, state)
}

// Modifiers updates modifier state
func (k *VirtualKeyboard) Modifiers(modsDepressed, modsLatched, modsLocked, group uint32) error {
	// Opcode 2: modifiers
	const opcode = 2
	return k.Context().SendRequest(k, opcode, modsDepressed, modsLatched, modsLocked, group)
}

// Destroy destroys the virtual keyboard
func (k *VirtualKeyboard) Destroy() error {
	// Opcode 3: destroy
	const opcode = 3
	err := k.Context().SendRequest(k, opcode)
	k.Context().Unregister(k)
	return err
}

// CreateDefaultKeymap creates a minimal XKB keymap file descriptor
func CreateDefaultKeymap() (int, uint32, error) {
	// Minimal XKB keymap
	keymap := `xkb_keymap {
	xkb_keycodes  { include "evdev+aliases(qwerty)"	};
	xkb_types     { include "complete"	};
	xkb_compat    { include "complete"	};
	xkb_symbols   { include "pc+us+inet(evdev)"	};
	xkb_geometry  { include "pc(pc105)"	};
};`

	// Create anonymous shared memory file
	size := len(keymap) + 1 // +1 for null terminator
	fd, err := wl.CreateAnonymousFile(int64(size))
	if err != nil {
		return -1, 0, err
	}

	// Map the memory
	data, err := wl.MapMemory(fd, size)
	if err != nil {
		_ = syscall.Close(fd)
		return -1, 0, err
	}
	defer func() { _ = wl.UnmapMemory(data) }()

	// Copy keymap to shared memory
	copy(data, keymap)
	data[len(keymap)] = 0 // null terminator

	// Seek to beginning for compositor to read
	_, err = syscall.Seek(fd, 0, 0)
	if err != nil {
		_ = syscall.Close(fd)
		return -1, 0, err
	}

	// Verify the fd is readable
	var stat syscall.Stat_t
	if err := syscall.Fstat(fd, &stat); err != nil {
		_ = syscall.Close(fd)
		return -1, 0, fmt.Errorf("fstat failed: %w", err)
	}

	// Return the keymap size INCLUDING null terminator
	// Wayland expects the full mmap size including the null byte
	// Safe conversion: size is controlled and small
	if size < 0 || size > 0x7FFFFFFF {
		_ = syscall.Close(fd)
		return -1, 0, fmt.Errorf("invalid keymap size: %d", size)
	}
	return fd, uint32(size), nil
}

// Dispatch handles incoming events (virtual keyboard has no events)
func (k *VirtualKeyboard) Dispatch(event *wl.Event) {
	// Virtual keyboard has no events
}
