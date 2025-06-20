// Package main demonstrates how to use the virtual_keyboard package to simulate keyboard input.
//
// This example shows how to:
// - Create a virtual keyboard manager
// - Create a virtual keyboard
// - Type text and perform keyboard operations
// - Handle modifiers and key combinations
// - Clean up resources properly
//
// Note: This is a demonstration of the API. In a real Wayland environment,
// you would need actual Wayland client library bindings.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bnema/libwldevices-go/virtual_keyboard"
)

// Additional key constants not defined in the package
const (
	// Function keys
	KEY_F1  = 59
	KEY_F2  = 60
	KEY_F3  = 61
	KEY_F4  = 62
	KEY_F5  = 63
	KEY_F6  = 64
	KEY_F7  = 65
	KEY_F8  = 66
	KEY_F9  = 67
	KEY_F10 = 68
	KEY_F11 = 87
	KEY_F12 = 88

	// Navigation keys
	KEY_UP       = 103
	KEY_DOWN     = 108
	KEY_LEFT     = 105
	KEY_RIGHT    = 106
	KEY_HOME     = 102
	KEY_END      = 107
	KEY_PAGEUP   = 104
	KEY_PAGEDOWN = 109
)

func main() {
	fmt.Println("Virtual Keyboard Example - Keyboard Input Simulation")
	fmt.Println("====================================================")

	// Create a context for the application
	ctx := context.Background()

	// Create a virtual keyboard manager
	fmt.Println("1. Creating virtual keyboard manager...")
	manager, err := virtual_keyboard.NewVirtualKeyboardManager(ctx)
	if err != nil {
		log.Fatalf("Failed to create virtual keyboard manager: %v", err)
	}
	defer func() {
		fmt.Println("9. Closing virtual keyboard manager...")
		if err := manager.Close(); err != nil {
			log.Printf("Error closing manager: %v", err)
		}
	}()

	// Create a virtual keyboard
	fmt.Println("2. Creating virtual keyboard...")
	keyboard, err := manager.CreateKeyboard()
	if err != nil {
		log.Printf("Failed to create virtual keyboard: %v", err)
		return
	}
	defer func() {
		fmt.Println("8. Closing virtual keyboard...")
		if err := keyboard.Close(); err != nil {
			log.Printf("Error closing keyboard: %v", err)
		}
	}()

	// The default keymap is set automatically during keyboard creation
	fmt.Println("3. Default keymap already configured")

	// Demonstrate basic key typing
	fmt.Println("4. Typing individual keys...")
	keys := []struct {
		key  uint32
		desc string
	}{
		{virtual_keyboard.KEY_H, "H"},
		{virtual_keyboard.KEY_E, "e"},
		{virtual_keyboard.KEY_L, "l"},
		{virtual_keyboard.KEY_L, "l"},
		{virtual_keyboard.KEY_O, "o"},
		{virtual_keyboard.KEY_SPACE, "Space"},
		{virtual_keyboard.KEY_W, "W"},
		{virtual_keyboard.KEY_O, "o"},
		{virtual_keyboard.KEY_R, "r"},
		{virtual_keyboard.KEY_L, "l"},
		{virtual_keyboard.KEY_D, "d"},
	}

	for _, key := range keys {
		fmt.Printf("   - Typing: %s\n", key.desc)
		if err := keyboard.TypeKey(key.key); err != nil {
			log.Printf("Error typing key %s: %v", key.desc, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("   - Pressing Enter")
	if err := keyboard.TypeKey(virtual_keyboard.KEY_ENTER); err != nil {
		log.Printf("Error typing Enter: %v", err)
	}

	// Demonstrate string typing
	fmt.Println("5. Typing strings...")
	strings := []string{
		"Hello, Wayland!",
		"This is a test of virtual keyboard input.",
		"Special characters: !@#$%^&*()",
		"Numbers: 1234567890",
		"Mixed case: AbCdEfGhIjKlMnOpQrStUvWxYz",
	}

	for _, str := range strings {
		fmt.Printf("   - Typing string: \"%s\"\n", str)
		if err := keyboard.TypeString(str); err != nil {
			log.Printf("Error typing string: %v", err)
		}
		// Press Enter after each string
		if err := keyboard.TypeKey(virtual_keyboard.KEY_ENTER); err != nil {
			log.Printf("Error typing Enter: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Demonstrate modifier keys
	fmt.Println("6. Testing modifier keys...")
	modifierTests := []struct {
		modifier uint32
		key      uint32
		desc     string
	}{
		{virtual_keyboard.KEY_LEFTCTRL, virtual_keyboard.KEY_C, "Ctrl+C (Copy)"},
		{virtual_keyboard.KEY_LEFTCTRL, virtual_keyboard.KEY_V, "Ctrl+V (Paste)"},
		{virtual_keyboard.KEY_LEFTCTRL, virtual_keyboard.KEY_Z, "Ctrl+Z (Undo)"},
		{virtual_keyboard.KEY_LEFTCTRL, virtual_keyboard.KEY_S, "Ctrl+S (Save)"},
		{virtual_keyboard.KEY_LEFTALT, virtual_keyboard.KEY_TAB, "Alt+Tab (Switch)"},
		{virtual_keyboard.KEY_LEFTCTRL, virtual_keyboard.KEY_Z, "Ctrl+Shift+Z (Redo) - Note: Shift not simulated in this example"},
	}

	for _, test := range modifierTests {
		fmt.Printf("   - Key combination: %s\n", test.desc)
		// Press modifier
		if err := keyboard.PressKey(test.modifier); err != nil {
			log.Printf("Error pressing modifier: %v", err)
			continue
		}
		// Press and release key
		if err := keyboard.TypeKey(test.key); err != nil {
			log.Printf("Error typing key: %v", err)
		}
		// Release modifier
		if err := keyboard.ReleaseKey(test.modifier); err != nil {
			log.Printf("Error releasing modifier: %v", err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// Demonstrate function keys
	fmt.Println("7. Testing function keys...")
	functionKeys := []struct {
		key  uint32
		desc string
	}{
		{KEY_F1, "F1"},
		{KEY_F2, "F2"},
		{KEY_F5, "F5 (Refresh)"},
		{KEY_F11, "F11 (Fullscreen)"},
		{KEY_F12, "F12"},
	}

	for _, fkey := range functionKeys {
		fmt.Printf("   - Function key: %s\n", fkey.desc)
		if err := keyboard.TypeKey(fkey.key); err != nil {
			log.Printf("Error typing function key: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Demonstrate arrow keys and navigation
	fmt.Println("   - Arrow keys and navigation...")
	navKeys := []struct {
		key  uint32
		desc string
	}{
		{KEY_UP, "Up Arrow"},
		{KEY_DOWN, "Down Arrow"},
		{KEY_LEFT, "Left Arrow"},
		{KEY_RIGHT, "Right Arrow"},
		{KEY_HOME, "Home"},
		{KEY_END, "End"},
		{KEY_PAGEUP, "Page Up"},
		{KEY_PAGEDOWN, "Page Down"},
	}

	for _, navKey := range navKeys {
		fmt.Printf("   - Navigation key: %s\n", navKey.desc)
		if err := keyboard.TypeKey(navKey.key); err != nil {
			log.Printf("Error typing navigation key: %v", err)
		}
		time.Sleep(150 * time.Millisecond)
	}

	// Demonstrate more complex operations
	fmt.Println("7. Performing complex keyboard operations...")

	// Simulate typing a paragraph with proper formatting
	fmt.Println("   - Typing formatted text with tabs and newlines")
	formattedText := []struct {
		action func() error
		desc   string
	}{
		{func() error { return keyboard.TypeString("Dear User,") }, "Type greeting"},
		{func() error { return keyboard.TypeKey(virtual_keyboard.KEY_ENTER) }, "New line"},
		{func() error { return keyboard.TypeKey(virtual_keyboard.KEY_ENTER) }, "Blank line"},
		{func() error { return keyboard.TypeKey(virtual_keyboard.KEY_TAB) }, "Tab indent"},
		{func() error { return keyboard.TypeString("This is a demonstration of virtual keyboard input.") }, "Type paragraph"},
		{func() error { return keyboard.TypeKey(virtual_keyboard.KEY_ENTER) }, "New line"},
		{func() error { return keyboard.TypeKey(virtual_keyboard.KEY_TAB) }, "Tab indent"},
		{func() error { return keyboard.TypeString("The virtual keyboard can simulate complex typing patterns.") }, "Type second paragraph"},
		{func() error { return keyboard.TypeKey(virtual_keyboard.KEY_ENTER) }, "New line"},
		{func() error { return keyboard.TypeKey(virtual_keyboard.KEY_ENTER) }, "Blank line"},
		{func() error { return keyboard.TypeString("Best regards,") }, "Type closing"},
		{func() error { return keyboard.TypeKey(virtual_keyboard.KEY_ENTER) }, "New line"},
		{func() error { return keyboard.TypeString("Virtual Keyboard Example") }, "Type signature"},
	}

	for _, action := range formattedText {
		fmt.Printf("     - %s\n", action.desc)
		if err := action.action(); err != nil {
			log.Printf("Error in formatted text action: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Demonstrate rapid typing
	fmt.Println("   - Rapid typing test")
	rapidText := "The quick brown fox jumps over the lazy dog. "
	for i := 0; i < 3; i++ {
		if err := keyboard.TypeString(rapidText); err != nil {
			log.Printf("Error in rapid typing: %v", err)
		}
	}

	// Demonstrate modifier state management
	fmt.Println("   - Testing modifier state management")
	if err := demonstrateModifierStates(keyboard); err != nil {
		log.Printf("Error in modifier state demo: %v", err)
	}

	fmt.Println("\nExample completed!")
}

// demonstrateModifierStates shows how to manage modifier key states
func demonstrateModifierStates(keyboard *virtual_keyboard.VirtualKeyboard) error {
	fmt.Println("     - Pressing and holding Shift")
	if err := keyboard.PressKey(virtual_keyboard.KEY_LEFTSHIFT); err != nil {
		return fmt.Errorf("failed to press shift: %v", err)
	}

	// Type some letters while shift is held
	shiftedText := "UPPERCASE TEXT"
	for _, char := range shiftedText {
		if char == ' ' {
			if err := keyboard.TypeKey(virtual_keyboard.KEY_SPACE); err != nil {
				return fmt.Errorf("failed to type space: %v", err)
			}
		} else if char >= 'A' && char <= 'Z' {
			key := virtual_keyboard.KEY_A + uint32(char - 'A')
			if err := keyboard.TypeKey(key); err != nil {
				return fmt.Errorf("failed to type character: %v", err)
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Println("     - Releasing Shift")
	if err := keyboard.ReleaseKey(virtual_keyboard.KEY_LEFTSHIFT); err != nil {
		return fmt.Errorf("failed to release shift: %v", err)
	}

	// Type some text without shift
	if err := keyboard.TypeString(" lowercase text"); err != nil {
		return fmt.Errorf("failed to type lowercase: %v", err)
	}

	fmt.Println("     - Testing Caps Lock")
	if err := keyboard.TypeKey(virtual_keyboard.KEY_CAPSLOCK); err != nil {
		return fmt.Errorf("failed to press caps lock: %v", err)
	}

	if err := keyboard.TypeString(" CAPS LOCK TEXT "); err != nil {
		return fmt.Errorf("failed to type caps lock text: %v", err)
	}

	// Turn off caps lock
	if err := keyboard.TypeKey(virtual_keyboard.KEY_CAPSLOCK); err != nil {
		return fmt.Errorf("failed to release caps lock: %v", err)
	}

	if err := keyboard.TypeString("normal text again"); err != nil {
		return fmt.Errorf("failed to type normal text: %v", err)
	}

	return nil
}

// demonstrateAdvancedFeatures shows more advanced virtual keyboard features
func demonstrateAdvancedFeatures(_ *virtual_keyboard.VirtualKeyboard) {
	fmt.Println("Advanced Keyboard Features:")

	// Demonstrate setting modifier state directly
	fmt.Println("   - Setting modifier states directly")
	// Note: The current API doesn't have MOD_* constants or SetModifiers function
	// This section would need to be implemented differently
	fmt.Println("     Note: Modifier state management would require manual key presses")

	// This functionality is not available in the current API

	// Demonstrate keypad input
	fmt.Println("   - Numeric keypad input")
	// Note: Keypad keys would need to be defined in the virtual_keyboard constants
	fmt.Println("     Note: Keypad keys demonstration skipped - constants not defined")

	// This functionality would need the keypad constants to be defined
}