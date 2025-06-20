// Package virtual_keyboard provides Go bindings for the virtual-keyboard-unstable-v1 Wayland protocol.
//
// This protocol allows clients to emulate a physical keyboard, enabling keyboard input injection
// into Wayland compositors. This is a complete, working implementation built on neurlang/wayland.
//
// # Basic Usage
//
//	// Create manager and keyboard
//	ctx := context.Background()
//	manager, err := NewVirtualKeyboardManager(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer manager.Close()
//
//	keyboard, err := manager.CreateKeyboard()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer keyboard.Close()
//
//	// Type text (handles uppercase/lowercase automatically)
//	keyboard.TypeString("Hello World!")
//
//	// Press individual keys
//	keyboard.TypeKey(KEY_ENTER)
//	keyboard.TypeKey(KEY_TAB)
//
//	// Manual key press/release
//	keyboard.Key(time.Now(), KEY_A, KeyStatePressed)
//	keyboard.Key(time.Now(), KEY_A, KeyStateReleased)
//
// # Protocol Specification
//
// Based on virtual-keyboard-unstable-v1 protocol.
// Supported by wlroots-based compositors (Sway, Hyprland, etc.).
package virtual_keyboard

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"github.com/bnema/libwldevices-go/internal/client"
	"github.com/bnema/libwldevices-go/internal/protocols"
)

// Common key constants (Linux input event codes)
const (
	KEY_A         = 30
	KEY_B         = 48
	KEY_C         = 46
	KEY_D         = 32
	KEY_E         = 18
	KEY_F         = 33
	KEY_G         = 34
	KEY_H         = 35
	KEY_I         = 23
	KEY_J         = 36
	KEY_K         = 37
	KEY_L         = 38
	KEY_M         = 50
	KEY_N         = 49
	KEY_O         = 24
	KEY_P         = 25
	KEY_Q         = 16
	KEY_R         = 19
	KEY_S         = 31
	KEY_T         = 20
	KEY_U         = 22
	KEY_V         = 47
	KEY_W         = 17
	KEY_X         = 45
	KEY_Y         = 21
	KEY_Z         = 44
	KEY_1         = 2
	KEY_2         = 3
	KEY_3         = 4
	KEY_4         = 5
	KEY_5         = 6
	KEY_6         = 7
	KEY_7         = 8
	KEY_8         = 9
	KEY_9         = 10
	KEY_0         = 11
	KEY_SPACE     = 57
	KEY_ENTER     = 28
	KEY_TAB       = 15
	KEY_BACKSPACE = 14
	KEY_ESC       = 1
	KEY_CAPSLOCK  = 58
	KEY_LEFTSHIFT = 42
	KEY_LEFTCTRL  = 29
	KEY_LEFTALT   = 56
	KEY_LEFTMETA  = 125
	
	// Additional keys for special characters
	KEY_MINUS        = 12  // - and _
	KEY_EQUAL        = 13  // = and +
	KEY_LEFTBRACE    = 26  // [ and {
	KEY_RIGHTBRACE   = 27  // ] and }
	KEY_SEMICOLON    = 39  // ; and :
	KEY_APOSTROPHE   = 40  // ' and "
	KEY_GRAVE        = 41  // ` and ~
	KEY_BACKSLASH    = 43  // \ and |
	KEY_COMMA        = 51  // , and <
	KEY_DOT          = 52  // . and >
	KEY_SLASH        = 53  // / and ?
	KEY_RIGHTSHIFT   = 54
)

// Key state constants
const (
	KEY_STATE_RELEASED = 0
	KEY_STATE_PRESSED  = 1
)

// Keymap format
const (
	KEYMAP_FORMAT_NO_KEYMAP = 0
	KEYMAP_FORMAT_XKB_V1    = 1
)

// KeyState represents the state of a key
type KeyState uint32

// Key state constants
const (
	KeyStateReleased KeyState = 0 // Key is released
	KeyStatePressed  KeyState = 1 // Key is pressed
)

// VirtualKeyboardManager manages virtual keyboard devices
type VirtualKeyboardManager struct {
	client  *client.Client
	manager *protocols.VirtualKeyboardManager
}

// VirtualKeyboard represents a virtual keyboard device
type VirtualKeyboard struct {
	keyboard  *protocols.VirtualKeyboard
	client    *client.Client
	keymapSet bool
}

