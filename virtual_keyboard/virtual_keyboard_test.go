package virtual_keyboard

import (
	"context"
	"testing"
	"time"
)

func TestNewVirtualKeyboardManager(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualKeyboardManager(ctx)
	if err != nil {
		t.Skipf("Skipping test - virtual keyboard manager not available: %v", err)
	}
	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	// Test manager cleanup
	err = manager.Close()
	if err != nil {
		t.Fatalf("Failed to close manager: %v", err)
	}
}

func TestVirtualKeyboardCreation(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualKeyboardManager(ctx)
	if err != nil {
		t.Skipf("Skipping test - virtual keyboard manager not available: %v", err)
	}
	defer manager.Close()

	// Test creating virtual keyboard
	keyboard, err := manager.CreateKeyboard()
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard: %v", err)
	}
	if keyboard == nil {
		t.Fatal("Keyboard should not be nil")
	}

	// Clean up
	_ = keyboard.Close()
}


func TestVirtualKeyboardKeys(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualKeyboardManager(ctx)
	if err != nil {
		t.Skipf("Skipping test - virtual keyboard manager not available: %v", err)
	}
	defer manager.Close()

	keyboard, err := manager.CreateKeyboard()
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard: %v", err)
	}
	defer func() { _ = keyboard.Close() }()

	// Test key press
	err = keyboard.Key(time.Now(), KEY_A, KeyStatePressed)
	if err != nil {
		t.Fatalf("Failed to press key: %v", err)
	}

	// Test key release
	err = keyboard.Key(time.Now(), KEY_A, KeyStateReleased)
	if err != nil {
		t.Fatalf("Failed to release key: %v", err)
	}

	// Test convenience methods
	err = keyboard.PressKey(KEY_B)
	if err != nil {
		t.Fatalf("Failed to press key with convenience method: %v", err)
	}

	err = keyboard.ReleaseKey(KEY_B)
	if err != nil {
		t.Fatalf("Failed to release key with convenience method: %v", err)
	}

	// Test TypeKey method
	err = keyboard.TypeKey(KEY_C)
	if err != nil {
		t.Fatalf("Failed to type key: %v", err)
	}
}

func TestVirtualKeyboardModifiers(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualKeyboardManager(ctx)
	if err != nil {
		t.Skipf("Skipping test - virtual keyboard manager not available: %v", err)
	}
	defer manager.Close()

	keyboard, err := manager.CreateKeyboard()
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard: %v", err)
	}
	defer func() { _ = keyboard.Close() }()

	// Test modifiers (using bit flags for common modifiers)
	// Shift = 1, Ctrl = 4, Alt = 8, etc (typical X11 modifier masks)
	err = keyboard.Modifiers(1|4, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to set modifiers: %v", err)
	}
}

func TestVirtualKeyboardClose(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualKeyboardManager(ctx)
	if err != nil {
		t.Skipf("Skipping test - virtual keyboard manager not available: %v", err)
	}
	defer manager.Close()

	keyboard, err := manager.CreateKeyboard()
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard: %v", err)
	}

	// Test close
	err = keyboard.Close()
	if err != nil {
		t.Fatalf("Failed to close keyboard: %v", err)
	}
}

func TestTypeString(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualKeyboardManager(ctx)
	if err != nil {
		t.Skipf("Skipping test - virtual keyboard manager not available: %v", err)
	}
	defer manager.Close()

	keyboard, err := manager.CreateKeyboard()
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard: %v", err)
	}
	defer func() { _ = keyboard.Close() }()

	// Test typing a string
	err = keyboard.TypeString("hello world")
	if err != nil {
		t.Fatalf("Failed to type string: %v", err)
	}

	// Test typing string with uppercase
	err = keyboard.TypeString("Hello World")
	if err != nil {
		t.Fatalf("Failed to type string with uppercase: %v", err)
	}
}


func TestKeyConstants(t *testing.T) {
	// Test that key constants are defined and have reasonable values
	keys := []struct {
		key   uint32
		name  string
		value uint32
	}{
		{KEY_A, "KEY_A", 30},
		{KEY_Z, "KEY_Z", 44},
		{KEY_0, "KEY_0", 11},
		{KEY_9, "KEY_9", 10},
		{KEY_SPACE, "KEY_SPACE", 57},
		{KEY_ENTER, "KEY_ENTER", 28},
		{KEY_ESC, "KEY_ESC", 1},
		{KEY_LEFTSHIFT, "KEY_LEFTSHIFT", 42},
		{KEY_LEFTCTRL, "KEY_LEFTCTRL", 29},
		{KEY_LEFTALT, "KEY_LEFTALT", 56},
	}

	for _, test := range keys {
		if test.key != test.value {
			t.Fatalf("%s should be %d, got %d", test.name, test.value, test.key)
		}
	}

	// Test key states
	if KEY_STATE_RELEASED != 0 {
		t.Fatal("KEY_STATE_RELEASED should be 0")
	}
	if KEY_STATE_PRESSED != 1 {
		t.Fatal("KEY_STATE_PRESSED should be 1")
	}
}

func TestKeymapFormatConstants(t *testing.T) {
	if KEYMAP_FORMAT_NO_KEYMAP != 0 {
		t.Fatal("KEYMAP_FORMAT_NO_KEYMAP should be 0")
	}
	if KEYMAP_FORMAT_XKB_V1 != 1 {
		t.Fatal("KEYMAP_FORMAT_XKB_V1 should be 1")
	}
}

func TestKeyStateConstants(t *testing.T) {
	// Test that KeyState enum values match the raw constants
	if uint32(KeyStateReleased) != KEY_STATE_RELEASED {
		t.Fatal("KeyStateReleased should equal KEY_STATE_RELEASED")
	}
	if uint32(KeyStatePressed) != KEY_STATE_PRESSED {
		t.Fatal("KeyStatePressed should equal KEY_STATE_PRESSED")
	}
}