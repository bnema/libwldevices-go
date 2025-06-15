package virtual_keyboard

import (
	"context"
	"testing"
)

func TestNewVirtualKeyboardManager(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualKeyboardManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard manager: %v", err)
	}
	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	// Test manager destruction
	err = manager.Destroy()
	if err != nil {
		t.Fatalf("Failed to destroy manager: %v", err)
	}
}

func TestVirtualKeyboardCreation(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualKeyboardManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard manager: %v", err)
	}
	defer manager.Destroy()

	// Test creating virtual keyboard
	keyboard, err := manager.CreateVirtualKeyboard(nil)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard: %v", err)
	}
	if keyboard == nil {
		t.Fatal("Keyboard should not be nil")
	}

	// Clean up
	keyboard.Destroy()
}

func TestVirtualKeyboardKeymap(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualKeyboardManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard manager: %v", err)
	}
	defer manager.Destroy()

	keyboard, err := manager.CreateVirtualKeyboard(nil)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard: %v", err)
	}
	defer keyboard.Destroy()

	// Test keymap without file descriptor (NO_KEYMAP format)
	err = keyboard.Keymap(KEYMAP_FORMAT_NO_KEYMAP, nil, 0)
	if err != nil {
		t.Fatalf("Failed to set no keymap: %v", err)
	}

	// Test invalid keymap format
	err = keyboard.Keymap(999, nil, 0)
	if err == nil {
		t.Fatal("Expected error for invalid keymap format")
	}

	// Test XKB format without file descriptor should fail
	err = keyboard.Keymap(KEYMAP_FORMAT_XKB_V1, nil, 100)
	if err == nil {
		t.Fatal("Expected error for XKB format without file descriptor")
	}
}

func TestVirtualKeyboardKeys(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualKeyboardManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard manager: %v", err)
	}
	defer manager.Destroy()

	keyboard, err := manager.CreateVirtualKeyboard(nil)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard: %v", err)
	}
	defer keyboard.Destroy()

	// Test key press
	err = keyboard.Key(0, KEY_A, KEY_STATE_PRESSED)
	if err != nil {
		t.Fatalf("Failed to press key: %v", err)
	}

	// Test key release
	err = keyboard.Key(0, KEY_A, KEY_STATE_RELEASED)
	if err != nil {
		t.Fatalf("Failed to release key: %v", err)
	}

	// Test convenience methods
	err = keyboard.KeyPress(KEY_B)
	if err != nil {
		t.Fatalf("Failed to press key with convenience method: %v", err)
	}

	err = keyboard.KeyRelease(KEY_B)
	if err != nil {
		t.Fatalf("Failed to release key with convenience method: %v", err)
	}

	// Test invalid key state
	err = keyboard.Key(0, KEY_A, 999)
	if err == nil {
		t.Fatal("Expected error for invalid key state")
	}
}

func TestVirtualKeyboardModifiers(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualKeyboardManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard manager: %v", err)
	}
	defer manager.Destroy()

	keyboard, err := manager.CreateVirtualKeyboard(nil)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard: %v", err)
	}
	defer keyboard.Destroy()

	// Test modifiers
	err = keyboard.Modifiers(MOD_SHIFT|MOD_CTRL, 0, 0, 0)
	if err != nil {
		t.Fatalf("Failed to set modifiers: %v", err)
	}
}

func TestVirtualKeyboardDestroy(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualKeyboardManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard manager: %v", err)
	}
	defer manager.Destroy()

	keyboard, err := manager.CreateVirtualKeyboard(nil)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard: %v", err)
	}

	// Test destroy
	err = keyboard.Destroy()
	if err != nil {
		t.Fatalf("Failed to destroy keyboard: %v", err)
	}

	// Test operations after destroy should fail
	err = keyboard.Key(0, KEY_A, KEY_STATE_PRESSED)
	if err == nil {
		t.Fatal("Expected error for operation on destroyed keyboard")
	}
}

func TestTypeKey(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualKeyboardManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard manager: %v", err)
	}
	defer manager.Destroy()

	keyboard, err := manager.CreateVirtualKeyboard(nil)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard: %v", err)
	}
	defer keyboard.Destroy()

	// Test typing a key
	err = TypeKey(keyboard, KEY_A)
	if err != nil {
		t.Fatalf("Failed to type key: %v", err)
	}
}

func TestTypeString(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualKeyboardManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard manager: %v", err)
	}
	defer manager.Destroy()

	keyboard, err := manager.CreateVirtualKeyboard(nil)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard: %v", err)
	}
	defer keyboard.Destroy()

	// Test typing a string
	err = TypeString(keyboard, "hello world")
	if err != nil {
		t.Fatalf("Failed to type string: %v", err)
	}

	// Test typing string with special characters
	err = TypeString(keyboard, "Hello, World!")
	if err != nil {
		t.Fatalf("Failed to type string with special characters: %v", err)
	}
}

