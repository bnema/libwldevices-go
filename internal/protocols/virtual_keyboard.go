package protocols

import (
	"os"
	"syscall"

	"github.com/neurlang/wayland/wl"
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
	ctx.Register(manager)
	return manager
}

// CreateVirtualKeyboard creates a new virtual keyboard
func (m *VirtualKeyboardManager) CreateVirtualKeyboard(seat *wl.Seat) (*VirtualKeyboard, error) {
	keyboard := NewVirtualKeyboard(m.Context())
	
	// Opcode 0: create_virtual_keyboard
	const opcode = 0
	
	err := m.Context().SendRequest(m, opcode, seat, keyboard)
	if err != nil {
		m.Context().Unregister(keyboard.Id())
		return nil, err
	}
	
	return keyboard, nil
}

// Destroy destroys the virtual keyboard manager (no destructor in protocol)
func (m *VirtualKeyboardManager) Destroy() error {
	m.Context().Unregister(m.Id())
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
	ctx.Register(keyboard)
	return keyboard
}

// Keymap sets the keyboard mapping
func (k *VirtualKeyboard) Keymap(format uint32, fd int, size uint32) error {
	// Opcode 0: keymap
	const opcode = 0
	
	// The neurlang/wayland library expects file descriptors as uintptr
	return k.Context().SendRequest(k, opcode, format, uintptr(fd), size)
}

// Key sends a key press/release event
func (k *VirtualKeyboard) Key(time, key, state uint32) error {
	// Opcode 1: key
	const opcode = 1
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
	k.Context().Unregister(k.Id())
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

	// Create a temporary file
	file, err := os.CreateTemp("", "keymap-*.xkb")
	if err != nil {
		return -1, 0, err
	}
	defer file.Close()

	// Write keymap
	_, err = file.WriteString(keymap)
	if err != nil {
		return -1, 0, err
	}

	// Get file descriptor
	fd := int(file.Fd())

	// Duplicate the fd so it remains valid after file.Close()
	newFd, err := syscall.Dup(fd)
	if err != nil {
		return -1, 0, err
	}

	return newFd, uint32(len(keymap)), nil
}

// Dispatch handles incoming events (virtual keyboard has no events)
func (k *VirtualKeyboard) Dispatch(event *wl.Event) {
	// Virtual keyboard has no events
}