// NewVirtualKeyboardManager creates a new virtual keyboard manager
func NewVirtualKeyboardManager(ctx context.Context) (*VirtualKeyboardManager, error) {
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
			return nil, fmt.Errorf("failed to create Wayland client: %w", result.err)
		}
		c = result.client
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled during client creation: %w", ctx.Err())
	}

	// Check if virtual keyboard protocol is available
	if !c.HasVirtualKeyboard() {
		c.Close()
		return nil, fmt.Errorf("zwp_virtual_keyboard_manager_v1 not available")
	}

	// Check context before binding
	select {
	case <-ctx.Done():
		_ = c.Close()
		return nil, fmt.Errorf("context cancelled before binding: %w", ctx.Err())
	default:
	}
	
	// Create the manager proxy
	manager := protocols.NewVirtualKeyboardManager(c.GetContext())

	// Bind to the global
	name := c.GetKeyboardManagerName()
	err := c.GetRegistry().Bind(name, protocols.VirtualKeyboardManagerInterface, 1, manager)
	if err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("failed to bind virtual keyboard manager: %w", err)
	}

	// Sync to ensure binding is complete
	if err := c.GetDisplay().Roundtrip(); err != nil {
		c.Close()
		return nil, fmt.Errorf("failed to roundtrip after binding: %w", err)
	}

	return &VirtualKeyboardManager{
		client:  c,
		manager: manager,
	}, nil
}

// CreateKeyboard creates a new virtual keyboard device
func (m *VirtualKeyboardManager) CreateKeyboard() (*VirtualKeyboard, error) {
	// Create virtual keyboard using the current seat
	keyboard, err := m.manager.CreateVirtualKeyboard(m.client.GetSeat())
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual keyboard: %w", err)
	}

	// Sync to ensure the keyboard is created
	if err := m.client.GetDisplay().Roundtrip(); err != nil {
		_ = keyboard.Destroy()
		return nil, fmt.Errorf("failed to roundtrip after creating keyboard: %w", err)
	}

	vk := &VirtualKeyboard{
		keyboard: keyboard,
		client:   m.client,
	}

	// Set default keymap
	if err := vk.setDefaultKeymap(); err != nil {
		_ = keyboard.Destroy()
		return nil, fmt.Errorf("failed to set default keymap: %w", err)
	}

	return vk, nil
}

// setDefaultKeymap sets a minimal default keymap
func (k *VirtualKeyboard) setDefaultKeymap() error {
	fd, size, err := protocols.CreateDefaultKeymap()
	if err != nil {
		return err
	}

	// Send the keymap
	err = k.keyboard.Keymap(KEYMAP_FORMAT_XKB_V1, fd, size)
	if err != nil {
		syscall.Close(fd)
		return err
	}

	k.keymapSet = true
	
	// Don't close the FD - the compositor needs to read it
	// The compositor will close it when done
	
	// Do a roundtrip to ensure the keymap is processed
	err = k.client.GetDisplay().Roundtrip()
	if err != nil {
		return fmt.Errorf("failed to roundtrip after keymap: %w", err)
	}

	// Note: FD is closed by the compositor after reading
	
	return nil
}

// Key sends a key press/release event
func (k *VirtualKeyboard) Key(timestamp time.Time, key uint32, state KeyState) error {
	if !k.keymapSet {
		return fmt.Errorf("keymap not set")
	}

	timeMs := uint32(timestamp.UnixNano() / 1000000)
	return k.keyboard.Key(timeMs, key, uint32(state))
}

// Modifiers updates the modifier state
func (k *VirtualKeyboard) Modifiers(modsDepressed, modsLatched, modsLocked, group uint32) error {
	if !k.keymapSet {
		return fmt.Errorf("keymap not set")
	}

	return k.keyboard.Modifiers(modsDepressed, modsLatched, modsLocked, group)
}

// Close releases the virtual keyboard device
func (k *VirtualKeyboard) Close() error {
	return k.keyboard.Destroy()
}

// Close releases the virtual keyboard manager
func (m *VirtualKeyboardManager) Close() error {
	if m.manager != nil {
		_ = m.manager.Destroy()
	}
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}

// Convenience methods for common operations

// PressKey presses a key (without releasing it)
func (k *VirtualKeyboard) PressKey(key uint32) error {
	return k.Key(time.Now(), key, KeyStatePressed)
}

// ReleaseKey releases a key
func (k *VirtualKeyboard) ReleaseKey(key uint32) error {
	return k.Key(time.Now(), key, KeyStateReleased)
}