func TestCharToKey(t *testing.T) {
	// Test basic letters
	key, shift := charToKey('a')
	if key != KEY_A || shift {
		t.Fatalf("Expected key=%d, shift=false for 'a', got key=%d, shift=%t", KEY_A, key, shift)
	}

	key, shift = charToKey('A')
	if key != KEY_A || !shift {
		t.Fatalf("Expected key=%d, shift=true for 'A', got key=%d, shift=%t", KEY_A, key, shift)
	}

	// Test numbers
	key, shift = charToKey('1')
	if key != KEY_1 || shift {
		t.Fatalf("Expected key=%d, shift=false for '1', got key=%d, shift=%t", KEY_1, key, shift)
	}

	key, shift = charToKey('!')
	if key != KEY_1 || !shift {
		t.Fatalf("Expected key=%d, shift=true for '!', got key=%d, shift=%t", KEY_1, key, shift)
	}

	// Test space
	key, shift = charToKey(' ')
	if key != KEY_SPACE || shift {
		t.Fatalf("Expected key=%d, shift=false for space, got key=%d, shift=%t", KEY_SPACE, key, shift)
	}

	// Test unsupported character
	key, shift = charToKey('€')
	if key != 0 || shift {
		t.Fatalf("Expected key=0, shift=false for unsupported character, got key=%d, shift=%t", key, shift)
	}
}

func TestModifierFunctions(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualKeyboardManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard manager: %v", err)
	}
	defer manager.Destroy()

	keyboard, err := manager.CreateVirtualKeyboard(nil)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard: %v", err)
	}
	defer keyboard.Destroy()

	// Test set modifiers
	err = SetModifiers(keyboard, MOD_SHIFT)
	if err != nil {
		t.Fatalf("Failed to set modifiers: %v", err)
	}

	// Test press modifiers
	err = PressModifiers(keyboard, MOD_CTRL|MOD_ALT)
	if err != nil {
		t.Fatalf("Failed to press modifiers: %v", err)
	}

	// Test release modifiers
	err = ReleaseModifiers(keyboard, MOD_CTRL|MOD_ALT)
	if err != nil {
		t.Fatalf("Failed to release modifiers: %v", err)
	}

	// Test key combo
	err = KeyCombo(keyboard, MOD_CTRL, KEY_C)
	if err != nil {
		t.Fatalf("Failed to perform key combo: %v", err)
	}
}

func TestVirtualKeyboardError(t *testing.T) {
	err := &VirtualKeyboardError{
		Code:    ERROR_NO_KEYMAP,
		Message: "test error",
	}

	expected := "virtual keyboard error 0: test error"
	if err.Error() != expected {
		t.Fatalf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestKeyConstants(t *testing.T) {
	// Test that key constants are defined and have reasonable values
	keys := []struct {
		key   uint32
		name  string
		min   uint32
		max   uint32
	}{
		{KEY_A, "KEY_A", 1, 255},
		{KEY_Z, "KEY_Z", 1, 255},
		{KEY_0, "KEY_0", 1, 255},
		{KEY_9, "KEY_9", 1, 255},
		{KEY_SPACE, "KEY_SPACE", 1, 255},
		{KEY_ENTER, "KEY_ENTER", 1, 255},
		{KEY_ESC, "KEY_ESC", 1, 255},
		{KEY_LEFTSHIFT, "KEY_LEFTSHIFT", 1, 255},
		{KEY_LEFTCTRL, "KEY_LEFTCTRL", 1, 255},
		{KEY_LEFTALT, "KEY_LEFTALT", 1, 255},
	}

	for _, test := range keys {
		if test.key < test.min || test.key > test.max {
			t.Fatalf("%s (%d) should be between %d and %d", test.name, test.key, test.min, test.max)
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

func TestModifierConstants(t *testing.T) {
	// Test that modifier constants are powers of 2 (bit flags)
	modifiers := []uint32{MOD_SHIFT, MOD_CAPS, MOD_CTRL, MOD_ALT, MOD_NUM, MOD_MOD3, MOD_LOGO, MOD_MOD5}
	
	for i, mod := range modifiers {
		expected := uint32(1 << i)
		if mod != expected {
			t.Fatalf("Modifier %d should be %d, got %d", i, expected, mod)
		}
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

func TestGetCurrentTime(t *testing.T) {
	// Test that getCurrentTime returns a uint32
	timestamp := getCurrentTime()
	_ = timestamp // Just make sure it compiles and returns something
}

func TestDestroyedManagerOperations(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualKeyboardManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual keyboard manager: %v", err)
	}

	// Destroy the manager
	err = manager.Destroy()
	if err != nil {
		t.Fatalf("Failed to destroy manager: %v", err)
	}

	// Operations on destroyed manager should fail
	_, err = manager.CreateVirtualKeyboard(nil)
	if err == nil {
		t.Fatal("Expected error for creating keyboard on destroyed manager")
	}

	// Second destroy should fail
	err = manager.Destroy()
	if err == nil {
		t.Fatal("Expected error for destroying already destroyed manager")
	}
}