// TypeKey presses and releases a key
func (k *VirtualKeyboard) TypeKey(key uint32) error {
	now := time.Now()
	if err := k.Key(now, key, KeyStatePressed); err != nil {
		return err
	}
	// Small delay between press and release
	time.Sleep(10 * time.Millisecond)
	now = time.Now()
	if err := k.Key(now, key, KeyStateReleased); err != nil {
		return err
	}
	// Don't do roundtrip after every key - let the example control this
	return nil
}

// TypeString types a string (basic ASCII support)
func (k *VirtualKeyboard) TypeString(text string) error {
	// Basic key mappings (no shift needed)
	keyMap := map[rune]uint32{
		'a': KEY_A, 'b': KEY_B, 'c': KEY_C, 'd': KEY_D, 'e': KEY_E,
		'f': KEY_F, 'g': KEY_G, 'h': KEY_H, 'i': KEY_I, 'j': KEY_J,
		'k': KEY_K, 'l': KEY_L, 'm': KEY_M, 'n': KEY_N, 'o': KEY_O,
		'p': KEY_P, 'q': KEY_Q, 'r': KEY_R, 's': KEY_S, 't': KEY_T,
		'u': KEY_U, 'v': KEY_V, 'w': KEY_W, 'x': KEY_X, 'y': KEY_Y,
		'z': KEY_Z,
		'0': KEY_0, '1': KEY_1, '2': KEY_2, '3': KEY_3, '4': KEY_4,
		'5': KEY_5, '6': KEY_6, '7': KEY_7, '8': KEY_8, '9': KEY_9,
		' ': KEY_SPACE, '\n': KEY_ENTER, '\t': KEY_TAB,
		'-': KEY_MINUS, '=': KEY_EQUAL, '[': KEY_LEFTBRACE, ']': KEY_RIGHTBRACE,
		';': KEY_SEMICOLON, '\'': KEY_APOSTROPHE, '`': KEY_GRAVE,
		'\\': KEY_BACKSLASH, ',': KEY_COMMA, '.': KEY_DOT, '/': KEY_SLASH,
	}

	// Characters that need shift
	shiftMap := map[rune]uint32{
		'A': KEY_A, 'B': KEY_B, 'C': KEY_C, 'D': KEY_D, 'E': KEY_E,
		'F': KEY_F, 'G': KEY_G, 'H': KEY_H, 'I': KEY_I, 'J': KEY_J,
		'K': KEY_K, 'L': KEY_L, 'M': KEY_M, 'N': KEY_N, 'O': KEY_O,
		'P': KEY_P, 'Q': KEY_Q, 'R': KEY_R, 'S': KEY_S, 'T': KEY_T,
		'U': KEY_U, 'V': KEY_V, 'W': KEY_W, 'X': KEY_X, 'Y': KEY_Y,
		'Z': KEY_Z,
		'!': KEY_1, '@': KEY_2, '#': KEY_3, '$': KEY_4, '%': KEY_5,
		'^': KEY_6, '&': KEY_7, '*': KEY_8, '(': KEY_9, ')': KEY_0,
		'_': KEY_MINUS, '+': KEY_EQUAL, '{': KEY_LEFTBRACE, '}': KEY_RIGHTBRACE,
		':': KEY_SEMICOLON, '"': KEY_APOSTROPHE, '~': KEY_GRAVE,
		'|': KEY_BACKSLASH, '<': KEY_COMMA, '>': KEY_DOT, '?': KEY_SLASH,
	}

	for _, char := range text {
		var key uint32
		var needShift bool

		// Check if it needs shift
		if shiftKey, ok := shiftMap[char]; ok {
			key = shiftKey
			needShift = true
		} else if normalKey, ok := keyMap[char]; ok {
			key = normalKey
			needShift = false
		} else {
			// Skip unsupported characters
			continue
		}

		if needShift {
			if err := k.PressKey(KEY_LEFTSHIFT); err != nil {
				return err
			}
			time.Sleep(5 * time.Millisecond) // Small delay after shift press
		}

		if err := k.TypeKey(key); err != nil {
			return err
		}

		if needShift {
			time.Sleep(5 * time.Millisecond) // Small delay before shift release
			if err := k.ReleaseKey(KEY_LEFTSHIFT); err != nil {
				return err
			}
		}

		// Small delay between characters
		time.Sleep(20 * time.Millisecond)
	}

	return nil
}
